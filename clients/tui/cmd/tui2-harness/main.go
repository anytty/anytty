// Command tui2-harness is the PTY-backed test driver for the tui2 smoke and
// acceptance scripts. `tui2-harness serve` speaks a tiny line protocol on
// stdin/stdout so shell scripts can use one set of primitives (spawn, send,
// resize, capture, clipboard, kill) whether tmux is installed or not.
//
// Protocol (one request per line, one response per request):
//
//	spawn <name> <cols> <rows> <base64 command>   -> OK | ERR ...
//	send <name> <base64 bytes>                    -> OK | ERR ...
//	sendhex <name> <hex bytes>                    -> OK | ERR ...
//	key <name> <base64 key name>                  -> OK | ERR ...
//	resize <name> <cols> <rows>                   -> OK | ERR ...
//	capture <name> <path>                         -> OK | ERR ...  (plain text)
//	capture_raw <name> <path>                     -> OK | ERR ...  (SGR runs)
//	cursor <name>                                 -> OK <x>,<y>
//	osc52 <name> <path>                           -> HIT | MISS
//	alive <name>                                  -> ALIVE | EXITED <code>
//	kill <name>                                   -> OK
//	killall                                       -> OK
//	ping                                          -> OK pong
//	quit                                          -> OK (then exits)
//
// Diagnostics go to stderr only; stdout carries protocol responses so the
// harness can never paint over a captured screen.
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/anytty/anytty/clients/tui/testing/harness"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "serve" {
		fmt.Fprintln(os.Stderr, "usage: tui2-harness serve")
		os.Exit(2)
	}
	if err := serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "tui2-harness:", err)
		os.Exit(1)
	}
}

func serve(in *os.File, out *os.File) error {
	manager := harness.NewManager()
	defer manager.KillAll()

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	writer := bufio.NewWriter(out)
	defer writer.Flush()
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		reply, quit := dispatch(manager, line)
		if _, err := fmt.Fprintln(writer, reply); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
		if quit {
			return nil
		}
	}
	return scanner.Err()
}

func dispatch(manager *harness.Manager, line string) (reply string, quit bool) {
	fields := strings.Fields(line)
	switch fields[0] {
	case "ping":
		return "OK pong", false
	case "quit":
		return "OK", true
	case "spawn":
		return spawn(manager, fields), false
	case "send":
		return send(manager, fields), false
	case "sendhex":
		return sendHex(manager, fields), false
	case "key":
		return sendKey(manager, fields), false
	case "resize":
		return resize(manager, fields), false
	case "capture":
		return capture(manager, fields, false), false
	case "capture_raw":
		return capture(manager, fields, true), false
	case "cursor":
		return cursor(manager, fields), false
	case "osc52":
		return osc52(manager, fields), false
	case "alive":
		return alive(manager, fields), false
	case "kill":
		return kill(manager, fields), false
	case "killall":
		manager.KillAll()
		return "OK", false
	default:
		return "ERR unknown command " + fields[0], false
	}
}

func spawn(manager *harness.Manager, fields []string) string {
	if len(fields) != 5 {
		return "ERR usage: spawn <name> <cols> <rows> <base64 command>"
	}
	cols, err := strconv.Atoi(fields[2])
	if err != nil {
		return "ERR cols: " + err.Error()
	}
	rows, err := strconv.Atoi(fields[3])
	if err != nil {
		return "ERR rows: " + err.Error()
	}
	command, err := decodeBase64(fields[4])
	if err != nil {
		return "ERR command: " + err.Error()
	}
	cfg := harness.Config{
		Argv: []string{shellPath(), "-c", string(command)},
		Cols: cols,
		Rows: rows,
	}
	if err := manager.Spawn(fields[1], cfg); err != nil {
		return "ERR spawn: " + err.Error()
	}
	return "OK"
}

func send(manager *harness.Manager, fields []string) string {
	session, errReply := lookup(manager, fields, "send", 3)
	if session == nil {
		return errReply
	}
	data, err := decodeBase64(fields[2])
	if err != nil {
		return "ERR data: " + err.Error()
	}
	if err := session.SendBytes([]byte(data)); err != nil {
		return "ERR send: " + err.Error()
	}
	return "OK"
}

func sendHex(manager *harness.Manager, fields []string) string {
	session, errReply := lookup(manager, fields, "sendhex", 3)
	if session == nil {
		return errReply
	}
	data, err := hex.DecodeString(fields[2])
	if err != nil {
		return "ERR hex: " + err.Error()
	}
	if err := session.SendBytes(data); err != nil {
		return "ERR sendhex: " + err.Error()
	}
	return "OK"
}

func sendKey(manager *harness.Manager, fields []string) string {
	session, errReply := lookup(manager, fields, "key", 3)
	if session == nil {
		return errReply
	}
	name, err := decodeBase64(fields[2])
	if err != nil {
		return "ERR key: " + err.Error()
	}
	if err := session.SendKeys(string(name)); err != nil {
		return "ERR key: " + err.Error()
	}
	return "OK"
}

func resize(manager *harness.Manager, fields []string) string {
	session, errReply := lookup(manager, fields, "resize", 4)
	if session == nil {
		return errReply
	}
	cols, err := strconv.Atoi(fields[2])
	if err != nil {
		return "ERR cols: " + err.Error()
	}
	rows, err := strconv.Atoi(fields[3])
	if err != nil {
		return "ERR rows: " + err.Error()
	}
	if err := session.Resize(cols, rows); err != nil {
		return "ERR resize: " + err.Error()
	}
	return "OK"
}

func capture(manager *harness.Manager, fields []string, raw bool) string {
	session, errReply := lookup(manager, fields, "capture", 3)
	if session == nil {
		return errReply
	}
	lines := session.CaptureText()
	if raw {
		lines = session.CaptureRaw()
	}
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}
	if err := os.WriteFile(fields[2], []byte(content), 0o644); err != nil {
		return "ERR capture: " + err.Error()
	}
	return "OK"
}

func cursor(manager *harness.Manager, fields []string) string {
	session, errReply := lookup(manager, fields, "cursor", 2)
	if session == nil {
		return errReply
	}
	x, y := session.Cursor()
	return fmt.Sprintf("OK %d,%d", x, y)
}

func osc52(manager *harness.Manager, fields []string) string {
	session, errReply := lookup(manager, fields, "osc52", 3)
	if session == nil {
		return errReply
	}
	text, ok := session.Clipboard()
	if !ok {
		return "MISS"
	}
	if err := os.WriteFile(fields[2], []byte(text), 0o644); err != nil {
		return "ERR osc52: " + err.Error()
	}
	return "HIT"
}

func alive(manager *harness.Manager, fields []string) string {
	session, errReply := lookup(manager, fields, "alive", 2)
	if session == nil {
		return errReply
	}
	if code := session.ExitCode(); code >= 0 {
		return "EXITED " + strconv.Itoa(code)
	}
	return "ALIVE"
}

func kill(manager *harness.Manager, fields []string) string {
	if len(fields) != 2 {
		return "ERR usage: kill <name>"
	}
	if err := manager.Kill(fields[1]); err != nil {
		return "ERR kill: " + err.Error()
	}
	return "OK"
}

// lookup validates the argument count and resolves the named session.
func lookup(manager *harness.Manager, fields []string, command string, count int) (*harness.Session, string) {
	if len(fields) != count {
		return nil, fmt.Sprintf("ERR usage: %s %s", command, usageTail(command))
	}
	session, err := manager.Session(fields[1])
	if err != nil {
		return nil, "ERR " + err.Error()
	}
	return session, ""
}

func usageTail(command string) string {
	switch command {
	case "send":
		return "<name> <base64 bytes>"
	case "sendhex":
		return "<name> <hex bytes>"
	case "key":
		return "<name> <base64 key name>"
	case "resize":
		return "<name> <cols> <rows>"
	case "capture", "capture_raw":
		return "<name> <path>"
	case "osc52":
		return "<name> <path>"
	default:
		return "<name>"
	}
}

func decodeBase64(value string) ([]byte, error) {
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	return base64.RawStdEncoding.DecodeString(value)
}

// shellPath returns the shell used to run command strings. $SHELL is honored
// when set; otherwise bash, then sh.
func shellPath() string {
	if shell := strings.TrimSpace(os.Getenv("TUI2_HARNESS_SHELL")); shell != "" {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}
	for _, candidate := range []string{"bash", "sh"} {
		if _, err := os.Stat("/bin/" + candidate); err == nil {
			return "/bin/" + candidate
		}
	}
	return "sh"
}
