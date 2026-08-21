package history

import (
	"fmt"
	"regexp"
	"regexp/syntax"
	"sort"
	"strings"
	"unicode/utf8"
)

const MaxHistorySearchPatternBytes = 4096

type HistorySearchMode string

const (
	HistorySearchModeText  HistorySearchMode = "text"
	HistorySearchModeGlob  HistorySearchMode = "glob"
	HistorySearchModeRegex HistorySearchMode = "regex"
)

type HistorySearchByteRange struct {
	Start int
	End   int
}

// HistorySearchPattern is a compiled, line-bound search expression. Required
// literals are conservative: every successful match must contain every value.
type HistorySearchPattern struct {
	mode             HistorySearchMode
	query            string
	regexp           *regexp.Regexp
	requiredLiterals []string
}

func CompileHistorySearchPattern(mode HistorySearchMode, query string) (*HistorySearchPattern, error) {
	if mode == "" {
		mode = HistorySearchModeText
	}
	if query == "" {
		return nil, historySearchPatternError(mode, "must not be empty")
	}
	if len(query) > MaxHistorySearchPatternBytes {
		return nil, historySearchPatternError(mode, fmt.Sprintf("must not exceed %d bytes", MaxHistorySearchPatternBytes))
	}
	result := &HistorySearchPattern{mode: mode, query: query}
	switch mode {
	case HistorySearchModeText:
		result.requiredLiterals = []string{query}
		return result, nil
	case HistorySearchModeGlob:
		expression, literals, err := historyGlobRegexp(query)
		if err != nil {
			return nil, historySearchPatternError(mode, err.Error())
		}
		result.requiredLiterals = literals
		result.regexp, err = regexp.Compile(expression)
		if err != nil {
			return nil, historySearchPatternError(mode, err.Error())
		}
	case HistorySearchModeRegex:
		compiled, err := regexp.Compile(query)
		if err != nil {
			return nil, historySearchPatternError(mode, err.Error())
		}
		parsed, err := syntax.Parse(query, syntax.Perl)
		if err != nil {
			return nil, historySearchPatternError(mode, err.Error())
		}
		result.regexp = compiled
		result.requiredLiterals = historyRegexRequiredLiterals(parsed)
	default:
		return nil, historySearchPatternError(mode, "unsupported search mode")
	}
	if result.regexp.MatchString("") {
		return nil, historySearchPatternError(mode, "must not match an empty string")
	}
	return result, nil
}

func (pattern *HistorySearchPattern) Mode() HistorySearchMode {
	if pattern == nil {
		return ""
	}
	return pattern.mode
}

func (pattern *HistorySearchPattern) Query() string {
	if pattern == nil {
		return ""
	}
	return pattern.query
}

func (pattern *HistorySearchPattern) RequiredLiterals() []string {
	if pattern == nil {
		return nil
	}
	return append([]string(nil), pattern.requiredLiterals...)
}

func (pattern *HistorySearchPattern) FindAllStringIndex(text string) []HistorySearchByteRange {
	if pattern == nil || text == "" {
		return nil
	}
	if pattern.mode != HistorySearchModeText {
		matches := pattern.regexp.FindAllStringIndex(text, -1)
		result := make([]HistorySearchByteRange, 0, len(matches))
		for _, match := range matches {
			if len(match) == 2 && match[1] > match[0] {
				result = append(result, HistorySearchByteRange{Start: match[0], End: match[1]})
			}
		}
		return result
	}
	var result []HistorySearchByteRange
	for offset := 0; offset <= len(text); {
		relative := strings.Index(text[offset:], pattern.query)
		if relative < 0 {
			break
		}
		start := offset + relative
		result = append(result, HistorySearchByteRange{Start: start, End: start + len(pattern.query)})
		_, size := utf8.DecodeRuneInString(text[start:])
		if size <= 0 {
			break
		}
		offset = start + size
	}
	return result
}

func historySearchPatternError(mode HistorySearchMode, reason string) error {
	return fmt.Errorf("%w: %s: %s", ErrHistoryInvalidSearchPattern, mode, reason)
}

func historyGlobRegexp(pattern string) (string, []string, error) {
	runes := []rune(pattern)
	var expression strings.Builder
	var literal strings.Builder
	var literals []string
	flushLiteral := func() {
		if literal.Len() == 0 {
			return
		}
		literals = append(literals, literal.String())
		literal.Reset()
	}
	expression.WriteString("^(?:")
	for index := 0; index < len(runes); index++ {
		value := runes[index]
		switch value {
		case '*':
			flushLiteral()
			expression.WriteString(".*")
		case '?':
			flushLiteral()
			expression.WriteByte('.')
		case '\\':
			if index+1 >= len(runes) {
				return "", nil, fmt.Errorf("trailing escape")
			}
			index++
			text := string(runes[index])
			literal.WriteString(text)
			expression.WriteString(regexp.QuoteMeta(text))
		case '[':
			flushLiteral()
			class, next, err := historyGlobClass(runes, index)
			if err != nil {
				return "", nil, err
			}
			expression.WriteString(class)
			index = next
		default:
			text := string(value)
			literal.WriteString(text)
			expression.WriteString(regexp.QuoteMeta(text))
		}
	}
	flushLiteral()
	expression.WriteString(")$")
	return expression.String(), normalizeHistorySearchLiterals(literals), nil
}

func historyGlobClass(pattern []rune, open int) (string, int, error) {
	index := open + 1
	if index >= len(pattern) {
		return "", open, fmt.Errorf("unterminated character class")
	}
	var class strings.Builder
	class.WriteByte('[')
	if pattern[index] == '!' || pattern[index] == '^' {
		class.WriteByte('^')
		index++
	}
	memberCount := 0
	for ; index < len(pattern); index++ {
		value := pattern[index]
		if value == ']' && memberCount > 0 {
			class.WriteByte(']')
			return class.String(), index, nil
		}
		if value == '\\' {
			if index+1 >= len(pattern) {
				return "", open, fmt.Errorf("unterminated character class escape")
			}
			index++
			value = pattern[index]
			class.WriteByte('\\')
			class.WriteRune(value)
			memberCount++
			continue
		}
		if value == ']' || value == '\\' {
			class.WriteByte('\\')
		}
		class.WriteRune(value)
		memberCount++
	}
	return "", open, fmt.Errorf("unterminated character class")
}

func historyRegexRequiredLiterals(expression *syntax.Regexp) []string {
	if expression == nil {
		return nil
	}
	var result []string
	switch expression.Op {
	case syntax.OpLiteral:
		if expression.Flags&syntax.FoldCase == 0 && len(expression.Rune) > 0 {
			result = append(result, string(expression.Rune))
		}
	case syntax.OpCapture, syntax.OpPlus:
		result = append(result, historyRegexRequiredLiterals(expression.Sub[0])...)
	case syntax.OpRepeat:
		if expression.Min > 0 {
			child := expression.Sub[0]
			if child.Op == syntax.OpLiteral && child.Flags&syntax.FoldCase == 0 && len(child.Rune)*expression.Min <= MaxHistorySearchPatternBytes {
				result = append(result, strings.Repeat(string(child.Rune), expression.Min))
			} else {
				result = append(result, historyRegexRequiredLiterals(child)...)
			}
		}
	case syntax.OpConcat:
		for _, child := range expression.Sub {
			result = append(result, historyRegexRequiredLiterals(child)...)
		}
	case syntax.OpAlternate:
		if len(expression.Sub) > 0 {
			common := make(map[string]struct{})
			for _, literal := range historyRegexRequiredLiterals(expression.Sub[0]) {
				common[literal] = struct{}{}
			}
			for _, child := range expression.Sub[1:] {
				childLiterals := make(map[string]struct{})
				for _, literal := range historyRegexRequiredLiterals(child) {
					childLiterals[literal] = struct{}{}
				}
				for literal := range common {
					if _, ok := childLiterals[literal]; !ok {
						delete(common, literal)
					}
				}
			}
			for literal := range common {
				result = append(result, literal)
			}
		}
	}
	return normalizeHistorySearchLiterals(result)
}

func normalizeHistorySearchLiterals(literals []string) []string {
	unique := make(map[string]struct{}, len(literals))
	for _, literal := range literals {
		if literal != "" {
			unique[literal] = struct{}{}
		}
	}
	result := make([]string, 0, len(unique))
	for literal := range unique {
		result = append(result, literal)
	}
	sort.Strings(result)
	return result
}
