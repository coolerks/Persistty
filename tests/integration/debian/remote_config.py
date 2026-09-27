"""私密连接参数只读仓库根 .env；不执行该文件，不回显参数。"""

from dataclasses import dataclass, field
import ipaddress
import os
from pathlib import Path
import re
import selectors
import stat
import subprocess
import time


ROOT = Path(__file__).resolve().parents[3]
KEYS = frozenset(("DEBIAN_USER", "DEBIAN_IP", "DEBIAN_PORT"))
ASSIGNMENT = re.compile(
    r"\s*([A-Z_][A-Z0-9_]*)\s*=\s*(?:\"([^\"\\]*)\"|'([^'\\]*)'|([^\s#]*))\s*(?:#.*)?"
)


class ConfigError(Exception):
    """固定类别，不含值、原始行或路径。"""


class TransportError(Exception):
    """固定类别，不包含传输错误正文或 argv。"""


@dataclass(frozen=True, repr=False)
class Connection:
    user: str = field(repr=False)
    ip: str = field(repr=False)
    port: int = field(repr=False)

    def __repr__(self):
        return "Connection(<private>)"


def parse_config(text):
    values = {}
    try:
        if any(ord(character) < 32 and character not in "\r\n\t" for character in text):
            raise ValueError
        for line in text.splitlines():
            if not line.strip() or line.lstrip().startswith("#"):
                continue
            match = ASSIGNMENT.fullmatch(line)
            if not match:
                raise ValueError
            key, *alternatives = match.groups()
            if key not in KEYS or key in values:
                raise ValueError
            values[key] = next(value for value in alternatives if value is not None)
            if "$" in values[key] or "`" in values[key]:
                raise ValueError
        if set(values) != KEYS:
            raise ValueError
        user, host, port = (values[key] for key in ("DEBIAN_USER", "DEBIAN_IP", "DEBIAN_PORT"))
        if not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_-]{0,31}", user):
            raise ValueError
        ipaddress.ip_address(host)
        if "%" in host or not re.fullmatch(r"[0-9]{1,5}", port) or not 1 <= int(port) <= 65535:
            raise ValueError
        return Connection(user, host, int(port))
    except (ValueError, StopIteration):
        raise ConfigError("Debian 连接配置格式或字段非法") from None


def read_config(repo_root=ROOT):
    try:
        fd = os.open(Path(repo_root) / ".env", os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
        with os.fdopen(fd, "rb") as source:
            metadata = os.fstat(source.fileno())
            if (not stat.S_ISREG(metadata.st_mode) or metadata.st_uid != os.getuid()
                    or stat.S_IMODE(metadata.st_mode) != 0o600 or metadata.st_size > 4096):
                raise ValueError
            raw = source.read(4097)
            if len(raw) > 4096:
                raise ValueError
        return parse_config(raw.decode("utf-8"))
    except (OSError, ValueError, UnicodeError):
        raise ConfigError("Debian 私密连接文件缺失或不安全") from None


def bounded_command(args):
    """只在内存捕获 stdout；stderr 不落盘、不回显。"""
    try:
        with subprocess.Popen(args, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL) as process:
            try:
                output = bytearray()
                deadline = time.monotonic() + 60
                with selectors.DefaultSelector() as selector:
                    selector.register(process.stdout, selectors.EVENT_READ)
                    while True:
                        remaining = deadline - time.monotonic()
                        if remaining <= 0 or not selector.select(remaining):
                            raise TransportError("Debian 传输超时")
                        chunk = os.read(process.stdout.fileno(), 65536)
                        if not chunk:
                            break
                        output.extend(chunk)
                        if len(output) > 1024 * 1024:
                            raise TransportError("Debian 输出超限")
                if process.wait(timeout=max(0.01, deadline - time.monotonic())):
                    raise TransportError("Debian 传输失败")
                return output.decode("utf-8")
            finally:
                if process.poll() is None:
                    process.kill()
                    process.wait()
    except (OSError, UnicodeError, subprocess.TimeoutExpired):
        raise TransportError("Debian 传输失败") from None


class Remote:
    def __init__(self, connection):
        self.connection = connection

    def options(self):
        return ["-F", "/dev/null", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes",
                "-o", "ConnectTimeout=10", "-o", "ControlPath=none", "-o", "ControlMaster=no"]

    def ssh(self, command):
        c = self.connection
        return bounded_command(["ssh", *self.options(), "-p", str(c.port),
                                "-l", c.user, c.ip, command])

    def scp(self, sources, directory):
        c = self.connection
        return bounded_command(["scp", *self.options(), "-P", str(c.port), "--",
                                *(str(path) for path in sources), f"{c.user}@[{c.ip}]:{directory}/"])


def redact(value, connection):
    if isinstance(value, dict):
        # 仅连接字段按端口身份去敏，不能改写恰好同值的 PID 或计数。
        port_fields = {"port", "debian_port", "ssh_port", "remote_port"}
        return {key: ("<remote-port>" if key.lower() in port_fields
                      and str(item) == str(connection.port) else redact(item, connection))
                for key, item in value.items()}
    if isinstance(value, list):
        return [redact(item, connection) for item in value]
    if isinstance(value, str):
        value = re.sub(r"user-\d+|user@\d+", lambda m: m.group(0).split(
            "-" if "-" in m.group(0) else "@")[0] + (
            "-" if "-" in m.group(0) else "@") + "<remote-uid>", value)
        value = re.sub(r"/run/user/\d+", "/run/user/<remote-uid>", value)
        value = re.sub(r"/home/[^/\s\"']+", "/home/<remote-user>", value)
        return value.replace(connection.user, "<remote-user>").replace(connection.ip, "<remote-ip>")
    return value
