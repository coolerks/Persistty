"""Local regressions; no remote connection."""
import importlib.util
import json
from pathlib import Path
import sys
import subprocess
import tempfile
import unittest
from unittest.mock import Mock, patch

BASE = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(BASE))
import run_remote

bridge_spec = importlib.util.spec_from_file_location("bridge_support", BASE / "bridge/probe.py")
bridge = importlib.util.module_from_spec(bridge_spec)
sys.modules["bridge_support"] = bridge
bridge_spec.loader.exec_module(bridge)
spec = importlib.util.spec_from_file_location("history_probe", Path(__file__).with_name("probe.py"))
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)


class HistoryTests(unittest.TestCase):
    def test_cache_failure_has_no_remote_allocation(self):
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory, "bridge-probe")
            binary.touch()
            remote = Mock()
            with patch.object(Path, "mkdir", side_effect=PermissionError):
                with self.assertRaises(PermissionError):
                    run_remote.run_history(remote, binary)
            remote.ssh.assert_not_called()
            remote.scp.assert_not_called()

    def test_optimized_modes_rejected_before_root_access(self):
        for mode in ("-O", "-OO"):
            code = "import sys; sys.path.insert(0," + repr(str(Path(__file__).parent)) + "); import test_history; test_history.probe.main()"
            result = subprocess.run([sys.executable, mode, "-c", code,
                "cleanup", "/tmp/foreign-directory"], stdout=subprocess.PIPE,
                stderr=subprocess.PIPE)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("禁止优化模式", result.stderr.decode())
            self.assertNotIn("FileNotFoundError", result.stderr.decode())

    def test_invalid_binary_has_no_transport(self):
        remote = Mock()
        with self.assertRaises(run_remote.TransportError):
            run_remote.run_history(remote, Path("/no/such/bridge-probe"))
        remote.ssh.assert_not_called()

    def test_atomic_phase_has_no_partial_file(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            probe.atomic(root,"phase","tui")
            self.assertEqual((root / "phase").read_text(),"tui")
            self.assertFalse((root / "phase.tmp").exists())

    def test_foreign_root_is_rejected_before_upload(self):
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory,"bridge-probe")
            binary.touch()
            remote = Mock()
            remote.ssh.return_value = "/tmp/foreign-directory"
            with self.assertRaises(run_remote.TransportError):
                run_remote.run_history(remote,binary)
            remote.scp.assert_not_called()

    def test_saved_evidence_is_summary_only_and_stable(self):
        evidence = json.loads(Path(__file__).with_name("evidence.json").read_text())
        self.assertNotIn('"frames"',json.dumps(evidence))
        self.assertNotIn('"capture"',json.dumps(evidence))
        self.assertFalse(evidence["redaction"]["connection_values_stored"])
        self.assertTrue(all(evidence["checks"].values()))
        samples = evidence["facts"]["samples"]
        for previous,current in zip(samples,samples[1:]):
            self.assertGreater(current["heartbeat"],previous["heartbeat"])
            self.assertEqual(current["input_count"],0)
            for role in ("server","pane"):
                self.assertEqual(current[role],samples[0][role])
        self.assertTrue(evidence["cleanup"]["root_removed"])
        self.assertTrue(all(unit["active"]=="inactive" for unit in evidence["cleanup"]["units"]))


if __name__ == "__main__": unittest.main()
