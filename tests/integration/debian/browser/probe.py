"""短时浏览器联调服务，只管理本次随机私有目录和 unit。"""

import json
import os
from pathlib import Path
import re
import shutil
import socket
import sys
import time
import urllib.request
import uuid

import support as b

stage = "arguments"


def unit(state, name, args):
    b.command(["systemd-run", "--user", "--quiet", "--unit=" + name,
               "--property=Type=exec", "--property=KillMode=control-group",
               "--property=Restart=no", "--property=RuntimeMaxSec=900",
               "--property=TimeoutStopSec=5", "--property=UMask=0077",
               "--property=StandardOutput=null", "--property=StandardError=null",
               "--setenv=LD_LIBRARY_PATH=" + state["library_path"], *args])


def save(state):
    Path(state["root"], "state.json").write_text(json.dumps(state) + "\n")


def layout_fixture(root):
    global stage
    stage = "layout_fixture"
    state = json.loads((root / "state.json").read_text())
    assert state["root"] == str(root)
    project = root / "project"
    assert project.is_dir() and not project.is_symlink()
    (root / "extra").mkdir(mode=0o700)
    for index in range(120):
        with (project / f"ui-fixture-{index:03d}.txt").open("x") as file:
            file.write("isolated layout fixture\n")
    return {"layout_fixture_ready": True, "fixture_files": 120, "folder_count": 2}


def start(root):
    global stage
    stage = "start_browser"
    os.chdir(root)
    token = uuid.uuid4().hex[:12]
    state = {"root": str(root), "socket": str(root / "tmux.sock"),
             "server_unit": f"persistty-browser-{token}-tmux.service",
             "web_unit": f"persistty-browser-{token}-web.service"}
    save(state)
    for package, pattern in (("tmux=3.5a-3", "tmux_*.deb"),
                             ("libevent-core-2.1-7t64=2.1.13-stable-1~deb13u1", "libevent-core-*.deb")):
        b.command(["apt-get", "download", package])
        files = list(root.glob(pattern))
        assert len(files) == 1
        b.command(["dpkg-deb", "-x", str(files[0]), str(root / "package")])
    state["tmux"] = str(root / "package/usr/bin/tmux")
    state["library_path"] = str(root / "package/usr/lib/x86_64-linux-gnu")
    (root / "tmux.conf").write_text("set -g exit-empty off\nset -g exit-unattached off\nset -g status off\n")
    save(state)
    unit(state, state["server_unit"], [state["tmux"], "-D", "-S", state["socket"],
                                        "-f", str(root / "tmux.conf")])
    b.wait_for(lambda: b.tmux(state, "list-sessions", check=False)[0] == 0)
    with socket.socket() as reservation:
        reservation.bind(("127.0.0.1", 0))
        port = reservation.getsockname()[1]
    state["port"] = port
    save(state)
    unit(state, state["web_unit"], [str(root / "runtime-probe"), "serve", str(root),
                                     state["tmux"], state["socket"], str(port)])
    b.wait_for(lambda: (root / "ready.json").exists())
    port = json.loads((root / "ready.json").read_text())["port"]
    assert isinstance(port, int) and 1 <= port <= 65535
    assert port == state["port"]
    save(state)
    return {"root": str(root), "port": port, "ready": True}


def cleanup(root):
    global stage
    stage = "cleanup_browser"
    state = json.loads((root / "state.json").read_text())
    assert state["root"] == str(root)
    units = [state["web_unit"], state["server_unit"]]
    for name in units:
        assert re.fullmatch(r"persistty-browser-[0-9a-f]{12}-(web|tmux)\.service", name)
    pids = []
    if "tmux" in state:
        status, output = b.tmux(state, "list-panes", "-a", "-F", "#{pane_pid}", check=False)
        if status == 0:
            pids = [int(line) for line in output.splitlines() if line.isdecimal()]
    for name in units:
        b.command(["systemctl", "--user", "stop", name], check=False)
        status, _ = b.command(["systemctl", "--user", "is-active", name], check=False)
        assert status != 0
        b.command(["systemctl", "--user", "reset-failed", name], check=False)
    deadline = time.monotonic() + 5
    while any(Path(f"/proc/{pid}").exists() for pid in pids) and time.monotonic() < deadline:
        time.sleep(0.1)
    assert not any(Path(f"/proc/{pid}").exists() for pid in pids)
    os.chdir("/tmp")
    shutil.rmtree(root)
    assert not root.exists()
    return {"units_inactive": True, "pane_processes_gone": True, "root_removed": True}


def workload_snapshot(root, state):
    counter = root / "project/w03-counter.log"
    port_file = root / "project/w03-http-port"
    b.wait_for(lambda: counter.is_file() and port_file.is_file())
    lines = len(counter.read_text().splitlines())
    port = int(port_file.read_text())
    assert 1 <= port <= 65535
    with urllib.request.urlopen(f"http://127.0.0.1:{port}/welcome.txt", timeout=3) as response:
        assert response.status == 200 and response.read(128)
    pane_ids = b.tmux(state, "list-panes", "-a", "-F", "#{pane_id}")[1].splitlines()
    def has_tui():
        return any("W03_DETERMINISTIC_TUI" in b.tmux(state, "capture-pane", "-p", "-t", pane)[1] for pane in pane_ids)
    b.wait_for(has_tui)
    tui = has_tui()
    return lines, tui


def restart(root, force=False):
    global stage
    stage = "load_state"
    state = json.loads((root / "state.json").read_text())
    assert state["root"] == str(root)
    assert re.fullmatch(r"persistty-browser-[0-9a-f]{12}-web\.service", state["web_unit"])
    before_pids = [int(line) for line in b.tmux(state, "list-panes", "-a", "-F", "#{pane_pid}")[1].splitlines()]
    assert before_pids
    before = {pid: b.identity(pid) for pid in before_pids}
    verify_workloads = len(before) == 3 and (root / "project/w03-counter.log").exists()
    if force or verify_workloads:
        stage = "workload_before"
        assert len(before) == 3
        counter_before, tui_before = workload_snapshot(root, state)
        assert tui_before
    old_web = b.command(["systemctl", "--user", "show", "-p", "MainPID", "--value", state["web_unit"]])[1].strip()
    (root / "ready.json").unlink()
    if force:
        stage = "kill_web"
        b.command(["systemctl", "--user", "kill", "--signal=SIGKILL", state["web_unit"]])
        b.wait_for(lambda: b.command(["systemctl", "--user", "is-active", state["web_unit"]], check=False)[0] != 0)
        stage = "restart_failed_web"
        b.command(["systemctl", "--user", "restart", state["web_unit"]])
    else:
        b.command(["systemctl", "--user", "restart", state["web_unit"]])
    stage = "wait_web"
    b.wait_for(lambda: (root / "ready.json").exists())
    new_web = b.command(["systemctl", "--user", "show", "-p", "MainPID", "--value", state["web_unit"]])[1].strip()
    port = json.loads((root / "ready.json").read_text())["port"]
    after_pids = [int(line) for line in b.tmux(state, "list-panes", "-a", "-F", "#{pane_pid}")[1].splitlines()]
    after = {pid: b.identity(pid) for pid in after_pids}
    checks = {"web_pid_changed": old_web != new_web and old_web != "0" and new_web != "0",
              "pane_identity_preserved": before == after,
              "port_stable": port == state["port"]}
    if force or verify_workloads:
        stage = "workload_after"
        time.sleep(1.5)
        counter_after, tui_after = workload_snapshot(root, state)
        checks.update({"counter_advanced": counter_after > counter_before,
                       "http_service_survived": True, "tui_screen_survived": tui_after})
    assert all(checks.values())
    return {"checks": checks, "pane_count": len(after)}


if __name__ == "__main__":
    if sys.flags.optimize:
        raise SystemExit("探针不允许 Python 优化模式")
    action, path = sys.argv[1:]
    root = Path(path)
    assert re.fullmatch(r"/tmp/persistty-browser-[A-Za-z0-9]{8}", str(root))
    assert action in ("start", "restart", "force-restart", "layout-fixture", "cleanup")
    try:
        result = start(root) if action == "start" else restart(root, action == "force-restart") if action in ("restart", "force-restart") else layout_fixture(root) if action == "layout-fixture" else cleanup(root)
    except Exception as error:
        result = {"failed_stage": stage, "error_kind": type(error).__name__}
    print(json.dumps(result))
