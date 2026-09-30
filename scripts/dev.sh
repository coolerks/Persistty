#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$ROOT"
case "${1:-}" in
  start|restart|--help) ;;
  stop)
    if [[ $# -ne 1 ]]; then
      printf '%s\n' 'stop 不接受额外选项。' >&2
      exit 2
    fi
    if [[ -x .cache/dev/dev-launcher ]]; then
      exec .cache/dev/dev-launcher "$@"
    fi
    printf '%s\n' '开发服务未启动。'
    exit 0
    ;;
  *) printf '%s\n' '用法: ./scripts/dev.sh start|stop|restart [选项]' >&2; exit 2 ;;
esac
if ! command -v go >/dev/null 2>&1; then
  printf '%s\n' '需要先安装项目要求的 Go 工具链。' >&2
  exit 1
fi
mkdir -p .cache/dev
go build -o .cache/dev/dev-launcher ./scripts/dev
exec .cache/dev/dev-launcher "$@"
