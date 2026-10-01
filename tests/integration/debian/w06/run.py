"""W05/W06 真实 Debian 隔离测试；连接参数与原始测试输出不回显。"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import selectors
import shlex
import signal
import stat
import subprocess
import sys
import time

PACKAGES = ("files", "search", "gitview", "toolrunner", "httpapi")
ROOT_PATTERN = r"/tmp/persistty-w06-accept-[A-Za-z0-9]{8}"


def capture(args, cwd, env, seconds=40):
    with subprocess.Popen(args, cwd=cwd, env=env, stdout=subprocess.PIPE,
                          stderr=subprocess.STDOUT, start_new_session=True) as process:
        try:
            data = bytearray()
            deadline = time.monotonic() + seconds
            with selectors.DefaultSelector() as selector:
                selector.register(process.stdout, selectors.EVENT_READ)
                while True:
                    remaining = deadline - time.monotonic()
                    if remaining <= 0 or not selector.select(remaining):
                        raise RuntimeError("output_timeout")
                    chunk = os.read(process.stdout.fileno(), 65536)
                    if not chunk:
                        break
                    data.extend(chunk)
                    if len(data) > 2 * 1024 * 1024:
                        raise RuntimeError("output_limit")
            return process.wait(timeout=max(0.01, deadline - time.monotonic())), data.decode("utf-8")
        finally:
            if process.poll() is None:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()


def probe(root):
    if not re.fullmatch(ROOT_PATTERN, str(root)):
        raise RuntimeError("root_identity")
    info = root.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o700:
        raise RuntimeError("private_root")
    env = {"PATH": "/usr/bin:/bin", "HOME": "/nonexistent", "TMPDIR": str(root),
           "GIT_CONFIG_GLOBAL": "/dev/null", "GIT_CONFIG_NOSYSTEM": "1"}
    versions = {}
    for tool, pattern in (("git", r"git version ([0-9.]+)"), ("rg", r"ripgrep ([0-9.]+)")):
        code, output = capture([tool, "--version"], root, env, 5)
        match = re.match(pattern, output)
        if code or not match:
            raise RuntimeError("required_tool")
        versions[tool] = match.group(1)
    (root / "internal/httpapi").mkdir(parents=True, mode=0o700)
    (root / "tests/fixtures").mkdir(parents=True, mode=0o700)
    (root / "search-git.json").rename(root / "tests/fixtures/search-git.json")
    results = {}
    for package in PACKAGES:
        binary = root / f"w06-{package}.test"
        info = binary.lstat()
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid():
            raise RuntimeError("binary_identity")
        binary.chmod(0o700)
        args = [str(binary), "-test.v", "-test.timeout=35s"]
        if package == "httpapi":
            args += ["-test.run=Test(SearchGit|SearchReplace|PreviewProtection|SaveBody)"]
        code, output = capture(args, root / "internal/httpapi" if package == "httpapi" else root, env)
        passed = re.findall(r"^--- PASS: (Test[A-Za-z0-9_]+)", output, re.MULTILINE)
        skipped = re.findall(r"^\s*--- SKIP: (Test[A-Za-z0-9_/]+)", output, re.MULTILINE)
        failed = re.findall(r"^\s*--- FAIL: (Test[A-Za-z0-9_/]+)", output, re.MULTILINE)
        results[package] = {"passed": passed, "skipped": skipped, "failed": failed,
                            "exit_code": code, "sha256": hashlib.sha256(binary.read_bytes()).hexdigest()}
    return {"redacted": True, "versions": versions, "packages": results,
            "passed": all(v["exit_code"] == 0 and v["passed"] and not v["skipped"] for v in results.values())}


def remote_run(directory):
    sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
    from remote_config import Remote, ROOT, TransportError, read_config
    binaries = [directory / f"w06-{package}.test" for package in PACKAGES]
    if not all(p.is_file() and not p.is_symlink() for p in binaries):
        raise RuntimeError("local_binaries")
    remote = Remote(read_config())
    root = remote.ssh("umask 077; mktemp -d /tmp/persistty-w06-accept-XXXXXXXX").strip()
    if not re.fullmatch(ROOT_PATTERN, root):
        raise TransportError("测试目录身份非法")
    result = None
    try:
        remote.scp([Path(__file__).resolve(), *binaries, ROOT / "tests/fixtures/search-git.json"], root)
        result = json.loads(remote.ssh(f"python3 -B {shlex.quote(root + '/run.py')} --probe {shlex.quote(root)}"))
    except Exception as error:
        result = {"passed": False, "error_kind": type(error).__name__}
    finally:
        try:
            remote.ssh(f"rm -rf -- {shlex.quote(root)}; test ! -e {shlex.quote(root)}")
            result["cleanup_verified"] = True
        except Exception:
            result = {"passed": False, "cleanup_verified": False, "recovery_root": root,
                      "error_kind": "cleanup_transport"}
    return result


def main():
    if sys.flags.optimize:
        raise SystemExit("验收探针不允许 Python 优化模式")
    parser = argparse.ArgumentParser(description=__doc__)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--binaries", type=Path)
    group.add_argument("--probe", type=Path)
    args = parser.parse_args()
    try:
        result = probe(args.probe) if args.probe else remote_run(args.binaries)
    except Exception as error:
        # 固定类别；不输出外部异常正文、SSH argv 或原始测试日志。
        result = {"passed": False, "error_kind": type(error).__name__}
    print(json.dumps(result, ensure_ascii=False, sort_keys=True))
    return 0 if result.get("passed") else 1


if __name__ == "__main__":
    raise SystemExit(main())
