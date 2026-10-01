"""验收 runner 的连接前拒绝与失败清理回归；不连接远端。"""

import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import types
import unittest
from unittest.mock import patch

SOURCE = Path(__file__).with_name("run.py")
SPEC = importlib.util.spec_from_file_location("w06_run", SOURCE)
runner = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(runner)


class RunnerTests(unittest.TestCase):
    def test_optimization_and_root_rejected_before_side_effects(self):
        result = subprocess.run([sys.executable, "-O", str(SOURCE), "--binaries", "/missing"],
                                capture_output=True, text=True, timeout=5)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("优化模式", result.stderr)
        with self.assertRaises(RuntimeError):
            runner.probe(Path("/tmp/foreign"))

    def test_upload_failure_still_cleans_exact_root_without_error_body(self):
        self.check_failure(False)

    def test_cleanup_failure_keeps_only_validated_recovery_identity(self):
        self.check_failure(True)

    def check_failure(self, cleanup_fails):
        calls = []
        class Remote:
            def __init__(self, connection):
                pass
            def ssh(self, command):
                calls.append(command)
                if len(calls) == 1:
                    return "/tmp/persistty-w06-accept-ABCDEFGH\n"
                if cleanup_fails:
                    raise RuntimeError("private-transmission-error")
                return ""
            def scp(self, sources, root):
                raise RuntimeError("private-transmission-error")
        module = types.SimpleNamespace(Remote=Remote, ROOT=SOURCE.parents[4],
                                       TransportError=RuntimeError, read_config=lambda: object())
        with tempfile.TemporaryDirectory() as directory:
            for package in runner.PACKAGES:
                Path(directory, f"w06-{package}.test").write_bytes(b"dummy")
            with patch.dict(sys.modules, {"remote_config": module}):
                result = runner.remote_run(Path(directory))
        self.assertEqual(len(calls), 2)
        self.assertEqual(calls[1], "rm -rf -- /tmp/persistty-w06-accept-ABCDEFGH; test ! -e /tmp/persistty-w06-accept-ABCDEFGH")
        self.assertFalse(result["passed"])
        self.assertEqual(result["cleanup_verified"], not cleanup_fails)
        self.assertNotIn("private-transmission-error", json.dumps(result))
        if cleanup_fails:
            self.assertEqual(result["recovery_root"], "/tmp/persistty-w06-accept-ABCDEFGH")


if __name__ == "__main__":
    unittest.main()
