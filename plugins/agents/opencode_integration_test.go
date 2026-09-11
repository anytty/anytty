package agents

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Opt-in integration starts only its own loopback OpenCode process with isolated
// configuration/data/cache. It does not call a model or alter hook trust.
func TestInstalledOpenCodeLoadsPlugin(t *testing.T) {
	if os.Getenv("ANYTTY_TEST_OPENCODE") != "1" {
		t.Skip("set ANYTTY_TEST_OPENCODE=1 to test installed OpenCode")
	}
	binary, err := exec.LookPath("opencode")
	if err != nil {
		t.Fatal(err)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	config := filepath.Join(dir, "config", "opencode")
	log := filepath.Join(dir, "events.jsonl")
	logJSON, _ := json.Marshal(log)
	receiver := filepath.Join(dir, "receiver")
	source := "#!" + node + "\nconst fs=require('node:fs');let s='';process.stdin.on('data',c=>s+=c);process.stdin.on('end',()=>fs.appendFileSync(" + string(logJSON) + ",s+'\\n'));\n"
	if err = os.WriteFile(receiver, []byte(source), 0700); err != nil {
		t.Fatal(err)
	}
	if err = InstallOpenCode(config, receiver); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	output, err := os.Create(filepath.Join(dir, "server.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	cmd := exec.Command(binary, "serve", "--hostname", "127.0.0.1", "--port", strconv.Itoa(port))
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+filepath.Join(dir, "config"), "XDG_DATA_HOME="+filepath.Join(dir, "data"), "XDG_CACHE_HOME="+filepath.Join(dir, "cache"), "XDG_STATE_HOME="+filepath.Join(dir, "state"), "OPENCODE_CONFIG_DIR="+config, "ANYTTY_TERMINAL_ID=integration-terminal", "ANYTTY_DAEMON_SOCKET=/isolated-hook-receiver.sock", "OPENCODE_SERVER_USERNAME=anytty-test", "OPENCODE_SERVER_PASSWORD=isolated-test-only")
	cmd.Stdout = output
	cmd.Stderr = output
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }() // only the process created above
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	client := &http.Client{Timeout: 30 * time.Second}
	request := func(method, path, body string) ([]byte, int, error) {
		req, err := http.NewRequest(method, base+path, strings.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
		req.SetBasicAuth("anytty-test", "isolated-test-only")
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		return data, res.StatusCode, err
	}
	ready := false
	for range 100 {
		_, status, err := request("GET", "/global/health", "")
		if err == nil && status == 200 {
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		data, _ := os.ReadFile(output.Name())
		t.Fatalf("isolated server did not start: %s", data)
	}
	data, status, err := request("POST", "/session", `{"title":"AnyTTY isolated plugin integration"}`)
	if err != nil || status != 200 {
		serverLog, _ := os.ReadFile(output.Name())
		t.Fatalf("create: %d %s %v server=%s", status, data, err, serverLog)
	}
	var session struct {
		ID string `json:"id"`
	}
	if err = json.Unmarshal(data, &session); err != nil || session.ID == "" {
		t.Fatalf("session: %s %v", data, err)
	}
	_, status, err = request("DELETE", "/session/"+session.ID, "")
	if err != nil || status != 200 {
		t.Fatalf("delete: %d %v", status, err)
	}
	for range 100 {
		data, _ = os.ReadFile(log)
		started, ended := false, false
		for _, line := range bytes.Split(bytes.TrimSpace(data), []byte{'\n'}) {
			var event Event
			if json.Unmarshal(line, &event) == nil && event.SessionID == session.ID {
				started = started || event.Kind == "start"
				ended = ended || event.Status == "exited"
			}
		}
		if started && ended {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	serverLog, _ := os.ReadFile(output.Name())
	t.Fatalf("official events missing, reports=%s server=%s", data, serverLog)
}
