package widgets

import (
	"reflect"
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
)

func TestRichTextBuildMergesAndStyles(t *testing.T) {
	root := Line(
		Styled("a", "x"),
		Styled("b", "x"),
		Styled("", "z"),
		Styled("c", "y"),
	).Build().Build()
	children := root.GetChildren()
	if len(children) != 2 {
		t.Fatalf("merged runs = %d, want 2 (%+v)", len(children), children)
	}
	if boxText(children[0]) != "ab" || children[0].GetStyle() != "x" {
		t.Fatalf("first run = %q/%q", boxText(children[0]), children[0].GetStyle())
	}
	if boxText(children[1]) != "c" || children[1].GetStyle() != "y" {
		t.Fatalf("second run = %q/%q", boxText(children[1]), children[1].GetStyle())
	}
}

func TestRichTextWidthPadsAndTextMeasures(t *testing.T) {
	rich := RichText{Spans: []Span{Styled("ab", "x")}, Width: 5}
	root := rich.Build().Build()
	if got := rowText(root); got != "ab   " {
		t.Fatalf("padded row = %q, want %q", got, "ab   ")
	}
	if got := rich.Text(); got != "ab" {
		t.Fatalf("RichText.Text = %q", got)
	}
}

func TestRichTextEmpty(t *testing.T) {
	root := Line().Build().Build()
	if children := root.GetChildren(); len(children) != 0 {
		t.Fatalf("empty rich text children = %d, want 0", len(children))
	}
	if got := (Line().Text()); got != "" {
		t.Fatalf("empty text = %q", got)
	}
	styled := Line(Styled("hi", "a"), Styled("!", "b")).Styled("z")
	if styled.Text() != "hi!" || len(styled.Spans) != 1 || styled.Spans[0].Style != "z" {
		t.Fatalf("RichText.Styled = %+v", styled)
	}
}

func TestWrapTextPlain(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		width int
		want  []string
	}{
		{"simple", "hello world", 5, []string{"hello", "world"}},
		{"spaces", "a b c d", 3, []string{"a b", "c d"}},
		{"hardbreak", "abcd ef", 3, []string{"abc", "d", "ef"}},
		{"newline", "ab\ncd", 5, []string{"ab", "cd"}},
		{"blankline", "ab\n\ncd", 5, []string{"ab", "", "cd"}},
		{"exact", "abcde", 5, []string{"abcde"}},
		{"one", "a", 1, []string{"a"}},
		{"empty", "", 5, []string{""}},
		{"leading spaces", "   x", 5, []string{"x"}},
		{"zero width", "abc", 0, nil},
		{"negative width", "abc", -1, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := WrapText(tc.in, tc.width); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("WrapText(%q, %d) = %#v, want %#v", tc.in, tc.width, got, tc.want)
			}
		})
	}
}

func TestWrapTextWideRunesNeverSplit(t *testing.T) {
	got := WrapText("你好", 3)
	if !reflect.DeepEqual(got, []string{"你", "好"}) {
		t.Fatalf("wide wrap = %#v", got)
	}
	if joined := WrapText("你好", 4); !reflect.DeepEqual(joined, []string{"你好"}) {
		t.Fatalf("wide exact = %#v", joined)
	}
	if got := WrapText("a你b", 2); !reflect.DeepEqual(got, []string{"a", "你", "b"}) {
		t.Fatalf("mixed wide wrap = %#v", got)
	}
	for _, line := range WrapText("你好世界", 3) {
		if sdk.DisplayWidth(line) > 3 {
			t.Fatalf("line %q exceeds width", line)
		}
	}
}

func TestWrapSpansPreservesStyles(t *testing.T) {
	spans := []Span{{Text: "ab", Style: "x"}, {Text: "cd", Style: "y"}}
	got := WrapSpans(spans, 2)
	if len(got) != 2 {
		t.Fatalf("wrapped rows = %d, want 2", len(got))
	}
	if got[0].Text() != "ab" || got[0].Spans[0].Style != "x" {
		t.Fatalf("row 0 = %+v", got[0])
	}
	if got[1].Text() != "cd" || got[1].Spans[0].Style != "y" {
		t.Fatalf("row 1 = %+v", got[1])
	}
	for _, row := range got {
		if row.Width != 2 {
			t.Fatalf("row width = %d, want 2", row.Width)
		}
	}
	if WrapSpans(spans, 0) != nil {
		t.Fatal("WrapSpans width 0 must be nil")
	}
}

func TestWrapSpansSplitsOneRunAcrossLines(t *testing.T) {
	got := WrapSpans([]Span{{Text: "abcd", Style: "x"}}, 3)
	if len(got) != 2 || got[0].Text() != "abc" || got[1].Text() != "d" {
		t.Fatalf("split run = %+v", got)
	}
	for _, row := range got {
		if len(row.Spans) != 1 || row.Spans[0].Style != "x" {
			t.Fatalf("style lost: %+v", row)
		}
	}
}

func TestRichTextWrapSpansMarksWrapped(t *testing.T) {
	rich := RichText{Spans: []Span{Styled("abcdef", "x")}}
	rows := rich.WrapSpans(3)
	if len(rows) != 2 || !rows[0].Wrap || !rows[1].Wrap {
		t.Fatalf("wrapped rows = %+v", rows)
	}
}

func TestParseEmphasis(t *testing.T) {
	got := ParseEmphasis("a **b** c")
	want := []Span{{Text: "a ", Style: ""}, {Text: "b", Style: "bold"}, {Text: " c", Style: ""}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseEmphasis = %#v, want %#v", got, want)
	}

	italic := ParseEmphasis("_em_")
	if !reflect.DeepEqual(italic, []Span{{Text: "em", Style: "italic"}}) {
		t.Fatalf("italic = %#v", italic)
	}

	mixed := ParseEmphasis("**bold** and _em_")
	want = []Span{
		{Text: "bold", Style: "bold"},
		{Text: " and ", Style: ""},
		{Text: "em", Style: "italic"},
	}
	if !reflect.DeepEqual(mixed, want) {
		t.Fatalf("mixed = %#v, want %#v", mixed, want)
	}
}

func TestParseEmphasisUnclosedAndEmptyStayLiteral(t *testing.T) {
	cases := map[string]string{
		"a **b":  "a **b",
		"_open":  "_open",
		"****":   "****",
		"__":     "__",
		"a ** b": "a ** b",
		"a _b c": "a _b c",
	}
	for in, want := range cases {
		got := ParseEmphasis(in)
		if len(got) != 1 || got[0].Text != want || got[0].Style != "" {
			t.Fatalf("ParseEmphasis(%q) = %#v, want literal %q", in, got, want)
		}
	}
}

func TestParseEmphasisBaseStyleComposes(t *testing.T) {
	got := ParseEmphasisStyled("**x** _y_ z", "muted")
	want := []Span{
		{Text: "x", Style: "muted;bold"},
		{Text: " ", Style: "muted"},
		{Text: "y", Style: "muted;italic"},
		{Text: " z", Style: "muted"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("base = %#v, want %#v", got, want)
	}
}
