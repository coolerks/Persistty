"""只验证私密文件解析、模拟传输和脱敏，不建立真实连接。"""

import contextlib
import io
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import Mock, patch


DEBIAN = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(DEBIAN))
import remote_config as config
import run_remote as runner


GOOD = "DEBIAN_USER=probe_user\nDEBIAN_IP=203.0.113.10\nDEBIAN_PORT=2222\n"


class RemoteConfigTests(unittest.TestCase):
    def test_literal_quotes_whitespace_comments_ipv6(self):
        c = config.parse_config("# fictional $ignored `comment`\n DEBIAN_USER = 'probe_user' # note\n"
                                'DEBIAN_IP="2001:db8::10"\nDEBIAN_PORT = 2222\n')
        self.assertEqual((c.user, c.ip, c.port), ("probe_user", "2001:db8::10", 2222))
        self.assertEqual(repr(c), "Connection(<private>)")

    def test_invalid_input_errors_never_echo_input(self):
        invalid = ["", GOOD + "DEBIAN_PORT=2222\n", GOOD + "OTHER=value\n",
                   GOOD.replace("probe_user", "$(touch marker)"),
                   GOOD.replace("probe_user", "`id`"), GOOD.replace("probe_user", "$USER"),
                   GOOD.replace("probe_user", "-unsafe"),
                   GOOD.replace("203.0.113.10", "999.0.0.1"),
                   GOOD.replace("203.0.113.10", "fe80::1%zone"),
                   GOOD.replace("2222", "0"), GOOD.replace("2222", "65536"),
                   GOOD.replace("2222", '"2222'), GOOD.replace("2222", "22;false"),
                   GOOD.replace("probe_user", "probe_user\x01")]
        for index, text in enumerate(invalid):
            with self.subTest(case=index), self.assertRaises(config.ConfigError) as error:
                config.parse_config(text)
            self.assertEqual(str(error.exception), "Debian 连接配置格式或字段非法")

    def test_only_owner_private_file_not_process_environment(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory, ".env")
            path.write_text(GOOD)
            path.chmod(0o600)
            owner = os.getuid()
            with patch.object(config.os, "getuid", return_value=owner + 1):
                with self.assertRaises(config.ConfigError):
                    config.read_config(directory)
            with patch.dict(os.environ, {"DEBIAN_USER": "hostile", "DEBIAN_IP": "192.0.2.20",
                                        "DEBIAN_PORT": "9999"}):
                self.assertEqual(config.read_config(directory).user, "probe_user")
            path.chmod(0o644)
            with self.assertRaises(config.ConfigError):
                config.read_config(directory)
            path.chmod(0o600)
            path.write_text("#" * 4097)
            with self.assertRaises(config.ConfigError):
                config.read_config(directory)
            path.unlink()
            with self.assertRaises(config.ConfigError):
                config.read_config(directory)
            source = Path(directory, "source")
            source.write_text(GOOD)
            source.chmod(0o600)
            path.symlink_to(source)
            with self.assertRaises(config.ConfigError):
                config.read_config(directory)
            path.unlink()
            os.mkfifo(path, 0o600)
            with self.assertRaises(config.ConfigError):
                config.read_config(directory)

    def test_mock_transport_exact_arguments_and_ipv6(self):
        remote = config.Remote(config.Connection("probe_user", "2001:db8::10", 2222))
        with patch.object(config, "bounded_command", return_value="") as command:
            remote.ssh("printf fixed")
            args = command.call_args.args[0]
            self.assertEqual(args[-6:], ["-p", "2222", "-l", "probe_user", "2001:db8::10", "printf fixed"])
            self.assertEqual(args[:3], ["ssh", "-F", "/dev/null"])
            remote.scp([Path("/tmp/fake-probe")], "/tmp/own-root")
            self.assertEqual(command.call_args.args[0][-1], "probe_user@[2001:db8::10]:/tmp/own-root/")

    def test_local_transport_failure_is_redacted_and_bounded(self):
        with self.assertRaises(config.TransportError) as error:
            config.bounded_command([sys.executable, "-c",
                                    "import sys; print('private failure',file=sys.stderr); sys.exit(1)"])
        self.assertEqual(str(error.exception), "Debian 传输失败")
        with self.assertRaises(config.TransportError) as error:
            config.bounded_command([sys.executable, "-c", "print('x' * (1024 * 1024 + 1))"])
        self.assertEqual(str(error.exception), "Debian 输出超限")

    def test_runner_cleanup_on_failed_terminal_check(self):
        remote = Mock()
        remote.ssh.side_effect = ["/tmp/persistty-terminal-ABCDEFGH\n", "{}",
                                  config.TransportError("Debian 传输失败"), '{"cleanup":true}']
        with patch.object(runner.time, "sleep"), self.assertRaises(config.TransportError):
            runner.run_terminal(remote)
        self.assertIn("cleanup", remote.ssh.call_args.args[0])
        self.assertIn("/tmp/persistty-terminal-ABCDEFGH", remote.ssh.call_args.args[0])

    def test_start_transport_failure_cleans_uploaded_unstarted_root(self):
        remote = Mock()
        remote.ssh.side_effect = ["/tmp/persistty-terminal-ABCDEFGH\n",
                                  config.TransportError("Debian 传输失败"),
                                  '{"unstarted_root_removed":true}']
        with self.assertRaises(config.TransportError):
            runner.run_terminal(remote)
        cleanup = remote.ssh.call_args.args[0]
        self.assertIn("test -f /tmp/persistty-terminal-ABCDEFGH/state.json", cleanup)
        self.assertIn("rm -f -- /tmp/persistty-terminal-ABCDEFGH/probe.py", cleanup)
        self.assertIn("rmdir -- /tmp/persistty-terminal-ABCDEFGH", cleanup)

    def test_local_transport_timeout_kills_only_child(self):
        with patch.object(config.time, "monotonic", side_effect=[0, 61]), \
                self.assertRaises(config.TransportError) as error:
            config.bounded_command([sys.executable, "-c", "import time; time.sleep(5)"])
        self.assertEqual(str(error.exception), "Debian 传输超时")

    def test_files_failure_cleans_exact_new_directory(self):
        remote = Mock()
        remote.ssh.side_effect = ["/tmp/persistty-files-upload.ABCDEFGH\n",
                                  config.TransportError("Debian 传输失败"), ""]
        with tempfile.TemporaryDirectory() as directory:
            probe, test = Path(directory, "fake-probe"), Path(directory, "fake-test")
            probe.touch()
            test.touch()
            with self.assertRaises(config.TransportError):
                runner.run_files(remote, probe, test)
        self.assertEqual(remote.ssh.call_args.args[0],
                         "rm -rf -- /tmp/persistty-files-upload.ABCDEFGH; test ! -e /tmp/persistty-files-upload.ABCDEFGH")

    def test_bad_remote_directory_never_uploads_or_cleans(self):
        remote = Mock()
        remote.ssh.return_value = "/tmp/not-owned\n"
        with self.assertRaises(config.TransportError):
            runner.run_terminal(remote)
        remote.scp.assert_not_called()
        self.assertEqual(remote.ssh.call_count, 1)

    def test_missing_file_fails_before_transport(self):
        with patch.object(runner, "read_config", side_effect=config.ConfigError("固定错误")), \
                patch.object(runner, "Remote") as remote, \
                patch.object(sys, "argv", ["run_remote.py", "terminal"]), \
                contextlib.redirect_stdout(io.StringIO()) as stdout, \
                contextlib.redirect_stderr(io.StringIO()) as stderr:
            self.assertEqual(runner.main(), 1)
            self.assertEqual(stdout.getvalue(), "")
            self.assertEqual(stderr.getvalue(), "固定错误\n")
            remote.assert_not_called()

    def test_bridge_failure_cleans_only_new_root(self):
        remote = Mock()
        remote.ssh.side_effect = ["/tmp/persistty-bridge-ABCDEFGH\n", "",
                                  config.TransportError("Debian 传输失败"), '{"root_removed":true}']
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory, "bridge-probe")
            binary.touch()
            with self.assertRaises(config.TransportError):
                runner.run_bridge(remote, binary)
        cleanup = remote.ssh.call_args.args[0]
        self.assertIn("/tmp/persistty-bridge-ABCDEFGH/state.json", cleanup)
        self.assertIn("cleanup /tmp/persistty-bridge-ABCDEFGH", cleanup)

    def test_bridge_bad_root_or_binary_has_no_upload(self):
        remote = Mock()
        with self.assertRaises(config.TransportError):
            runner.run_bridge(remote, Path("/nonexistent/bridge-probe"))
        remote.ssh.assert_not_called()
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory, "bridge-probe")
            binary.touch()
            remote.ssh.return_value = "/tmp/not-own"
            with self.assertRaises(config.TransportError):
                runner.run_bridge(remote, binary)
        remote.scp.assert_not_called()

    def test_bridge_cleanup_failure_records_only_exact_own_root(self):
        remote = Mock()
        remote.ssh.side_effect = ["/tmp/persistty-bridge-ABCDEFGH\n", "", "{}", "{}",
                                  config.TransportError("Debian 传输失败")]
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory, "bridge-probe")
            binary.touch()
            with patch.object(runner.time, "sleep"), self.assertRaises(config.TransportError) as error:
                runner.run_bridge(remote, binary)
        self.assertEqual(str(error.exception),
                         "Debian bridge 清理未验证；仅核查自身实验目录 /tmp/persistty-bridge-ABCDEFGH")

    def test_missing_field_cannot_use_process_environment(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory, ".env")
            path.write_text("DEBIAN_USER=probe_user\nDEBIAN_IP=203.0.113.10\n")
            path.chmod(0o600)
            with patch.dict(os.environ, {"DEBIAN_PORT": "2222"}), \
                    self.assertRaises(config.ConfigError):
                config.read_config(directory)

    def test_invalid_utf8_and_port_boundaries(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory, ".env")
            path.write_bytes(b"\xff")
            path.chmod(0o600)
            with self.assertRaises(config.ConfigError) as error:
                config.read_config(directory)
            self.assertEqual(str(error.exception), "Debian 私密连接文件缺失或不安全")
        for port in ("1", "65535"):
            self.assertEqual(config.parse_config(GOOD.replace("2222", port)).port, int(port))

    def test_connection_port_fields_do_not_rewrite_process_identity(self):
        raw = {"port": 2222, "nested": {"DEBIAN_PORT": "2222", "ssh_port": 2222},
               "pid": 2222, "start_ticks": 2222, "heartbeat": 2222}
        safe = config.redact(raw, config.parse_config(GOOD))
        self.assertEqual(safe["port"], "<remote-port>")
        self.assertEqual(safe["nested"], {"DEBIAN_PORT": "<remote-port>", "ssh_port": "<remote-port>"})
        for key in ("pid", "start_ticks", "heartbeat"):
            self.assertEqual(safe[key], 2222)

    def test_redaction_preserves_comparable_observation_identity(self):
        text = (DEBIAN / "terminal/evidence.json").read_text()
        data = json.loads(text)
        self.assertEqual(data["redaction"]["kind"], "redacted_observation")
        self.assertFalse(data["redaction"]["connection_values_stored"])
        self.assertNotRegex(text, r"user-\d+|user@\d+|/run/user/\d+|/home/[^<]")
        baseline = data["start"]["baseline"]
        samples = data["start"]["samples"] + [data["new_ssh_connection"]]
        for sample in samples:
            for role in ("server", "pane"):
                self.assertEqual(sample[role], baseline[role])
                self.assertIn("user-<remote-uid>", sample[role]["cgroup"])
                self.assertNotIn(data["start"]["web_unit"], sample[role]["cgroup"])
        self.assertGreater(samples[-1]["heartbeat"], baseline["heartbeat"])
        raw = {"cgroup": "user-1234.slice/user@1234.service",
               "home": "/home/probe_user/own", "target": "203.0.113.10", "pid": 42}
        safe = config.redact(raw, config.parse_config(GOOD))
        self.assertEqual(safe["cgroup"], "user-<remote-uid>.slice/user@<remote-uid>.service")
        self.assertEqual(safe["pid"], 42)
        self.assertNotIn("probe_user", json.dumps(safe))
        self.assertNotIn("203.0.113.10", json.dumps(safe))


if __name__ == "__main__":
    unittest.main()
