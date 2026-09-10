package input

import "testing"

func TestDigitAliasEncodingDoesNotStealShortcutIdentity(t *testing.T) {
	for _, digit := range "2345678" {
		token := "ctrl-" + string(digit)
		if !ShortcutKeyRequiresEnhancedKeyboard(token) {
			t.Fatalf("%s must remain disambiguated", token)
		}
		binding := Binding{Key: KeyChar, Char: string(digit), Ctrl: true, RequiresKeyboardDisambiguation: true}
		for _, protocol := range []KeyboardProtocol{KeyboardProtocolKittyCSIU, KeyboardProtocolXTermModifyOtherKeys} {
			event := InputEvent{Kind: EventKindKey, Key: KeyChar, Char: string(digit), Ctrl: true, KeyboardProtocol: protocol}
			if !bindingMatches(binding, event) {
				t.Fatalf("%s over %s should match", token, protocol)
			}
			encoded, _ := ctrlCharBytes(string(digit))
			event.Char = string(encoded)
			if bindingMatches(binding, event) {
				t.Fatalf("%s over %s must not match alias %q", token, protocol, encoded)
			}
		}
	}
}

func TestCtrlSlashNormalRouteProducesControlByte(t *testing.T) {
	for _, protocol := range []KeyboardProtocol{KeyboardProtocolKittyCSIU, KeyboardProtocolXTermModifyOtherKeys} {
		intent := RouteWithOptions(InputEvent{Kind: EventKindKey, Key: KeyChar, Char: "/", Ctrl: true, KeyboardProtocol: protocol}, RouteOptions{})
		if intent.Kind != IntentTerminalInput || string(intent.Bytes) != "\x1f" {
			t.Fatalf("%s: %+v", protocol, intent)
		}
	}
}
