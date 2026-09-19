// Package ansi parses raw PTY output into a fixed-size cell grid that the
// terminal component renders. It is the only v2 package that reads escape
// sequences; consumers get plain cells whose style is a render token, so no
// escape bytes ever reach the framebuffer.
//
// The parser is incremental: callers feed arbitrary byte chunks (half a
// UTF-8 rune, half a CSI sequence) and can snapshot the screen at any time.
// Unrecognized sequences are skipped without disturbing later parsing.
//
// Handled: printable text (wide graphemes, combining marks, emoji), CR/LF/BS/
// TAB, SGR attributes and colors (basic, 256-color and truecolor), cursor
// movement (A/B/C/D/E/F/G/H/f/d), erase (J/K), scrolling (S/T, region r is
// simplified to full screen), save/restore cursor (s/u and ESC 7/8) and the
// mode bits the routing layer needs (?25, ?1049, ?1000/1002/1003, ?1006,
// ?2004).
package ansi
