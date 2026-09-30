"""通过本地 loopback 暂时转发隔离浏览器探针，不回显 SSH 身份。"""

import os
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from remote_config import Remote, read_config  # noqa: E402


def main():
    if len(sys.argv) != 2 or not sys.argv[1].isdecimal():
        raise SystemExit(2)
    port = int(sys.argv[1])
    if not 1 <= port <= 65535:
        raise SystemExit(2)
    connection = read_config()
    args = ["ssh", *Remote(connection).options(), "-o", "ExitOnForwardFailure=yes",
            "-N", "-L", f"127.0.0.1:18080:127.0.0.1:{port}",
            "-p", str(connection.port), "-l", connection.user, connection.ip]
    null = os.open(os.devnull, os.O_WRONLY)
    os.dup2(null, 2)
    os.close(null)
    os.execvp("ssh", args)


if __name__ == "__main__":
    main()
