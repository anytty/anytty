package main

import (
	"io"
	"os/exec"
)

// process is one running layout program: the host writes frames to Stdin and
// reads frames from Stdout. Stop terminates it and releases the pipes.
type process interface {
	Stdin() io.Writer
	Stdout() io.Reader
	Stop()
}

// execProcess is the real subprocess implementation.
type execProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func startExecProcess(argv []string) (process, error) {
	return startExecProcessTo(argv, nil)
}

// startExecProcessTo is startExecProcess with an optional stderr sink (the
// -dev stderr ring buffer). A nil stderr keeps the production behavior: the
// child's stderr is discarded.
func startExecProcessTo(argv []string, stderr io.Writer) (process, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stderr = stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, err
	}
	return &execProcess{cmd: cmd, stdin: stdin, stdout: stdout}, nil
}

func (p *execProcess) Stdin() io.Writer  { return p.stdin }
func (p *execProcess) Stdout() io.Reader { return p.stdout }

func (p *execProcess) Stop() {
	_ = p.stdin.Close()
	_ = p.stdout.Close()
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	_ = p.cmd.Wait()
}
