package harness

import (
	"bytes"
	"testing"
)

func TestKeyBytesNamedKeys(t *testing.T) {
	cases := []struct {
		name string
		want []byte
	}{
		{"Enter", []byte{'\r'}},
		{"Escape", []byte{0x1b}},
		{"Tab", []byte{'\t'}},
		{"Down", []byte{0x1b, '[', 'B'}},
		{"PageUp", []byte{0x1b, '[', '5', '~'}},
		{"C-p", []byte{0x10}},
		{"C-c", []byte{0x03}},
		{"C-d", []byte{0x04}},
		{"C-q", []byte{0x11}},
		{"percent", []byte("percent")},
		{"%", []byte("%")},
		{"h", []byte("h")},
	}
	for _, tc := range cases {
		if got := KeyBytes(tc.name); !bytes.Equal(got, tc.want) {
			t.Errorf("KeyBytes(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}
