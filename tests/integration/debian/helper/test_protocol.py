import os
from pathlib import Path
import tempfile
import unittest

from protocol import Grant, Simulation


def credential(text=b"synthetic-only"):
    reader, writer = os.pipe()
    os.write(writer, text)
    os.close(writer)
    return reader


class ProtocolTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="persistty-helper-")
        self.addCleanup(self.temp.cleanup)
        self.file = Path(self.temp.name) / "sample.txt"
        self.file.write_bytes(b"initial")
        self.protocol = Simulation()

    def save(self, grant, *, cancelled=False, password=b"synthetic-only"):
        fd = credential(password)
        try:
            self.protocol.save(grant, b"changed", fd, cancelled=cancelled)
        finally:
            os.close(fd)

    def test_one_file_one_use(self):
        grant = self.protocol.authorize("one", self.file)
        self.save(grant)
        self.assertEqual(self.file.read_bytes(), b"changed")
        with self.assertRaises(PermissionError):
            self.save(grant)
        with self.assertRaises(ValueError):
            self.protocol.authorize("one", self.file)

    def test_changed_content_rejected_and_grant_consumed(self):
        grant = self.protocol.authorize("two", self.file)
        self.file.write_bytes(b"external")
        with self.assertRaises(FileExistsError):
            self.save(grant)
        with self.assertRaises(PermissionError):
            self.save(grant)
        self.assertEqual(self.file.read_bytes(), b"external")

    def test_symlink_replacement_rejected(self):
        other = Path(self.temp.name) / "other.txt"
        other.write_bytes(b"outside")
        grant = self.protocol.authorize("three", self.file)
        self.file.unlink()
        self.file.symlink_to(other)
        with self.assertRaises(OSError):
            self.save(grant)
        self.assertEqual(other.read_bytes(), b"outside")

    def test_cancel_and_bad_credential_consume_grant(self):
        cancelled = self.protocol.authorize("four", self.file)
        with self.assertRaises(InterruptedError):
            self.save(cancelled, cancelled=True)
        with self.assertRaises(PermissionError):
            self.save(cancelled)
        bad = self.protocol.authorize("five", self.file)
        with self.assertRaises(PermissionError):
            self.save(bad, password=b"incorrect")
        with self.assertRaises(PermissionError):
            self.save(bad)
        self.assertEqual(self.file.read_bytes(), b"initial")

    def test_nonce_cannot_be_reissued_or_redirected(self):
        grant = self.protocol.authorize("six", self.file)
        with self.assertRaises(ValueError):
            self.protocol.authorize("six", self.file)
        other = Path(self.temp.name) / "other.txt"
        other.write_bytes(b"other")
        forged = Grant(grant.nonce, other, grant.version)
        with self.assertRaises(PermissionError):
            self.save(forged)
        with self.assertRaises(PermissionError):
            self.save(grant)
        self.assertEqual(other.read_bytes(), b"other")


if __name__ == "__main__":
    unittest.main()
