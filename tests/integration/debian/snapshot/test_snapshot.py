"""Snapshot runner regressions; no remote connection."""
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import Mock, patch

BASE = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(BASE))
import run_remote


class SnapshotTests(unittest.TestCase):
    def test_snapshot_dispatch_selects_separate_analyzer(self):
        remote = Mock()
        binary = Path("/tmp/bridge-probe")
        with patch.object(run_remote, "run_history", return_value={"ok": True}) as history:
            self.assertEqual(run_remote.run_snapshot(remote, binary), {"ok": True})
            history.assert_called_once_with(remote, binary, analysis_kind="snapshot")

    def test_invalid_analysis_before_transport(self):
        remote = Mock()
        with self.assertRaises(run_remote.TransportError):
            run_remote.run_history(remote, Path("/tmp/bridge-probe"), analysis_kind="foreign")
        remote.ssh.assert_not_called()

    def test_cache_failure_before_allocate(self):
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory, "bridge-probe")
            binary.touch()
            remote = Mock()
            with patch.object(Path, "mkdir", side_effect=PermissionError):
                with self.assertRaises(PermissionError):
                    run_remote.run_snapshot(remote, binary)
            remote.ssh.assert_not_called()
            remote.scp.assert_not_called()

    def test_summary_only_evidence(self):
        path = Path(__file__).with_name("evidence.json")
        if not path.exists(): self.fail("real evidence missing")
        evidence = json.loads(path.read_text())
        text = json.dumps(evidence)
        for forbidden in ('"capture"', '"serialized"', '"cells"'):
            self.assertNotIn(forbidden, text)
        self.assertTrue(all(evidence["checks"].values()))
        self.assertEqual(evidence["analysis"]["real_single_connection"]["record"], "tui120")
        self.assertTrue(evidence["cleanup"]["root_removed"])
        self.assertTrue(all(item["active"] == "inactive" for item in evidence["cleanup"]["units"]))
        samples = evidence["facts"]["samples"]
        for previous, current in zip(samples, samples[1:]):
            self.assertGreater(current["heartbeat"], previous["heartbeat"])
            self.assertEqual(current["input_count"], 0)
            for role in ("server", "pane"):
                self.assertEqual(current[role], samples[0][role])


if __name__ == "__main__": unittest.main()
