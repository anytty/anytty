#!/usr/bin/env bash
# 一键构建并启动 TUI v2（单机本地）。
# 用法：bash clients/tui/scripts/run.sh [宿主参数...]
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT_DIR"

# 定位 Go 工具链：PATH 优先，否则用本机已安装的工具链。
if ! command -v go >/dev/null 2>&1; then
  for candidate in "$HOME"/.local/share/go-toolchains/*/bin /usr/local/go/bin; do
    if [[ -x "$candidate/go" ]]; then
      export PATH="$candidate:$PATH"
      export GOROOT="$(dirname "$candidate")"
      break
    fi
  done
fi
if ! command -v go >/dev/null 2>&1; then
  echo "找不到 go：请安装 Go，或把工具链 bin 目录加入 PATH（例如 ~/.local/share/go-toolchains/*/bin）" >&2
  exit 1
fi

# 代理/校验设置只在未显式配置时兜底，避免影响你自己的环境。
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
export GOSUMDB="${GOSUMDB:-off}"
export GOTOOLCHAIN="${GOTOOLCHAIN:-local}"
export GOWORK="${GOWORK:-off}"
export GOFLAGS="${GOFLAGS:--mod=readonly}"

BIN_DIR="${TUI2_BIN_DIR:-$ROOT_DIR/clients/tui/.bin}"
mkdir -p "$BIN_DIR"

echo "building tui2 host + shell ..." >&2
go build -o "$BIN_DIR/tui2" ./clients/tui/cmd/tui2
go build -o "$BIN_DIR/tui2-shell" ./clients/tui/cmd/tui2-shell

if [[ "${TUI2_BUILD_ONLY:-}" != "" ]]; then
  echo "built: $BIN_DIR/tui2  $BIN_DIR/tui2-shell"
  exit 0
fi

# 在 anytty 会话里嵌套运行时给出提示（宿主本身不拦截，只是提醒）。
if [[ -n "${ANYTTY:-}" ]]; then
  echo "注意：检测到 ANYTTY 环境（嵌套运行）。若显示异常，请在普通终端里运行。" >&2
fi

# 测试/日常：默认读取老版本的 endpoint registry（只读），免去重新配对。
# 显式设置 TUI2_ENDPOINTS 时以你的为准（支持多路径，冒号分隔）。
if [[ -z "${TUI2_ENDPOINTS:-}" && -f "$HOME/.config/anytty/endpoints.yaml" ]]; then
  export TUI2_ENDPOINTS="$HOME/.config/anytty/endpoints.yaml"
  echo "endpoints: 读取 $TUI2_ENDPOINTS（只读；可用 TUI2_ENDPOINTS 覆盖）" >&2
fi

# 日志默认写 $XDG_STATE_HOME/anytty/tui2.log（-log-file/TUI2_LOG_FILE 覆盖），
# 共享层连接诊断只进文件；默认路由 local-unix + 凭据可用的 ssh，
# 需要拨 webrtc/cloud 时设置 TUI2_ROUTES（如 local-unix,direct-webrtc-tcp）。
echo "log: ${TUI2_LOG_FILE:-${XDG_STATE_HOME:-$HOME/.local/state}/anytty/tui2.log}（tail -f 查看）" >&2

# 可用 TUI2_SHELL_CMD 覆盖布局程序（例如 Python 参考实现）。
shell_cmd="${TUI2_SHELL_CMD:-$BIN_DIR/tui2-shell}"
exec "$BIN_DIR/tui2" -shell "$shell_cmd" "$@"
