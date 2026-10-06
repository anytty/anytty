package widgets

import (
	"math"
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1 MB"},
		{3 * 1024 * 1024, "3 MB"},
		{1024 * 1024 * 1024, "1 GB"},
		{5*1024*1024*1024 + 512*1024*1024, "5.5 GB"},
		{-2048, "-2 KB"},
		{math.MaxInt64, "8 EB"},
	}
	for _, tc := range cases {
		if got := FormatBytes(tc.in); got != tc.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatCount(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{7, "7"},
		{999, "999"},
		{1000, "1,000"},
		{1234, "1,234"},
		{1234567, "1,234,567"},
		{math.MaxInt64, "9,223,372,036,854,775,807"},
		{-1000, "-1,000"},
		{-1, "-1"},
	}
	for _, tc := range cases {
		if got := FormatCount(tc.in); got != tc.want {
			t.Errorf("FormatCount(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatDurationBoundaries(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{0, "0s"},
		{500 * time.Nanosecond, "500ns"},
		{999 * time.Nanosecond, "999ns"},
		{1500 * time.Nanosecond, "2µs"},
		{250 * time.Microsecond, "250µs"},
		{1500 * time.Microsecond, "1.5ms"},
		{250 * time.Millisecond, "250ms"},
		{1000 * time.Millisecond, "1s"},
		{1500 * time.Millisecond, "1.5s"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{60 * time.Second, "1m"},
		{3600 * time.Second, "1h"},
		{3661 * time.Second, "1h1m1s"},
		{3600*time.Second + 5*time.Second, "1h5s"},
		{-90 * time.Second, "-1m30s"},
	}
	for _, tc := range cases {
		if got := FormatDuration(tc.in); got != tc.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatPercentAndFloat(t *testing.T) {
	percent := []struct {
		f      float64
		digits int
		want   string
	}{
		{0, 0, "0%"},
		{0.125, 1, "12.5%"},
		{1, 0, "100%"},
		{-0.5, 0, "-50%"},
		{1.0 / 3.0, 2, "33.33%"},
		{0.5, -1, "50%"},
	}
	for _, tc := range percent {
		if got := FormatPercent(tc.f, tc.digits); got != tc.want {
			t.Errorf("FormatPercent(%v, %d) = %q, want %q", tc.f, tc.digits, got, tc.want)
		}
	}
	if got := FormatFloat(3.14159, 2, 0); got != "3.14" {
		t.Errorf("FormatFloat = %q, want 3.14", got)
	}
	if got := FormatFloat(3.5, 0, 6); got != "     4" {
		t.Errorf("FormatFloat padded = %q, want right-aligned 4", got)
	}
}

func TestFormatNonFinite(t *testing.T) {
	if got := FormatPercent(math.NaN(), 1); got != "NaN%" {
		t.Errorf("NaN percent = %q", got)
	}
	if got := FormatPercent(math.Inf(1), 0); got != "+Inf%" {
		t.Errorf("Inf percent = %q", got)
	}
	if got := FormatFloat(math.NaN(), 2, 0); got != "NaN" {
		t.Errorf("NaN float = %q", got)
	}
	if got := PadLeft("x", 4); got != "   x" {
		t.Errorf("PadLeft = %q", got)
	}
	if got := PadRight("x", 4); got != "x   " {
		t.Errorf("PadRight = %q", got)
	}
	if got := PadLeft("toolong", 2); got != "toolong" {
		t.Errorf("PadLeft must not truncate: %q", got)
	}
}

func TestScaleValue(t *testing.T) {
	cases := []struct {
		in     float64
		want   float64
		suffix string
	}{
		{0, 0, ""},
		{999, 999, ""},
		{1500, 1.5, "K"},
		{2_000_000, 2, "M"},
		{1_000_000_000, 1, "G"},
		{1_000_000_000_000, 1, "T"},
		{1_000_000_000_000_000, 1, "P"},
		{1e18, 1, "E"},
		{-1500, -1.5, "K"},
	}
	for _, tc := range cases {
		got, suffix := ScaleValue(tc.in)
		if got != tc.want || suffix != tc.suffix {
			t.Errorf("ScaleValue(%v) = (%v, %q), want (%v, %q)", tc.in, got, suffix, tc.want, tc.suffix)
		}
	}
	if v, suffix := ScaleValue(math.NaN()); !math.IsNaN(v) || suffix != "" {
		t.Errorf("ScaleValue(NaN) = (%v, %q)", v, suffix)
	}
	if v, suffix := ScaleValue(math.Inf(1)); !math.IsInf(v, 1) || suffix != "" {
		t.Errorf("ScaleValue(+Inf) = (%v, %q)", v, suffix)
	}
}
