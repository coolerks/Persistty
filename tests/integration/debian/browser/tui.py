"""合成全屏终端负载，避免测试依赖主机进程和本地化输出。"""

import curses
import os
import sys
import time


def run(screen):
    curses.raw()
    curses.noecho()
    screen.keypad(True)
    screen.nodelay(True)
    curses.mousemask(curses.ALL_MOUSE_EVENTS)
    sys.stdout.write("\x1b[?2004h")
    sys.stdout.flush()
    event = "READY"
    tick = 0
    try:
        while True:
            size = os.get_terminal_size()
            if screen.getmaxyx() != (size.lines, size.columns):
                curses.resizeterm(size.lines, size.columns)
            try:
                key = screen.get_wch()
                if key == curses.KEY_MOUSE:
                    curses.getmouse()
                    event = "MOUSE_RECEIVED"
                elif key == curses.KEY_LEFT:
                    event = "LEFT_RECEIVED"
                elif isinstance(key, str):
                    event = "KEY: " + format(ord(key), "02x")
            except curses.error:
                pass
            screen.erase()
            for row, text in enumerate(("W03_DETERMINISTIC_TUI", "Unicode: 雪", f"TICK: {tick}", event)):
                if row < size.lines:
                    screen.addnstr(row, 0, text, max(1, size.columns - 1))
            screen.refresh()
            tick += 1
            time.sleep(0.05)
    finally:
        sys.stdout.write("\x1b[?2004l")
        sys.stdout.flush()


if __name__ == "__main__":
    curses.wrapper(run)
