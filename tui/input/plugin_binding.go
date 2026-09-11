package input

// MatchPluginKey reuses terminal protocol normalization and alias handling from
// the core shortcut router; plugin bindings do not implement a second parser.
func MatchPluginKey(token string, event InputEvent) bool {
	key, ok := parseShortcutKeyToken(token)
	if !ok {
		return false
	}
	return bindingMatches(Binding{Key: key.Key, Char: key.Char, Ctrl: key.Ctrl, Alt: key.Alt, Shift: key.Shift, RequiresKeyboardDisambiguation: ShortcutKeyRequiresEnhancedKeyboard(token)}, event)
}
