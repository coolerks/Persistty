"""Synthetic-only Landlock experiment in the caller's private mktemp directory."""

import json
import os
from pathlib import Path
import subprocess
import sys
import time


def command(argv):
    return subprocess.run(argv, cwd="/", env={"PATH": "/usr/bin:/bin", "HOME": "/nonexistent"},
                          stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=8, check=False)


def run(root):
    if not root.is_dir() or root.is_symlink() or root.stat().st_mode & 0o077:
        raise RuntimeError("private root required")
    binary = root / "cli-probe"
    if not binary.is_file() or binary.is_symlink():
        raise RuntimeError("probe binary missing")
    workspace = root / "workspace"
    outside = root / "outside"
    workspace.mkdir(mode=0o700)
    outside.mkdir(mode=0o700)
    (workspace / "inside.txt").write_text("SENTINEL_INSIDE\n", encoding="utf-8")
    (workspace / "-样本.txt").write_text("SENTINEL_UNICODE\n", encoding="utf-8")
    (outside / "secret.txt").write_text("SENTINEL_OUTSIDE\n", encoding="utf-8")
    (workspace / "escape").symlink_to(outside, target_is_directory=True)

    def attempt(kind, target):
        result = command([str(binary), str(workspace), kind, str(target)])
        return result.returncode, result.stdout[:4096]

    allowed_code, allowed = attempt("cat", workspace / "inside.txt")
    unicode_code, unicode_text = attempt("cat", workspace / "-样本.txt")
    denied_code, denied = attempt("cat", outside / "secret.txt")
    link_code, link = attempt("cat", workspace / "escape" / "secret.txt")
    invalid_code, _ = attempt("unknown", workspace / "inside.txt")
    rg_code, rg = attempt("rg", workspace)
    # Replace a child directory between setup and the next command.
    nested = workspace / "nested"
    nested.mkdir()
    old = workspace / "old-nested"
    nested.rename(old)
    nested.symlink_to(outside, target_is_directory=True)
    swapped_result = command([str(binary), str(workspace), "cat", str(nested / "secret.txt")])
    swapped_code, swapped = swapped_result.returncode, swapped_result.stdout[:4096]
    nested.unlink()
    old.rename(nested)

    fifo = workspace / "blocked-fifo"
    os.mkfifo(fifo, 0o600)
    blocked = subprocess.Popen([str(binary), str(workspace), "cat", str(fifo)],
                               cwd="/", env={"PATH": "/usr/bin:/bin", "HOME": "/nonexistent"},
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        time.sleep(0.15)
        still_running = blocked.poll() is None
        blocked.terminate()
        cancelled_code = blocked.wait(timeout=3)
    finally:
        if blocked.poll() is None:
            blocked.kill()
            blocked.wait(timeout=3)

    git_available = Path("/usr/bin/git").is_file()
    git_ok = False
    git_code = None
    git_errors = {}
    if git_available:
        repo = workspace / "repo"
        repo.mkdir()
        init = command(["/usr/bin/git", "init", "-q", str(repo)])
        if init.returncode == 0:
            (repo / "tracked.txt").write_text("synthetic\n", encoding="utf-8")
            added = command(["/usr/bin/git", "-C", str(repo), "add", "tracked.txt"])
            committed = command(["/usr/bin/git", "-C", str(repo), "-c", "user.name=Probe",
                                 "-c", "user.email=probe@example.invalid", "commit", "-qm", "probe"])
            if added.returncode == 0 and committed.returncode == 0:
                git_result = command([str(binary), str(workspace), "git", str(repo)])
                git_code, git_out = git_result.returncode, git_result.stdout[:4096]
                git_errors = {key: needle in git_result.stderr.lower() for key, needle in {
                    "permission": b"permission denied", "not_repository": b"not a git repository",
                    "unsafe": b"dubious ownership", "missing": b"no such file",
                    "config": b"config", "dev_null": b"/dev/null",
                    "proc": b"/proc", "repo": str(repo).encode(),
                    "git_objects": b"objects", "git_head": b"head"}.items()}
                git_ok = git_code == 0 and len(git_out.strip()) == 40

    checks = {
        "allowed_read": allowed_code == 0 and allowed == b"SENTINEL_INSIDE\n",
        "unicode_dash_path": unicode_code == 0 and unicode_text == b"SENTINEL_UNICODE\n",
        "outside_denied": denied_code != 0 and b"SENTINEL_OUTSIDE" not in denied,
        "symlink_denied": link_code != 0 and b"SENTINEL_OUTSIDE" not in link,
        "invalid_command_denied": invalid_code == 80,
        "cancel_reaped": still_running and cancelled_code == -15,
        "parent_swap_denied": swapped_code != 0 and b"SENTINEL_OUTSIDE" not in swapped,
        "rg_private_data_not_read": rg_code in (0, 2) and b"SENTINEL_INSIDE" in rg
        and b"SENTINEL_OUTSIDE" not in rg,
        "git_read_only": git_ok,
    }
    return {"checks": checks, "git_available": git_available,
            "rg_available": Path("/usr/bin/rg").is_file(),
            "git_errors": git_errors,
            "statuses": {"allowed": allowed_code, "unicode": unicode_code,
                         "cancel": cancelled_code, "denied": denied_code,
                         "symlink": link_code, "invalid": invalid_code,
                         "swap": swapped_code, "rg": rg_code,
                         "git": git_code}}


def main():
    if len(sys.argv) != 2:
        return 2
    root = Path(sys.argv[1])
    try:
        result = run(root)
    except (OSError, RuntimeError, subprocess.TimeoutExpired):
        result = {"probe_failed": True, "stage": "execution"}
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
