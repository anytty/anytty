package main

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestGlobalTimeoutBoundsCompleteCommand(t *testing.T) {
	timeout := 20 * time.Millisecond
	command := &cobra.Command{
		Use: "timeout-probe",
		RunE: func(cmd *cobra.Command, _ []string) error {
			<-cmd.Context().Done()
			return classifyCLIError(cmd.Context().Err())
		},
	}
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	wrapGlobalTimeout(command, &timeout)
	started := time.Now()
	err := command.Execute()
	if cliExitCode(err) != 7 || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("global timeout error = %v, exit=%d", err, cliExitCode(err))
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("global timeout took %s", elapsed)
	}
}

func TestGlobalTimeoutRejectsNegativeDuration(t *testing.T) {
	command := newRootCmd()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"--timeout", "-1s", "config", "paths"})
	if err := command.Execute(); cliExitCode(err) != 2 {
		t.Fatalf("negative global timeout = %v, exit=%d", err, cliExitCode(err))
	}
}
