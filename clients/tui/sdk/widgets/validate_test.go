package widgets

import "testing"

func TestRequiredValidator(t *testing.T) {
	v := Required("required")
	if got := v(""); got != "required" {
		t.Fatalf("empty = %q", got)
	}
	if got := v("   "); got != "required" {
		t.Fatalf("whitespace = %q", got)
	}
	if got := v(" ok "); got != "" {
		t.Fatalf("value = %q", got)
	}
}

func TestMinMaxLenValidators(t *testing.T) {
	min := MinLen(3, "short")
	if got := min(""); got != "" {
		t.Fatalf("empty should defer to Required, got %q", got)
	}
	if got := min("ab"); got != "short" {
		t.Fatalf("ab = %q", got)
	}
	if got := min("abc"); got != "" {
		t.Fatalf("abc = %q", got)
	}
	if got := min("中中中"); got != "" {
		t.Fatalf("3 runes of CJK = %q", got)
	}
	if got := MinLen(0, "short")("x"); got != "" {
		t.Fatalf("n<=0 should disable, got %q", got)
	}

	max := MaxLen(3, "long")
	if got := max(""); got != "" {
		t.Fatalf("empty = %q", got)
	}
	if got := max("abcd"); got != "long" {
		t.Fatalf("abcd = %q", got)
	}
	if got := max("abc"); got != "" {
		t.Fatalf("abc = %q", got)
	}
	if got := MaxLen(0, "long")("anything"); got != "" {
		t.Fatalf("n<=0 should disable, got %q", got)
	}
}

func TestPatternValidator(t *testing.T) {
	digits := Pattern(`^\d+$`, "digits only")
	if got := digits(""); got != "" {
		t.Fatalf("empty = %q", got)
	}
	if got := digits("123"); got != "" {
		t.Fatalf("123 = %q", got)
	}
	if got := digits("12a"); got != "digits only" {
		t.Fatalf("12a = %q", got)
	}

	broken := Pattern(`[`, "bad pattern")
	if got := broken(""); got != "" {
		t.Fatalf("broken pattern on empty = %q", got)
	}
	if got := broken("x"); got != "bad pattern" {
		t.Fatalf("broken pattern must fail closed, got %q", got)
	}
}

func TestEmailValidator(t *testing.T) {
	v := Email("bad email")
	cases := map[string]string{
		"":                    "",
		"a@b.co":              "",
		"first.last+tag@x.io": "",
		"no-at-sign":          "bad email",
		"a@b":                 "bad email",
		"a@b.":                "bad email",
		"a b@c.d":             "bad email",
	}
	for value, want := range cases {
		if got := v(value); got != want {
			t.Fatalf("Email(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestIntRangeValidator(t *testing.T) {
	v := IntRange(1, 10, "out of range")
	cases := map[string]string{
		"":    "",
		"1":   "",
		"10":  "",
		" 5 ": "",
		"0":   "out of range",
		"11":  "out of range",
		"x":   "out of range",
		"1.5": "out of range",
	}
	for value, want := range cases {
		if got := v(value); got != want {
			t.Fatalf("IntRange(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestOneOfValidator(t *testing.T) {
	v := OneOf([]string{"red", "green", "blue"}, "pick one")
	if got := v(""); got != "" {
		t.Fatalf("empty = %q", got)
	}
	if got := v("green"); got != "" {
		t.Fatalf("green = %q", got)
	}
	if got := v("yellow"); got != "pick one" {
		t.Fatalf("yellow = %q", got)
	}
}

func TestAllAndCustomValidators(t *testing.T) {
	v := All(Required("required"), MinLen(3, "short"), Email("bad email"))
	if got := v(""); got != "required" {
		t.Fatalf("empty = %q, want first error", got)
	}
	if got := v("ab"); got != "short" {
		t.Fatalf("ab = %q", got)
	}
	if got := v("not-an-email"); got != "bad email" {
		t.Fatalf("not-an-email = %q", got)
	}
	if got := v("a@b.co"); got != "" {
		t.Fatalf("valid = %q", got)
	}

	withNil := All(nil, Required("required"))
	if got := withNil(""); got != "required" {
		t.Fatalf("nil entry must be skipped, got %q", got)
	}
	if got := All()("anything"); got != "" {
		t.Fatalf("no validators = %q", got)
	}

	if got := Custom(nil)("x"); got != "" {
		t.Fatalf("Custom(nil) = %q", got)
	}
	if got := Custom(func(string) string { return "nope" })("x"); got != "nope" {
		t.Fatalf("Custom = %q", got)
	}
}
