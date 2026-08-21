package history

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCompileHistorySearchPatternModesAndUnicode(t *testing.T) {
	tests := []struct {
		name     string
		mode     HistorySearchMode
		query    string
		text     string
		want     []HistorySearchByteRange
		literals []string
	}{
		{name: "text overlaps", mode: HistorySearchModeText, query: "aba", text: "ababa", want: []HistorySearchByteRange{{Start: 0, End: 3}, {Start: 2, End: 5}}, literals: []string{"aba"}},
		{name: "regex unicode", mode: HistorySearchModeRegex, query: `中文-[0-9]+-🚀.*UUID=`, text: "x中文-42-🚀 ok UUID=abc", want: []HistorySearchByteRange{{Start: 1, End: 24}}, literals: []string{"-🚀", "UUID=", "中文-"}},
		{name: "glob unicode", mode: HistorySearchModeGlob, query: `*用户/????/详情*`, text: "INFO 用户/四二42/详情 done", want: []HistorySearchByteRange{{Start: 0, End: 32}}, literals: []string{"/详情", "用户/"}},
		{name: "glob class", mode: HistorySearchModeGlob, query: `error-[!0-8][0-9]`, text: "error-99", want: []HistorySearchByteRange{{Start: 0, End: 8}}, literals: []string{"error-"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pattern, err := CompileHistorySearchPattern(test.mode, test.query)
			if err != nil {
				t.Fatal(err)
			}
			if got := pattern.FindAllStringIndex(test.text); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("matches=%#v want=%#v", got, test.want)
			}
			if got := pattern.RequiredLiterals(); !reflect.DeepEqual(got, test.literals) {
				t.Fatalf("literals=%q want=%q", got, test.literals)
			}
		})
	}
}

func TestCompileHistorySearchPatternRejectsUnsafeExpressions(t *testing.T) {
	for _, test := range []struct {
		mode  HistorySearchMode
		query string
	}{
		{HistorySearchModeRegex, "["},
		{HistorySearchModeRegex, "a*"},
		{HistorySearchModeGlob, "[abc"},
		{HistorySearchModeGlob, "*"},
		{HistorySearchModeText, ""},
		{HistorySearchModeText, strings.Repeat("x", MaxHistorySearchPatternBytes+1)},
		{"unknown", "value"},
	} {
		if _, err := CompileHistorySearchPattern(test.mode, test.query); !errors.Is(err, ErrHistoryInvalidSearchPattern) {
			t.Fatalf("mode=%q query=%q error=%v", test.mode, test.query, err)
		}
	}
}

func TestRegexRequiredLiteralsStayConservative(t *testing.T) {
	tests := []struct {
		query string
		want  []string
	}{
		{query: `prefix-(foo|bar)-suffix`, want: []string{"-suffix", "prefix-"}},
		{query: `0{9}`, want: []string{"000000000"}},
		{query: `(?i)missing-[a-z]+`, want: nil},
		{query: `[A-Z]{9}`, want: nil},
	}
	for _, test := range tests {
		pattern, err := CompileHistorySearchPattern(HistorySearchModeRegex, test.query)
		if err != nil {
			t.Fatal(err)
		}
		if got := pattern.RequiredLiterals(); !reflect.DeepEqual(got, test.want) {
			t.Fatalf("query=%q literals=%q want=%q", test.query, got, test.want)
		}
	}
}
