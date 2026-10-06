package widgets

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Validator reports "" when a value is acceptable, or the message to render
// under the field. Validators are pure functions, so they compose freely and
// can be driven by a Form or called directly.
type Validator func(string) string

// Required rejects a value that is empty after trimming surrounding
// whitespace. Unlike the other validators it does not skip empty input: it is
// the one validator whose whole job is to inspect emptiness.
func Required(msg string) Validator {
	return func(value string) string {
		if strings.TrimSpace(value) == "" {
			return msg
		}
		return ""
	}
}

// MinLen rejects a value shorter than n runes. An empty value is left to
// Required, so MinLen alone accepts "". n <= 0 disables the check.
func MinLen(n int, msg string) Validator {
	return func(value string) string {
		if value == "" || n <= 0 {
			return ""
		}
		if utf8.RuneCountInString(value) < n {
			return msg
		}
		return ""
	}
}

// MaxLen rejects a value longer than n runes. An empty value passes and
// n <= 0 disables the check.
func MaxLen(n int, msg string) Validator {
	return func(value string) string {
		if value == "" || n <= 0 {
			return ""
		}
		if utf8.RuneCountInString(value) > n {
			return msg
		}
		return ""
	}
}

// Pattern rejects a value that does not fully match the regular expression re.
// A pattern that fails to compile fails closed: every non-empty value is
// rejected with msg, since a broken pattern must never silently pass data.
// Empty values pass so Pattern composes with Required.
func Pattern(re, msg string) Validator {
	compiled, err := regexp.Compile(re)
	if err != nil {
		return func(value string) string {
			if value == "" {
				return ""
			}
			return msg
		}
	}
	return func(value string) string {
		if value == "" {
			return ""
		}
		if !compiled.MatchString(value) {
			return msg
		}
		return ""
	}
}

// emailPattern is intentionally small: one @, a non-empty local part and a
// dotted domain. Full RFC 5322 is a parsing problem, not a validation one.
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Email rejects values that are not a plausible email address. Empty values
// pass so Email composes with Required.
func Email(msg string) Validator {
	return func(value string) string {
		if value == "" {
			return ""
		}
		if !emailPattern.MatchString(value) {
			return msg
		}
		return ""
	}
}

// IntRange rejects a value that is not a base-10 integer inside [min, max].
// Empty values pass so IntRange composes with Required.
func IntRange(min, max int, msg string) Validator {
	return func(value string) string {
		if strings.TrimSpace(value) == "" {
			return ""
		}
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return msg
		}
		if n < min || n > max {
			return msg
		}
		return ""
	}
}

// OneOf rejects a value absent from values. Empty values pass so OneOf
// composes with Required.
func OneOf(values []string, msg string) Validator {
	return func(value string) string {
		if value == "" {
			return ""
		}
		for _, allowed := range values {
			if value == allowed {
				return ""
			}
		}
		return msg
	}
}

// All runs validators in order and returns the first non-empty message. A nil
// entry is skipped. No validators means always valid.
func All(validators ...Validator) Validator {
	return func(value string) string {
		for _, validate := range validators {
			if validate == nil {
				continue
			}
			if msg := validate(value); msg != "" {
				return msg
			}
		}
		return ""
	}
}

// Custom adapts an arbitrary function into a Validator. A nil fn is a no-op.
func Custom(fn func(string) string) Validator {
	if fn == nil {
		return func(string) string { return "" }
	}
	return Validator(fn)
}
