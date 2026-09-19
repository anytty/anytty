// Package render turns styled lines into a cell grid and into ANSI byte
// frames for the host TTY. It is the only place in tui2 that produces escape
// sequences; kernel and components stay byte-free (PROTOCOL §9.6).
//
// The package has four responsibilities:
//
//   - display width: grapheme-aware width for CJK, emoji, ZWJ sequences,
//     flags, skin-tone modifiers, variation selectors and combining marks;
//   - styles: explicit theme-free style strings sent by layout programs
//     ("fg:#RRGGBB;bg:#RRGGBB;bold;dim;reverse") translated verbatim to SGR,
//     plus the named tokens builtin components use internally;
//   - Framebuffer: a (x,y) cell grid with styles, wide-grapheme cells and a
//     cursor, blitted with kernel/composition output (Line values);
//   - screen sequences and minimal-diff output: alt screen, cursor, mouse
//     (1000/1002/1006), bracket paste and per-row diffs.
package render
