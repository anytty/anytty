package sdk

import "github.com/anytty/anytty/clients/tui/sdk/builder"

// RuneWidth returns the display cells of one rune (sdk/builder).
func RuneWidth(r rune) int { return builder.RuneWidth(r) }

// DisplayWidth returns the display cells of s.
func DisplayWidth(s string) int { return builder.DisplayWidth(s) }

// Truncate cuts s to at most maxWidth display cells.
func Truncate(s string, maxWidth int) string { return builder.Truncate(s, maxWidth) }
