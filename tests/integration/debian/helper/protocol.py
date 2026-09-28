"""Nonprivileged, synthetic single-file authorization protocol experiment."""

from dataclasses import dataclass
import hashlib
import os
from pathlib import Path
import stat


def fingerprint_fd(fd):
    identity = os.fstat(fd)
    if not stat.S_ISREG(identity.st_mode):
        raise ValueError("not a regular file")
    digest = hashlib.sha256()
    os.lseek(fd, 0, os.SEEK_SET)
    while chunk := os.read(fd, 65536):
        digest.update(chunk)
    return identity.st_dev, identity.st_ino, identity.st_size, identity.st_mtime_ns, digest.digest()


def fingerprint(path):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC)
    try:
        return fingerprint_fd(fd)
    finally:
        os.close(fd)


@dataclass(frozen=True)
class Grant:
    nonce: str
    file: Path
    version: tuple


class Simulation:
    def __init__(self):
        self._grants = {}
        self._seen = set()

    def authorize(self, nonce, file):
        if not nonce or nonce in self._seen:
            raise ValueError("invalid nonce")
        file = Path(file).absolute()
        grant = Grant(nonce, file, fingerprint(file))
        self._seen.add(nonce)
        self._grants[nonce] = grant
        return grant

    def save(self, grant, content, password_fd, *, cancelled=False):
        # Consume before any fallible operation so failure never extends a grant.
        current = self._grants.pop(grant.nonce, None)
        if current != grant:
            raise PermissionError("grant spent or changed")
        if cancelled:
            raise InterruptedError("cancelled")
        password = bytearray(os.read(password_fd, 256))
        try:
            if password != b"synthetic-only":
                raise PermissionError("credential rejected")
        finally:
            password[:] = b"\0" * len(password)
        # This nonprivileged experiment writes only caller-owned temporary files.
        fd = os.open(grant.file, os.O_RDWR | os.O_NOFOLLOW | os.O_CLOEXEC)
        try:
            if fingerprint_fd(fd) != grant.version:
                raise FileExistsError("file changed or replaced")
            os.ftruncate(fd, 0)
            os.lseek(fd, 0, os.SEEK_SET)
            os.write(fd, content)
            os.fsync(fd)
        finally:
            os.close(fd)
