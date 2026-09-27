"""Fixed synthetic history and real curses experiment; never user content."""
import argparse
import curses
import hashlib
import json
import os
from pathlib import Path
import re
import select
import stat
import sys
import time
import uuid

import bridge_support as b


def atomic(root, name, value):
    tmp = root / (name + ".tmp")
    tmp.write_text(str(value))
    tmp.replace(root / name)


def workload(root):
    (root / "input_count").write_text("0")
    input_bytes = 0
    tick = 0
    last = ""
    screen = None
    deadline = time.monotonic() + 280
    while time.monotonic() < deadline:
        tick += 1
        atomic(root, "heartbeat", tick)
        if select.select([0],[],[],0)[0]:
            received = os.read(0,65536)
            if received:
                input_bytes += len(received)
                atomic(root,"input_count",input_bytes)
        phase = (root / "phase").read_text() if (root / "phase").exists() else "plain"
        if phase != last:
            if phase == "plain":
                for i in range(1, 201):
                    os.write(1, f"HIST_{i:04d} \x1b[31m雪\x1b[0m e\u0301\r\n".encode())
            elif phase == "gap":
                for i in range(1, 81):
                    os.write(1, f"GAP_{i:04d}\r\n".encode())
            elif phase == "tui":
                screen = curses.initscr()
                curses.noecho()
                curses.cbreak()
                curses.start_color()
                curses.init_pair(1, curses.COLOR_GREEN, curses.COLOR_BLACK)
            elif phase == "exit":
                curses.nocbreak()
                curses.echo()
                curses.endwin()
                screen = None
                os.write(1, b"TUI_EXIT_D07\r\n")
            last = phase
            atomic(root, "phase_ack", phase)
        if screen is not None:
            size = os.get_terminal_size(1)
            if screen.getmaxyx() != (size.lines, size.columns):
                curses.resizeterm(size.lines, size.columns)
            screen.erase()
            screen.addstr(0, 0, "CURSES_D07", curses.color_pair(1))
            screen.addstr(1, 0, f"SIZE_{size.columns}x{size.lines}")
            screen.addstr(2, 0, "雪 e\u0301")
            screen.move(3, 5)
            screen.refresh()
            atomic(root, "tui_size", f"{size.columns}x{size.lines}")
        time.sleep(0.05)


def phase(state, name):
    root = Path(state["root"])
    atomic(root, "phase", name)
    b.wait_for(lambda: (root / "phase_ack").read_text() == name)


def record(state, cols=100, rows=30):
    result = json.loads(b.command([str(Path(state["root"], "bridge-probe")), "client",
        "--port", str(state["listen_port"]), "--token-file", str(Path(state["root"], "token")),
        "--mode", "record", "--duration", "1s", "--cols", str(cols), "--rows", str(rows)])[1])
    assert result["resize_ack"] and result["bytes"] <= 256*1024
    assert not result["stats"]["active"] and result["stats"]["started"] == result["stats"]["reaped"]
    return result


def start(root):
    token = uuid.uuid4().hex[:12]
    state = {"root":str(root), "socket":str(root / "tmux.sock"),
        "session":"persistty_probe_" + token,
        "server_unit":f"persistty-bridge-{token}-tmux.service",
        "web_unit":f"persistty-bridge-{token}-web.service",
        "client_unit":f"persistty-bridge-{token}-client.service"}
    b.store(state)
    for package, pattern in (("tmux=3.5a-3", "tmux_*.deb"),
        ("libevent-core-2.1-7t64=2.1.13-stable-1~deb13u1", "libevent-core-*.deb")):
        state["stage"] = "download"
        b.store(state)
        b.command(["apt-get", "download", package])
        files = list(root.glob(pattern))
        assert len(files) == 1
        b.command(["dpkg-deb", "-x", str(files[0]), str(root / "package")])
    state["tmux"] = str(root / "package/usr/bin/tmux")
    state["library_path"] = str(root / "package/usr/lib/x86_64-linux-gnu")
    config = root / "tmux.conf"
    config.write_text("set -g exit-empty off\nset -g exit-unattached off\nset -g status off\nset -g history-limit 5000\n")
    (root / "token").write_text(uuid.uuid4().hex + uuid.uuid4().hex)
    (root / "token").chmod(0o600)
    state["stage"] = "start_units"
    b.store(state)
    b.unit(state, state["server_unit"], [state["tmux"], "-D", "-S", state["socket"], "-f", str(config)])
    b.wait_for(lambda: b.tmux(state, "list-sessions", check=False)[0] == 0)
    b.tmux(state, "new-session", "-d", "-s", state["session"], "-x", "100", "-y", "30",
        "env", "TERM=xterm-256color", "LC_ALL=C.UTF-8", sys.executable, str(root / "probe.py"), "workload", str(root))
    b.wait_for(lambda: (root / "phase_ack").exists())
    b.snapshot(state, "ordinary_history_created")
    b.start_web(state)
    records = {"plain":record(state)}
    state["stage"] = "capture_gap"
    b.store(state)
    status, capture = b.tmux(state, "capture-pane", "-p", "-e", "-S", "-200", "-t", state["session"], check=False)
    state["stage"] = "capture_status_" + str(status) + "_history_markers_" + str(capture.count("HIST_"))
    b.store(state)
    assert status == 0 and "HIST_0001" in capture
    state["stage"] = "advance_gap"
    b.store(state)
    phase(state, "gap")
    state["stage"] = "record_gap"
    b.store(state)
    records["gap"] = record(state)
    b.snapshot(state, "detached_output_reconnected")
    phase(state, "tui")
    records["tui100"] = record(state)
    saved = b.tmux(state, "capture-pane", "-a", "-p", "-t", state["session"])[1]
    current = b.tmux(state, "capture-pane", "-p", "-t", state["session"])[1]
    assert "CURSES_D07" in current and "CURSES_D07" not in saved
    state["alternate_on_during_curses"] = b.tmux(state,"display-message","-p","-t",state["session"],"#{alternate_on}")[1].strip()
    assert state["alternate_on_during_curses"] == "1"
    state["stage"] = "tui_resize"
    b.store(state)
    for cols, rows in ((80,24), (120,40)):
        records[f"tui{cols}"] = record(state, cols, rows)
        assert (root / "tui_size").read_text() == f"{cols}x{rows}"
        b.snapshot(state, f"curses_redrawn_{cols}x{rows}")
    records["tui_reconnect"] = record(state, 120,40)
    b.snapshot(state, "same_curses_reconnected")
    phase(state, "exit")
    records["exit"] = record(state,120,40)
    state["alternate_on_after_exit"] = b.tmux(state,"display-message","-p","-t",state["session"],"#{alternate_on}")[1].strip()
    assert state["alternate_on_after_exit"] == "0"
    b.snapshot(state, "curses_exited_same_pane")
    assert all(sample["input_count"] == 0 for sample in state["samples"])
    state["stage"] = "complete"
    b.store(state)
    print(json.dumps({"records":records, "capture":capture,
        "facts":{"samples":state["samples"], "curses_saved_grid_distinct":True,
            "alternate_on_during_curses":state["alternate_on_during_curses"],
            "alternate_on_after_exit":state["alternate_on_after_exit"],
            "tmux_version": b.command([state["tmux"], "-V"], env={**os.environ,"LD_LIBRARY_PATH":state["library_path"]})[1].strip(),
            "binary_sha256": hashlib.sha256((root / "bridge-probe").read_bytes()).hexdigest()}}))


def main():
    if sys.flags.optimize:
        raise RuntimeError("禁止优化模式")
    parser = argparse.ArgumentParser()
    parser.add_argument("phase",choices=("start","workload","cleanup"))
    parser.add_argument("root",type=Path)
    args = parser.parse_args()
    root = args.root
    assert root.parent == Path("/tmp")
    assert re.fullmatch(r"persistty-history-[A-Za-z0-9]{8}",root.name) and not root.is_symlink()
    info = root.stat()
    assert os.getuid() != 0 and info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700
    os.chdir(root)
    try:
        if args.phase == "start": start(root)
        elif args.phase == "workload": workload(root)
        else: b.cleanup(root)
    except Exception:
        stage = args.phase
        if (root / "state.json").exists(): stage = json.loads((root / "state.json").read_text()).get("stage",stage)
        print(json.dumps({"probe_failed":True,"stage":stage}))


if __name__ == "__main__": main()
