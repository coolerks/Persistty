"""本地编排判定回归；不连接目标机。"""

import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location("bridge_probe", Path(__file__).with_name("probe.py"))
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)


class ProbeTests(unittest.TestCase):
    def test_optimization_rejected_before_resources(self):
        result = subprocess.run([sys.executable, "-O", str(Path(probe.__file__)), "start", "/tmp/not-own"],
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("禁止优化模式", result.stderr.decode())

    def test_command_failure_does_not_echo_child_error(self):
        with self.assertRaisesRegex(RuntimeError, "实验子命令失败") as error:
            probe.command([sys.executable, "-c", "import sys; print('private',file=sys.stderr); sys.exit(2)"])
        self.assertNotIn("private", str(error.exception))

    def test_output_limit(self):
        with self.assertRaisesRegex(RuntimeError, "输出超限"):
            probe.command([sys.executable, "-c", "print('x' * (1024 * 1024 + 1))"])

    def test_stalled_heartbeat_rejected_even_above_baseline(self):
        with tempfile.TemporaryDirectory() as directory:
            Path(directory, "heartbeat").write_text("5")
            Path(directory, "input_count").write_text("1")
            server = {"pid": 10, "start_ticks": 1, "cgroup": "server.service"}
            pane = {"pid": 20, "start_ticks": 2, "cgroup": "tmux-spawn-test.scope"}
            state = {"root": directory, "session": "probe", "server_unit": "server.service", "web_unit": "web.service",
                     "baseline": {"server": server, "pane": pane, "heartbeat": 1},
                     "samples": [{"heartbeat": 5}]}
            with patch.object(probe, "tmux", return_value=(0, "10|20|%0")), \
                    patch.object(probe, "identity", side_effect=[server, pane]), \
                    self.assertRaisesRegex(AssertionError, "心跳未逐次增长"):
                probe.snapshot(state, "stalled")

    def test_cleanup_rejects_foreign_unit_before_stop(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "state.json").write_text(json.dumps({"root": str(root),
                "session": "persistty_probe_0123456789ab", "client_unit": "business.service"}))
            with patch.object(probe, "command") as command, self.assertRaises(AssertionError):
                probe.cleanup(root)
            command.assert_not_called()

    def test_evidence_identity_progress_and_cleanup(self):
        for name in ("evidence.json", "evidence-review.json"):
            with self.subTest(evidence=name):
                evidence = json.loads(Path(__file__).with_name(name).read_text())
                self.assertFalse(evidence["redaction"]["connection_values_stored"])
                samples = evidence["new_ssh_connection"]["samples"]
                self.assertEqual(len(samples), 13)
                for previous, current in zip(samples, samples[1:]):
                    self.assertGreater(current["heartbeat"], previous["heartbeat"])
                    for role in ("server", "pane"):
                        self.assertEqual(current[role], samples[0][role])
                    self.assertEqual(current["input_count"], 1)
                self.assertTrue(evidence["cleanup"]["root_removed"])
                self.assertTrue(all(unit["active"] == "inactive" for unit in evidence["cleanup"]["units"]))
                self.assertTrue(all(value is True for value in evidence["start"]["exercise"]["checks"].values()))


if __name__ == "__main__":
    unittest.main()
