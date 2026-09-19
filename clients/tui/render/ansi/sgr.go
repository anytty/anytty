package ansi

import (
	"strconv"
	"strings"

	"github.com/anytty/anytty/clients/tui/render"
)

// sgrState is the absolute graphic rendition state at the cursor. Colors are
// stored as normalized SGR parameter runs ("31", "38;5;196",
// "38;2;255;0;0") so cells carry exactly what the program asked for.
type sgrState struct {
	bold      bool
	dim       bool
	italic    bool
	underline bool
	reverse   bool
	strike    bool

	fg string
	bg string
}

// token renders the state as a framebuffer token.
func (s sgrState) token() render.Token {
	params := s.params()
	if params == "" {
		return render.TokenDefault
	}
	return render.ANSIToken(params)
}

func (s sgrState) params() string {
	parts := make([]string, 0, 8)
	if s.bold {
		parts = append(parts, "1")
	}
	if s.dim {
		parts = append(parts, "2")
	}
	if s.italic {
		parts = append(parts, "3")
	}
	if s.underline {
		parts = append(parts, "4")
	}
	if s.reverse {
		parts = append(parts, "7")
	}
	if s.strike {
		parts = append(parts, "9")
	}
	if s.fg != "" {
		parts = append(parts, s.fg)
	}
	if s.bg != "" {
		parts = append(parts, s.bg)
	}
	return strings.Join(parts, ";")
}

func (s *sgrState) reset() { *s = sgrState{} }

// apply folds one SGR parameter list (groups split on ';', sub-parameters on
// ':') into the state. Unknown codes are ignored, never fatal.
func (s *sgrState) apply(groups [][]int) {
	if len(groups) == 0 {
		s.reset()
		return
	}
	for i := 0; i < len(groups); i++ {
		group := groups[i]
		code := 0
		if len(group) > 0 && group[0] >= 0 {
			code = group[0]
		}
		switch code {
		case 0:
			s.reset()
		case 1:
			s.bold = true
		case 2:
			s.dim = true
		case 3:
			s.italic = true
		case 4:
			s.underline = true
		case 7:
			s.reverse = true
		case 9:
			s.strike = true
		case 21:
			s.bold = false
		case 22:
			s.bold, s.dim = false, false
		case 23:
			s.italic = false
		case 24:
			s.underline = false
		case 27:
			s.reverse = false
		case 29:
			s.strike = false
		case 39:
			s.fg = ""
		case 49:
			s.bg = ""
		case 38, 48:
			params, used, ok := extendedColor(groups[i:])
			if !ok {
				continue
			}
			if code == 38 {
				s.fg = strconv.Itoa(code) + ";" + params
			} else {
				s.bg = strconv.Itoa(code) + ";" + params
			}
			i += used - 1
		default:
			switch {
			case code >= 30 && code <= 37, code >= 90 && code <= 97:
				s.fg = strconv.Itoa(code)
			case code >= 40 && code <= 47, code >= 100 && code <= 107:
				s.bg = strconv.Itoa(code)
			}
		}
	}
}

// extendedColor resolves the color payload after a 38/48 code. It accepts
// the semicolon forms (38;5;n, 38;2;r;g;b) and the colon forms
// (38:5:n, 38:2[:colorspace]:r:g:b), normalizing to "5;n" / "2;r;g;b".
// It returns the number of groups consumed so the caller can skip them.
func extendedColor(groups [][]int) (string, int, bool) {
	first := groups[0]
	if len(first) > 1 {
		return colorParams(first[1:])
	}
	if len(groups) < 2 || len(groups[1]) == 0 {
		return "", 1, false
	}
	sub := groups[1][0]
	switch sub {
	case 5:
		if len(groups) < 3 || len(groups[2]) == 0 || groups[2][0] < 0 {
			return "", 1, false
		}
		params, _, ok := colorParams([]int{5, groups[2][0]})
		return params, 3, ok
	case 2:
		if len(groups) < 5 {
			return "", 1, false
		}
		vals := []int{2, 0, 0, 0}
		for j := 0; j < 3; j++ {
			if len(groups[2+j]) == 0 || groups[2+j][0] < 0 {
				return "", 1, false
			}
			vals[1+j] = groups[2+j][0]
		}
		params, _, ok := colorParams(vals)
		return params, 5, ok
	default:
		return "", 1, false
	}
}

// colorParams validates and normalizes a color payload: [5, n] or
// [2, r, g, b]. A colorspace slot in the colon form is ignored.
func colorParams(parts []int) (string, int, bool) {
	switch parts[0] {
	case 5:
		if len(parts) < 2 {
			return "", 1, false
		}
		n := parts[1]
		if n < 0 || n > 255 {
			return "", 1, false
		}
		return "5;" + strconv.Itoa(n), 1, true
	case 2:
		if len(parts) < 4 {
			return "", 1, false
		}
		values := parts[len(parts)-3:]
		for _, v := range values {
			if v < 0 || v > 255 {
				return "", 1, false
			}
		}
		return "2;" + strconv.Itoa(values[0]) + ";" + strconv.Itoa(values[1]) + ";" + strconv.Itoa(values[2]), 1, true
	default:
		return "", 1, false
	}
}
