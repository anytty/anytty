package kernel

import "testing"

func TestRuneWidth(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want int
	}{
		{"ascii", 'a', 1},
		{"space", ' ', 1},
		{"box drawing", '─', 1},
		{"star ambiguous", '★', 1},
		{"cjk han", '你', 2},
		{"cjk unified", '一', 2},
		{"hangul jamo", 'ᄀ', 2},
		{"fullwidth", 'Ａ', 2},
		{"emoji grinning", '😀', 2},
		{"emoji rocket", '🚀', 2},
		{"combining acute", '\u0301', 0},
		{"variation selector", '\uFE0F', 0},
		{"zero width joiner", '\u200D', 0},
		{"zero width space", '\u200B', 0},
		{"newline", '\n', 0},
		{"tab control", '\t', 0},
		{"nul", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RuneWidth(tt.r); got != tt.want {
				t.Fatalf("RuneWidth(%q) = %d, want %d", tt.r, got, tt.want)
			}
		})
	}
}

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"ascii", "abc", 3},
		{"cjk", "你好", 4},
		{"mixed", "a你b", 4},
		{"emoji pair", "😀😀", 4},
		{"combining", "e\u0301", 1},
		{"box", "──", 2},
		{"emoji zwj sequence", "👨\u200D👩", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DisplayWidth(tt.in); got != tt.want {
				t.Fatalf("DisplayWidth(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"zero budget", "hello", 0, ""},
		{"negative budget", "hello", -3, ""},
		{"shorter than budget", "hi", 5, "hi"},
		{"exact", "hello", 5, "hello"},
		{"clip ascii", "hello", 3, "hel"},
		{"wide rune fits", "你好", 2, "你"},
		{"wide rune split avoided", "你好", 3, "你"},
		{"wide rune needs two cells", "你好", 1, ""},
		{"cjk fits", "你好", 4, "你好"},
		{"emoji fits", "😀a", 2, "😀"},
		{"emoji split avoided", "😀a", 1, ""},
		{"combining kept", "e\u0301x", 1, "e\u0301"},
		{"mixed clip", "a你b", 3, "a你"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Truncate(tt.in, tt.max); got != tt.want {
				t.Fatalf("Truncate(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
		})
	}
}
