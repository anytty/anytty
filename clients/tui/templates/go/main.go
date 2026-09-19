// Command template: the smallest useful anytty layout program in Go: a process
// speaking the tui2 protocol on stdin/stdout through the official Go SDK. The
// host owns the terminal; the program owns every box. Draws tab bar · two
// terminal slots · footer · picker; handles Ctrl-T, Ctrl-F, click, Esc/Ctrl-Q.
// 改哪里: header/footer 文字 · key 键位 · pick 的 enter 行为 · view 的盒子树。
// Dev loop: bash clients/tui/scripts/dev.sh clients/tui/templates/go
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

type app struct {
	client                             *sdk.Client
	tabs                               []string
	slots                              [2]string // terminal source ids, empty = unbound
	sources                            []*pb.Source
	active, focus, pickSel, cols, rows int
	picker                             bool
	status                             string
}

func main() {
	a := &app{tabs: []string{"main"}, cols: 80, rows: 24}
	a.client = sdk.New(os.Stdin, os.Stdout, sdk.Handlers{
		Hello:   func(h *pb.Hello) { a.cols, a.rows = int(h.GetCols()), int(h.GetRows()); a.commit() },
		Sources: a.onSources,
		Key:     func(k *pb.KeyEvent) { a.key(k.GetKey()) },
		Mouse: func(m *pb.MouseEvent) {
			if m.GetAction() == "press" {
				a.mouse(m.GetNode())
				a.commit()
			}
		},
		Resize: func(cols, rows int) { a.cols, a.rows = cols, rows; a.commit() },
		Notice: func(level, message string) { a.status = message; a.commit() },
	})
	if err := a.client.Loop(); err != nil {
		fmt.Fprintln(os.Stderr, "template:", err)
		os.Exit(1)
	}
}
func (a *app) newTab() { a.active = len(a.tabs); a.tabs = append(a.tabs, fmt.Sprint(len(a.tabs)+1)) }
func (a *app) commit() {
	keys := sdk.Keys{Claim: []string{"ctrl-t", "ctrl-f", "esc"}}
	if a.picker {
		keys = sdk.Keys{All: true}
	}
	_ = a.client.Commit(a.view(), keys)
}

// view declares the whole UI: tab bar · slots · footer · picker overlay.
func (a *app) view() *pb.Box {
	// 改哪里: tab 条（图标、标题、+ 按钮）。
	header := sdk.Row(sdk.Text(" main ").Style("tab_inactive")).ID("header").Height(1)
	for i, name := range a.tabs {
		label, style := " "+name+" ", "tab_inactive"
		if i == a.active {
			label, style = "["+name+"]", "tab_active"
		}
		header.Child(sdk.Text(label).ID(fmt.Sprintf("tab:%d", i)).Style(style).Input("mouse"))
	}
	header.Child(sdk.Text(" + ").ID("tab:new").Style("tab_inactive"))
	body := sdk.Box().Flow("row").Flex(1)
	for i, source := range a.slots {
		id := fmt.Sprintf("slot:%d", i)
		if source == "" {
			body.Child(sdk.Col(sdk.Text("  [空槽]"), sdk.Text("  Ctrl-F 选择终端")).
				ID(id).Style("muted").Flex(1).Input("mouse"))
		} else {
			body.Child(sdk.Terminal(source).ID(id).Flex(1).Focused(i == a.focus).
				Input("key", "paste", "wheel").Props(map[string]string{"chrome.title": "muted"}))
		}
	}
	// 改哪里: footer 键位提示。
	hints := "Ctrl-F picker · Ctrl-T tab · click focus · Esc quit"
	if a.picker {
		hints = "↑/↓ select · Enter bind · Esc close"
	}
	status := fmt.Sprintf("tab %d/%d", a.active+1, len(a.tabs))
	if a.status != "" {
		status = a.status + " │ " + status
	}
	root := sdk.Col(header, body, sdk.Row(sdk.Text(hints).Style("muted"), sdk.Text(" ").Flex(1),
		sdk.Text(sdk.Truncate(status, a.cols/2)).Style("status")).ID("footer").Height(1))
	if !a.picker {
		return root.Build()
	}
	// 改哪里: picker 列出 sources；+ New terminal 走 terminal.create。
	overlay := sdk.Col(sdk.Text(" select a terminal").Style("muted")).ID("picker")
	for i := 0; i <= len(a.sources); i++ {
		label, state, style, marker := "+ New terminal", "create", "", "  "
		if i < len(a.sources) {
			src := a.sources[i]
			label, state = src.GetTitle(), src.GetEndpoint()+" · live"
			if src.GetExited() {
				state = src.GetEndpoint() + " · exited"
			}
		}
		if i == a.pickSel {
			style, marker = "selection", "> "
		}
		overlay.Child(sdk.Text(marker + label + "  " + state).ID(fmt.Sprintf("pick:%d", i)).Style(style).Input("mouse"))
	}
	overlay.Child(sdk.Text(" enter bind · esc close").Style("muted"))
	width, height := min(46, a.cols-2), len(a.sources)+4
	root.Child(overlay.Pos((a.cols-width)/2, (a.rows-height)/2).Width(width).Height(height))
	return root.Build()
}

// 改哪里: 全部键位。
func (a *app) key(key string) {
	if a.picker {
		switch key {
		case "up":
			a.pickSel = max(0, a.pickSel-1)
		case "down":
			a.pickSel = min(len(a.sources), a.pickSel+1)
		case "enter":
			a.pick()
		case "esc", "ctrl-f":
			a.picker = false
		}
	} else {
		switch key {
		case "ctrl-f":
			a.picker, a.pickSel = true, 0
		case "ctrl-t":
			a.newTab()
		case "esc":
			_, _ = a.client.Emit("system.quit", nil, nil)
		}
	}
	a.commit()
}

// 改哪里: picker 的 enter 行为（terminal.create / terminal.attach 示例）。
func (a *app) pick() {
	if a.pickSel >= len(a.sources) {
		a.picker, a.status = false, "creating terminal"
		_, _ = a.client.Emit("terminal.create", &pb.MethodParams{Endpoint: "local"}, func(resp *pb.Response) {
			a.status = "create failed: " + resp.GetError()
			if resp.GetOk() {
				a.slots[a.focus] = "terminal:" + resp.GetData().GetEndpoint() + ":" + resp.GetData().GetId()
				a.status = "bound " + resp.GetData().GetId()
			}
			a.commit()
		})
		return
	}
	src := a.sources[a.pickSel]
	endpoint, id, fit := "local", src.GetId(), true
	if parts := strings.SplitN(src.GetId(), ":", 3); len(parts) == 3 {
		endpoint, id = parts[1], parts[2]
	}
	a.picker, a.status = false, "binding "+id
	_, _ = a.client.Emit("terminal.attach", &pb.MethodParams{Endpoint: endpoint, Id: id, Fit: &fit},
		func(resp *pb.Response) {
			a.status = "attach failed: " + resp.GetError()
			if resp.GetOk() {
				a.slots[a.focus], a.status = src.GetId(), "bound "+id
			}
			a.commit()
		})
}
func (a *app) mouse(node string) {
	var index int
	switch {
	case strings.HasPrefix(node, "slot:"):
		if _, err := fmt.Sscanf(node, "slot:%d", &index); err == nil && index >= 0 && index < len(a.slots) {
			a.focus = index
		}
	case strings.HasPrefix(node, "pick:"):
		_, _ = fmt.Sscanf(node, "pick:%d", &index)
		a.pickSel = min(max(index, 0), len(a.sources))
		a.pick()
	}
}
func (a *app) onSources(items []*pb.Source) {
	a.sources = a.sources[:0]
	for _, src := range items {
		if src.GetKind() == "terminal" {
			a.sources = append(a.sources, src)
		}
	}
	if a.slots[0] == "" && a.slots[1] == "" {
		// 改哪里: 冷启动自动绑定第一个终端；想总是弹选择器就改成 a.picker, a.pickSel = true, 0。
		if len(a.sources) > 0 {
			a.slots[0] = a.sources[0].GetId()
		} else {
			a.picker, a.pickSel = true, 0
		}
	}
	a.commit()
}
