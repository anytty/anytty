package history

import vterm "github.com/anytty/anytty/vterm/vterm"

// TerminalSemanticSource 是 history 唯一允许消费的 terminal semantics source。
// domain owner 是 vterm：vterm 解码 PTY bytes 和 resize events，history 只接收
// 已排序 transaction，不能 replay raw。
type TerminalSemanticSource = vterm.TerminalSemanticSource

// TerminalSemanticSize 携带一个 semantic transaction 关联的 PTY size。history
// 只把它用于 vterm/linehist 投影边界，不能用它重写 sealed logical lines。
type TerminalSemanticSize = vterm.TerminalSemanticSize

// TerminalSemanticTransaction 是一个已排序的 vterm write/resize boundary；linehist
// 只消费其中的 eviction seal 信号与 clear/resize/alt 边界。
type TerminalSemanticTransaction = vterm.TerminalSemanticTransaction

// TerminalSemanticOp 是 vterm 产出的已排序 terminal operation；当前 linehist
// 默认路径不再用它重放第二份 screen model。
type TerminalSemanticOp = vterm.TerminalSemanticOp

// TerminalSemanticCell 是 vterm cell payload，用来构建 logical-line 和 hot-screen
// projection content；需要进入 history 时必须复制成 history-owned payload。
type TerminalSemanticCell = vterm.TerminalSemanticCell

// TerminalSemanticCellRun 是 vterm 在 scroll-out proof 中保留的 styled text run。
// history 只能把它展开成 history-owned cells，不能把 run slice 当成独立 truth。
type TerminalSemanticCellRun = vterm.TerminalSemanticCellRun

// TerminalSemanticStyle 是 vterm style token 形状。history 保存 terminal
// semantics，默认主题解析交给 viewer。
type TerminalSemanticStyle = vterm.TerminalSemanticStyle

// TerminalSemanticScrollOut 表示同一个 transaction 内 primary screen 行离开可见区域。
// linehist 把它作为 seal-on-eviction 输入，而不是第二份 store。
type TerminalSemanticScrollOut = vterm.TerminalSemanticScrollOut

// TerminalSemanticScrollbackRowAppend 是 vterm 挂在 ordered op 上的 scrollback
// row payload，例如 ED2 清屏时旧可见行离开 primary viewport。
type TerminalSemanticScrollbackRowAppend = vterm.ScrollbackRowAppend

// TerminalSemanticFrame 是 vterm 为一个 transaction 产出的 current primary 或 alt
// fixed-grid frame。旧 screen-backed path 已不再用它创建 ordinary sealed history。
type TerminalSemanticFrame = vterm.TerminalSemanticFrame
