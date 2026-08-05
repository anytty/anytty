package vterm

import (
	"strings"
	"testing"
)

func TestVTermTitleCallback(t *testing.T) {
	var capturedTitle string
	titleHandler := func(title string) {
		capturedTitle = title
	}

	vt := New(80, 24, 1000, nil)
	vt.SetTitleHandler(titleHandler)

	// OSC 2 ; title ST (where ST is ESC \)
	_, err := vt.Write([]byte("\x1b]2;Test Title\x1b\\"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if capturedTitle != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", capturedTitle)
	}
}

func TestVTermTitleCallbackNotSetDoesNotPanic(t *testing.T) {
	vt := New(80, 24, 1000, nil)
	// Don't set title handler

	// Should not panic
	_, err := vt.Write([]byte("\x1b]2;Test Title\x1b\\"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestVTermTitleBELPreservesFollowingPromptText(t *testing.T) {
	var capturedTitle string
	vt := New(80, 24, 1000, nil)
	vt.SetTitleHandler(func(title string) {
		capturedTitle = title
	})

	if _, err := vt.Write([]byte("\x1b]2;anytty-prompt\x07anytty$ ")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if capturedTitle != "anytty-prompt" {
		t.Fatalf("expected title anytty-prompt, got %q", capturedTitle)
	}
	rendered := strings.Join(vt.RenderLines(), "\n")
	if !strings.Contains(rendered, "anytty$") {
		t.Fatalf("expected prompt text to remain visible, got %q", rendered)
	}
}

func TestVTermLongTitleWithinParserBuffer(t *testing.T) {
	title := strings.Repeat("x", 32*1024)
	var capturedTitle string
	vt := New(80, 24, 1000, nil)
	vt.SetTitleHandler(func(title string) {
		capturedTitle = title
	})

	if _, err := vt.Write([]byte("\x1b]2;" + title + "\x07prompt")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if capturedTitle != title {
		t.Fatalf("expected long title to survive parser buffer, got len=%d want=%d", len(capturedTitle), len(title))
	}
	rendered := strings.Join(vt.RenderLines(), "\n")
	if !strings.Contains(rendered, "prompt") {
		t.Fatalf("expected prompt text after long title, got %q", rendered)
	}
}

func TestVTermWriteWithDamageOSCTitleAndWorkingDirectoryKeepSemanticText(t *testing.T) {
	var capturedTitle string
	var capturedCWD string
	vt := New(80, 24, 1000, nil)
	vt.SetTitleHandler(func(title string) {
		capturedTitle = title
	})
	vt.SetWorkingDirectoryHandler(func(path string) {
		capturedCWD = path
	})

	_, err, damage := vt.WriteWithDamage([]byte("\x1b]2;anytty-title\x07\x1b]7;file://host/srv/app\x1b\\prompt$ "))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if capturedTitle != "anytty-title" {
		t.Fatalf("expected title callback, got %q", capturedTitle)
	}
	if capturedCWD != "file://host/srv/app" {
		t.Fatalf("expected working directory callback, got %q", capturedCWD)
	}
	if !semanticOpsContainText(damage.SemanticOps, "prompt$ ") {
		t.Fatalf("OSC title/cwd batch should keep following prompt as semantic text, ops=%#v damage=%#v", damage.SemanticOps, damage)
	}
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpTitle {
			t.Fatalf("OSC title is vterm-owned state, not a history semantic op, got %#v in %#v", op, damage.SemanticOps)
		}
	}
}

func TestVTermWorkingDirectoryCallback(t *testing.T) {
	var captured string
	vt := New(80, 24, 1000, nil)
	vt.SetWorkingDirectoryHandler(func(path string) {
		captured = path
	})

	if _, err := vt.Write([]byte("\x1b]7;file://host/srv/app\x1b\\")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if captured != "file://host/srv/app" {
		t.Fatalf("expected working directory callback, got %q", captured)
	}
}

func TestVTermOSC8HyperlinkDoesNotLeakControlBytes(t *testing.T) {
	vt := New(80, 24, 1000, nil)

	if _, err := vt.Write([]byte("\x1b]8;id=anytty;https://example.test\x1b\\linked\x1b]8;;\x1b\\ tail")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	row := vt.ScreenRowView(0)
	if got := rowToString(row); !strings.Contains(got, "linked tail") {
		t.Fatalf("expected hyperlink text to remain visible, got %q", got)
	}
	for i := 0; i < len("linked"); i++ {
		if row[i].LinkURL != "https://example.test" || row[i].LinkParams != "id=anytty" {
			t.Fatalf("expected linked cell %d to keep hyperlink, got %#v", i, row[i])
		}
	}
	if row[len("linked")+1].LinkURL != "" || row[len("linked")+1].LinkParams != "" {
		t.Fatalf("expected trailing text to reset hyperlink, got %#v", row[len("linked")+1])
	}
}

func TestVTermC1OSC8HyperlinkKeepsSemanticLinkText(t *testing.T) {
	vt := New(80, 24, 1000, nil)

	raw := string([]byte{0x9d}) + "8;id=c1;https://example.test/c1" + string([]byte{0x9c}) +
		"linked" +
		string([]byte{0x9d}) + "8;;" + string([]byte{0x9c}) +
		" tail"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("WriteWithDamage failed: %v", err)
	}
	if !semanticOpsContainText(damage.SemanticOps, "linked") || !semanticOpsContainText(damage.SemanticOps, " tail") {
		t.Fatalf("expected C1 OSC8 text to remain in semantic ops, ops=%#v damage=%#v", damage.SemanticOps, damage)
	}

	row := vt.ScreenRowView(0)
	if got := rowToString(row); !strings.Contains(got, "linked tail") {
		t.Fatalf("expected C1 OSC8 hyperlink text to remain visible, got %q damage=%#v", got, damage)
	}
	for i := 0; i < len("linked"); i++ {
		if row[i].LinkURL != "https://example.test/c1" || row[i].LinkParams != "id=c1" {
			t.Fatalf("expected C1 OSC8 linked cell %d to keep hyperlink, got %#v", i, row[i])
		}
	}
	if row[len("linked")+1].LinkURL != "" || row[len("linked")+1].LinkParams != "" {
		t.Fatalf("expected trailing text to reset C1 OSC8 hyperlink, got %#v", row[len("linked")+1])
	}
	for _, forbidden := range []string{"8;id=c1", "https://example.test/c1", "8;;"} {
		if strings.Contains(rowToString(row), forbidden) {
			t.Fatalf("C1 OSC8 control payload must not render as text %q, got %q", forbidden, rowToString(row))
		}
	}
}

func TestVTermUnsupportedPrivateModeDoesNotLeakControlBytes(t *testing.T) {
	vt := New(80, 24, 1000, nil)

	if _, err := vt.Write([]byte("\x1b[?9999hprompt\x1b[?9999l")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	rendered := strings.Join(vt.RenderLines(), "\n")
	if !strings.Contains(rendered, "prompt") {
		t.Fatalf("expected text after unsupported private mode to remain visible, got %q", rendered)
	}
	for _, leaked := range []string{"[?9999h", "[?9999l"} {
		if strings.Contains(rendered, leaked) {
			t.Fatalf("expected unsupported private mode bytes not to render, got %q", rendered)
		}
	}
}
