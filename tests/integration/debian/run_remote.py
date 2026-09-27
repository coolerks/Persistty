"""显式授权后使用 .env 运行隔离探针；只输出脱敏结构化结果。"""

import argparse
import json
from pathlib import Path
import re
import shlex
import sys
import time

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


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("kind", choices=("terminal", "files"))
    parser.add_argument("--probe", type=Path, default=Path("/tmp/persistty-files-probe"))
    parser.add_argument("--test", type=Path, default=Path("/tmp/persistty-files-test"))
    args = parser.parse_args()
    try:
        connection = read_config()
        remote = Remote(connection)
        result = (run_terminal(remote) if args.kind == "terminal"
                  else run_files(remote, args.probe.resolve(), args.test.resolve()))
        print(json.dumps({"redaction": {"kind": "redacted_observation",
                          "connection_values_stored": False},
                          **redact(result, connection)}, ensure_ascii=False, indent=2))
    except (ConfigError, TransportError) as error:
        print(str(error), file=sys.stderr)
        return 1
    except Exception:
        print("Debian 实验结果解析或执行失败；仅核查本次实验资源", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
