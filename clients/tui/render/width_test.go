package render

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
		{"cjk", '你', 2},
		{"fullwidth", 'Ａ', 2},
		{"hangul", '한', 2},
		{"combining acute", '\u0301', 0},
		{"variation selector", '\uFE0F', 0},
		{"zero width joiner", '\u200D', 0},
		{"tab control", '\t', 0},
		{"newline control", '\n', 0},
		{"esc control", '\x1b', 0},
		{"nul", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RuneWidth(tt.r); got != tt.want {
				t.Fatalf("RuneWidth(%U) = %d, want %d", tt.r, got, tt.want)
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
		{"precomposed e acute", "\u00e9", 1},
		{"combining e acute", "e\u0301", 1},
		{"two combining e acute", "e\u0301e\u0301", 2},
		{"lone combining mark", "\u0301", 0},
		{"family zwj", "👨‍👩‍👧‍👦", 2},
		{"family zwj with text", "a👨‍👩‍👧‍👦b", 4},
		{"flag", "🇨🇳", 2},
		{"victory hand variation", "✌️", 2},
		{"victory hand bare", "✌", 1},
		{"skin tone", "👍🏽", 2},
		{"emoji plus cjk", "😀你", 4},
		{"zero width joiner alone", "\u200d", 0},
		{"control chars", "a\tb\nc", 3},
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
		{"negative budget", "hello", -2, ""},
		{"fits", "hi", 5, "hi"},
		{"exact", "hello", 5, "hello"},
		{"clips ascii", "hello", 3, "hel"},
		{"wide fits", "你", 2, "你"},
		{"wide split avoided", "你好", 3, "你"},
		{"wide needs two cells", "你好", 1, ""},
		{"family never split", "a👨‍👩‍👧‍👦b", 3, "a👨‍👩‍👧‍👦"},
		{"flag never split", "🇨🇳🇨🇳", 3, "🇨🇳"},
		{"combining keeps base", "e\u0301x", 1, "e\u0301"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Truncate(tt.in, tt.max); got != tt.want {
				t.Fatalf("Truncate(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{"pad ascii", "ab", 4, "ab  "},
		{"clip ascii", "abcd", 2, "ab"},
		{"clip wide", "你好", 3, "你 "},
		{"zero width", "ab", 0, ""},
		{"exact", "你", 2, "你"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PadRight(tt.in, tt.width); got != tt.want {
				t.Fatalf("PadRight(%q, %d) = %q, want %q", tt.in, tt.width, got, tt.want)
			}
		})
	}
}

func TestClusters(t *testing.T) {
	got := Clusters("a👨‍👩‍👧‍👦🇨🇳")
	if len(got) != 3 {
		t.Fatalf("Clusters length = %d, want 3 (%+v)", len(got), got)
	}
	if got[0].Text != "a" || got[0].Width != 1 {
		t.Fatalf("cluster 0 = %+v, want a/1", got[0])
	}
	if got[1].Text != "👨‍👩‍👧‍👦" || got[1].Width != 2 {
		t.Fatalf("cluster 1 = %+v, want family/2", got[1])
	}
	if got[2].Text != "🇨🇳" || got[2].Width != 2 {
		t.Fatalf("cluster 2 = %+v, want flag/2", got[2])
	}
	if Clusters("") != nil {
		t.Fatalf("Clusters(\"\") should be nil")
	}
}
