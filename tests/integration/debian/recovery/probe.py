"""Probe tmux control-mode snapshot boundaries on an isolated Debian session."""

import argparse
import base64
import curses
import fcntl
import json
import os
from pathlib import Path
import pty
import re
import select
import signal
import stat
import struct
import subprocess
import sys
import termios
import time
import uuid

import bridge_support as b


def atomic(root, name, value):
    path = root / (name + ".tmp")
    path.write_text(str(value))
    path.replace(root / name)


class ControlClient:
    def __init__(self, state):
        self.process = subprocess.Popen(
            [state["tmux"], "-N", "-C", "-S", state["socket"],
             "attach-session", "-t", "=" + state["session"]],
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
            env={**os.environ, "LD_LIBRARY_PATH": state["library_path"], "TERM": "xterm-256color"},
        )
        self.buffer = bytearray()
        self.lines = []

    def read_until(self, predicate, timeout=10, start=0):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if any(predicate(line) for line in self.lines[start:]):
                return
            ready, _, _ = select.select([self.process.stdout], [], [], min(0.2, deadline - time.monotonic()))
            if not ready:
                continue
            chunk = os.read(self.process.stdout.fileno(), 65536)
            if not chunk:
                raise RuntimeError("control client exited")
            self.buffer.extend(chunk)
            if len(self.buffer) > 1024 * 1024:
                raise RuntimeError("control output exceeds bound")
            while b"\n" in self.buffer:
                line, _, rest = self.buffer.partition(b"\n")
                self.lines.append(bytes(line))
                self.buffer = bytearray(rest)

    def command(self, command, blocks=1):
        start = len(self.lines)
        self.process.stdin.write((command + "\n").encode())
        self.process.stdin.flush()
        self.read_until(lambda line: line.startswith((b"%end ", b"%error ")) and
                        sum(item.startswith((b"%end ", b"%error "))
                            for item in self.lines[start:]) >= blocks,
                        start=start)
        segment = self.lines[start:]
        output = []
        cursor = 0
        for _ in range(blocks):
            begin = next(index for index in range(cursor, len(segment))
                         if segment[index].startswith(b"%begin "))
            end = next(index for index in range(begin + 1, len(segment))
                       if segment[index].startswith((b"%end ", b"%error ")))
            assert segment[end].startswith(b"%end "), "control command failed"
            assert segment[begin].split()[1:3] == segment[end].split()[1:3]
            output.extend(segment[begin + 1:end])
            cursor = end + 1
        self.last_segment = segment[:cursor]
        return output, start + cursor - 1

    def close(self):
        if self.process.poll() is None:
            self.process.terminate()
            self.process.wait(timeout=5)


class AttachClient:
    def __init__(self, state, *, readonly=True, cols=100, rows=30):
        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))
        self.master = master
        args = [state["tmux"], "-N", "-S", state["socket"], "attach-session", "-t", "=" + state["session"]]
        if readonly:
            args.extend(["-f", "read-only,ignore-size"])
        self.process = subprocess.Popen(
            args,
            stdin=slave, stdout=slave, stderr=subprocess.DEVNULL,
            env={**os.environ, "LD_LIBRARY_PATH": state["library_path"], "TERM": "xterm-256color"},
            start_new_session=True,
        )
        os.close(slave)
        self.output = bytearray()
        self.forced_close = False

    def resize(self, cols, rows):
        fcntl.ioctl(self.master, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))
        os.kill(self.process.pid, signal.SIGWINCH)

    def read_until(self, marker, timeout=5):
        deadline = time.monotonic() + timeout
        while marker not in self.output:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise RuntimeError("attach output deadline")
            if not select.select([self.master], [], [], remaining)[0]:
                continue
            chunk = os.read(self.master, 65536)
            if not chunk:
                raise RuntimeError("attach client exited")
            self.output.extend(chunk)
            if len(self.output) > 256 * 1024:
                raise RuntimeError("attach output exceeds bound")

    def close(self):
        if self.master is not None:
            os.close(self.master)
            self.master = None
        if self.process.poll() is None:
            self.process.terminate()
            try:
                self.process.wait(timeout=1)
            except subprocess.TimeoutExpired:
                self.process.kill()
                self.process.wait(timeout=10)
                self.forced_close = True


def workload(root):
    atomic(root, "heartbeat", 1)
    atomic(root, "input_count", 0)
    for number in range(1, 401):
        os.write(1, f"SEQ_{number:04d}\r\n".encode())
        atomic(root, "stream_count", number)
        atomic(root, "heartbeat", number)
        time.sleep(0.005)
    os.write(1, b"\x1b[31")
    atomic(root, "partial_written", 1)
    b.wait_for(lambda: (root / "release_partial").exists())
    os.write(1, b"mPENDING_RED\x1b[0m\r\n")
    os.write(1, b"\xe9\x9b")
    atomic(root, "utf8_written", 1)
    b.wait_for(lambda: (root / "release_utf8").exists())
    os.write(1, b"\xaa UTF8_DONE\r\n")
    os.write(1, b"\x1b]8;;https://example.test")
    atomic(root, "osc_written", 1)
    b.wait_for(lambda: (root / "release_osc").exists())
    os.write(1, b"\x1b\\OSC_DONE\r\n")
    screen = curses.initscr()
    curses.noecho()
    curses.cbreak()
    curses.start_color()
    curses.init_pair(1, curses.COLOR_GREEN, curses.COLOR_BLACK)
    atomic(root, "tui_started", 1)
    heartbeat = 401
    while not (root / "release_tui").exists():
        heartbeat += 1
        atomic(root, "heartbeat", heartbeat)
        size = os.get_terminal_size(1)
        if screen.getmaxyx() != (size.lines, size.columns):
            curses.resizeterm(size.lines, size.columns)
        screen.erase()
        screen.addstr(0, 0, "CURSES_W03", curses.color_pair(1))
        screen.addstr(1, 0, "ACTIVE_ALT_SCREEN")
        screen.addstr(2, 0, f"SIZE_{size.columns}x{size.lines}")
        screen.move(2, 3)
        screen.refresh()
        atomic(root, "tui_size", f"{size.columns}x{size.lines}")
        time.sleep(0.05)
    curses.nocbreak()
    curses.echo()
    curses.endwin()
    atomic(root, "finished", 1)
    b.wait_for(lambda: (root / "start_burst").exists())
    for number in range(1, 5001):
        os.write(1, f"BURST_{number:04d}\r\n".encode())
        if number % 100 == 0:
            heartbeat += 1
            atomic(root, "heartbeat", heartbeat)
    atomic(root, "burst_done", 5000)
    time.sleep(5)


def marker_numbers(lines):
    return [int(match) for line in lines for match in re.findall(rb"SEQ_(\d{4})", line)]


def atomic_pending(client, pane, marker):
    output, boundary = client.command(
        f"capture-pane -p -e -t {pane} ; "
        f"display-message -p {marker} ; "
        f"capture-pane -p -P -t {pane}", blocks=3)
    split = next((index for index, line in enumerate(output) if line == marker.encode()), None)
    assert split is not None
    first_begin = next(index for index, line in enumerate(client.last_segment)
                       if line.startswith(b"%begin "))
    assert not any(line.startswith(b"%output ")
                   for line in client.last_segment[first_begin:]), \
        "pane notification interleaved with atomic capture"
    return output[:split], output[split + 1:], boundary


def start(root):
    token = uuid.uuid4().hex[:12]
    state = {"root": str(root), "socket": str(root / "tmux.sock"),
             "session": "persistty_probe_" + token,
             "server_unit": f"persistty-bridge-{token}-tmux.service",
             "web_unit": f"persistty-bridge-{token}-web.service",
             "client_unit": f"persistty-bridge-{token}-client.service",
             "stage": "download"}
    b.store(state)
    for package, pattern in (("tmux=3.5a-3", "tmux_*.deb"),
                             ("libevent-core-2.1-7t64=2.1.13-stable-1~deb13u1", "libevent-core-*.deb")):
        b.command(["apt-get", "download", package])
        files = list(root.glob(pattern))
        assert len(files) == 1
        b.command(["dpkg-deb", "-x", str(files[0]), str(root / "package")])
    state["tmux"] = str(root / "package/usr/bin/tmux")
    state["library_path"] = str(root / "package/usr/lib/x86_64-linux-gnu")
    (root / "tmux.conf").write_text("set -g exit-empty off\nset -g exit-unattached off\n"
                                    "set -g status off\nset -g history-limit 120\n")
    state["stage"] = "start_tmux"
    b.store(state)
    b.unit(state, state["server_unit"], [state["tmux"], "-D", "-S", state["socket"],
                                         "-f", str(root / "tmux.conf")])
    b.wait_for(lambda: b.tmux(state, "list-sessions", check=False)[0] == 0)
    b.tmux(state, "new-session", "-d", "-s", state["session"], "-x", "100", "-y", "30",
           "-e", "PERSISTTY_INITIAL_CWD=" + str(root),
           "-e", "PERSISTTY_DISPLAY_NAME=probe",
           "env", "TERM=xterm-256color", "LC_ALL=C.UTF-8", sys.executable,
           str(root / "probe.py"), "workload", str(root))
    session_env_status, session_env_value = b.tmux(state, "show-environment", "-t",
                                                    "=" + state["session"],
                                                    "PERSISTTY_INITIAL_CWD", check=False)
    probe_status, _ = b.tmux(state, "show-options", "-gqv", "exit-empty", check=False)
    set_status, _ = b.tmux(state, "set-option", "-w", "-t", "=" + state["session"] + ":0",
                            "history-limit", "120", check=False)
    metadata_status, _ = b.tmux(state, "set-option", "-t", "=" + state["session"],
                                 "@persistty_initial_cwd", str(root), check=False)
    metadata_read_status, metadata_value = b.tmux(state, "show-option", "-qv", "-t",
                                                    "=" + state["session"], "@persistty_initial_cwd",
                                                    check=False)
    window_metadata_status, _ = b.tmux(state, "set-option", "-w", "-t",
                                       "=" + state["session"] + ":0",
                                       "@persistty_initial_cwd", str(root), check=False)
    window_read_status, window_value = b.tmux(state, "show-option", "-wqv", "-t",
                                               "=" + state["session"] + ":0",
                                               "@persistty_initial_cwd", check=False)
    product_commands_ok = (probe_status == set_status == window_metadata_status == window_read_status == session_env_status == 0
                           and session_env_value.strip() == "PERSISTTY_INITIAL_CWD=" + str(root)
                           and window_value.strip() == str(root))
    b.wait_for(lambda: (root / "stream_count").exists())
    b.snapshot(state, "before_control_attach")
    state["stage"] = "control_snapshot"
    b.store(state)
    client = ControlClient(state)
    attach_clients = []
    try:
        client.read_until(lambda line: line.startswith(b"%session-changed "))
        b.wait_for(lambda: int((root / "stream_count").read_text()) >= 40)
        stream_attach = AttachClient(state)
        attach_clients.append(stream_attach)
        b.wait_for(lambda: "read-only" in b.tmux(state, "list-clients", "-F", "#{client_flags}")[1])
        pane = state["baseline"]["pane_id"]
        snapshot, boundary = client.command(f"capture-pane -p -e -S - -t {pane}")
        client.read_until(lambda line: b"SEQ_0400" in line, timeout=10)
        stream_attach.read_until(b"SEQ_0400", timeout=10)
        stream_attach.close()
        attach_clients.remove(stream_attach)
        before = marker_numbers(snapshot)
        after = marker_numbers(line for line in client.lines[boundary + 1:]
                               if line.startswith(b"%output "))
        state["stage"] = "pending_sequence"
        b.store(state)
        b.wait_for(lambda: (root / "partial_written").exists())
        plain_status, history_plain = b.tmux(state, "capture-pane", "-p", "-S", "-", "-E", "-1", "-t", pane, check=False)
        plain_history_size = b.tmux(state, "display-message", "-p", "-t", pane, "#{history_size}")[1].strip()
        pending_grid, pending_prefix, pending_boundary = atomic_pending(
            client, pane, "__PENDING_BOUNDARY__")
        (root / "release_partial").touch()
        client.read_until(lambda line: b"PENDING_RED" in line)
        pending_live = [line for line in client.lines[pending_boundary + 1:]
                        if line.startswith(b"%output ") and b"PENDING_RED" in line]
        state["stage"] = "utf8_sequence"
        b.store(state)
        b.wait_for(lambda: (root / "utf8_written").exists())
        _, utf8_prefix, utf8_boundary = atomic_pending(client, pane, "__UTF8_BOUNDARY__")
        utf8_attach = AttachClient(state)
        attach_clients.append(utf8_attach)
        utf8_attach.read_until(b"SEQ_0400")
        (root / "release_utf8").touch()
        client.read_until(lambda line: b"UTF8_DONE" in line, start=utf8_boundary + 1)
        utf8_attach.read_until(b"UTF8_DONE")
        utf8_attach.close()
        attach_clients.remove(utf8_attach)
        state["stage"] = "osc_sequence"
        b.store(state)
        b.wait_for(lambda: (root / "osc_written").exists())
        _, osc_prefix, osc_boundary = atomic_pending(client, pane, "__OSC_BOUNDARY__")
        osc_attach = AttachClient(state)
        attach_clients.append(osc_attach)
        osc_attach.read_until(b"UTF8_DONE")
        (root / "release_osc").touch()
        client.read_until(lambda line: b"OSC_DONE" in line, start=osc_boundary + 1)
        osc_attach.read_until(b"OSC_DONE")
        osc_attach.close()
        attach_clients.remove(osc_attach)
        state["stage"] = "alternate_screen"
        b.store(state)
        b.wait_for(lambda: (root / "tui_started").exists())
        b.wait_for(lambda: "CURSES_W03" in b.tmux(state, "capture-pane", "-p", "-t", pane)[1])
        alternate = b.tmux(state, "display-message", "-p", "-t", pane, "#{alternate_on}")[1].strip()
        tui_attach = AttachClient(state)
        attach_clients.append(tui_attach)
        tui_attach.read_until(b"ACTIVE_ALT_SCREEN")
        current, _ = client.command(f"capture-pane -p -e -t {pane}")
        saved, _ = client.command(f"capture-pane -a -p -t {pane}")
        tui_attach.close()
        attach_clients.remove(tui_attach)
        client.close()
        b.wait_for(lambda: int((root / "heartbeat").read_text()) > state["samples"][-1]["heartbeat"])
        b.snapshot(state, "all_observers_detached_during_tui")
        client = ControlClient(state)
        client.read_until(lambda line: line.startswith(b"%session-changed "))
        recovered_current, _ = client.command(f"capture-pane -p -e -t {pane}")
        b.wait_for(lambda: int((root / "heartbeat").read_text()) > state["samples"][-1]["heartbeat"])
        b.snapshot(state, "new_owner_attached_during_tui")
        state["stage"] = "resize_roles"
        b.store(state)
        pane_size = lambda: b.tmux(state, "display-message", "-p", "-t", pane,
                                    "#{window_width}x#{window_height}")[1].strip()
        owner_attach = AttachClient(state, readonly=False)
        attach_clients.append(owner_attach)
        b.wait_for(lambda: pane_size() == "100x30")
        small_viewer = AttachClient(state, cols=40, rows=10)
        attach_clients.append(small_viewer)
        small_viewer.resize(60, 20)
        time.sleep(0.2)
        observer_did_not_resize = pane_size() == "100x30"
        owner_attach.resize(80, 24)
        b.wait_for(lambda: pane_size() == "80x24" and
                   (root / "tui_size").read_text() == "80x24")
        owner_resized_tui = True
        owner_attach.resize(100, 30)
        b.wait_for(lambda: pane_size() == "100x30" and
                   (root / "tui_size").read_text() == "100x30")
        small_viewer.close()
        attach_clients.remove(small_viewer)
        owner_attach.close()
        attach_clients.remove(owner_attach)
        alternate_status, history_alternate = b.tmux(state, "capture-pane", "-p", "-S", "-", "-E", "-1", "-t", pane, check=False)
        saved_status, history_saved = b.tmux(state, "capture-pane", "-a", "-p", "-S", "-", "-t", pane, check=False)
        (root / "release_tui").touch()
        b.wait_for(lambda: (root / "finished").exists())
        exit_status, history_after_exit = b.tmux(state, "capture-pane", "-p", "-S", "-", "-E", "-1", "-t", pane, check=False)
        state["stage"] = "slow_observer"
        b.store(state)
        slow_viewer = AttachClient(state)
        attach_clients.append(slow_viewer)
        state["stage"] = "slow_observer_burst"
        b.store(state)
        (root / "start_burst").touch()
        b.wait_for(lambda: (root / "burst_done").exists())
        state["stage"] = "slow_observer_snapshot"
        b.store(state)
        b.snapshot(state, "slow_observer_did_not_stop_pane")
        state["stage"] = "slow_observer_close"
        b.store(state)
        slow_viewer.close()
        attach_clients.remove(slow_viewer)
        plain_numbers = marker_numbers(history_plain.encode().splitlines())
        alternate_numbers = marker_numbers(history_alternate.encode().splitlines())
        saved_numbers = marker_numbers(history_saved.encode().splitlines())
        exit_numbers = marker_numbers(history_after_exit.encode().splitlines())
        facts = {"tmux_version": b.command([state["tmux"], "-V"],
                                           env={**os.environ, "LD_LIBRARY_PATH": state["library_path"]})[1].strip(),
                 "product_command_statuses": [probe_status, set_status, metadata_status,
                                                metadata_read_status, window_metadata_status,
                                                window_read_status, session_env_status],
                 "snapshot_count": len(before), "snapshot_last": max(before, default=0),
                 "live_count": len(after), "live_first": min(after, default=0),
                 "missing_markers": sorted(set(range(1, 401)) - set(before + after))[:20],
                 "duplicate_markers": sorted({n for n in before + after if (before + after).count(n) > 1})[:20],
                 "pending_grid_has_prefix": any(b"\\033[31" in line or b"\x1b[31" in line
                                                for line in pending_grid),
                 "pending_prefix_has_csi": any(b"\\033[31" in line or b"\x1b[31" in line
                                               for line in pending_prefix),
                 "pending_prefix_hex": [line.hex() for line in pending_prefix[:2]],
                 "pending_live_has_suffix_without_prefix": any(b"mPENDING_RED" in line and
                                                               b"\\033[31" not in line and b"\x1b[31" not in line
                                                               for line in pending_live),
                 "utf8_prefix_hex": [line.hex() for line in utf8_prefix[:2]],
                 "osc_prefix_hex": [line.hex() for line in osc_prefix[:2]],
                 "utf8_attach_has_complete_character": b"\xe9\x9b\xaa" in utf8_attach.output,
                 "osc_attach_has_following_text": b"OSC_DONE" in osc_attach.output,
                 "tui_attach_has_current_screen": b"CURSES_W03" in tui_attach.output,
                 "history_plain_count": len(plain_numbers),
                 "history_capture_statuses": [plain_status, alternate_status, saved_status, exit_status],
                 "history_plain_size": plain_history_size,
                 "history_plain_bounds": [min(plain_numbers, default=0), max(plain_numbers, default=0)],
                 "history_plain_first_line": history_plain.splitlines()[0][:24] if history_plain else "",
                 "history_alternate_count": len(alternate_numbers),
                 "history_saved_count": len(saved_numbers),
                 "history_after_exit_count": len(exit_numbers),
                 "history_after_exit_bounds": [min(exit_numbers, default=0), max(exit_numbers, default=0)],
                 "bounded_history_retained_latest": int(plain_history_size) <= 120 and
                                                    plain_numbers == list(range(min(plain_numbers), 372)) and
                                                    min(plain_numbers) > 1,
                 "alternate_on": alternate,
                 "current_has_tui": any(b"CURSES_W03" in line for line in current),
                 "new_owner_has_tui": any(b"CURSES_W03" in line for line in recovered_current),
                 "observer_did_not_resize": observer_did_not_resize,
                 "owner_resized_tui": owner_resized_tui,
                 "slow_observer_burst_finished": (root / "burst_done").read_text() == "5000",
                 "slow_observer_required_force_close": slow_viewer.forced_close,
                 "saved_has_tui": any(b"CURSES_W03" in line for line in saved)}
        checks = {
            "control_markers_ordered_without_gap": before + after == list(range(1, 401)),
            "control_snapshot_requires_pending_csi": facts["pending_prefix_has_csi"] and
                                                     facts["pending_live_has_suffix_without_prefix"],
            "control_pending_utf8_not_captured": utf8_prefix == [b""],
            "normal_attach_recovers_split_utf8": facts["utf8_attach_has_complete_character"],
            "normal_attach_recovers_osc_and_tui": facts["osc_attach_has_following_text"] and
                                                  facts["tui_attach_has_current_screen"],
            "normal_history_bounded": facts["bounded_history_retained_latest"],
            "normal_history_survives_alternate": len(alternate_numbers) > 0 and
                                                 alternate_numbers == list(range(min(alternate_numbers), max(alternate_numbers) + 1)),
            "new_owner_same_tui": facts["new_owner_has_tui"] and
                                  all(sample["input_count"] == 0 for sample in state["samples"]),
            "role_resize_rules": observer_did_not_resize and owner_resized_tui,
            "slow_observer_does_not_stop_pane": facts["slow_observer_burst_finished"],
            "product_tmux_command_forms": product_commands_ok,
        }
        state["stage"] = "complete"
        b.store(state)
        records = {"utf8": base64.b64encode(utf8_attach.output).decode(),
                   "osc": base64.b64encode(osc_attach.output).decode(),
                   "tui": base64.b64encode(tui_attach.output).decode(),
                   "stream": base64.b64encode(stream_attach.output).decode()}
        print(json.dumps({"facts": facts, "checks": checks, "records": records}))
    finally:
        for attach in attach_clients:
            attach.close()
        client.close()


def main():
    if sys.flags.optimize:
        raise RuntimeError("禁止优化模式")
    parser = argparse.ArgumentParser()
    parser.add_argument("phase", choices=("start", "workload", "cleanup"))
    parser.add_argument("root", type=Path)
    args = parser.parse_args()
    root = args.root
    assert root.parent == Path("/tmp") and re.fullmatch(r"persistty-recovery-[A-Za-z0-9]{8}", root.name)
    assert not root.is_symlink()
    info = root.stat()
    assert os.getuid() != 0 and info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700
    os.chdir(root)
    if args.phase == "workload":
        workload(root)
        return
    try:
        if args.phase == "start":
            start(root)
        else:
            b.cleanup(root)
    except Exception as error:
        stage = args.phase
        if (root / "state.json").exists():
            stage = json.loads((root / "state.json").read_text()).get("stage", stage)
        print(json.dumps({"probe_failed": True, "stage": stage,
                          "error_type": type(error).__name__}))


if __name__ == "__main__":
    main()
