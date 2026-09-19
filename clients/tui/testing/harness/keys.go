package harness

// namedKeys maps tmux-style key names to the byte sequences a terminal sends.
// Unknown names fall back to their literal text, mirroring `tmux send-keys`.
var namedKeys = map[string][]byte{
	"Enter":     {'\r'},
	"Return":    {'\r'},
	"CR":        {'\r'},
	"Escape":    {0x1b},
	"Esc":       {0x1b},
	"Tab":       {'\t'},
	"Space":     {' '},
	"BSpace":    {0x7f},
	"Backspace": {0x7f},
	"Up":        {0x1b, '[', 'A'},
	"Down":      {0x1b, '[', 'B'},
	"Right":     {0x1b, '[', 'C'},
	"Left":      {0x1b, '[', 'D'},
	"Home":      {0x1b, '[', 'H'},
	"End":       {0x1b, '[', 'F'},
	"PageUp":    {0x1b, '[', '5', '~'},
	"PgUp":      {0x1b, '[', '5', '~'},
	"PageDown":  {0x1b, '[', '6', '~'},
	"PgDn":      {0x1b, '[', '6', '~'},
	"Insert":    {0x1b, '[', '2', '~'},
	"IC":        {0x1b, '[', '2', '~'},
	"Delete":    {0x1b, '[', '3', '~'},
	"DC":        {0x1b, '[', '3', '~'},
	"F1":        {0x1b, 'O', 'P'},
	"F2":        {0x1b, 'O', 'Q'},
	"F3":        {0x1b, 'O', 'R'},
	"F4":        {0x1b, 'O', 'S'},
	"F5":        {0x1b, '[', '1', '5', '~'},
	"F6":        {0x1b, '[', '1', '7', '~'},
	"F7":        {0x1b, '[', '1', '8', '~'},
	"F8":        {0x1b, '[', '1', '9', '~'},
	"F9":        {0x1b, '[', '2', '0', '~'},
	"F10":       {0x1b, '[', '2', '1', '~'},
	"F11":       {0x1b, '[', '2', '3', '~'},
	"F12":       {0x1b, '[', '2', '4', '~'},
}

// KeyBytes resolves one tmux-style key name to the bytes to write. A "C-x"
// name becomes the matching control character; any other unknown name is sent
// as its literal UTF-8 text.
func KeyBytes(name string) []byte {
	if bytes, ok := namedKeys[name]; ok {
		return append([]byte(nil), bytes...)
	}
	if len(name) == 2 && (name[0] == 'C' || name[0] == 'c') && name[1] == '-' {
		return controlByte(name[1])
	}
	if len(name) == 3 && name[1] == '-' {
		switch name[0] {
		case 'C', 'c':
			return controlByte(name[2])
		case '^':
			return controlByte(name[2])
		}
	}
	return []byte(name)
}

// controlByte maps a control key suffix to its control code: a-z, @.._ and ?
// follow the ASCII caret notation.
func controlByte(ch byte) []byte {
	switch {
	case ch == '?':
		return []byte{0x7f}
	case ch == ' ':
		return []byte{0x00}
	case ch >= 'a' && ch <= 'z':
		return []byte{ch - 'a' + 1}
	case ch >= 'A' && ch <= 'Z':
		return []byte{ch - 'A' + 1}
	case ch >= '@' && ch <= '_':
		return []byte{ch - '@'}
	default:
		return []byte{ch}
	}
}
