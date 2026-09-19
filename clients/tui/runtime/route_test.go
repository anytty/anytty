package runtime

import "testing"

func terminalFocus(inputs ...string) *Focus {
	return &Focus{ID: "terminal:local:main", IsTerminal: true, MouseTracking: true, Input: inputs}
}

func componentFocus(inputs ...string) *Focus {
	return &Focus{ID: "picker:local:1", IsTerminal: false, Input: inputs}
}

func TestRoute(t *testing.T) {
	tests := []struct {
		name string
		ev   InputEvent
		st   RouteState
		want Destination
	}{
		// Priority 1: reserved keys.
		{"ctrl-q always host", InputEvent{Kind: InputKey, Key: KeyCtrlQ}, RouteState{Focus: terminalFocus("key")}, DestinationHost},
		{"ctrl-q host under overlay", InputEvent{Kind: InputKey, Key: KeyCtrlQ}, RouteState{CoreOverlay: true}, DestinationHost},
		{"ctrl-c host under overlay", InputEvent{Kind: InputKey, Key: KeyCtrlC}, RouteState{CoreOverlay: true, Focus: terminalFocus("key")}, DestinationHost},
		{"ctrl-c unclaimed goes pty", InputEvent{Kind: InputKey, Key: KeyCtrlC}, RouteState{Focus: terminalFocus("key")}, DestinationPTY},
		{"ctrl-c unclaimed no focus goes program", InputEvent{Kind: InputKey, Key: KeyCtrlC}, RouteState{}, DestinationProgram},

		// Priority 2: core overlay owns everything.
		{"overlay owns key", InputEvent{Kind: InputKey, Key: "a"}, RouteState{CoreOverlay: true, KeysAll: true}, DestinationHost},
		{"overlay owns paste", InputEvent{Kind: InputPaste}, RouteState{CoreOverlay: true, Focus: terminalFocus("paste")}, DestinationHost},
		{"overlay owns mouse", InputEvent{Kind: InputMouse, HitFocused: true}, RouteState{CoreOverlay: true, Focus: terminalFocus("mouse")}, DestinationHost},
		{"overlay owns wheel", InputEvent{Kind: InputWheel}, RouteState{CoreOverlay: true, Focus: terminalFocus("wheel")}, DestinationHost},

		// Priority 3: keys.all.
		{"keys.all sends key to program", InputEvent{Kind: InputKey, Key: "a"}, RouteState{KeysAll: true, Focus: terminalFocus("key")}, DestinationProgram},
		{"keys.all does not capture paste", InputEvent{Kind: InputPaste}, RouteState{KeysAll: true, Focus: terminalFocus("paste")}, DestinationPTY},
		{"keys.all does not capture mouse", InputEvent{Kind: InputMouse, HitFocused: true}, RouteState{KeysAll: true, Focus: terminalFocus("mouse")}, DestinationPTY},

		// Priority 4: claim.
		{"claim hit sends key to program", InputEvent{Kind: InputKey, Key: "ctrl-f"}, RouteState{Claim: []string{"ctrl-p", "ctrl-f"}, Focus: terminalFocus("key")}, DestinationProgram},
		{"claim miss falls through to pty", InputEvent{Kind: InputKey, Key: "a"}, RouteState{Claim: []string{"ctrl-p"}, Focus: terminalFocus("key")}, DestinationPTY},
		{"claim does not capture paste", InputEvent{Kind: InputPaste}, RouteState{Claim: []string{"a"}, Focus: terminalFocus("paste")}, DestinationPTY},

		// Priority 5: focused content source, key.
		{"focused terminal key to pty", InputEvent{Kind: InputKey, Key: "a"}, RouteState{Focus: terminalFocus("key")}, DestinationPTY},
		{"focused terminal without key input goes program", InputEvent{Kind: InputKey, Key: "a"}, RouteState{Focus: terminalFocus("paste")}, DestinationProgram},
		{"focused non-terminal key to component", InputEvent{Kind: InputKey, Key: "a"}, RouteState{Focus: componentFocus("key")}, DestinationComponent},
		{"focused non-terminal without key input goes program", InputEvent{Kind: InputKey, Key: "a"}, RouteState{Focus: componentFocus("mouse")}, DestinationProgram},

		// Priority 5: paste.
		{"focused terminal paste to pty", InputEvent{Kind: InputPaste}, RouteState{Focus: terminalFocus("paste")}, DestinationPTY},
		{"focused terminal without paste goes program", InputEvent{Kind: InputPaste}, RouteState{Focus: terminalFocus("key")}, DestinationProgram},
		{"focused non-terminal paste goes program", InputEvent{Kind: InputPaste}, RouteState{Focus: componentFocus("paste")}, DestinationProgram},

		// Priority 5: mouse (hit focused + terminal mouse tracking).
		{"focused tracking terminal mouse to pty", InputEvent{Kind: InputMouse, HitFocused: true}, RouteState{Focus: terminalFocus("mouse")}, DestinationPTY},
		{"mouse without tracking goes program", InputEvent{Kind: InputMouse, HitFocused: true}, RouteState{Focus: &Focus{ID: "terminal:local:main", IsTerminal: true, Input: []string{"mouse"}}}, DestinationProgram},
		{"mouse missing focused hit goes program", InputEvent{Kind: InputMouse, HitFocused: false}, RouteState{Focus: terminalFocus("mouse")}, DestinationProgram},
		{"mouse on focused non-terminal goes program", InputEvent{Kind: InputMouse, HitFocused: true}, RouteState{Focus: componentFocus("mouse")}, DestinationProgram},

		// Priority 5: wheel (focused + tracking + panel declares wheel).
		{"wheel with all three conditions to pty", InputEvent{Kind: InputWheel}, RouteState{Focus: terminalFocus("wheel")}, DestinationPTY},
		{"wheel without tracking goes program", InputEvent{Kind: InputWheel}, RouteState{Focus: &Focus{ID: "terminal:local:main", IsTerminal: true, Input: []string{"wheel"}}}, DestinationProgram},
		{"wheel without declaration goes program", InputEvent{Kind: InputWheel}, RouteState{Focus: terminalFocus("key")}, DestinationProgram},
		{"wheel on non-terminal goes program", InputEvent{Kind: InputWheel}, RouteState{Focus: componentFocus("wheel")}, DestinationProgram},

		// Priority 6: nothing above matched.
		{"no focus key goes program", InputEvent{Kind: InputKey, Key: "a"}, RouteState{}, DestinationProgram},
		{"no focus paste goes program", InputEvent{Kind: InputPaste}, RouteState{}, DestinationProgram},
		{"no focus mouse goes program", InputEvent{Kind: InputMouse, HitFocused: true}, RouteState{}, DestinationProgram},
		{"no focus wheel goes program", InputEvent{Kind: InputWheel}, RouteState{}, DestinationProgram},
		{"focused missing key input with claim miss goes program", InputEvent{Kind: InputKey, Key: "z"}, RouteState{Claim: []string{"q"}, Focus: terminalFocus()}, DestinationProgram},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Route(tc.ev, tc.st); got != tc.want {
				t.Fatalf("Route(%+v, %+v) = %v, want %v", tc.ev, tc.st, got, tc.want)
			}
		})
	}
}

func TestFocusAccepts(t *testing.T) {
	var nilFocus *Focus
	if nilFocus.Accepts(InputKey) {
		t.Fatal("nil focus must accept nothing")
	}
	f := componentFocus("key", "mouse")
	if !f.Accepts(InputKey) || !f.Accepts(InputMouse) {
		t.Fatal("declared inputs must be accepted")
	}
	if f.Accepts(InputWheel) {
		t.Fatal("undeclared input must not be accepted")
	}
}

func TestRouteStateClaims(t *testing.T) {
	st := RouteState{}
	if st.claims("ctrl-p") {
		t.Fatal("empty claim must not match")
	}
	st.Claim = []string{"ctrl-p", "ctrl-f"}
	if !st.claims("ctrl-f") || st.claims("ctrl-q") {
		t.Fatal("claim lookup mismatch")
	}
}
