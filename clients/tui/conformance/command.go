package conformance

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// Start implements Factory for an external candidate program. The command is
// argv, never a shell string, so quoting rules live in the caller.
func (f CommandFactory) Start() (Endpoint, error) {
	if len(f.Argv) == 0 {
		return nil, errors.New("empty candidate command")
	}
	cmd := exec.Command(f.Argv[0], f.Argv[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &commandEndpoint{cmd: cmd, stdin: stdin, stdout: stdout, stderr: stderr}, nil
}

type commandEndpoint struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr *bytes.Buffer
	closed bool
}

func (c *commandEndpoint) Write(data []byte) error {
	_, err := c.stdin.Write(data)
	return err
}

func (c *commandEndpoint) Read(data []byte) (int, error) { return c.stdout.Read(data) }

func (c *commandEndpoint) CloseInput() error {
	if c.closed {
		return nil
	}
	c.closed = true
	return c.stdin.Close()
}

func (c *commandEndpoint) Wait() (int, error) {
	err := c.cmd.Wait()
	if err == nil {
		return 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if detail := c.stderr.String(); detail != "" {
			return exit.ExitCode(), fmt.Errorf("exit %d: %s", exit.ExitCode(), trimOutput(detail))
		}
		return exit.ExitCode(), nil
	}
	return 1, err
}

func (c *commandEndpoint) Kill() {
	_ = c.stdin.Close()
	_ = c.stdout.Close()
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
}

func trimOutput(text string) string {
	if len(text) > 400 {
		return text[:400] + "..."
	}
	return text
}
