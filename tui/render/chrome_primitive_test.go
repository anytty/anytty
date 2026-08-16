package render

import (
	"strings"
	"testing"
)

func TestChromePrimitivePaneActionSlotsMatchHitRegions(t *testing.T) {
	panel := PanelVM{
		ID:     "pane-1",
		Title:  "shell",
		Active: true,
		Chrome: PanelChromeVM{Actions: []ChromeActionVM{
			paneChromeActionVM(ActionPaneZoom, StyleAccent),
			paneChromeActionVM(ActionPaneSplitRight, StyleAccent),
			paneChromeActionVM(ActionPaneSplitDown, StyleAccent),
			paneChromeActionVM(ActionPaneClose, StyleAccent),
		}},
	}
	rect := Rect{W: 44, H: 8}
	primitive := PaneChromePrimitive(panel, rect, StyleAccent)
	regions := appendPaneActionRegions(nil, panel, rect, panel.ID, rect)

	if len(primitive.ActionSlots) != len(regions) {
		t.Fatalf("primitive action slots must match hit regions slots=%#v regions=%#v", primitive.ActionSlots, regions)
	}
	for index, slot := range primitive.ActionSlots {
		region := regions[index]
		if slot.ActionID != region.ActionID || slot.Rect != region.Rect || slot.Text == "" {
			t.Fatalf("slot %d must drive hit region slot=%#v region=%#v", index, slot, region)
		}
	}
}

func TestChromePrimitivePaneOwnerTokenDrivesTakeOwnerHitRegion(t *testing.T) {
	panel := PanelVM{
		ID:     "pane-2",
		Title:  "123",
		Active: true,
		Chrome: PanelChromeVM{
			Terminal: TerminalChromeVM{
				Title:       ChromeSlotVM{Text: "123", Style: StyleAccent},
				State:       ChromeSlotVM{Text: paneChromeRunningGlyph(), Style: StyleSuccess},
				AttachCount: 2,
				Owner:       ChromeSlotVM{Text: "◇ follow", Style: StyleMuted},
				TakeOwner:   true,
				TerminalID:  "term-1",
			},
			Actions: []ChromeActionVM{paneChromeActionVM(ActionPaneClose, StyleAccent)},
		},
	}
	rect := Rect{W: 72, H: 8}
	primitive := PaneChromePrimitive(panel, rect, StyleAccent)
	regions := appendPaneActionRegions(nil, panel, rect, panel.ID, rect)

	found := false
	for index, slot := range primitive.ActionSlots {
		if slot.ActionID != ActionTerminalTakeResizeOwner.String() {
			continue
		}
		found = true
		if !strings.Contains(slot.Text, "◇ follow") || regions[index].ActionID != slot.ActionID || regions[index].Rect != slot.Rect {
			t.Fatalf("owner token should be visible hit slot, slot=%#v regions=%#v", slot, regions)
		}
	}
	if !found {
		t.Fatalf("expected owner token action slot, got %#v", primitive.ActionSlots)
	}
}

func TestChromePrimitiveTerminalSizeLockButtonPrecedesTitleAndDrivesHitRegion(t *testing.T) {
	panel := terminalChromeSlotTestPanel("pane-1", "shell")
	panel.Chrome.Actions = []ChromeActionVM{paneChromeActionVM(ActionPaneClose, StyleAccent)}
	panel.Chrome.Terminal.Owner = ChromeSlotVM{Text: "◆ owner", Style: StyleSuccess}
	panel.Chrome.Terminal.TakeOwner = false
	panel.Chrome.Terminal.CanLockSize = true
	rect := Rect{W: 72, H: 8}
	primitive := PaneChromePrimitive(panel, rect, StyleAccent)
	unlocked, ok := chromeSlotByAction(primitive.ActionSlots, ActionResizeLayoutLock.String())
	if !ok || unlocked.Text != paneChromeBracketToken(paneChromeSizeUnlockGlyph()) {
		t.Fatalf("unlocked terminal should expose size lock button before title, got %#v", primitive.ActionSlots)
	}
	if primitive.Title.Text == "" || unlocked.Rect.X >= primitive.Title.Rect.X {
		t.Fatalf("size lock button should sit before title slot, title=%#v lock=%#v", primitive.Title, unlocked)
	}
	regions := appendPaneActionRegions(nil, panel, rect, panel.ID, rect)
	if !hitRegionsContainActionRect(regions, ActionResizeLayoutLock.String(), unlocked.Rect) {
		t.Fatalf("size lock slot should drive matching hit region, slot=%#v regions=%#v", unlocked, regions)
	}

	panel.Chrome.Terminal.Locked = true
	lockedPrimitive := PaneChromePrimitive(panel, rect, StyleAccent)
	locked, ok := chromeSlotByAction(lockedPrimitive.ActionSlots, ActionResizeLayoutLock.String())
	if !ok || locked.Text != paneChromeBracketToken(paneChromeSizeLockGlyph()) || locked.Rect.X != unlocked.Rect.X {
		t.Fatalf("locked terminal should keep same action slot with locked glyph, unlocked=%#v locked=%#v", unlocked, locked)
	}
}

func TestChromePrimitiveFollowerSizeLockGlyphIsDisplayOnly(t *testing.T) {
	panel := terminalChromeSlotTestPanel("pane-1", "shell")
	panel.Chrome.Actions = []ChromeActionVM{paneChromeActionVM(ActionPaneClose, StyleAccent)}
	rect := Rect{W: 72, H: 8}
	primitive := PaneChromePrimitive(panel, rect, StyleAccent)
	if _, ok := chromeSlotByAction(primitive.ActionSlots, ActionResizeLayoutLock.String()); ok {
		t.Fatalf("follower terminal should display lock glyph without action slot, got %#v", primitive.ActionSlots)
	}
	regions := appendPaneActionRegions(nil, panel, rect, panel.ID, rect)
	if hitRegionsContainAction(regions, ActionResizeLayoutLock.String()) {
		t.Fatalf("follower terminal should not drive size lock hit region, regions=%#v", regions)
	}
	if !chromeLabelsContainText(primitive.LabelSlots, paneChromeBracketToken(paneChromeSizeUnlockGlyph())) {
		t.Fatalf("follower terminal should still display size lock glyph, labels=%#v", primitive.LabelSlots)
	}
}

func TestChromePrimitiveFloatingActionSlotsMatchHitRegions(t *testing.T) {
	rect := Rect{X: 10, Y: 4, W: 30, H: 8}
	primitive := FloatingChromePrimitive(FloatingVM{ID: "float-1", Rect: rect, Active: true}, rect, StyleAccent)
	regions := appendFloatingActionRegions(nil, FloatingVM{ID: "float-1", Rect: rect, Active: true}, rect, "float-1", Rect{W: 80, H: 24})

	if len(primitive.ActionSlots) != len(regions) {
		t.Fatalf("primitive floating slots must match hit regions slots=%#v regions=%#v", primitive.ActionSlots, regions)
	}
	for index, slot := range primitive.ActionSlots {
		region := regions[index]
		if slot.ActionID != region.ActionID || slot.Rect != region.Rect || slot.Text == "" {
			t.Fatalf("floating slot %d must drive hit region slot=%#v region=%#v", index, slot, region)
		}
		if region.PaneID != "float-1" || !region.Floating {
			t.Fatalf("floating slot %d should carry floating panel identity, got %#v", index, region)
		}
	}
}

func TestChromePrimitiveFloatingTerminalOwnerTokenDrivesTakeOwnerHitRegion(t *testing.T) {
	rect := Rect{X: 2, Y: 1, W: 84, H: 8}
	floating := FloatingVM{ID: "float-1", Title: "123", Rect: rect, Active: true, Chrome: FloatingChromeVM{
		Terminal: TerminalChromeVM{
			Title:       ChromeSlotVM{Text: "123", Style: StyleAccent},
			State:       ChromeSlotVM{Text: paneChromeRunningGlyph(), Style: StyleSuccess},
			AttachCount: 2,
			Owner:       ChromeSlotVM{Text: "◇ follow", Style: StyleMuted},
			TakeOwner:   true,
			TerminalID:  "term-1",
		},
		Actions: []ChromeActionVM{
			paneChromeActionVM(ActionFloatingCenter, StyleAccent),
			paneChromeActionVM(ActionFloatingCollapse, StyleAccent),
			paneChromeActionVM(ActionPaneZoom, StyleAccent),
			paneChromeActionVM(ActionFloatingClose, StyleAccent),
		},
	}}
	primitive := FloatingChromePrimitive(floating, rect, StyleAccent)
	regions := appendFloatingActionRegions(nil, floating, rect, "float-1", Rect{W: 100, H: 24})

	found := false
	for _, slot := range primitive.ActionSlots {
		if slot.ActionID == ActionTerminalTakeResizeOwner.String() && strings.Contains(slot.Text, "◇ follow") {
			found = true
			if !hitRegionsContainActionRect(regions, slot.ActionID, slot.Rect) {
				t.Fatalf("floating owner token should drive matching hit region, slot=%#v regions=%#v", slot, regions)
			}
		}
	}
	if !found {
		t.Fatalf("expected floating owner token action slot, got %#v", primitive.ActionSlots)
	}
}

func TestChromePrimitiveFloatingSizeLockStaysInTerminalLabel(t *testing.T) {
	rect := Rect{X: 2, Y: 1, W: 84, H: 8}
	floating := FloatingVM{ID: "float-1", Title: "123", Rect: rect, Active: true, Chrome: FloatingChromeVM{
		Terminal: TerminalChromeVM{
			Title:       ChromeSlotVM{Text: "123", Style: StyleAccent},
			TerminalID:  "term-1",
			Locked:      true,
			CanLockSize: true,
		},
		Actions: []ChromeActionVM{
			paneChromeActionVM(ActionFloatingCenter, StyleAccent),
			paneChromeActionVM(ActionFloatingCollapse, StyleAccent),
			paneChromeActionVM(ActionPaneZoom, StyleAccent),
			paneChromeActionVM(ActionFloatingClose, StyleAccent),
		},
	}}
	primitive := FloatingChromePrimitive(floating, rect, StyleAccent)
	lock, ok := chromeSlotByAction(primitive.ActionSlots, ActionResizeLayoutLock.String())
	if !ok || lock.Text != paneChromeBracketToken(paneChromeSizeLockGlyph()) {
		t.Fatalf("floating terminal label should expose size lock action slot, got %#v", primitive.ActionSlots)
	}
	if len(primitive.LabelSlots) == 0 || lock.Rect.X >= primitive.LabelSlots[len(primitive.LabelSlots)-1].Rect.X {
		t.Fatalf("floating size lock slot should sit before terminal title, lock=%#v labels=%#v", lock, primitive.LabelSlots)
	}
	if chromeSlotsContainAction(floatingChromeControlSlots(primitive.ActionSlots), ActionResizeLayoutLock.String()) {
		t.Fatalf("floating right control cluster must not duplicate terminal size lock slot, got %#v", primitive.ActionSlots)
	}
}

func TestChromePrimitiveTerminalRightSlotsStayAnchoredAcrossTitleLengths(t *testing.T) {
	rect := Rect{W: 86, H: 8}
	short := terminalChromeSlotTestPanel("pane-short", "sh")
	long := terminalChromeSlotTestPanel("pane-long", strings.Repeat("long-title-", 8))
	shortSlots := PaneChromePrimitive(short, rect, StyleAccent).ActionSlots
	longSlots := PaneChromePrimitive(long, rect, StyleAccent).ActionSlots

	for _, actionID := range []string{ActionTerminalTakeResizeOwner.String(), ActionPaneClose.String()} {
		shortSlot, ok := chromeSlotByAction(shortSlots, actionID)
		if !ok {
			t.Fatalf("missing short slot %s in %#v", actionID, shortSlots)
		}
		longSlot, ok := chromeSlotByAction(longSlots, actionID)
		if !ok {
			t.Fatalf("missing long slot %s in %#v", actionID, longSlots)
		}
		if shortSlot.Rect.X != longSlot.Rect.X || shortSlot.Rect.W != longSlot.Rect.W {
			t.Fatalf("slot %s should stay anchored across title lengths short=%#v long=%#v", actionID, shortSlot, longSlot)
		}
	}
	if chromeSlotsContainText(shortSlots, "size:") || chromeSlotsContainText(longSlots, "layout:") {
		t.Fatalf("terminal chrome slots should not expose debug meta, short=%#v long=%#v", shortSlots, longSlots)
	}
}

func TestChromePrimitiveNarrowTerminalKeepsOwnerBeforeShareAndLifecycle(t *testing.T) {
	panel := terminalChromeSlotTestPanel("pane-1", "tiny")
	primitive := PaneChromePrimitive(panel, Rect{W: 34, H: 8}, StyleAccent)
	owner, ok := chromeSlotByAction(primitive.ActionSlots, ActionTerminalTakeResizeOwner.String())
	if !ok || !strings.Contains(owner.Text, "follow") {
		t.Fatalf("narrow terminal should keep owner action before share/lifecycle slots, got %#v", primitive.ActionSlots)
	}
	closeSlot, ok := chromeSlotByAction(primitive.ActionSlots, ActionPaneClose.String())
	if !ok || closeSlot.Rect.W <= 0 {
		t.Fatalf("narrow terminal should retain close action, got %#v", primitive.ActionSlots)
	}
	if chromeSlotsContainText(primitive.ActionSlots, "x2") || chromeSlotsContainText(primitive.ActionSlots, paneChromeRunningGlyph()) {
		t.Fatalf("narrow terminal should drop share/lifecycle before owner action, got %#v", primitive.ActionSlots)
	}
}

func terminalChromeSlotTestPanel(id string, title string) PanelVM {
	return PanelVM{
		ID:     id,
		Title:  title,
		Active: true,
		Chrome: PanelChromeVM{
			Terminal: TerminalChromeVM{
				Title:       ChromeSlotVM{Text: title, Style: StyleAccent},
				State:       ChromeSlotVM{Text: paneChromeRunningGlyph(), Style: StyleSuccess},
				AttachCount: 2,
				Owner:       ChromeSlotVM{Text: "◇ follow", Style: StyleMuted},
				TakeOwner:   true,
				TerminalID:  "term-1",
			},
			Actions: []ChromeActionVM{
				paneChromeActionVM(ActionPaneZoom, StyleAccent),
				paneChromeActionVM(ActionPaneSplitRight, StyleAccent),
				paneChromeActionVM(ActionPaneSplitDown, StyleAccent),
				paneChromeActionVM(ActionPaneClose, StyleAccent),
			},
		},
	}
}

func chromeSlotByAction(slots []ChromeSlot, actionID string) (ChromeSlot, bool) {
	for _, slot := range slots {
		if slot.ActionID == actionID {
			return slot, true
		}
	}
	return ChromeSlot{}, false
}

func chromeSlotsContainText(slots []ChromeSlot, text string) bool {
	for _, slot := range slots {
		if strings.Contains(slot.Text, text) {
			return true
		}
	}
	return false
}

func chromeLabelsContainText(slots []ChromeSlot, text string) bool {
	for _, slot := range slots {
		if strings.Contains(slot.Text, text) {
			return true
		}
	}
	return false
}

func chromeSlotsContainAction(slots []ChromeSlot, actionID string) bool {
	for _, slot := range slots {
		if slot.ActionID == actionID {
			return true
		}
	}
	return false
}

func hitRegionsContainAction(regions []HitRegion, actionID string) bool {
	for _, region := range regions {
		if region.ActionID == actionID {
			return true
		}
	}
	return false
}

func hitRegionsContainActionRect(regions []HitRegion, actionID string, rect Rect) bool {
	for _, region := range regions {
		if region.ActionID == actionID && region.Rect == rect {
			return true
		}
	}
	return false
}

func TestChromePrimitiveToastSpecPreservesCurrentGeometry(t *testing.T) {
	rect := Rect{X: 20, Y: 3, W: 42, H: 5}
	primitive := ToastChromePrimitive(ToastVM{ID: "toast-1", Title: "pane.split"}, rect)

	if primitive.Rect != rect || primitive.Layer != LayerToast || primitive.Style != StyleToast {
		t.Fatalf("toast primitive must preserve current geometry/style, got %#v", primitive)
	}
	if primitive.ContentRect != (Rect{X: 22, Y: 5, W: 38, H: 1}) {
		t.Fatalf("toast text rect should match current centered toast geometry, got %#v", primitive.ContentRect)
	}
}
