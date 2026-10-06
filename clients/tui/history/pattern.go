package history

import (
	"errors"
	"regexp"
	"strings"
)

// Pattern uses the provider's mode semantics: literal text, anchored glob or
// unanchored Go regexp; empty matches are rejected. Local command PTYs use it
// when no persistent provider exists.
func Pattern(mode, query string) (*regexp.Regexp, error) {
	if query == "" || len(query) > 4096 {
		return nil, errors.New("terminal search: query must be 1..4096 bytes")
	}
	expression := query
	switch mode {
	case "", "text":
		expression = regexp.QuoteMeta(query)
	case "regex":
	case "glob":
		var b strings.Builder
		b.WriteString("^(?:")
		runes := []rune(query)
		for i := 0; i < len(runes); i++ {
			switch runes[i] {
			case '*':
				b.WriteString(".*")
			case '?':
				b.WriteByte('.')
			case '\\':
				i++
				if i == len(runes) {
					return nil, errors.New("terminal search: trailing glob escape")
				}
				b.WriteString(regexp.QuoteMeta(string(runes[i])))
			case '[':
				b.WriteByte('[')
				i++
				if i < len(runes) && runes[i] == '!' {
					b.WriteByte('^')
					i++
				}
				for ; i < len(runes) && runes[i] != ']'; i++ {
					b.WriteRune(runes[i])
				}
				if i == len(runes) {
					return nil, errors.New("terminal search: unclosed glob class")
				}
				b.WriteByte(']')
			default:
				b.WriteString(regexp.QuoteMeta(string(runes[i])))
			}
		}
		b.WriteString(")$")
		expression = b.String()
	default:
		return nil, errors.New("terminal search: unknown mode")
	}
	re, err := regexp.Compile(expression)
	if err != nil {
		return nil, err
	}
	if re.MatchString("") {
		return nil, errors.New("terminal search: pattern must not match an empty string")
	}
	return re, nil
}
