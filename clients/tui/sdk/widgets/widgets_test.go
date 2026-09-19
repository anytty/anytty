package widgets

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func findBox(root *pb.Box, id string) *pb.Box {
	if root == nil {
		return nil
	}
	if root.GetId() == id {
		return root
	}
	for _, child := range root.GetChildren() {
		if found := findBox(child, id); found != nil {
			return found
		}
	}
	return nil
}

func boxText(b *pb.Box) string {
	if b == nil || b.GetContent() == nil {
		return ""
	}
	return b.GetContent().GetText()
}

func TestTabBarStructure(t *testing.T) {
	bar := TabBar{
		Left: &Segment{Text: " local ", Style: "chrome", ID: "workspace"},
		Items: []TabItem{
			{ID: "tab:0", Title: "1:1", Active: true},
			{ID: "tab:1", Title: "2:2"},
		},
		Plus:   true,
		PlusID: "tab:new",
	}
	root := bar.Build().ID("header").Height(1).Build()

	workspace := findBox(root, "workspace")
	if workspace == nil || workspace.GetStyle() != "chrome" {
		t.Fatalf("workspace box = %+v", workspace)
	}
	active := findBox(root, "tab:0")
	if active == nil || active.GetStyle() != DefaultActiveStyle || boxText(active) != "[1:1]" {
		t.Fatalf("active tab = %+v", active)
	}
	if len(active.GetInput()) != 1 || active.GetInput()[0] != "mouse" {
		t.Fatalf("active tab input = %v", active.GetInput())
	}
	inactive := findBox(root, "tab:1")
	if inactive == nil || inactive.GetStyle() != DefaultInactiveStyle || boxText(inactive) != " 2:2 " {
		t.Fatalf("inactive tab = %+v", inactive)
	}
	plus := findBox(root, "tab:new")
	if plus == nil || boxText(plus) != " + " {
		t.Fatalf("plus box = %+v", plus)
	}
}

func TestStatusBarStructureAndAlignment(t *testing.T) {
	bar := StatusBar{
		Left:  []Segment{{Text: "NORMAL", Style: "status"}},
		Right: []Segment{{Text: "tab 1/2", Style: "status"}, {Text: "12:00", Style: "muted"}},
		Width: 30,
	}
	root := bar.Build().Build()
	children := root.GetChildren()
	if len(children) != 6 {
		t.Fatalf("status bar children = %d, want 6 (left, sep, spacer, 2 right segments + sep)", len(children))
	}
	if got := boxText(children[0]); got != "NORMAL" {
		t.Fatalf("left = %q", got)
	}
	if got := boxText(children[1]); got != " "+DefaultSeparator+" " {
		t.Fatalf("separator = %q", got)
	}
	spacer := boxText(children[2])
	if sdk.DisplayWidth(spacer) <= 0 {
		t.Fatalf("spacer = %q, want a positive gap", spacer)
	}
	right := boxText(children[3]) + boxText(children[4]) + boxText(children[5])
	if got := sdk.DisplayWidth("NORMAL" + " " + DefaultSeparator + " " + spacer + right); got != 30 {
		t.Fatalf("status row width = %d, want 30 (%q)", got, "NORMAL"+" "+DefaultSeparator+" "+spacer+right)
	}
	if got := bar.Text(); got != "NORMAL "+DefaultSeparator+" tab 1/2 "+DefaultSeparator+" 12:00" {
		t.Fatalf("StatusBar.Text = %q", got)
	}
}

func TestStatusBarTrimsLeftFirst(t *testing.T) {
	bar := StatusBar{
		Left:  []Segment{{Text: "LONGHINTS", Style: "muted"}, {Text: "extra", Style: "muted"}},
		Right: []Segment{{Text: "12:00", Style: "muted"}},
		Width: 12,
	}
	root := bar.Build().Build()
	var text string
	for _, child := range root.GetChildren() {
		text += boxText(child)
	}
	if sdk.DisplayWidth(text) > 12 {
		t.Fatalf("trimmed row width = %d (%q), want <= 12", sdk.DisplayWidth(text), text)
	}
	if !contains(boxText(root.GetChildren()[len(root.GetChildren())-1]), "12:00") {
		t.Fatalf("right segment must survive trimming: %q", text)
	}
}

func TestCardCentersAndBorders(t *testing.T) {
	card := Card{
		ID:     "slot-1",
		Title:  "空槽",
		Lines:  []string{"Ctrl-F 选择终端", "Ctrl-P 面板命令"},
		Width:  20,
		Height: 8,
		Style:  DefaultBorderStyle,
		Center: true,
	}
	root := card.Build().Build()
	if root.GetSize().GetWidth() != 20 || root.GetSize().GetHeight() != 8 {
		t.Fatalf("card size = %+v", root.GetSize())
	}
	children := root.GetChildren()
	if len(children) != 8 {
		t.Fatalf("card rows = %d, want 8 (declared height)", len(children))
	}
	if top := boxText(children[0]); !contains(top, "空槽") || !strings.HasPrefix(top, "┌─") || !strings.HasSuffix(top, "┐") {
		t.Fatalf("card top rule = %q", top)
	}
	if children[0].GetStyle() != DefaultBorderStyle {
		t.Fatalf("card border style = %q, want %q", children[0].GetStyle(), DefaultBorderStyle)
	}
	if got := boxText(children[7]); got != "└"+strings.Repeat("─", 18)+"┘" {
		t.Fatalf("card bottom rule = %q", got)
	}
	row := children[3].GetChildren()
	if len(row) != 3 || boxText(row[0]) != "│" || boxText(row[2]) != "│" {
		t.Fatalf("card side bars = %+v", row)
	}
	hint := boxText(row[1])
	if !contains(hint, "Ctrl-F 选择终端") || !strings.HasPrefix(hint, " ") {
		t.Fatalf("card hint must be centered: %q", hint)
	}
	if width := sdk.DisplayWidth(hint); width != 18 {
		t.Fatalf("card hint width = %d, want 18", width)
	}
}

func TestDividerBuildsGutterRun(t *testing.T) {
	vertical := Divider{ID: "divider:0", Vertical: true, Length: 4, Style: "fg:#aabbcc", Input: []string{"mouse"}}.Build().Build()
	if vertical.GetId() != "divider:0" || boxText(vertical) != "││││" {
		t.Fatalf("vertical divider = %+v text %q", vertical, boxText(vertical))
	}
	if vertical.GetSize().GetWidth() != 1 || vertical.GetSize().GetHeight() != 4 {
		t.Fatalf("vertical divider size = %+v", vertical.GetSize())
	}
	if vertical.GetStyle() != "fg:#aabbcc" || vertical.GetInput()[0] != "mouse" {
		t.Fatalf("vertical divider style/input = %q %v", vertical.GetStyle(), vertical.GetInput())
	}
	horizontal := Divider{Length: 3}.Build().Build()
	if boxText(horizontal) != "───" || horizontal.GetSize().GetHeight() != 1 {
		t.Fatalf("horizontal divider = %+v text %q", horizontal, boxText(horizontal))
	}
	if horizontal.GetStyle() != DefaultSepStyle {
		t.Fatalf("horizontal divider default style = %q", horizontal.GetStyle())
	}
}

func TestFrameDrawsChromeAndRowIDs(t *testing.T) {
	frame := Frame{
		ID:     "overlay:x",
		Title:  "terminals",
		Width:  20,
		Height: 5,
		Style:  "fg:#ffffff",
		Rows: []FrameRow{
			{Text: "select", ID: "pick:0", Input: []string{"mouse"}},
		},
	}
	root := frame.Build().Build()
	children := root.GetChildren()
	if len(children) != 5 {
		t.Fatalf("frame rows = %d, want 5", len(children))
	}
	if top := boxText(children[0]); !contains(top, "terminals") {
		t.Fatalf("frame top rule = %q", top)
	}
	if children[0].GetStyle() != "fg:#ffffff" {
		t.Fatalf("frame style = %q, want the explicit program style", children[0].GetStyle())
	}
	if got := boxText(children[4]); got != "└"+strings.Repeat("─", 18)+"┘" {
		t.Fatalf("frame bottom rule = %q", got)
	}
	row := findBox(root, "pick:0")
	if row == nil {
		t.Fatal("frame row id is lost")
	}
	// The row is the inner content of one body row (side bars 1 cell each):
	// the declared geometry is what the host's hit test consumes.
	if row.GetSize().GetWidth() != 18 || row.GetSize().GetHeight() != 1 {
		t.Fatalf("frame row size = %+v, want 18x1", row.GetSize())
	}
	if len(row.GetInput()) != 1 || row.GetInput()[0] != "mouse" {
		t.Fatalf("frame row input = %v, want mouse", row.GetInput())
	}
}

func TestButtonHotKeyAndClickHandling(t *testing.T) {
	button := Button{ID: "btn:new", Text: "New terminal", Hot: "N"}
	root := button.Build()
	box := root.Build()
	if box.GetId() != "btn:new" {
		t.Fatalf("button id = %q", box.GetId())
	}
	children := box.GetChildren()
	if len(children) != 2 {
		t.Fatalf("hot button children = %d, want hot/post (the hot char leads the text)", len(children))
	}
	if got := boxText(children[0]); got != "N" {
		t.Fatalf("hot run = %q", got)
	}
	if children[0].GetStyle() != DefaultButtonStyle {
		t.Fatalf("hot run style = %q", children[0].GetStyle())
	}
	if got := boxText(children[1]); got != "ew terminal" {
		t.Fatalf("post run = %q", got)
	}

	// Caller-owned dispatch: the widget never emits methods; a click is
	// resolved by the host's hit test against the stable id, so the id and
	// the input flag are the whole widget contract (kernel.HitTest itself is
	// covered by kernel tests and the tmux acceptance).
	container := sdk.Row(root, sdk.Text("   ")).Width(20).Build()
	if container.GetChildren()[0].GetId() != "btn:new" {
		t.Fatalf("button id/position lost in a container: %v", container.GetChildren()[0])
	}
	if len(container.GetChildren()[0].GetInput()) != 1 || container.GetChildren()[0].GetInput()[0] != "mouse" {
		t.Fatalf("button input = %v, want mouse", container.GetChildren()[0].GetInput())
	}
}

func TestKeyHintSegments(t *testing.T) {
	hint := KeyHint{Mode: "SCROLL", Keys: []string{"PgUp/PgDn scroll", "y copy"}, ID: "keyhint"}
	root := hint.Build().Build()
	children := root.GetChildren()
	if len(children) != 3 {
		t.Fatalf("keyhint children = %d, want mode/sep/keys", len(children))
	}
	if boxText(children[0]) != "SCROLL" || children[0].GetStyle() != DefaultStatusStyle {
		t.Fatalf("mode box = %+v", children[0])
	}
	if boxText(children[2]) != "PgUp/PgDn scroll · y copy" {
		t.Fatalf("keys box = %q", boxText(children[2]))
	}
	if got := hint.Text(); got != "SCROLL "+DefaultSeparator+" PgUp/PgDn scroll · y copy" {
		t.Fatalf("KeyHint.Text = %q", got)
	}
}

func contains(text, want string) bool {
	for i := 0; i+len(want) <= len(text); i++ {
		if text[i:i+len(want)] == want {
			return true
		}
	}
	return false
}
