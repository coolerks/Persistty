"""不创建远端服务的探针判定回归。"""

import importlib.util
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("probe", Path(__file__).with_name("probe.py"))
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)


class ProbeTests(unittest.TestCase):
    def test_optimized_execution_rejected_before_resources(self):
        result = subprocess.run([sys.executable, "-O", str(Path(probe.__file__)),
                                 "start", "/tmp/not-a-probe"], capture_output=True, timeout=5)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("禁止优化模式".encode(), result.stderr)

    def test_command_output_limit(self):
        with self.assertRaisesRegex(RuntimeError, "1 MiB"):
            probe.command([sys.executable, "-c", "print('x' * (1024 * 1024 + 1))"])

    def test_snapshot_rejects_stale_since_last_sample(self):
        server = {"pid": 101, "start_ticks": 1, "cgroup": "own.service"}
        pane = {"pid": 102, "start_ticks": 2, "cgroup": "own.service"}
        with tempfile.TemporaryDirectory() as root:
            Path(root, "heartbeat").write_text("9")
            state = {"root": root, "baseline": {"server": server, "pane": pane,
                     "heartbeat": 1}, "samples": [{"heartbeat": 9}]}
            reply = subprocess.CompletedProcess([], 0, "101|102|100x30|%0", "")
            with patch.object(probe, "tmux", return_value=reply), patch.object(
                    probe, "identity", side_effect=[server, pane]):
                with self.assertRaisesRegex(AssertionError, "自上次采样"):
                    probe.snapshot(state, "stopped")


if __name__ == "__main__":
    unittest.main()
