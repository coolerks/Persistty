"""真实 Go/PTY/WS 的独立 Debian 编排，不接触产品服务。"""

import argparse
import json
import os
from pathlib import Path
import re
import select
import shutil
import stat
import subprocess
import sys
import tempfile
import termios
import time
import tty
import uuid


def command(args, check=True, env=None):
    with tempfile.TemporaryFile(dir=".") as output:
        process = subprocess.run(args, stdout=output, stderr=subprocess.DEVNULL,
                                 timeout=25, env=env)
        output.seek(0)
        raw = output.read(1024 * 1024 + 1)
    if len(raw) > 1024 * 1024 or (check and process.returncode):
        raise RuntimeError("实验子命令失败或输出超限")
    return process.returncode, raw.decode("utf-8")


def wait_for(test):
    deadline = time.monotonic() + 8
    while time.monotonic() < deadline:
        if test():
            return
        time.sleep(0.1)
    raise RuntimeError("实验就绪超时")


def identity(pid):
    text = Path(f"/proc/{pid}/stat").read_text()
    fields = text[text.rfind(")") + 2:].split()
    return {"pid": int(pid), "start_ticks": int(fields[19]),
            "cgroup": Path(f"/proc/{pid}/cgroup").read_text().strip()}


def store(state):
    Path(state["root"], "state.json").write_text(json.dumps(state) + "\n")


def tmux(state, *args, check=True):
    return command([state["tmux"], "-N", "-S", state["socket"], *args], check,
                   {**os.environ, "LD_LIBRARY_PATH": state["library_path"]})


def unit(state, name, args):
    command(["systemd-run", "--user", "--quiet", "--unit=" + name,
             "--property=Type=exec", "--property=KillMode=control-group",
             "--property=Restart=no", "--property=RuntimeMaxSec=300",
             "--property=TimeoutStopSec=5", "--property=UMask=0077",
             "--property=StandardOutput=null", "--property=StandardError=null",
             "--setenv=LD_LIBRARY_PATH=" + state["library_path"], *args])


def snapshot(state, label):
    facts = tmux(state, "list-panes", "-t", "=" + state["session"], "-F",
                 "#{pid}|#{pane_pid}|#{pane_id}")[1].strip().split("|")
    assert len(facts) == 3
    result = {"label": label, "server": identity(int(facts[0])),
              "pane": identity(int(facts[1])), "pane_id": facts[2],
              "heartbeat": int(Path(state["root"], "heartbeat").read_text()),
              "input_count": int(Path(state["root"], "input_count").read_text()),
              "clients": tmux(state, "list-clients", "-F", "#{client_pid}")[1].splitlines()}
    assert state["server_unit"] in result["server"]["cgroup"].replace("\\x2d", "-")
    assert state["web_unit"] not in result["server"]["cgroup"].replace("\\x2d", "-")
    assert state["web_unit"] not in result["pane"]["cgroup"].replace("\\x2d", "-")
    if state.get("baseline"):
        for role in ("server", "pane"):
            assert result[role] == state["baseline"][role], "原进程身份改变"
        assert result["heartbeat"] > state["samples"][-1]["heartbeat"], "心跳未逐次增长"
    scope = result["pane"]["cgroup"].split("/")[-1]
    if scope != state["server_unit"]:
        assert re.fullmatch(r"tmux-spawn-[A-Za-z0-9-]+\.scope", scope)
        state["pane_scope"] = scope
    state.setdefault("baseline", result)
    state.setdefault("samples", []).append(result)
    store(state)
    return result


def start_web(state):
    root = Path(state["root"])
    ready = root / "ready.json"
    ready.unlink(missing_ok=True)
    unit(state, state["web_unit"], [str(root / "bridge-probe"), "serve",
         "--tmux-bin", state["tmux"], "--socket", state["socket"],
         "--session", state["session"], "--token-file", str(root / "token"),
         "--ready-file", str(ready)])
    wait_for(ready.exists)
    data = json.loads(ready.read_text())
    assert set(data) == {"port", "pid"}
    assert isinstance(data["port"], int) and 1 <= data["port"] <= 65535
    assert stat.S_IMODE(ready.stat().st_mode) == 0o600
    state["web_identity"] = identity(data["pid"])
    assert state["web_unit"] in state["web_identity"]["cgroup"].replace("\\x2d", "-")
    state["listen_port"] = data["port"]
    store(state)


def client(state, mode):
    result = json.loads(command([str(Path(state["root"], "bridge-probe")), "client",
                               "--port", str(state["listen_port"]), "--token-file",
                               str(Path(state["root"], "token")), "--mode", mode,
                               "--duration", "20s"])[1])
    required = ({"no_auth", "wrong_origin", "single_attach_409", "valid_resize",
                 "split_unicode_binary_roundtrip", "invalid_resize_1008", "oversize_1009",
                 "detach_reconnect_reaped", "resources_bounded"} if mode == "exercise" else {"binary_output"})
    assert all(result["checks"].get(key) is True for key in required), "客户端断言缺失"
    facts = result["stats"]
    assert facts["active"] is False and facts["started"] == facts["reaped"], "attach 未回收"
    return result


def hold(state):
    root = Path(state["root"])
    ready = root / "client-ready.json"
    ready.unlink(missing_ok=True)
    unit(state, state["client_unit"], [str(root / "bridge-probe"), "client",
         "--port", str(state["listen_port"]), "--token-file", str(root / "token"),
         "--mode", "hold", "--ready-file", str(ready)])
    wait_for(ready.exists)
    wait_for(lambda: len(tmux(state, "list-clients")[1].splitlines()) == 1)
    pid = int(tmux(state, "list-clients", "-F", "#{client_pid}")[1].strip())
    attach = identity(pid)
    assert state["web_unit"] in attach["cgroup"].replace("\\x2d", "-")
    state.setdefault("attach_identities", []).append(attach)
    store(state)


def start(root):
    token = uuid.uuid4().hex[:12]
    state = {"root": str(root), "socket": str(root / "tmux.sock"),
             "session": "persistty_probe_" + token,
             "server_unit": f"persistty-bridge-{token}-tmux.service",
             "web_unit": f"persistty-bridge-{token}-web.service",
             "client_unit": f"persistty-bridge-{token}-client.service"}
    store(state)
    for package, pattern in (("tmux=3.5a-3", "tmux_*.deb"),
                             ("libevent-core-2.1-7t64=2.1.13-stable-1~deb13u1", "libevent-core-*.deb")):
        state["stage"] = "download_tmux" if pattern.startswith("tmux") else "download_dependency"
        store(state)
        command(["apt-get", "download", package])
        files = list(root.glob(pattern))
        assert len(files) == 1
        command(["dpkg-deb", "-x", str(files[0]), str(root / "package")])
    state["tmux"] = str(root / "package/usr/bin/tmux")
    state["library_path"] = str(root / "package/usr/lib/x86_64-linux-gnu")
    state["version"] = command([state["tmux"], "-V"], env={**os.environ,
                                "LD_LIBRARY_PATH": state["library_path"]})[1].strip()
    store(state)
    config = root / "tmux.conf"
    config.write_text("set -g exit-empty off\nset -g exit-unattached off\nset -g status off\n")
    (root / "token").write_text(uuid.uuid4().hex + uuid.uuid4().hex)
    (root / "token").chmod(0o600)
    state["stage"] = "start_tmux"
    store(state)
    unit(state, state["server_unit"], [state["tmux"], "-D", "-S", state["socket"], "-f", str(config)])
    wait_for(lambda: tmux(state, "list-sessions", check=False)[0] == 0)
    tmux(state, "new-session", "-d", "-s", state["session"], "-x", "100", "-y", "30",
         sys.executable, str(root / "probe.py"), "workload", str(root))
    wait_for(lambda: (root / "heartbeat").exists() and (root / "input_count").exists())
    snapshot(state, "before_go_attach")
    state["stage"] = "start_go"
    store(state)
    start_web(state)
    state["stage"] = "client_exercise"
    store(state)
    state["exercise"] = client(state, "exercise")
    time.sleep(0.4)
    snapshot(state, "after_ws_exercise")
    state["expected_input_count"] = state["samples"][-1]["input_count"]
    assert state["expected_input_count"] == 1, "输入并非仅执行一次"
    store(state)
    print(json.dumps(state))


def workload(root):
    tty.setraw(0, termios.TCSANOW)
    count, heartbeat, pending = 0, 0, bytearray()
    (root / "input_count").write_text("0")
    deadline = time.monotonic() + 280
    while time.monotonic() < deadline:
        heartbeat += 1
        temporary = root / "heartbeat.tmp"
        temporary.write_text(str(heartbeat))
        temporary.replace(root / "heartbeat")
        if select.select([0], [], [], 0.15)[0]:
            raw = os.read(0, 65536)
            if not raw:
                break
            pending.extend(raw)
            assert len(pending) <= 131072
            while b"\n" in pending:
                line, _, remaining = pending.partition(b"\n")
                pending = bytearray(remaining)
                assert line == "BRIDGE_INPUT_雪".encode("utf-8"), "不是实验输入"
                count += 1
                temporary = root / "input_count.tmp"
                temporary.write_text(str(count))
                temporary.replace(root / "input_count")
                payload = "BRIDGE_ACK_雪\r\n".encode("utf-8")
                os.write(1, payload[:12])
                os.write(1, payload[12:])
        os.write(1, b"BRIDGE_HEARTBEAT\r\n")


def check(state):
    time.sleep(0.4)
    snapshot(state, "new_ssh_connection")
    for phase in ("stop", "restart", "sigkill"):
        state["stage"] = "lifecycle_" + phase
        store(state)
        hold(state)
        time.sleep(0.4)
        snapshot(state, "attached_before_go_" + phase)
        if phase == "stop":
            command(["systemctl", "--user", "stop", state["web_unit"]])
        elif phase == "restart":
            Path(state["root"], "ready.json").unlink()
            previous = state["web_identity"]["pid"]
            command(["systemctl", "--user", "restart", state["web_unit"]])
            wait_for(lambda: Path(state["root"], "ready.json").exists())
            ready = json.loads(Path(state["root"], "ready.json").read_text())
            assert ready["pid"] != previous
            state["restart_identity"] = identity(ready["pid"])
            state["listen_port"] = ready["port"]
            state["restart_observation"] = client(state, "observe")
            time.sleep(0.4)
            sample = snapshot(state, "live_after_systemctl_restart")
            assert sample["input_count"] == state["expected_input_count"]
            command(["systemctl", "--user", "stop", state["web_unit"]])
        else:
            command(["systemctl", "--user", "kill", "--signal=SIGKILL", state["web_unit"]])
            command(["systemctl", "--user", "reset-failed", state["web_unit"]], check=False)
        wait_for(lambda: not tmux(state, "list-clients")[1].strip())
        command(["systemctl", "--user", "stop", state["client_unit"]], check=False)
        command(["systemctl", "--user", "reset-failed", state["client_unit"]], check=False)
        time.sleep(0.4)
        sample = snapshot(state, "after_go_" + phase)
        assert sample["input_count"] == state["expected_input_count"], "输入被自动重放"
        start_web(state)
        observed = client(state, "observe")
        state.setdefault("observations", []).append(observed)
        time.sleep(0.4)
        sample = snapshot(state, "reconnected_after_" + phase)
        assert sample["input_count"] == state["expected_input_count"], "重连自动输入"
    store(state)
    print(json.dumps(state))


def cleanup(root):
    state = json.loads((root / "state.json").read_text())
    assert state["root"] == str(root)
    assert re.fullmatch(r"persistty_probe_[0-9a-f]{12}", state["session"])
    token = state["session"].removeprefix("persistty_probe_")
    for role in ("client", "web", "tmux"):
        key = "server_unit" if role == "tmux" else role + "_unit"
        assert state[key] == f"persistty-bridge-{token}-{role}.service", "非实验 unit"
    if state.get("pane_scope"):
        assert state["pane_scope"] == state["baseline"]["pane"]["cgroup"].split("/")[-1]
    results = []
    for key in ("client_unit", "web_unit", "server_unit"):
        name = state[key]
        assert re.fullmatch(r"persistty-bridge-[0-9a-f]{12}-(web|tmux|client)\.service", name)
        command(["systemctl", "--user", "stop", name], check=False)
        status, text = command(["systemctl", "--user", "is-active", name], check=False)
        assert status != 0
        command(["systemctl", "--user", "reset-failed", name], check=False)
        results.append({"unit": name, "active": text.strip()})
    if state.get("pane_scope"):
        scope = state["pane_scope"]
        assert re.fullmatch(r"tmux-spawn-[A-Za-z0-9-]+\.scope", scope)
        before = state["baseline"]["pane"]
        if Path(f"/proc/{before['pid']}").exists():
            assert identity(before["pid"]) != before or scope in identity(before["pid"])["cgroup"]
        command(["systemctl", "--user", "stop", scope], check=False)
        status, text = command(["systemctl", "--user", "is-active", scope], check=False)
        assert status != 0
        results.append({"unit": scope, "active": text.strip()})
    for role in ("server", "pane"):
        before = state.get("baseline", {}).get(role)
        if before and Path(f"/proc/{before['pid']}").exists():
            assert identity(before["pid"]) != before, "实验进程残留"
    shutil.rmtree(root)
    assert not root.exists()
    print(json.dumps({"units": results, "root_removed": True}))


def main():
    if sys.flags.optimize:
        raise RuntimeError("禁止优化模式")
    parser = argparse.ArgumentParser()
    parser.add_argument("phase", choices=("start", "check", "cleanup", "workload"))
    parser.add_argument("root", type=Path)
    args = parser.parse_args()
    root = args.root
    assert root.parent == Path("/tmp") and re.fullmatch(r"persistty-bridge-[A-Za-z0-9]{8}", root.name)
    assert not root.is_symlink()
    info = root.stat()
    assert os.getuid() != 0 and info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700
    os.chdir(root)
    try:
        if args.phase == "start":
            start(root)
        elif args.phase == "check":
            check(json.loads((root / "state.json").read_text()))
        elif args.phase == "cleanup":
            cleanup(root)
        else:
            workload(root)
    except Exception:
        if args.phase == "workload":
            raise
        stage = args.phase
        try:
            stage = json.loads((root / "state.json").read_text()).get("stage", stage)
        except Exception:
            pass
        print(json.dumps({"probe_failed": True, "phase": args.phase, "stage": stage}))


if __name__ == "__main__":
    main()
