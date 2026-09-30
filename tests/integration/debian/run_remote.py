"""显式授权后使用 .env 运行隔离探针；只输出脱敏结构化结果。"""

import argparse
import json
from pathlib import Path
import re
import shlex
import sys
import time
import os
import subprocess
import tempfile

from remote_config import ConfigError, Remote, ROOT, TransportError, read_config, redact


def run_terminal(remote):
    root = remote.ssh("umask 077; mktemp -d /tmp/persistty-terminal-XXXXXXXX").strip()
    if not re.fullmatch(r"/tmp/persistty-terminal-[A-Za-z0-9]{8}", root):
        raise TransportError("Debian 临时目录身份非法")
    started = False
    result = {}
    try:
        remote.scp([ROOT / "tests/integration/debian/terminal/probe.py"], root)
        started = True
        result["start"] = json.loads(remote.ssh(f"python3 {shlex.quote(root + '/probe.py')} start {shlex.quote(root)}"))
        time.sleep(1)
        result["new_ssh_connection"] = json.loads(remote.ssh(
            f"python3 {shlex.quote(root + '/probe.py')} check {shlex.quote(root)}"))
    finally:
        if started:
            # start 可能已自行清理；只对本次明确目录执行 cleanup。
            result["cleanup"] = json.loads(remote.ssh(
                f"if test -f {shlex.quote(root + '/state.json')}; then "
                f"python3 {shlex.quote(root + '/probe.py')} cleanup {shlex.quote(root)}; "
                f"elif test -d {shlex.quote(root)}; then "
                f"rm -f -- {shlex.quote(root + '/probe.py')}; rmdir -- {shlex.quote(root)} && "
                "printf '{\"unstarted_root_removed\":true}'; "
                "else printf '{\"already_removed\":true}'; fi"))
        else:
            remote.ssh(f"rm -f -- {shlex.quote(root + '/probe.py')} ; rmdir -- {shlex.quote(root)}")
    return result


def run_files(remote, probe, test):
    if not probe.is_file() or not test.is_file():
        raise TransportError("Debian 本地探针文件缺失")
    root = remote.ssh("umask 077; mktemp -d /tmp/persistty-files-upload.XXXXXXXX").strip()
    if not re.fullmatch(r"/tmp/persistty-files-upload\.[A-Za-z0-9]{8}", root):
        raise TransportError("Debian 临时目录身份非法")
    result = {}
    try:
        remote.scp([probe, test], root)
        result["probe"] = json.loads(remote.ssh(
            f"TMPDIR={shlex.quote(root)} timeout 20s {shlex.quote(root + '/' + probe.name)}"))
        # 测试正文仅用于判断退出码，不进入报告。
        remote.ssh(f"TMPDIR={shlex.quote(root)} timeout 25s {shlex.quote(root + '/' + test.name)} -test.timeout=20s")
        result["tests_passed"] = True
    finally:
        remote.ssh(f"rm -rf -- {shlex.quote(root)}; test ! -e {shlex.quote(root)}")
    result["cleanup_verified"] = True
    return result


def run_bridge(remote, binary):
    if not binary.is_file() or binary.name != "bridge-probe":
        raise TransportError("Debian bridge 本地二进制缺失或名称非法")
    root = remote.ssh("umask 077; mktemp -d /tmp/persistty-bridge-XXXXXXXX").strip()
    if not re.fullmatch(r"/tmp/persistty-bridge-[A-Za-z0-9]{8}", root):
        raise TransportError("Debian 临时目录身份非法")
    result = {}
    try:
        remote.scp([ROOT / "tests/integration/debian/bridge/probe.py", binary], root)
        remote.ssh(f"chmod 700 -- {shlex.quote(root + '/bridge-probe')}")
        result["start"] = json.loads(remote.ssh(
            f"python3 -B {shlex.quote(root + '/probe.py')} start {shlex.quote(root)}"))
        if result["start"].get("probe_failed"):
            raise TransportError("Debian bridge 实验失败阶段 " + result["start"]["stage"])
        time.sleep(1)
        result["new_ssh_connection"] = json.loads(remote.ssh(
            f"python3 -B {shlex.quote(root + '/probe.py')} check {shlex.quote(root)}"))
        if result["new_ssh_connection"].get("probe_failed"):
            raise TransportError("Debian bridge 实验失败阶段 " + result["new_ssh_connection"]["stage"])
    finally:
        try:
            result["cleanup"] = json.loads(remote.ssh(
                f"if test -f {shlex.quote(root + '/state.json')}; then "
                f"python3 -B {shlex.quote(root + '/probe.py')} cleanup {shlex.quote(root)}; "
                f"else rm -f -- {shlex.quote(root + '/probe.py')} {shlex.quote(root + '/bridge-probe')}; "
                f"rmdir -- {shlex.quote(root)} && printf '{{\"unstarted_root_removed\":true}}'; fi"))
            if result["cleanup"].get("probe_failed"):
                raise TransportError("Debian bridge 清理判定失败")
        except Exception:
            # 随机自有 ROOT 不是连接身份；失败时保留精确恢复入口。
            raise TransportError("Debian bridge 清理未验证；仅核查自身实验目录 " + root) from None
    return result


def run_history(remote, binary, *, analysis_kind="history"):
    if analysis_kind not in ("history", "snapshot"):
        raise TransportError("Debian 解析种类非法")
    if not binary.is_file() or binary.name != "bridge-probe":
        raise TransportError("Debian history 本地二进制缺失或名称非法")
    cache = ROOT / ".cache"
    cache.mkdir(exist_ok=True)
    root = remote.ssh("umask 077; mktemp -d /tmp/persistty-history-XXXXXXXX").strip()
    if not re.fullmatch(r"/tmp/persistty-history-[A-Za-z0-9]{8}", root):
        raise TransportError("Debian history 临时目录身份非法")
    result = {}
    try:
        remote.scp([binary], root)
        # 复用审查过的隔离与精确清理 helper；名称不覆盖 D06 probe。
        remote.scp([ROOT / "tests/integration/debian/bridge/probe.py"], root + "/")
        remote.ssh(f"mv -- {shlex.quote(root + '/probe.py')} {shlex.quote(root + '/bridge_support.py')}")
        remote.scp([ROOT / "tests/integration/debian/history/probe.py"], root)
        remote.ssh(f"chmod 700 -- {shlex.quote(root + '/bridge-probe')}")
        raw = json.loads(remote.ssh(f"python3 -B {shlex.quote(root + '/probe.py')} start {shlex.quote(root)}"))
        if raw.get("probe_failed"):
            raise TransportError("Debian history 实验失败阶段 " + raw["stage"])
        fd, path = tempfile.mkstemp(prefix="history-",suffix=".json",dir=cache)
        try:
            with os.fdopen(fd,"w") as file: json.dump({"records":raw["records"],"capture":raw["capture"]},file)
            process = subprocess.run(["node", str(ROOT / ("tests/integration/debian/" + analysis_kind + "/analyze.mjs")),path],
                stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,timeout=45)
            if process.returncode or len(process.stdout)>65536:
                raise TransportError("Debian history 解析失败")
            parsed = json.loads(process.stdout)
            if not all(parsed["checks"].values()):
                raise TransportError("Debian history 解析断言失败")
            result = {"facts":raw["facts"],**parsed}
        finally:
            Path(path).unlink(missing_ok=True)
    finally:
        try:
            result["cleanup"] = json.loads(remote.ssh(
                f"if test -f {shlex.quote(root + '/state.json')}; then "
                f"python3 -B {shlex.quote(root + '/probe.py')} cleanup {shlex.quote(root)}; "
                f"else rm -f -- {shlex.quote(root + '/probe.py')} {shlex.quote(root + '/bridge_support.py')} "
                f"{shlex.quote(root + '/bridge-probe')}; rmdir -- {shlex.quote(root)} && "
                "printf '{\"unstarted_root_removed\":true}'; fi"))
            if result["cleanup"].get("probe_failed"): raise TransportError("清理失败")
        except Exception:
            raise TransportError("Debian history 清理未验证；仅核查自身实验目录 " + root) from None
    return result


def run_snapshot(remote, binary):
    # Reuse a new isolated history run, selecting one connection in a separate analyzer.
    return run_history(remote, binary, analysis_kind="snapshot")


def run_recovery(remote, binary=None):
    root = remote.ssh("umask 077; mktemp -d /tmp/persistty-recovery-XXXXXXXX").strip()
    if not re.fullmatch(r"/tmp/persistty-recovery-[A-Za-z0-9]{8}", root):
        raise TransportError("Debian recovery 临时目录身份非法")
    result = {}
    try:
        remote.scp([ROOT / "tests/integration/debian/bridge/probe.py"], root)
        remote.ssh(f"mv -- {shlex.quote(root + '/probe.py')} {shlex.quote(root + '/bridge_support.py')}")
        remote.scp([ROOT / "tests/integration/debian/recovery/probe.py"], root)
        if binary is not None:
            if not binary.is_file() or binary.name != "runtime-probe":
                raise TransportError("Debian runtime probe 二进制缺失或名称非法")
            remote.scp([binary], root)
            remote.ssh(f"chmod 700 -- {shlex.quote(root + '/runtime-probe')}")
        result = json.loads(remote.ssh(
            f"python3 -B {shlex.quote(root + '/probe.py')} start {shlex.quote(root)}"))
        if result.get("probe_failed"):
            raise TransportError("Debian recovery 实验失败阶段 " + result["stage"]
                                 + " 类别 " + result.get("error_type", "unknown"))
        cache = ROOT / ".cache"
        cache.mkdir(exist_ok=True)
        fd, path = tempfile.mkstemp(prefix="recovery-", suffix=".json", dir=cache)
        try:
            with os.fdopen(fd, "w") as file:
                json.dump(result.pop("records"), file)
            process = subprocess.run(
                ["node", str(ROOT / "tests/integration/debian/recovery/analyze.mjs"), path],
                stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, timeout=45)
            if process.returncode or len(process.stdout) > 65536:
                raise TransportError("Debian recovery xterm 解析失败")
            analyzed = json.loads(process.stdout)
            result["xterm"] = analyzed["summary"]
            result["checks"].update(analyzed["checks"])
            if binary is not None:
                product = json.loads(remote.ssh(
                    f"LD_LIBRARY_PATH={shlex.quote(root + '/package/usr/lib/x86_64-linux-gnu')} "
                    f"timeout 30s {shlex.quote(root + '/runtime-probe')} "
                    f"{shlex.quote(root)} {shlex.quote(root + '/package/usr/bin/tmux')} "
                    f"{shlex.quote(root + '/tmux.sock')}"))
                if product.get("failed_stage") or not product.get("checks"):
                    raise TransportError("Debian 产品探针失败阶段 " + product.get("failed_stage", "unknown"))
                result["checks"].update({"product_" + key: value for key, value in product["checks"].items()})
            if not all(result["checks"].values()):
                failed = ",".join(key for key, value in result["checks"].items() if not value)
                raise TransportError("Debian recovery 断言失败: " + failed
                                     + " command_statuses=" + str(result["facts"].get("product_command_statuses")))
        finally:
            Path(path).unlink(missing_ok=True)
    finally:
        try:
            result["cleanup"] = json.loads(remote.ssh(
                f"if test -f {shlex.quote(root + '/state.json')}; then "
                f"python3 -B {shlex.quote(root + '/probe.py')} cleanup {shlex.quote(root)}; "
                f"else rm -f -- {shlex.quote(root + '/probe.py')} {shlex.quote(root + '/bridge_support.py')} "
                f"{shlex.quote(root + '/runtime-probe')}; "
                f"rmdir -- {shlex.quote(root)} && printf '{{\"unstarted_root_removed\":true}}'; fi"))
            if result["cleanup"].get("probe_failed"):
                raise TransportError("Debian recovery 清理失败")
        except Exception:
            raise TransportError("Debian recovery 清理未验证；仅核查自身实验目录 " + root) from None
    return result


def run_browser_start(remote, binary):
    if not binary.is_file() or binary.name != "runtime-probe":
        raise TransportError("Debian browser 本地二进制缺失或名称非法")
    root = remote.ssh("umask 077; mktemp -d /tmp/persistty-browser-XXXXXXXX").strip()
    if not re.fullmatch(r"/tmp/persistty-browser-[A-Za-z0-9]{8}", root):
        raise TransportError("Debian browser 临时目录身份非法")
    try:
        remote.scp([ROOT / "tests/integration/debian/bridge/probe.py", binary], root)
        remote.ssh(f"mv -- {shlex.quote(root + '/probe.py')} {shlex.quote(root + '/support.py')}")
        remote.scp([ROOT / "tests/integration/debian/browser/probe.py", ROOT / "tests/integration/debian/browser/tui.py"], root)
        remote.ssh(f"chmod 700 -- {shlex.quote(root + '/runtime-probe')}")
        result = json.loads(remote.ssh(
            f"python3 -B {shlex.quote(root + '/probe.py')} start {shlex.quote(root)}"))
        if result.get("failed_stage"):
            raise TransportError("Debian 浏览器启动探针失败阶段 " + result["failed_stage"])
        return result
    except Exception:
        try:
            cleanup_output = remote.ssh(f"if test -f {shlex.quote(root + '/state.json')}; then "
                       f"python3 -B {shlex.quote(root + '/probe.py')} cleanup {shlex.quote(root)}; "
                       f"else rm -rf -- {shlex.quote(root)}; fi")
            if cleanup_output.strip() and json.loads(cleanup_output).get("failed_stage"):
                raise TransportError("清理探针未通过")
        except Exception:
            raise TransportError("Debian browser 启动失败且清理未验证；仅核查本次实验目录 " + root) from None
        raise


def run_browser_cleanup(remote, root):
    if not re.fullmatch(r"/tmp/persistty-browser-[A-Za-z0-9]{8}", root):
        raise TransportError("Debian browser 清理目录身份非法")
    result = json.loads(remote.ssh(
        f"python3 -B {shlex.quote(root + '/probe.py')} cleanup {shlex.quote(root)}"))
    if result.get("failed_stage"):
        raise TransportError("Debian 浏览器清理探针失败阶段 " + result["failed_stage"])
    return result


def run_browser_layout(remote, root):
    if not re.fullmatch(r"/tmp/persistty-browser-[A-Za-z0-9]{8}", root):
        raise TransportError("Debian browser fixture 目录身份非法")
    result = json.loads(remote.ssh(
        f"python3 -B {shlex.quote(root + '/probe.py')} layout-fixture {shlex.quote(root)}"))
    if result.get("failed_stage"):
        raise TransportError("Debian 布局 fixture 失败阶段 " + result["failed_stage"])
    return result


def run_browser_restart(remote, root, force=False):
    if not re.fullmatch(r"/tmp/persistty-browser-[A-Za-z0-9]{8}", root):
        raise TransportError("Debian browser 重启目录身份非法")
    result = json.loads(remote.ssh(
        f"python3 -B {shlex.quote(root + '/probe.py')} {'force-restart' if force else 'restart'} {shlex.quote(root)}"))
    if result.get("failed_stage"):
        raise TransportError("Debian 浏览器重启探针失败阶段 " + result["failed_stage"]
                             + "（" + result.get("error_kind", "unknown") + "）")
    return result


def run_cli(remote, binary):
    if not binary.is_file() or binary.name != "cli-probe":
        raise TransportError("Debian CLI 本地二进制缺失或名称非法")
    root = remote.ssh("umask 077; mktemp -d /tmp/persistty-cli-XXXXXXXX").strip()
    if not re.fullmatch(r"/tmp/persistty-cli-[A-Za-z0-9]{8}", root):
        raise TransportError("Debian CLI 临时目录身份非法")
    result = {}
    try:
        remote.scp([binary, ROOT / "tests/integration/debian/cli/probe.py"], root)
        remote.ssh(f"chmod 700 -- {shlex.quote(root + '/cli-probe')}")
        raw = remote.ssh(f"python3 -B {shlex.quote(root + '/probe.py')} {shlex.quote(root)}")
        result = json.loads(raw)
        if result.get("probe_failed") or len(result.get("checks", {})) != 9 or not all(result["checks"].values()):
            checks = result.get("checks", {})
            failed = ",".join(key for key, passed in checks.items() if not passed)
            raise TransportError("Debian CLI 隔离实验未通过: " + (failed or result.get("stage", "unknown"))
                                 + " statuses=" + json.dumps(result.get("statuses", {}), sort_keys=True)
                                 + " git_errors=" + json.dumps(result.get("git_errors", {}), sort_keys=True))
    finally:
        # Only the newly returned, validated private mktemp root is removed.
        remote.ssh(f"rm -rf -- {shlex.quote(root)}; test ! -e {shlex.quote(root)}")
    result["cleanup_verified"] = True
    return result


def run_helper_capabilities(remote):
    # Read-only existence checks; no sudo invocation or policy file contents.
    source = ("import json,os; "
              "print(json.dumps({'sudo_binary':os.path.isfile('/usr/bin/sudo'),"
              "'pam_service_entry':os.path.isfile('/etc/pam.d/sudo'),"
              "'sudoedit_binary':os.path.isfile('/usr/bin/sudoedit')}))")
    return json.loads(remote.ssh("python3 -B -c " + shlex.quote(source)))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("kind", choices=("terminal", "files", "bridge", "history", "snapshot", "recovery", "browser-start", "browser-restart", "browser-force-restart", "browser-cleanup", "browser-layout-fixture", "cli", "helper"))
    parser.add_argument("--probe", type=Path, default=Path("/tmp/persistty-files-probe"))
    parser.add_argument("--test", type=Path, default=Path("/tmp/persistty-files-test"))
    parser.add_argument("--binary", type=Path, default=Path("/tmp/bridge-probe"))
    parser.add_argument("--root", default="")
    args = parser.parse_args()
    try:
        connection = read_config()
        remote = Remote(connection)
        if args.kind == "terminal":
            result = run_terminal(remote)
        elif args.kind == "bridge":
            result = run_bridge(remote, args.binary.resolve())
        elif args.kind == "history":
            result = run_history(remote, args.binary.resolve())
        elif args.kind == "snapshot":
            result = run_snapshot(remote, args.binary.resolve())
        elif args.kind == "recovery":
            result = run_recovery(remote, args.binary.resolve() if args.binary.name == "runtime-probe" else None)
        elif args.kind == "browser-start":
            result = run_browser_start(remote, args.binary.resolve())
        elif args.kind == "browser-cleanup":
            result = run_browser_cleanup(remote, args.root)
        elif args.kind == "browser-layout-fixture":
            result = run_browser_layout(remote, args.root)
        elif args.kind == "browser-restart":
            result = run_browser_restart(remote, args.root)
        elif args.kind == "browser-force-restart":
            result = run_browser_restart(remote, args.root, force=True)
        elif args.kind == "cli":
            result = run_cli(remote, args.binary.resolve())
        elif args.kind == "helper":
            result = run_helper_capabilities(remote)
        else:
            result = run_files(remote, args.probe.resolve(), args.test.resolve())
        print(json.dumps({"redaction": {"kind": "redacted_observation",
                          "connection_values_stored": False},
                          **redact(result, connection)}, ensure_ascii=False, indent=2))
    except (ConfigError, TransportError) as error:
        print(str(error), file=sys.stderr)
        return 1
    except Exception as error:
        print("Debian 实验结果解析或执行失败（" + type(error).__name__
              + "）；仅核查本次实验资源", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
