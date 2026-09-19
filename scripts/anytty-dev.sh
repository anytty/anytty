#!/usr/bin/env bash
# anytty-dev.sh — 隔离的开发/调试环境。
#
# 目标：后续构建、调试新的 anytty 绝不覆盖或连接 hs 上正在使用的
#   ~/.config/anytty、~/.local/state/anytty 与默认 socket。
#
# 做法：所有 XDG 目录、pool socket、plugin registry 全部加 -dev 后缀，
# 并始终显式使用 --socket <dev socket>。子命令：
#   demo       一键体验：构建 + 隔离终端池 + 新 TUI（tui2 + tui2-shell）
#   stop       停止 dev 终端池
#   pool ...   以 dev socket 运行终端池（参数透传；兼容旧 daemon 子命令）
#   cli ...    以 dev socket 运行任意 anytty-dev 子命令（参数透传）
#   build      构建 anytty-dev / tui2 / tui2-shell
#   test       运行新 TUI 测试（clients/tui）
#   endpoints  重新生成 dev 版 endpoints.yaml
#   env        打印当前隔离环境变量
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEV_ROOT="${ANYTTY_DEV_ROOT:-$HOME/.local/share/anytty-dev}"

export XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config/anytty-dev}"
export XDG_STATE_HOME="${XDG_STATE_HOME:-$HOME/.local/state/anytty-dev}"
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-$XDG_STATE_HOME/run}"
export ANYTTY_DEV_SOCKET="${ANYTTY_DEV_SOCKET:-$XDG_RUNTIME_DIR/anytty-v2-wire7-dev.sock}"
# 既有 hs 终端池默认占用 0.0.0.0:41120；dev 终端池只绑 loopback 的独立端口。
export ANYTTY_DIRECT_SIGNALING_LISTEN="${ANYTTY_DIRECT_SIGNALING_LISTEN:-127.0.0.1:44120}"
export ANYTTY_DIRECT_ICE_TCP_LISTEN="${ANYTTY_DIRECT_ICE_TCP_LISTEN:-127.0.0.1:44121}"

# 构建工具链（隔离在用户目录，不写系统路径）。
GOTOOLCHAIN_BIN="${GOTOOLCHAIN_BIN:-$HOME/.local/share/go-toolchains/go1.26.7/bin}"
if [[ -d "$GOTOOLCHAIN_BIN" ]]; then
  export PATH="$GOTOOLCHAIN_BIN:$PATH"
fi
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
export GOSUMDB="${GOSUMDB:-off}"
export GOWORK=off

mkdir -p "$XDG_CONFIG_HOME/anytty" "$XDG_STATE_HOME" "$XDG_RUNTIME_DIR" "$DEV_ROOT/bin"

ENDPOINTS="$XDG_CONFIG_HOME/anytty/endpoints.yaml"
write_endpoints() {
  cat > "$ENDPOINTS" <<YAML
version: 3
default: local
endpoints:
  local:
    label: local-dev
    enabled: true
    connect_mode: auto
    routes:
      local:
        kind: local-unix
        enabled: true
        socket: "$ANYTTY_DEV_SOCKET"
YAML
}

case "${1:-help}" in
  env)
    printf 'XDG_CONFIG_HOME=%s\n' "$XDG_CONFIG_HOME"
    printf 'XDG_STATE_HOME=%s\n' "$XDG_STATE_HOME"
    printf 'XDG_RUNTIME_DIR=%s\n' "$XDG_RUNTIME_DIR"
    printf 'ANYTTY_DEV_SOCKET=%s\n' "$ANYTTY_DEV_SOCKET"
    printf 'DEV_ROOT=%s\n' "$DEV_ROOT"
    ;;
  endpoints)
    write_endpoints
    printf 'wrote %s\n' "$ENDPOINTS"
    ;;
  build)
    [[ -f "$ENDPOINTS" ]] || write_endpoints
    cd "$ROOT_DIR"
    go build -tags anytty_dev_commands -o "$DEV_ROOT/bin/anytty-dev" ./cmd/anytty
    go build -o "$DEV_ROOT/bin/tui2" ./clients/tui/cmd/tui2
    go build -o "$DEV_ROOT/bin/tui2-shell" ./clients/tui/cmd/tui2-shell
    printf 'built into %s/bin\n' "$DEV_ROOT"
    ;;
  test)
    cd "$ROOT_DIR"
    go test ./clients/tui/...
    ;;
  pool | daemon)
    shift
    exec "$DEV_ROOT/bin/anytty-dev" --socket "$ANYTTY_DEV_SOCKET" pool "$@"
    ;;
  cli)
    shift
    exec "$DEV_ROOT/bin/anytty-dev" --socket "$ANYTTY_DEV_SOCKET" "$@"
    ;;
  demo)
    # 一键体验：确保 dev registry + 构建 + 隔离终端池，然后进入 TUI。
    shift
    [[ -f "$ENDPOINTS" ]] || write_endpoints
    if [[ ! -x "$DEV_ROOT/bin/anytty-dev" ]]; then
      bash "${BASH_SOURCE[0]}" build >&2
    fi
    "$DEV_ROOT/bin/anytty-dev" --socket "$ANYTTY_DEV_SOCKET" pool start >/dev/null 2>&1 || true
    export TUI2_BIN="${TUI2_BIN:-$DEV_ROOT/bin/tui2}"
    export TUI2_SHELL="${TUI2_SHELL:-$DEV_ROOT/bin/tui2-shell}"
    exec "$DEV_ROOT/bin/anytty-dev" --socket "$ANYTTY_DEV_SOCKET" "$@"
    ;;
  stop)
    "$DEV_ROOT/bin/anytty-dev" --socket "$ANYTTY_DEV_SOCKET" pool stop
    ;;
  *)
    sed -n '2,16p' "${BASH_SOURCE[0]}"
    ;;
esac
