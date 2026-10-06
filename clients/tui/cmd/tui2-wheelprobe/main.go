// Command tui2-wheelprobe is a tiny full-screen terminal program used to
// inspect the mouse bytes that a tui2 host forwards to a TUI child. It keeps
// DEC mouse tracking enabled, like Codex/OpenCode, and renders every decoded
// event instead of implementing scrollback itself.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/term"
)

const (
	enterAltScreen = "\x1b[?1049h\x1b[?25l\x1b[?1000h\x1b[?1006h"
	exitAltScreen  = "\x1b[?1006l\x1b[?1000l\x1b[?25h\x1b[?1049l"
)

type mouseEvent struct {
	Code  int
	X     int
	Y     int
	Final byte
}

func (e mouseEvent) wheel() (string, bool) {
	if e.Code&64 == 0 {
		return "", false
	}
	switch e.Code & 3 {
	case 0:
		return "up", true
	case 1:
		return "down", true
	default:
		return "wheel", true
	}
}

// parseSGRMouse decodes one SGR 1006 mouse report. It returns incomplete for
// a prefix that may become a complete report when more bytes arrive.
func parseSGRMouse(buf []byte) (event mouseEvent, consumed int, complete bool) {
	if len(buf) == 0 || buf[0] != 0x1b {
		return mouseEvent{}, 0, false
	}
	if len(buf) < 3 {
		return mouseEvent{}, 0, false
	}
	if buf[1] != '[' || buf[2] != '<' {
		return mouseEvent{}, 1, true
	}
	end := bytes.IndexByte(buf[3:], 'M')
	lower := bytes.IndexByte(buf[3:], 'm')
	if end < 0 || (lower >= 0 && lower < end) {
		end = lower
	}
	if end < 0 {
		return mouseEvent{}, 0, false
	}
	end += 3
	fields := strings.Split(string(buf[3:end]), ";")
	if len(fields) != 3 {
		return mouseEvent{}, end + 1, true
	}
	code, errCode := strconv.Atoi(fields[0])
	x, errX := strconv.Atoi(fields[1])
	y, errY := strconv.Atoi(fields[2])
	if errCode != nil || errX != nil || errY != nil {
		return mouseEvent{}, end + 1, true
	}
	return mouseEvent{Code: code, X: x, Y: y, Final: buf[end]}, end + 1, true
}

func main() {
	maxEvents := flag.Int("max-events", 24, "number of decoded events to keep on screen")
	logPath := flag.String("log", "", "append raw bytes and decoded events to a file for host diagnostics")
	flag.Parse()
	if *maxEvents < 1 {
		*maxEvents = 1
	}
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		fmt.Fprintln(os.Stderr, "tui2-wheelprobe: stdin and stdout must be a terminal")
		os.Exit(1)
	}
	state, err := term.MakeRaw(os.Stdin.Fd())
	if err != nil {
		fmt.Fprintln(os.Stderr, "tui2-wheelprobe: raw mode:", err)
		os.Exit(1)
	}
	defer term.Restore(os.Stdin.Fd(), state)
	var trace *os.File
	if *logPath != "" {
		trace, err = os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			fmt.Fprintln(os.Stderr, "tui2-wheelprobe: log:", err)
			return
		}
		defer trace.Close()
	}
	tracef := func(format string, args ...any) {
		if trace != nil {
			_, _ = fmt.Fprintf(trace, format+"\n", args...)
		}
	}
	fmt.Print(enterAltScreen)
	defer fmt.Print(exitAltScreen)

	events := make([]string, 0, *maxEvents)
	raw := make([]byte, 0, 64)
	paint := func() {
		var b strings.Builder
		b.WriteString("\x1b[H\x1b[2Jtui2 wheel probe\r\n")
		b.WriteString("DEC 1000 + SGR 1006 enabled; press q or Ctrl-C to quit.\r\n\r\n")
		if len(events) == 0 {
			b.WriteString("waiting for child TUI mouse events...\r\n")
		}
		for _, line := range events {
			b.WriteString(line)
			b.WriteString("\r\n")
		}
		_, _ = os.Stdout.WriteString(b.String())
	}
	paint()

	buf := make([]byte, 4096)
	for {
		n, readErr := os.Stdin.Read(buf)
		if n > 0 {
			tracef("raw=%q", buf[:n])
			raw = append(raw, buf[:n]...)
			for len(raw) > 0 {
				ev, consumed, complete := parseSGRMouse(raw)
				if !complete {
					if raw[0] == 0x1b && len(raw) < 3 {
						break
					}
					if len(raw) >= 3 && raw[0] == 0x1b && raw[1] == '[' && raw[2] == '<' {
						break
					}
					consumed = 1
				}
				if consumed == 0 {
					break
				}
				chunk := append([]byte(nil), raw[:consumed]...)
				raw = raw[consumed:]
				if complete && ev.Code&64 != 0 {
					direction, _ := ev.wheel()
					tracef("event=wheel direction=%s code=%d x=%d y=%d raw=%q", direction, ev.Code, ev.X, ev.Y, chunk)
					events = append(events, fmt.Sprintf("wheel=%s code=%d x=%d y=%d raw=%q", direction, ev.Code, ev.X, ev.Y, chunk))
				} else if complete {
					tracef("event=mouse code=%d x=%d y=%d final=%q raw=%q", ev.Code, ev.X, ev.Y, ev.Final, chunk)
					events = append(events, fmt.Sprintf("mouse code=%d x=%d y=%d final=%q raw=%q", ev.Code, ev.X, ev.Y, ev.Final, chunk))
				} else if len(chunk) == 1 && (chunk[0] == 'q' || chunk[0] == 0x03) {
					return
				}
				if len(events) > *maxEvents {
					events = events[len(events)-*maxEvents:]
				}
				paint()
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				fmt.Fprintln(os.Stderr, "tui2-wheelprobe: read:", readErr)
			}
			return
		}
	}
}
