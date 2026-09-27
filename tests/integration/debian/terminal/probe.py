"""仅在自身临时目录执行的 Debian tmux/systemd 生命周期探针。"""

import argparse
import fcntl
import json
import os
from pathlib import Path
import pty
import select
import shutil
import signal
import stat
import struct
import subprocess
import sys
import tempfile
import termios
import time
import uuid


def command(args, check=True, env=None):
    # 捕获到本实验目录，避免命令异常输出导致无界内存占用。
    limit = 1024 * 1024
    with tempfile.TemporaryFile(dir=".") as stdout, tempfile.TemporaryFile(dir=".") as stderr:
        completed = subprocess.run(args, stdout=stdout, stderr=stderr, timeout=20, env=env)
        stdout.seek(0)
        stderr.seek(0)
        out, err = stdout.read(limit + 1), stderr.read(limit + 1)
        if len(out) > limit or len(err) > limit:
            raise RuntimeError("命令输出超过单流 1 MiB 限制")
        result = subprocess.CompletedProcess(args, completed.returncode,
                                             out.decode("utf-8", errors="replace"),
                                             err.decode("utf-8", errors="replace"))
    if check and result.returncode:
        raise RuntimeError(f"命令失败 {args[0]}: {result.stderr[:1500]}")
    return result


def wait_for(test):
    deadline = time.monotonic() + 8
    while time.monotonic() < deadline:
        if test():
            return
        time.sleep(0.1)
    raise RuntimeError("就绪轮询超时")


def identity(pid):
    text = Path(f"/proc/{pid}/stat").read_text()
    fields = text[text.rfind(")") + 2:].split()
    return {"pid": int(pid), "start_ticks": int(fields[19]),
            "cgroup": Path(f"/proc/{pid}/cgroup").read_text().strip()}


def tmux(state, *args, check=True):
    return command([state["tmux"], "-N", "-S", state["socket"], *args], check,
                   {**os.environ, "LD_LIBRARY_PATH": state["library_path"]})


def unit(state, name, args):
    command(["systemd-run", "--user", "--quiet", "--unit=" + name,
             "--property=Type=exec", "--property=KillMode=control-group",
             "--property=Restart=no", "--property=RuntimeMaxSec=180",
             "--property=TimeoutStopSec=5", "--property=UMask=0077",
             "--setenv=LD_LIBRARY_PATH=" + state["library_path"], *args])


def snapshot(state, label):
    facts = tmux(state, "list-panes", "-t", "=probe", "-F",
                 "#{pid}|#{pane_pid}|#{pane_width}x#{pane_height}|#{pane_id}").stdout.strip().split("|")
    assert len(facts) == 4, f"无法查询唯一 pane: {facts}"
    server, pane = int(facts[0]), int(facts[1])
    result = {"label": label, "server": identity(server), "pane": identity(pane),
              "heartbeat": int(Path(state["root"], "heartbeat").read_text()),
              "dimensions": facts[2], "pane_id": facts[3],
              "clients": tmux(state, "list-clients", "-F",
                              "#{client_pid}:#{client_flags}:#{client_width}x#{client_height}").stdout.strip().splitlines()}
    previous = state.get("baseline")
    if previous:
        assert result["server"] == previous["server"], "server 身份变化"
        assert result["pane"] == previous["pane"], "pane 身份变化"
        last = state["samples"][-1]
        assert result["heartbeat"] > last["heartbeat"], "自上次采样以来心跳未增长"
    assert state["server_unit"] in result["server"]["cgroup"].replace("\\x2d", "-"), result
    assert state["web_unit"] not in result["pane"]["cgroup"], result
    scope = result["pane"]["cgroup"].split("/")[-1]
    if scope != state["server_unit"]:
        assert scope.startswith("tmux-spawn-") and scope.endswith(".scope"), result
        state["pane_scope"] = scope
    state.setdefault("baseline", result)
    state.setdefault("samples", []).append(result)
    store(state)
    return result


def store(state):
    Path(state["root"], "state.json").write_text(json.dumps(state, indent=2) + "\n")


def start(root):
    token = uuid.uuid4().hex[:12]
    state = {"root": str(root), "socket": str(root / "tmux.sock"),
             "server_unit": f"persistty-spike-{token}-tmux.service",
             "web_unit": f"persistty-spike-{token}-web.service",
             "observer_unit": f"persistty-spike-{token}-observer.service"}
    store(state)
    command(["apt-get", "download", "tmux=3.5a-3"])
    packages = list(root.glob("tmux_*.deb"))
    assert len(packages) == 1
    command(["dpkg-deb", "-x", str(packages[0]), str(root / "package")])
    dependency = "libevent-core-2.1-7t64=2.1.13-stable-1~deb13u1"
    command(["apt-get", "download", dependency])
    libraries = list(root.glob("libevent-core-*.deb"))
    assert len(libraries) == 1
    command(["dpkg-deb", "-x", str(libraries[0]), str(root / "package")])
    state["dependency"] = dependency
    state["library_path"] = str(root / "package/usr/lib/x86_64-linux-gnu")
    state["tmux"] = str(root / "package/usr/bin/tmux")
    env = {**os.environ, "LD_LIBRARY_PATH": state["library_path"]}
    state["version"] = command([state["tmux"], "-V"], env=env).stdout.strip()
    state["systemd"] = command(["systemd-run", "--version"]).stdout.splitlines()[0]
    store(state)
    missing = command([state["tmux"], "-N", "-S", str(root / "missing.sock"),
                       "new-session", "-d", "-s", "must_not_exist"], check=False, env=env)
    assert missing.returncode != 0 and not (root / "missing.sock").exists()
    state["missing_server_rejected"] = True
    config = root / "tmux.conf"
    config.write_text("set -g exit-empty off\nset -g exit-unattached off\n"
                      "set -g status off\nset -g history-limit 5000\n")
    unit(state, state["server_unit"], [state["tmux"], "-D", "-S", state["socket"], "-f", str(config)])
    wait_for(lambda: tmux(state, "show-options", "-s", "exit-empty", check=False).returncode == 0)
    time.sleep(1)
    assert tmux(state, "show-options", "-s", "exit-empty").stdout.strip() == "exit-empty off"
    assert not tmux(state, "list-sessions").stdout.strip()
    state["empty_server_alive"] = True
    state["socket_mode"] = oct(stat.S_IMODE(Path(state["socket"]).stat().st_mode))
    assert stat.S_IMODE(Path(state["socket"]).stat().st_mode) & 0o077 == 0
    script = str(Path(__file__).resolve())
    tmux(state, "new-session", "-d", "-s", "probe", "-x", "100", "-y", "30",
         sys.executable, script, "heartbeat", str(root))
    wait_for(lambda: (root / "heartbeat").exists())
    snapshot(state, "before_attach")
    unit(state, state["web_unit"], [sys.executable, script, "attach", str(root)])
    wait_for(lambda: len(tmux(state, "list-clients").stdout.splitlines()) == 1)
    attach_pid = int(tmux(state, "list-clients", "-F", "#{client_pid}").stdout.strip())
    state["attach_identity"] = identity(attach_pid)
    assert state["web_unit"] in state["attach_identity"]["cgroup"].replace("\\x2d", "-"), state["attach_identity"]
    time.sleep(0.5)
    snapshot(state, "attached")
    command(["systemctl", "--user", "stop", state["web_unit"]])
    wait_for(lambda: not tmux(state, "list-clients").stdout.strip())
    time.sleep(0.5)
    snapshot(state, "after_web_stop")
    # transient unit 停止后会被卸载，重新启动同一名称模拟新 Web 实例。
    unit(state, state["web_unit"], [sys.executable, script, "attach", str(root)])
    wait_for(lambda: len(tmux(state, "list-clients").stdout.splitlines()) == 1)
    command(["systemctl", "--user", "restart", state["web_unit"]])
    wait_for(lambda: len(tmux(state, "list-clients").stdout.splitlines()) == 1)
    time.sleep(0.5)
    snapshot(state, "after_web_restart")
    command(["systemctl", "--user", "kill", "--signal=SIGKILL", state["web_unit"]])
    wait_for(lambda: not tmux(state, "list-clients").stdout.strip())
    time.sleep(0.5)
    snapshot(state, "after_web_sigkill")
    command(["systemctl", "--user", "reset-failed", state["web_unit"]])
    unit(state, state["web_unit"], [sys.executable, script, "attach", str(root)])
    wait_for(lambda: len(tmux(state, "list-clients").stdout.splitlines()) == 1)
    unit(state, state["observer_unit"], [sys.executable, script, "observer", str(root)])
    wait_for(lambda: len(tmux(state, "list-clients").stdout.splitlines()) == 2)
    time.sleep(0.5)
    observed = snapshot(state, "with_small_readonly_observer")
    assert observed["dimensions"] == "100x30", "观察端改变了尺寸"
    assert any("read-only" in flags and "ignore-size" in flags for flags in observed["clients"])
    for name in (state["observer_unit"], state["web_unit"]):
        command(["systemctl", "--user", "stop", name])
    time.sleep(0.5)
    snapshot(state, "after_all_attach_disappear")
    state["history_capture"] = tmux(state, "capture-pane", "-p", "-t",
                                   state["baseline"]["pane_id"], "-S", "-5000").stdout.strip().splitlines()[-3:]
    store(state)
    print(json.dumps(state, indent=2))


def heartbeat(root):
    deadline = time.monotonic() + 170
    count = 0
    while time.monotonic() < deadline:
        count += 1
        temporary = root / "heartbeat.tmp"
        temporary.write_text(str(count))
        temporary.replace(root / "heartbeat")
        print(f"probe-heartbeat {count}", flush=True)
        time.sleep(0.2)


def attach(root, observer):
    state = json.loads((root / "state.json").read_text())
    pid, master = pty.fork()
    if pid == 0:
        os.environ["TERM"] = "xterm-256color"
        size = (10, 40) if observer else (30, 100)
        fcntl.ioctl(0, termios.TIOCSWINSZ, struct.pack("HHHH", *size, 0, 0))
        args = [state["tmux"], "-N", "-S", state["socket"], "attach-session", "-t", "=probe"]
        if observer:
            args += ["-f", "read-only,ignore-size"]
        os.execv(state["tmux"], args)
    try:
        deadline = time.monotonic() + 120
        while time.monotonic() < deadline:
            readable, _, _ = select.select([master], [], [], 0.2)
            if readable:
                try:
                    if not os.read(master, 65536):
                        break
                except OSError:
                    break
    finally:
        os.close(master)
        try:
            os.kill(pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
        os.waitpid(pid, 0)


def cleanup(root):
    state = json.loads((root / "state.json").read_text())
    results = []
    for key in ("observer_unit", "web_unit", "server_unit"):
        name = state[key]
        assert name.startswith("persistty-spike-") and name.endswith(".service")
        command(["systemctl", "--user", "stop", name], check=False)
        active = command(["systemctl", "--user", "is-active", name], check=False)
        assert active.returncode != 0, "自身 unit 清理失败"
        command(["systemctl", "--user", "reset-failed", name], check=False)
        results.append({"unit": name, "active": active.stdout.strip()})
    if state.get("pane_scope"):
        scope = state["pane_scope"]
        assert scope.startswith("tmux-spawn-") and scope.endswith(".scope")
        before = state["baseline"]["pane"]
        alive = Path(f"/proc/{before['pid']}").exists()
        if alive:
            current = identity(before["pid"])
            assert current != before or scope in current["cgroup"]
        results.append({"pane_after_server_stop": alive,
                        "scope_before_explicit_cleanup": command(
                            ["systemctl", "--user", "is-active", scope], check=False).stdout.strip()})
        command(["systemctl", "--user", "stop", scope], check=False)
        active = command(["systemctl", "--user", "is-active", scope], check=False)
        assert active.returncode != 0, "自身派生 scope 清理失败"
        results.append({"unit": scope, "active": active.stdout.strip()})
    # 停止唯一实验 server unit 后，已记录 pane/server 不再运行。
    for key in ("server", "pane"):
        before = state.get("baseline", {}).get(key)
        if before and Path(f"/proc/{before['pid']}").exists():
            assert identity(before["pid"]) != before, "实验进程未清理"
    print(json.dumps({"cleanup": results, "root_removed": str(root)}, indent=2))
    shutil.rmtree(root)


def main():
    if sys.flags.optimize:
        raise RuntimeError("禁止优化模式，实验断言不得被禁用")
    parser = argparse.ArgumentParser()
    parser.add_argument("phase", choices=("start", "check", "cleanup", "heartbeat", "attach", "observer"))
    parser.add_argument("root", type=Path)
    args = parser.parse_args()
    assert not args.root.is_symlink(), "临时根不能为链接"
    root = args.root.resolve()
    assert root.parent == Path("/tmp") and root.name.startswith("persistty-terminal-"), "不是批准的临时根"
    info = root.stat()
    assert info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700
    assert os.getuid() != 0, "禁止 root 实验"
    os.chdir(root)
    if args.phase == "start":
        try:
            start(root)
        except Exception:
            cleanup(root)
            raise
    elif args.phase == "cleanup":
        cleanup(root)
    elif args.phase == "heartbeat":
        heartbeat(root)
    elif args.phase in ("attach", "observer"):
        attach(root, args.phase == "observer")
    else:
        state = json.loads((root / "state.json").read_text())
        sample = snapshot(state, "new_ssh_connection")
        store(state)
        print(json.dumps(sample, indent=2))


if __name__ == "__main__":
    main()
