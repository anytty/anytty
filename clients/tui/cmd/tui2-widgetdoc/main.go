// Command tui2-widgetdoc generates the SDK widget catalog
// clients/tui/docs/WIDGETS.zh-CN.md.
//
// Every picture in the catalog is rendered through the real host pipeline
// (runtime.Session -> kernel layout -> compositor), so the glyphs and geometry
// are exactly what a program sees. Only cell text is kept: style tokens are
// resolved against the terminal theme at runtime, so a picture shows glyphs
// and layout, never colors. The output is deterministic (fixed entry order and
// sample data, no timestamps, no map iteration) and drift-checked by
// main_test.go.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/clients/tui/runtime"
	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/widgets"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// catalogPath is relative to the repository root (the directory with go.mod).
const catalogPath = "clients/tui/docs/WIDGETS.zh-CN.md"

func main() {
	check := flag.Bool("check", false, "verify the catalog is up to date; exit 1 with a hint on drift")
	stdout := flag.Bool("stdout", false, "print the catalog to stdout instead of writing the file")
	flag.Parse()

	root, err := repoRoot()
	if err != nil {
		fatal(err)
	}
	doc, err := generate(root)
	if err != nil {
		fatal(err)
	}
	if *stdout {
		if _, err := os.Stdout.Write(doc.Bytes()); err != nil {
			fatal(err)
		}
		return
	}
	path := filepath.Join(root, catalogPath)
	if *check {
		onDisk, err := os.ReadFile(path)
		if err != nil {
			fatal(err)
		}
		if !bytes.Equal(onDisk, doc.Bytes()) {
			fmt.Fprintf(os.Stderr, "tui2-widgetdoc: %s is stale\nrun: go run ./clients/tui/cmd/tui2-widgetdoc\n", catalogPath)
			os.Exit(1)
		}
		return
	}
	if err := os.WriteFile(path, doc.Bytes(), 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "tui2-widgetdoc:", err)
	os.Exit(1)
}

// repoRoot walks up from the working directory until it finds go.mod.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found above %s (run from inside the repository)", dir)
		}
		dir = parent
	}
}

// entry is one catalog item. build == nil means "no picture": the entry is a
// code snippet plus optional generated markdown text.
type entry struct {
	name    string
	purpose string
	code    string
	cols    int
	rows    int
	build   func() *pb.Box
	// glyphs are box-drawing or content glyphs the rendered picture must
	// contain; TestEveryEntryRendersContent enforces them.
	glyphs []string
	text   string
}

type section struct {
	title   string
	entries []entry
}

// generate renders the whole catalog into a buffer. root is used only to read
// the token constants from the widgets source (the single source of truth).
func generate(root string) (*bytes.Buffer, error) {
	sections, err := catalog(root)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	writeHeader(&b)
	for _, sec := range sections {
		fmt.Fprintf(&b, "## %s\n\n", sec.title)
		for _, e := range sec.entries {
			if err := writeEntry(&b, e); err != nil {
				return nil, fmt.Errorf("%s: %w", e.name, err)
			}
		}
	}
	return &b, nil
}

func writeHeader(b *bytes.Buffer) {
	b.WriteString("# SDK 组件图鉴（WIDGETS）\n\n")
	b.WriteString("本图鉴为 `clients/tui/sdk/widgets` 的每个组件给出「渲染图」：图由真实的宿主管线生成\n")
	b.WriteString("（`runtime.NewSession` → `HandleView` → `ComposeFrame`），取的是内核/合成器排出的单元格\n")
	b.WriteString("文本，所以看到的是字形与几何布局。颜色与样式 token 由终端主题在运行时解析，图上不体现，\n")
	b.WriteString("同一张图在深色/浅色主题下一致。\n\n")
	b.WriteString("- 重新生成：`go run ./clients/tui/cmd/tui2-widgetdoc`\n")
	b.WriteString("- 检查漂移（提交前/CI）：`go run ./clients/tui/cmd/tui2-widgetdoc -check`\n")
	b.WriteString("- 只打印不落盘：`go run ./clients/tui/cmd/tui2-widgetdoc -stdout`\n")
	b.WriteString("- 图由生成器产出，不要手改；`clients/tui/cmd/tui2-widgetdoc/main_test.go` 会逐字节比对。\n\n")
	b.WriteString("Go / Python / TS 三语言共享同一套组件与命名（命名风格随语言惯例），组件清单与用法见\n")
	b.WriteString("`clients/tui/docs/SDK-GUIDE.zh-CN.md` §4.1。\n\n")
}

func writeEntry(b *bytes.Buffer, e entry) error {
	fmt.Fprintf(b, "### `%s`\n\n%s\n\n", e.name, e.purpose)
	if e.code != "" {
		fmt.Fprintf(b, "```go\n%s\n```\n\n", e.code)
	}
	if e.build != nil {
		picture, err := renderEntry(e)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "```\n%s\n```\n\n", picture)
	}
	if e.text != "" {
		b.WriteString(strings.TrimRight(e.text, "\n"))
		b.WriteString("\n\n")
	}
	return nil
}

// renderEntry lays one widget out through the real host session and returns
// the frame text with trailing spaces and blank tail rows trimmed.
func renderEntry(e entry) (string, error) {
	cols, rows := e.cols, e.rows
	if cols <= 0 {
		cols = 40
	}
	if rows <= 0 {
		rows = 12
	}
	session := runtime.NewSession(runtime.Options{ViewID: "widgetdoc", Cols: cols, Rows: rows}, &bytes.Buffer{}, &bytes.Buffer{})
	if err := session.HandleView(&pb.View{Epoch: 1, Rev: 1, Root: e.build(), Keys: &pb.Keys{All: true}}); err != nil {
		return "", err
	}
	return frameText(session.ComposeFrame(nil, nil)), nil
}

// frameText flattens a composed frame to plain text: cell text per row,
// trailing spaces trimmed per line and blank tail rows dropped.
func frameText(frame *render.Frame) string {
	lines := make([]string, 0, frame.Rows())
	for y := 0; y < frame.Rows(); y++ {
		var row strings.Builder
		for x := 0; x < frame.Cols(); x++ {
			row.WriteString(frame.CellAt(x, y).Text)
		}
		lines = append(lines, strings.TrimRight(row.String(), " "))
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// styleTokenRows reads the exported Style* constants from tokens.go so the
// generated token table cannot drift from the source.
func styleTokenRows(root string) ([][2]string, error) {
	path := filepath.Join(root, "clients/tui/sdk/widgets/tokens.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	var rows [][2]string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			values, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range values.Names {
				if !strings.HasPrefix(name.Name, "Style") || i >= len(values.Values) {
					continue
				}
				lit, ok := values.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					return nil, fmt.Errorf("tokens.go: %s: %w", name.Name, err)
				}
				rows = append(rows, [2]string{name.Name, value})
			}
		}
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("tokens.go: no Style* constants found")
	}
	return rows, nil
}

// frameRows is the terse constructor for FrameRow slices.
func frameRows(texts ...string) []widgets.FrameRow {
	out := make([]widgets.FrameRow, len(texts))
	for i, text := range texts {
		out[i] = widgets.FrameRow{Text: text}
	}
	return out
}

// rectsText formats solved rects for the text-form geometry entries.
func rectsText(rects []widgets.Rect) string {
	parts := make([]string, len(rects))
	for i, r := range rects {
		parts[i] = fmt.Sprintf("(%d,%d %dx%d)", r.X, r.Y, r.W, r.H)
	}
	return strings.Join(parts, " ")
}

func catalog(root string) ([]section, error) {
	tokens, err := styleTokenRows(root)
	if err != nil {
		return nil, err
	}
	return []section{
		{
			title: "布局 / 容器",
			entries: []entry{
				{
					name:    "SplitLayout",
					purpose: "按权重把空间分成若干 pane，返回每个 pane 与分隔条的 `Rect`；分隔条自己用 `Divider` 画，拖拽由程序处理。",
					code: "layout := widgets.SplitLayout{Orient: \"row\", Weights: []int{1, 2}}\n" +
						"panes, gaps := layout.Rects(36, 6)\n" +
						"for _, p := range panes {\n" +
						"    view.Child(widgets.Card{Title: \"pane\", Width: p.W, Height: p.H}.Build().Pos(p.X, p.Y))\n" +
						"}",
					cols:   36,
					rows:   6,
					build:  splitLayoutView,
					glyphs: []string{"┌", "│", "└"},
				},
				{
					name:    "FloatingLayer",
					purpose: "程序侧浮窗：`Frame` 加 `Pos`，合成时压在常规流之上；`Collapsed` 只留标题行。",
					code: "float := widgets.FloatingLayer{Title: \"floating\", X: 5, Y: 2, Width: 18, Height: 4,\n" +
						"    Rows: []widgets.FrameRow{{Text: \"drag to move\"}, {Text: \"click to collapse\"}}}\n" +
						"view := sdk.Stack(base.Build(), float.Build()).Build()",
					cols:   28,
					rows:   8,
					build:  floatingLayerView,
					glyphs: []string{"┌", "floating", "└"},
				},
				{
					name:    "Card",
					purpose: "带边框的文本框，可选水平/垂直居中；空槽位、对话框的积木，边框由 `Frame` 程序侧绘制。",
					code: "card := widgets.Card{Title: \"empty slot\", Lines: []string{\"press ctrl-b\"},\n" +
						"    Width: 22, Height: 6, Center: true}\n" +
						"view := sdk.Col(card.Build()).Build()",
					cols:   24,
					rows:   6,
					build:  cardView,
					glyphs: []string{"┌", "press ctrl-b", "└"},
				},
				{
					name:    "Frame / FrameRow",
					purpose: "程序侧带行标记的框：顶边嵌标题、左右竖线、底边；`FrameRow` 可带命中 `ID` 与 `Input`。",
					code: "frame := widgets.Frame{Title: \"logs\", Width: 26, Height: 6, Rows: []widgets.FrameRow{\n" +
						"    {Text: \"alpha  ok\"},\n" +
						"    {Text: \"beta   running\", ID: \"log:beta\", Input: []string{\"mouse\"}},\n" +
						"    {Text: \"gamma  dead\"},\n" +
						"}}\n" +
						"view := sdk.Col(frame.Build()).Build()",
					cols:   26,
					rows:   6,
					build:  frameView,
					glyphs: []string{"┌", "logs", "│", "└"},
				},
				{
					name: "BorderBox",
					purpose: "自选边框字形的容器（`BorderNormal` / `BorderRounded` / `BorderThick` / `BorderDouble`），" +
						"标题嵌上边框，内容放 `Child`。",
					code: "box := widgets.BorderBox{Title: \"rounded\", Width: 11, Height: 3,\n" +
						"    Border: widgets.BorderRounded()}\n" +
						"view := sdk.Col(box.Build()).Build()",
					cols:   23,
					rows:   6,
					build:  borderBoxView,
					glyphs: []string{"┌", "╭", "┏", "╔", "└", "╰", "┗", "╚"},
				},
				{
					name: "Divider",
					purpose: "一格宽/高的分隔条：横向 `─`、竖向 `│`（竖向按行构建，整列可见），" +
						"作为 pane 间隙时可拖拽（ID/Input 由程序命中）。",
					code: "view := sdk.Col(\n" +
						"    widgets.Divider{Length: 20}.Build(),\n" +
						"    sdk.Row(panes[0], widgets.Divider{Vertical: true, Length: 3}.Build(), panes[1]),\n" +
						").Build()",
					cols:   22,
					rows:   7,
					build:  dividerView,
					glyphs: []string{"─", "│"},
				},
				{
					name:    "Rect / Distribute",
					purpose: "纯几何：`Rect` 是矩形值，`Distribute` 按整数比例分格子（至少 1 格，余数给最后一个 pane）。",
					code: "sizes := widgets.Distribute(80, []int{1, 2, 1}) // → [20 40 20]\n" +
						"panes, gaps := widgets.SplitLayout{Weights: []int{1, 2}}.Rects(30, 6)",
					text: fmt.Sprintf("无需渲染图，`Distribute` 的输出即可说明：\n\n"+
						"```\nDistribute(80, []int{1, 2, 1}) → %v\nDistribute(10, []int{1, 1, 1}) → %v\nRects(30, 6) → panes %s / gaps %s\n```",
						widgets.Distribute(80, []int{1, 2, 1}),
						widgets.Distribute(10, []int{1, 1, 1}),
						rectsText(panesRects(30, 6)),
						rectsText(gapsRects(30, 6))),
				},
			},
		},
		{
			title: "Chrome / 导航",
			entries: []entry{
				{
					name:    "TabBar / TabItem",
					purpose: "顶部标签条：左侧 workspace 段 + 每个标签一个可点击文本框（active 用 `[title]`）+ 可选 `+`。",
					code: "bar := widgets.TabBar{\n" +
						"    Left:  &widgets.Segment{Text: \" ws \"},\n" +
						"    Items: []widgets.TabItem{{ID: \"tab:main\", Title: \"main\", Active: true}},\n" +
						"    Plus:  true,\n" +
						"}\n" +
						"view := sdk.Col(bar.Build()).Build()",
					cols:   30,
					rows:   2,
					build:  tabBarView,
					glyphs: []string{"[main]", " ws ", "+"},
				},
				{
					name:    "StatusBar / Segment",
					purpose: "一行状态栏：左/右两组 `Segment`，`Width > 0` 时右组贴右边，空间不够先丢尾段再截文本。",
					code: "bar := widgets.StatusBar{\n" +
						"    Left:  []widgets.Segment{{Text: \"▸ connected\", Style: \"ok\"}},\n" +
						"    Right: []widgets.Segment{{Text: \"100x30\"}},\n" +
						"    Width: 32,\n" +
						"}\n" +
						"view := sdk.Col(bar.Build()).Build()",
					cols:   32,
					rows:   2,
					build:  statusBarView,
					glyphs: []string{"▸ connected", "100x30", "│"},
				},
				{
					name:    "TitleBar",
					purpose: "一行标题条：左侧 `Segment` 组 + 右侧按钮组（`Button`），`Width` 右对齐并截断左组。",
					code: "bar := widgets.TitleBar{\n" +
						"    Left:    []widgets.Segment{{Text: \"terminal:local:alpha\"}},\n" +
						"    Buttons: []widgets.Button{{Text: \"[x]\", Hot: \"x\", ID: \"close\"}},\n" +
						"    Width:   32,\n" +
						"}\n" +
						"view := sdk.Col(bar.Build()).Build()",
					cols:   32,
					rows:   2,
					build:  titleBarView,
					glyphs: []string{"terminal:local:alpha", "[x]"},
				},
				{
					name:    "Footer",
					purpose: "底部栏：可选场景 badge + 左侧按键组 + 右侧摘要段，`Width` 右对齐、空间不足先截左组。",
					code: "footer := widgets.Footer{\n" +
						"    Badge:    widgets.Segment{Text: \"NORMAL\"}, HasBadge: true,\n" +
						"    Groups:   []widgets.Segment{{Text: \"ctrl-b 前缀\"}},\n" +
						"    Right:    []widgets.Segment{{Text: \"connected\"}},\n" +
						"    Width:    56,\n" +
						"}\n" +
						"view := sdk.Col(footer.Build()).Build()",
					cols:   56,
					rows:   2,
					build:  footerView,
					glyphs: []string{"NORMAL", "ctrl-b 前缀", "connected"},
				},
				{
					name:    "KeyHint",
					purpose: "页脚左侧的按键提示：当前模式 + 一组键位，模式与键位分别用不同 token 渲染。",
					code: "hint := widgets.KeyHint{Mode: \"PREFIX\", Keys: []string{\"q detach\", \"n new\", \"x close\"}}\n" +
						"view := sdk.Col(hint.Build()).Build()",
					cols:   36,
					rows:   1,
					build:  keyHintView,
					glyphs: []string{"PREFIX", "q detach", "·"},
				},
				{
					name:    "Button",
					purpose: "可点击文本框：协议 `Input(\"mouse\")` + 程序命中处理；`Hot` 字符单独成段方便标快捷键。",
					code: "button := widgets.Button{Text: \"[ Connect ]\", Hot: \"C\", ID: \"connect\"}\n" +
						"view := sdk.Row(button.Build()).Build()",
					cols:   30,
					rows:   1,
					build:  buttonView,
					glyphs: []string{"[ Connect ]", "[ Quit ]"},
				},
				{
					name:    "Picker / PickerRow",
					purpose: "带边框的可选列表（选择器浮层）：行可带命中 `ID`，`Selected` 行加 `▸` 标记。",
					code: "picker := widgets.Picker{Title: \"terminals\", Width: 26, Height: 6,\n" +
						"    Rows: []widgets.PickerRow{\n" +
						"        {Text: \"local:alpha\", ID: \"t1\", Selected: true},\n" +
						"        {Text: \"local:beta\", ID: \"t2\"},\n" +
						"    }}\n" +
						"view := sdk.Col(picker.Build()).Build()",
					cols:   26,
					rows:   6,
					build:  pickerView,
					glyphs: []string{"┌", "▸", "└"},
				},
				{
					name:    "Toast",
					purpose: "临时一行提示：程序把它放进下一帧、再下一帧移除即消失，没有宿主计时器。",
					code: "toast := widgets.Toast{Text: \" copied 2 cells \", Style: \"overlay\", ID: \"toast\"}\n" +
						"view.Child(toast.Build().Pos(2, 3))",
					cols:   24,
					rows:   4,
					build:  toastView,
					glyphs: []string{"copied 2 cells"},
				},
			},
		},
		{
			title: "内容",
			entries: []entry{
				{
					name:    "List",
					purpose: "窗口化单行列表：只渲染可见行，`Height` 含 Header/Footer，选中行加标记；`Follow` 让选中不越窗。",
					code: "list := widgets.List{\n" +
						"    Header: \"terminals\", Items: []string{\"alpha\", \"beta\", \"gamma\"},\n" +
						"    Height: 6, Selected: 2, Footer: \"j/k move\",\n" +
						"}\n" +
						"view := sdk.Col(list.Build()).Build()",
					cols:   28,
					rows:   6,
					build:  listView,
					glyphs: []string{"terminals", "▸ gamma", "j/k move"},
				},
				{
					name:    "VirtualList",
					purpose: "`List` 的富行版本（`ListRow`）：逐行样式、`Disabled` 置灰，其余窗口化规则相同。",
					code: "list := widgets.VirtualList{Height: 6, Selected: 1, Rows: []widgets.ListRow{\n" +
						"    {Text: \"alpha  connected\", ID: \"r0\"},\n" +
						"    {Text: \"gamma  exited\", ID: \"r2\", Disabled: true},\n" +
						"}}\n" +
						"view := sdk.Col(list.Build()).Build()",
					cols:   28,
					rows:   6,
					build:  virtualListView,
					glyphs: []string{"sessions", "▸ beta", "enter attach"},
				},
				{
					name:    "Table / Column",
					purpose: "表头 + 对齐/截断的单元格：`Column` 支持固定宽、`Flex` 弹性、对齐与 `Format`，CJK 截断安全。",
					code: "table := widgets.Table{Width: 40, Rule: true, Selected: 1, Columns: []widgets.Column{\n" +
						"    {Title: \"session\", Flex: 2},\n" +
						"    {Title: \"status\", Flex: 1},\n" +
						"    {Title: \"uptime\", Align: widgets.AlignRight},\n" +
						"}}\n" +
						"view := sdk.Col(table.Build()).Build()",
					cols:   40,
					rows:   6,
					build:  tableView,
					glyphs: []string{"session", "status", "uptime", "─"},
				},
				{
					name:    "TextInput",
					purpose: "单行编辑器状态 + 纯 `Build`：`HandleKey` 吃编辑键，焦点行渲染光标格（样式剥离后仍是该字符）。",
					code: "input := widgets.TextInput{Value: []rune(\"anytty\"), Cursor: 3,\n" +
						"    Width: 24, Focused: true, Placeholder: \"type a name\"}\n" +
						"view := sdk.Col(input.Build()).Build()",
					cols:   24,
					rows:   2,
					build:  textInputView,
					glyphs: []string{"anytty", "type a session name"},
				},
				{
					name:    "TextArea",
					purpose: "多行编辑器：光标是 rune 下标，`RowOffset`/`ColOffset` 控制可视窗口，只渲染可见行。",
					code: "area := widgets.TextArea{Value: []rune(\"line one\\nline two\"), Cursor: 8,\n" +
						"    Width: 24, Height: 4, Focused: true}\n" +
						"view := sdk.Col(area.Build()).Build()",
					cols:   24,
					rows:   4,
					build:  textAreaView,
					glyphs: []string{"line one", "line three"},
				},
				{
					name:    "Modal",
					purpose: "程序侧对话框：`Frame` + `FloatingLayer`，可居中、可加遮罩；保留/移除由程序在下一帧决定。",
					code: "modal := widgets.Modal{Title: \"close session\", Width: 22, Height: 6,\n" +
						"    Center: true, Backdrop: true, ParentWidth: 34, ParentHeight: 10,\n" +
						"    Rows: []widgets.FrameRow{{Text: \"Close alpha?\"}, {Text: \"\"},\n" +
						"        {Text: \"[ Enter ] yes\"}, {Text: \"[ Esc ]   no\"}}}\n" +
						"view := modal.Build().Build()",
					cols:   34,
					rows:   10,
					build:  modalView,
					glyphs: []string{"┌", "close session", "Close alpha?", "└"},
				},
				{
					name:    "Menu / MenuItem",
					purpose: "键盘菜单浮层：热键/禁用/分隔线，`Move`/`Hotkey` 改选中，`Value` 给程序派发。",
					code: "menu := widgets.Menu{Title: \"actions\", Width: 20, X: 6, Y: 1, Selected: 1,\n" +
						"    Items: []widgets.MenuItem{\n" +
						"        {ID: \"copy\", Label: \"Copy\", Hotkey: \"c\"},\n" +
						"        {Separator: true},\n" +
						"        {ID: \"kill\", Label: \"Kill\", Disabled: true},\n" +
						"    }}\n" +
						"view := sdk.Stack(base.Build(), menu.Build()).Build()",
					cols:   28,
					rows:   8,
					build:  menuView,
					glyphs: []string{"┌", "▸ Paste", "───", "└"},
				},
				{
					name:    "RichText / Span",
					purpose: "一行内多段样式：相邻同一样式自动合并，`WrapText`/`WrapSpans` 做 CJK 安全换行。",
					code: "line := widgets.RichText{Width: 30, Spans: []widgets.Span{\n" +
						"    widgets.Styled(\"status \", \"muted\"),\n" +
						"    widgets.Styled(\"ok\", \"success\"),\n" +
						"    widgets.Styled(\" · 2 panes\", \"accent\"),\n" +
						"}}\n" +
						"view := sdk.Col(line.Build()).Build()",
					cols:   30,
					rows:   1,
					build:  richTextView,
					glyphs: []string{"status", "ok", "2 panes"},
				},
				{
					name:    "Scrollbar",
					purpose: "按比例的滚动指示器：thumb 大小 = `Visible*Length/Total`，`OffsetAt` 把点击位置折成 offset；纵横两用。",
					code: "bar := widgets.Scrollbar{Total: 100, Visible: 20, Offset: 30, Height: 6}\n" +
						"view := sdk.Col(bar.Build()).Build()\n" +
						"offset := bar.OffsetAt(clickY)",
					cols:   24,
					rows:   10,
					build:  scrollbarView,
					glyphs: []string{"┃", "━"},
				},
			},
		},
		{
			title: "交互",
			entries: []entry{
				{
					name:    "ContextMenu",
					purpose: "锚定在点击处的 `Menu`：`Open` 记录点击并把菜单夹回父矩形内，`Build` 输出定位浮层；程序决定点击是否关闭。",
					code: "cm := widgets.ContextMenu{Menu: widgets.Menu{Title: \"row actions\", Width: 20,\n" +
						"    Items: []widgets.MenuItem{{ID: \"open\", Label: \"Open\", Hotkey: \"o\"}}},\n" +
						"    ParentWidth: 30, ParentHeight: 9, Margin: 1}\n" +
						"cm.Open(mx, my)\n" +
						"view := sdk.Stack(base.Build(), cm.Build()).Build()",
					cols:   30,
					rows:   9,
					build:  contextMenuView,
					glyphs: []string{"┌", "row actions", "Open", "└"},
				},
				{
					name:    "HitRegion / Hit",
					purpose: "命中测试：矩形 + id，`Hit` 返回第一个包含 (x,y) 的区域。",
					code: "regions := []widgets.HitRegion{{ID: \"ok\", X: 2, Y: 1, W: 8, H: 1}}\n" +
						"if region, ok := widgets.Hit(regions, mx, my); ok {\n" +
						"    dispatch(region.ID)\n" +
						"}",
				},
				{
					name:    "ListRegions / TableRegions",
					purpose: "把 `List`/`Table` 的可见行翻译成命中区域；`RowAt` 由 y 反查行号（越界返回 false）。",
					code: "regions := widgets.ListRegions(list, originX, originY, width)\n" +
						"regions = widgets.TableRegions(table, originX, originY)\n" +
						"row, ok := list.RowAt(my - originY)",
				},
				{
					name:    "ApplyWheel / OnWheel / TableScroll",
					purpose: "滚轮增量 → 新 offset：`ApplyWheel` 是纯函数，`List.OnWheel`/`VirtualList.OnWheel`/`TableScroll` 是现成接线。",
					code: "offset = widgets.ApplyWheel(offset, total, visible, delta)\n" +
						"list.OnWheel(ev) // 就地更新 Offset\n" +
						"offset = widgets.TableScroll(offset, total, visible, delta)",
				},
				{
					name:    "Drag / SelectRange",
					purpose: "拖拽选择：`Begin`/`Update`/`End` 得到归一化矩形；`SelectRange`/`ListSelectRange` 把 y0..y1 折成行区间。",
					code: "var drag widgets.Drag\n" +
						"drag.Begin(x, y, \"left\")\n" +
						"drag.Update(x2, y2)\n" +
						"rect, ok := drag.End()\n" +
						"lo, hi, ok := widgets.ListSelectRange(list, y0, y1)",
				},
				{
					name:    "ClickTracker",
					purpose: "单/双/三击判定：同一位置在双击窗口内连续点击返回 1/2/3，超时或移位重新计数。",
					code: "var clicks widgets.ClickTracker\n" +
						"n := clicks.Click(x, y, at) // 1 单击、2 双击、3 三击",
				},
				{
					name:    "Hover",
					purpose: "悬停追踪：`Update` 吃鼠标位置返回命中 id，`StyleFor(id)` 只给被悬停的 id 叠加 hover 样式。",
					code: "var hover widgets.Hover\n" +
						"id := hover.Update(mx, my)\n" +
						"box.Style(hover.StyleFor(id))",
				},
			},
		},
		{
			title: "表单",
			entries: []entry{
				{
					name:    "Form / Field",
					purpose: "字段列 + 焦点导航 + 聚合校验：`Next`/`Prev` 跳过禁用/只读字段，`Values`/`Validate` 读值与查错。",
					code: "form := widgets.Form{Width: 32, Focus: 0, Fields: []widgets.Field{\n" +
						"    {ID: \"name\", Label: \"Name\", Required: true,\n" +
						"        Input: &widgets.TextInput{Value: []rune(\"anytty\"), Width: 28}},\n" +
						"    {ID: \"email\", Label: \"Email\", Help: \"used for alerts\",\n" +
						"        Input: &widgets.TextInput{Placeholder: \"you@example.com\", Width: 28}},\n" +
						"}}\n" +
						"view := sdk.Col(form.Build()).Build()",
					cols:   32,
					rows:   5,
					build:  formView,
					glyphs: []string{"Name *", "anytty", "used for alerts"},
				},
				{
					name:    "Validator",
					purpose: "校验器是 `func(string) string`（返回错误文案，空串通过），可直接挂到 `Field.Validate`。",
					code: "field.Validate = widgets.All(\n" +
						"    widgets.Required(\"required\"),\n" +
						"    widgets.MinLen(3, \"too short\"),\n" +
						")\n" +
						"msg := field.Validate(field.Text())",
					text: "| 校验器 | 语义 |\n" +
						"|---|---|\n" +
						"| `Required(msg)` | 去空白后为空则失败 |\n" +
						"| `MinLen(n, msg)` / `MaxLen(n, msg)` | rune 长度下限 / 上限 |\n" +
						"| `Pattern(re, msg)` | 正则必须匹配 |\n" +
						"| `Email(msg)` | 邮箱形状 |\n" +
						"| `IntRange(min, max, msg)` | 整数字面量且落在闭区间 |\n" +
						"| `OneOf(values, msg)` | 必须是枚举之一 |\n" +
						"| `All(...)` | 全部通过；返回第一个错误 |\n" +
						"| `Custom(fn)` | 自定义 `func(string) string` |\n" +
						"| `DayValidator(min, max, msg)` | `YYYY-MM-DD` 且落在日期区间 |",
				},
				{
					name:    "Select / Option",
					purpose: "下拉选择状态机 + 纯 `Build`：`Open` 时 `Build` 输出下拉列表，支持 typeahead、禁用项与占位符。",
					code: "select := widgets.Select{Label: \"profile\", Width: 22, Value: \"dev\",\n" +
						"    Options: []widgets.Option{{Value: \"dev\", Label: \"development\"},\n" +
						"        {Value: \"prod\", Label: \"production\"}}}\n" +
						"select.Open = true // Build 输出下拉列表\n" +
						"view := sdk.Col(select.Build()).Build()",
					cols:   22,
					rows:   6,
					build:  selectView,
					glyphs: []string{"profile", "▾", "▸ production"},
				},
				{
					name:    "Calendar",
					purpose: "Mon-first 月历（6 行 × 7 列）：`Month`/`Selected`/`Today`/`Marked` 都由调用方给，不读挂钟。",
					code: "cal := widgets.Calendar{\n" +
						"    Month:    widgets.Date{Year: 2026, Month: 9, Day: 1},\n" +
						"    Selected: widgets.Date{Year: 2026, Month: 9, Day: 18},\n" +
						"    Today:    widgets.Date{Year: 2026, Month: 9, Day: 22},\n" +
						"}\n" +
						"view := sdk.Col(cal.Build()).Build()",
					cols:   22,
					rows:   8,
					build:  calendarView,
					glyphs: []string{"September 2026", "Mo", "Su", "18"},
				},
				{
					name:    "Date",
					purpose: "纯日期值（年/月/日）：`AddDays`/`AddMonths` 进位并夹住月末，`ParseDate` 拒绝被 time 静默归一化的日期。",
					code: "date, err := widgets.ParseDate(\"2026-09-18\")\n" +
						"next := date.AddMonths(1)\n" +
						"leap := widgets.Date{Year: 2024, Month: 1, Day: 31}.AddMonths(1)",
					text: fmt.Sprintf("日期运算示例（生成时计算）：\n\n"+
						"```\nDate{2026, 1, 31}.AddMonths(1) → %s\nDate{2024, 1, 31}.AddMonths(1) → %s\nDate{2026, 9, 18}.AddDays(7)  → %s\nParseDate(\"2024-02-30\")       → error（非法日期）\n```",
						widgets.Date{Year: 2026, Month: 1, Day: 31}.AddMonths(1),
						widgets.Date{Year: 2024, Month: 1, Day: 31}.AddMonths(1),
						widgets.Date{Year: 2026, Month: 9, Day: 18}.AddDays(7)),
				},
				{
					name:    "DateField",
					purpose: "`TextInput` 加日期校验：`NewDateField` 自动挂 `DayValidator`，`Date()` 读回解析后的日期。",
					code: "input := &widgets.TextInput{Placeholder: \"YYYY-MM-DD\", Width: 28}\n" +
						"field := widgets.NewDateField(\"due\", \"Due date\", input)\n" +
						"field.Min = widgets.Date{Year: 2026, Month: 1, Day: 1}\n" +
						"date, ok := field.Date()",
				},
			},
		},
		{
			title: "反馈",
			entries: []entry{
				{
					name:    "ProgressBar",
					purpose: "定宽进度条：`Value`/`Max` + 可选 label/百分比，glyph 可换；没有计时器，进度由程序推进。",
					code: "bar := widgets.ProgressBar{Label: \"upload \", Value: 7, Max: 10, Width: 20}\n" +
						"view := sdk.Col(bar.Build()).Build()",
					cols:   32,
					rows:   1,
					build:  progressBarView,
					glyphs: []string{"upload", "█", "░", "70%"},
				},
				{
					name:    "Spinner",
					purpose: "动画的一帧：`Frame` 由程序自己的 tick 推进（宿主没有 timer），默认盲文帧序列可整体替换。",
					code: "spinner := widgets.Spinner{Frame: 3, Label: \" starting\"}\n" +
						"view := sdk.Col(spinner.Build()).Build()",
					cols:   20,
					rows:   4,
					build:  spinnerView,
					glyphs: []string{"⠋", "⠙", "⠹", "⠸", "starting"},
				},
				{
					name:    "Badge",
					purpose: "小号状态标签（状态、计数、按键标记）：一个文本框 + 样式 token。",
					code: "badge := widgets.Badge{Text: \"● live\", Style: \"ok\", ID: \"live\"}\n" +
						"view := sdk.Row(badge.Build()).Build()",
					cols:   24,
					rows:   1,
					build:  badgeView,
					glyphs: []string{"● live", "beta", "2 panes"},
				},
				{
					name: "Tags / Tag",
					purpose: "一行标签：`Tag` 可带命中 `ID` 与样式，`Separator` 决定间隔；点击派发同 Button/Picker。" +
						"（没有独立的 `Tabs` 组件，标签页请用 `TabBar`）",
					code: "tags := widgets.Tags{Separator: \" · \", Items: []widgets.Tag{\n" +
						"    {Text: \"go\", ID: \"t1\"},\n" +
						"    {Text: \"python\", ID: \"t2\"},\n" +
						"}}\n" +
						"view := sdk.Col(tags.Build()).Build()",
					cols:   24,
					rows:   1,
					build:  tagsView,
					glyphs: []string{"go", " · ", "ts"},
				},
			},
		},
		{
			title: "图表",
			entries: []entry{
				{
					name:    "Sparkline",
					purpose: "一行八分块迷你图：`Min`/`Max` 可选，缺省从数据推导；超宽序列按 `Width` 分桶平均。",
					code: "spark := widgets.Sparkline{Values: []float64{1, 3, 2, 5, 4, 8, 6, 9, 7, 4, 6, 8}, Width: 16}\n" +
						"view := sdk.Col(spark.Build()).Build()",
					cols:   20,
					rows:   2,
					build:  sparklineView,
					glyphs: []string{"▁", "█", "cpu 12 samples"},
				},
				{
					name:    "BarChart",
					purpose: "柱状图：纵向用八分块子格（`Height` 行）或横向整块；`Max <= 0` 自动量程，`Labels` 可选。",
					code: "chart := widgets.BarChart{Values: []float64{3, 7, 5, 9, 6, 2}, Height: 5,\n" +
						"    Labels: []string{\"mo\", \"tu\", \"we\", \"th\", \"fr\", \"sa\"}, Width: 26}\n" +
						"view := sdk.Col(chart.Build()).Build()",
					cols:   26,
					rows:   6,
					build:  barChartView,
					glyphs: []string{"█", "▇", "▅", "mo", "sa"},
				},
				{
					name:    "Heatmap",
					purpose: "矩阵热力图：值可以是锯齿行，缺格按最低档渲染；`RowLabels`/`ColLabels` 可选。",
					code: "heat := widgets.Heatmap{RowLabels: []string{\"am\", \"pm\", \"night\"}, Values: [][]float64{\n" +
						"    {0, 1, 3, 6, 9, 7, 4, 2, 1, 0},\n" +
						"    {2, 4, 8, 9, 7, 5, 3, 2, 1, 0},\n" +
						"}}\n" +
						"view := sdk.Col(heat.Build()).Build()",
					cols:   16,
					rows:   3,
					build:  heatmapView,
					glyphs: []string{"·", "░", "▒", "▓", "█", "am"},
				},
				{
					name:    "Meter / Gauge",
					purpose: "浮点值的标签条：复用 `ProgressBar` 的字形，`ShowValue` 显示 `值/上限`；`Gauge` 是别名。",
					code: "meter := widgets.Meter{Label: \"disk \", Value: 72, Max: 100, Width: 16, ShowValue: true}\n" +
						"view := sdk.Col(meter.Build()).Build()",
					cols:   30,
					rows:   1,
					build:  meterView,
					glyphs: []string{"disk", "█", "░", "72/100"},
				},
				{
					name:    "Legend",
					purpose: "一行图例：每个 `LegendItem` 是彩色 marker + label，与图表配合解释颜色含义。",
					code: "legend := widgets.Legend{Items: []widgets.LegendItem{\n" +
						"    {Label: \"ok\", Color: \"ok\"},\n" +
						"    {Label: \"dead\", Color: \"danger\"},\n" +
						"}}\n" +
						"view := sdk.Col(legend.Build()).Build()",
					cols:   30,
					rows:   1,
					build:  legendView,
					glyphs: []string{"■", "ok", "warning", "dead"},
				},
			},
		},
		{
			title: "主题 / 非视觉",
			entries: []entry{
				{
					name: "Theme",
					purpose: "设计 token 集合：`DarkTheme()` 映射宿主 token（随宿主主题），`LightTheme()` 固定浅色；" +
						"`ThemedXxx(theme)` 预填各组件。",
					code: "theme := widgets.DarkTheme()\n" +
						"list := widgets.ThemedList(theme)\n" +
						"table := widgets.ThemedTable(theme)\n" +
						"style := widgets.WithBold(\"fg:#6d3bd4\")",
					text: tokenTable(tokens),
				},
				{
					name:    "Format*",
					purpose: "数字/文本格式化纯函数：输出固定、无 ANSI、无 CJK 宽度逻辑，图表与状态栏直接复用。",
					code: "text := widgets.FormatBytes(1536)          // 1.5 KB\n" +
						"count := widgets.FormatCount(1234567)     // 1,234,567\n" +
						"uptime := widgets.FormatDuration(elapsed) // 1m30s",
					text: formatTable(),
				},
				{
					name:    "Terminal",
					purpose: "终端内容源：`sdk.Terminal(sourceID)` 绑定宿主终端，渲染由宿主的真实 PTY 决定。",
					code: "view := sdk.Col(\n" +
						"    sdk.Terminal(\"terminal:local:alpha\"),\n" +
						").Build()",
					text: "图鉴跳过 `sdk.Terminal`：它需要宿主提供真实的终端源（PTY），离线渲染只会得到空白内容。",
				},
			},
		},
	}, nil
}

func tokenTable(tokens [][2]string) string {
	var b strings.Builder
	b.WriteString("可用 token 常量（来自 `clients/tui/sdk/widgets/tokens.go`）：\n\n")
	b.WriteString("| 常量 | 值 |\n|---|---|\n")
	for _, row := range tokens {
		fmt.Fprintf(&b, "| `%s` | `%s` |\n", row[0], row[1])
	}
	return b.String()
}

type formatExample struct {
	call   string
	output string
}

func formatExamples() []formatExample {
	scaled, suffix := widgets.ScaleValue(1500)
	return []formatExample{
		{"FormatBytes(1536)", widgets.FormatBytes(1536)},
		{"FormatCount(1234567)", widgets.FormatCount(1234567)},
		{"FormatDuration(90*time.Second)", widgets.FormatDuration(90 * time.Second)},
		{"FormatPercent(0.125, 1)", widgets.FormatPercent(0.125, 1)},
		{"FormatFloat(3.14159, 2, 6)", widgets.FormatFloat(3.14159, 2, 6)},
		{"ScaleValue(1500)", widgets.FormatFloat(scaled, 1, 0) + suffix},
		{`PadLeft("42", 5)`, widgets.PadLeft("42", 5)},
		{`PadRight("42", 5)`, widgets.PadRight("42", 5)},
	}
}

func formatTable() string {
	var b strings.Builder
	b.WriteString("| 调用 | 输出 |\n|---|---|\n")
	for _, example := range formatExamples() {
		fmt.Fprintf(&b, "| `%s` | `%s` |\n", example.call, example.output)
	}
	return b.String()
}

func panesRects(width, height int) []widgets.Rect {
	panes, _ := widgets.SplitLayout{Weights: []int{1, 2}}.Rects(width, height)
	return panes
}

func gapsRects(width, height int) []widgets.Rect {
	_, gaps := widgets.SplitLayout{Weights: []int{1, 2}}.Rects(width, height)
	return gaps
}

// ---------------------------------------------------------------------------
// Pictures

func splitLayoutView() *pb.Box {
	layout := widgets.SplitLayout{Weights: []int{1, 2}, Gap: 1}
	panes, gaps := layout.Rects(36, 6)
	view := sdk.Box()
	for i, pane := range panes {
		card := widgets.Card{Title: fmt.Sprintf("pane %d", i+1), Lines: []string{"slot"}, Width: pane.W, Height: pane.H}
		view.Child(card.Build().Pos(pane.X, pane.Y))
	}
	for _, gap := range gaps {
		view.Child(widgets.Divider{Vertical: true, Length: gap.H}.Build().Pos(gap.X, gap.Y))
	}
	return view.Build()
}

func floatingLayerView() *pb.Box {
	base := widgets.Frame{Title: "workspace", Width: 28, Height: 8, Rows: frameRows("pane 1", "pane 2", "pane 3", "pane 4", "pane 5")}
	float := widgets.FloatingLayer{Title: "floating", X: 5, Y: 2, Width: 18, Height: 4, Rows: frameRows("drag to move", "click to collapse")}
	return sdk.Stack(base.Build(), float.Build()).Build()
}

func cardView() *pb.Box {
	card := widgets.Card{Title: "empty slot", Lines: []string{"press ctrl-b"}, Width: 22, Height: 6, Center: true}
	return sdk.Col(card.Build()).Build()
}

func frameView() *pb.Box {
	frame := widgets.Frame{
		Title:  "logs",
		Width:  26,
		Height: 6,
		Rows: []widgets.FrameRow{
			{Text: "alpha  ok"},
			{Text: "beta   running", ID: "log:beta", Input: []string{"mouse"}},
			{Text: "gamma  dead"},
		},
	}
	return sdk.Col(frame.Build()).Build()
}

func borderBoxView() *pb.Box {
	box := func(title string, set widgets.BorderSet) *sdk.Builder {
		return widgets.BorderBox{Title: title, Width: 11, Height: 3, Border: set}.Build()
	}
	return sdk.Col(
		sdk.Row(box("normal", widgets.BorderNormal()), sdk.Text(" "), box("rounded", widgets.BorderRounded())),
		sdk.Row(box("thick", widgets.BorderThick()), sdk.Text(" "), box("double", widgets.BorderDouble())),
	).Build()
}

func dividerView() *pb.Box {
	return sdk.Col(
		sdk.Text("horizontal"),
		widgets.Divider{Length: 20}.Build(),
		sdk.Text("vertical"),
		sdk.Row(
			sdk.Col(sdk.Text("L1"), sdk.Text("L2"), sdk.Text("L3")),
			widgets.Divider{Vertical: true, Length: 3}.Build(),
			sdk.Col(sdk.Text("R1"), sdk.Text("R2"), sdk.Text("R3")),
		),
	).Build()
}

func tabBarView() *pb.Box {
	bar := widgets.TabBar{
		Left:  &widgets.Segment{Text: " ws "},
		Items: []widgets.TabItem{{ID: "tab:main", Title: "main", Active: true}, {ID: "tab:logs", Title: "logs"}},
		Plus:  true,
	}
	return sdk.Col(bar.Build(), sdk.Text(" body ")).Build()
}

func statusBarView() *pb.Box {
	bar := widgets.StatusBar{
		Left:  []widgets.Segment{{Text: "▸ connected", Style: "ok"}, {Text: "2 panes"}},
		Right: []widgets.Segment{{Text: "100x30"}, {Text: "14:02"}},
		Width: 32,
	}
	return sdk.Col(sdk.Text(" body"), bar.Build()).Build()
}

func titleBarView() *pb.Box {
	bar := widgets.TitleBar{
		Left:    []widgets.Segment{{Text: "terminal:local:alpha"}},
		Buttons: []widgets.Button{{Text: "[x]", Hot: "x", ID: "close"}},
		Width:   32,
	}
	return sdk.Col(bar.Build(), sdk.Text(" body ")).Build()
}

func footerView() *pb.Box {
	footer := widgets.Footer{
		Badge:    widgets.Segment{Text: "NORMAL"},
		HasBadge: true,
		Groups:   []widgets.Segment{{Text: "ctrl-b 前缀"}, {Text: "ctrl-q 退出"}},
		Right:    []widgets.Segment{{Text: "2 panes"}, {Text: "connected"}},
		Width:    56,
	}
	return sdk.Col(sdk.Text(" body"), footer.Build()).Build()
}

func keyHintView() *pb.Box {
	hint := widgets.KeyHint{Mode: "PREFIX", Keys: []string{"q detach", "n new", "x close"}}
	return sdk.Col(hint.Build()).Build()
}

func buttonView() *pb.Box {
	return sdk.Row(
		widgets.Button{Text: "[ Connect ]", Hot: "C", ID: "connect"}.Build(),
		sdk.Text("  "),
		widgets.Button{Text: "[ Quit ]", Hot: "Q", ID: "quit"}.Build(),
	).Build()
}

func pickerView() *pb.Box {
	picker := widgets.Picker{
		Title:  "terminals",
		Width:  26,
		Height: 6,
		Rows: []widgets.PickerRow{
			{Text: "local:alpha", ID: "t1", Selected: true},
			{Text: "local:beta", ID: "t2"},
			{Text: "dev:build", ID: "t3"},
			{Text: "old:dead", ID: "t4", Selectable: false},
		},
	}
	return sdk.Col(picker.Build()).Build()
}

func toastView() *pb.Box {
	view := sdk.Box()
	view.Child(sdk.Text(" body row 1").Pos(0, 0))
	view.Child(sdk.Text(" body row 2").Pos(0, 1))
	view.Child(widgets.Toast{Text: " copied 2 cells ", Style: "overlay", ID: "toast"}.Build().Pos(2, 3))
	return view.Build()
}

func listView() *pb.Box {
	list := widgets.List{
		Header:   "terminals",
		Items:    []string{"alpha", "beta", "gamma", "delta", "epsilon"},
		Height:   6,
		Selected: 2,
		Footer:   "j/k move",
	}
	return sdk.Col(list.Build()).Build()
}

func virtualListView() *pb.Box {
	list := widgets.VirtualList{
		Header:   "sessions",
		Height:   6,
		Selected: 1,
		Rows: []widgets.ListRow{
			{Text: "alpha  connected", ID: "r0"},
			{Text: "beta   connected", ID: "r1"},
			{Text: "gamma  exited", ID: "r2", Disabled: true},
			{Text: "delta  connected", ID: "r3"},
			{Text: "old    dead", ID: "r4", Disabled: true},
		},
		Footer: "enter attach",
	}
	return sdk.Col(list.Build()).Build()
}

func tableView() *pb.Box {
	table := widgets.Table{
		Width:    40,
		Rule:     true,
		Selected: 1,
		Columns: []widgets.Column{
			{Title: "session", Flex: 2},
			{Title: "status", Flex: 1},
			{Title: "uptime", Align: widgets.AlignRight},
		},
		Rows: [][]string{
			{"alpha", "ok", "2h10m"},
			{"beta", "ok", "18m"},
			{"gamma", "dead", "3d"},
		},
	}
	return sdk.Col(table.Build()).Build()
}

func textInputView() *pb.Box {
	value := widgets.TextInput{Value: []rune("anytty"), Cursor: 3, Width: 24, Focused: true, ID: "name"}
	placeholder := widgets.TextInput{Placeholder: "type a session name", Width: 24}
	return sdk.Col(value.Build(), placeholder.Build()).Build()
}

func textAreaView() *pb.Box {
	area := widgets.TextArea{
		Value:   []rune("line one\nline two\nline three"),
		Cursor:  8,
		Width:   24,
		Height:  4,
		Focused: true,
	}
	return sdk.Col(area.Build()).Build()
}

func modalView() *pb.Box {
	modal := widgets.Modal{
		ID:           "confirm",
		Title:        "close session",
		Width:        22,
		Height:       6,
		Center:       true,
		Backdrop:     true,
		ParentWidth:  34,
		ParentHeight: 10,
		Rows: []widgets.FrameRow{
			{Text: "Close alpha?"},
			{Text: ""},
			{Text: "[ Enter ] yes"},
			{Text: "[ Esc ]   no"},
		},
	}
	return modal.Build().Build()
}

func menuView() *pb.Box {
	base := widgets.Frame{Title: "pane", Width: 28, Height: 8, Rows: frameRows("content")}
	menu := widgets.Menu{
		Title:    "actions",
		Width:    20,
		X:        6,
		Y:        1,
		Selected: 1,
		Items: []widgets.MenuItem{
			{ID: "copy", Label: "Copy", Hotkey: "c"},
			{ID: "paste", Label: "Paste", Hotkey: "v"},
			{Separator: true},
			{ID: "close", Label: "Close", Hotkey: "x"},
			{ID: "kill", Label: "Kill", Hotkey: "k", Disabled: true},
		},
	}
	return sdk.Stack(base.Build(), menu.Build()).Build()
}

func richTextView() *pb.Box {
	line := widgets.RichText{Width: 30, Spans: []widgets.Span{
		widgets.Styled("status ", "muted"),
		widgets.Styled("ok", "success"),
		widgets.Styled(" · ", "muted"),
		widgets.Styled("2 panes", "accent"),
		widgets.Styled(" · ", "muted"),
		widgets.Styled("14:02", "muted"),
	}}
	return sdk.Col(line.Build()).Build()
}

func scrollbarView() *pb.Box {
	return sdk.Col(
		sdk.Text("vertical"),
		widgets.Scrollbar{Total: 100, Visible: 20, Offset: 30, Height: 6}.Build(),
		sdk.Text("horizontal"),
		widgets.Scrollbar{Total: 100, Visible: 20, Offset: 30, Width: 24}.Build(),
	).Build()
}

func contextMenuView() *pb.Box {
	cm := widgets.ContextMenu{
		Menu: widgets.Menu{
			Title: "row actions",
			Width: 20,
			Items: []widgets.MenuItem{
				{ID: "open", Label: "Open", Hotkey: "o"},
				{ID: "copy", Label: "Copy", Hotkey: "c"},
				{Separator: true},
				{ID: "delete", Label: "Delete", Hotkey: "d"},
			},
		},
		ParentWidth:  30,
		ParentHeight: 9,
		Margin:       1,
	}
	cm.Open(12, 5)
	base := widgets.Frame{Title: "table", Width: 30, Height: 9, Rows: frameRows("row 1", "row 2", "row 3")}
	return sdk.Stack(base.Build(), cm.Build()).Build()
}

func formView() *pb.Box {
	form := widgets.Form{
		Width: 32,
		Focus: 0,
		Fields: []widgets.Field{
			{ID: "name", Label: "Name", Required: true, Input: &widgets.TextInput{Value: []rune("anytty"), Width: 28}},
			{ID: "email", Label: "Email", Help: "used for alerts", Input: &widgets.TextInput{Placeholder: "you@example.com", Width: 28}},
		},
	}
	return sdk.Col(form.Build()).Build()
}

func selectView() *pb.Box {
	closed := widgets.Select{
		Label: "profile",
		Width: 22,
		Value: "dev",
		Options: []widgets.Option{
			{Value: "dev", Label: "development"},
			{Value: "prod", Label: "production"},
		},
	}
	open := widgets.Select{
		Width: 22,
		Open:  true,
		Value: "prod",
		Options: []widgets.Option{
			{Value: "dev", Label: "development"},
			{Value: "prod", Label: "production"},
			{Value: "old", Label: "legacy", Disabled: true},
		},
	}
	return sdk.Col(closed.Build(), sdk.Text("open:"), open.Build()).Build()
}

func calendarView() *pb.Box {
	cal := widgets.Calendar{
		Month:    widgets.Date{Year: 2026, Month: 9, Day: 1},
		Selected: widgets.Date{Year: 2026, Month: 9, Day: 18},
		Today:    widgets.Date{Year: 2026, Month: 9, Day: 22},
		Marked:   map[widgets.Date]string{{Year: 2026, Month: 9, Day: 9}: "release"},
		Width:    22,
	}
	return sdk.Col(cal.Build()).Build()
}

func progressBarView() *pb.Box {
	bar := widgets.ProgressBar{Label: "upload ", Value: 7, Max: 10, Width: 20}
	return sdk.Col(bar.Build()).Build()
}

func spinnerView() *pb.Box {
	return sdk.Col(
		widgets.Spinner{Frame: 0, Label: " starting"}.Build(),
		widgets.Spinner{Frame: 1, Label: " starting"}.Build(),
		widgets.Spinner{Frame: 2, Label: " starting"}.Build(),
		widgets.Spinner{Frame: 3, Label: " starting"}.Build(),
	).Build()
}

func badgeView() *pb.Box {
	return sdk.Row(
		widgets.Badge{Text: "● live", Style: "ok", ID: "b1"}.Build(),
		sdk.Text(" "),
		widgets.Badge{Text: "beta", Style: "accent"}.Build(),
		sdk.Text(" "),
		widgets.Badge{Text: "2 panes", Style: "muted"}.Build(),
	).Build()
}

func tagsView() *pb.Box {
	tags := widgets.Tags{
		Items:     []widgets.Tag{{Text: "go", ID: "t1"}, {Text: "python", ID: "t2"}, {Text: "ts", ID: "t3"}},
		Separator: " · ",
	}
	return sdk.Col(tags.Build()).Build()
}

func sparklineView() *pb.Box {
	spark := widgets.Sparkline{Values: []float64{1, 3, 2, 5, 4, 8, 6, 9, 7, 4, 6, 8}, Width: 16}
	return sdk.Col(sdk.Text("cpu 12 samples"), spark.Build()).Build()
}

func barChartView() *pb.Box {
	chart := widgets.BarChart{
		Values:   []float64{3, 7, 5, 9, 6, 2},
		Labels:   []string{"mo", "tu", "we", "th", "fr", "sa"},
		Width:    26,
		Height:   5,
		Selected: 3,
	}
	return sdk.Col(chart.Build()).Build()
}

func heatmapView() *pb.Box {
	heat := widgets.Heatmap{
		Values: [][]float64{
			{0, 1, 3, 6, 9, 7, 4, 2, 1, 0},
			{2, 4, 8, 9, 7, 5, 3, 2, 1, 0},
			{1, 2, 3, 4, 5, 6, 5, 4, 3, 2},
		},
		RowLabels: []string{"am", "pm", "night"},
	}
	return sdk.Col(heat.Build()).Build()
}

func meterView() *pb.Box {
	meter := widgets.Meter{Label: "disk ", Value: 72, Max: 100, Width: 16, ShowValue: true}
	return sdk.Col(meter.Build()).Build()
}

func legendView() *pb.Box {
	legend := widgets.Legend{Items: []widgets.LegendItem{
		{Label: "ok", Color: "ok"},
		{Label: "warning", Color: "warning"},
		{Label: "dead", Color: "danger"},
	}}
	return sdk.Col(legend.Build()).Build()
}
