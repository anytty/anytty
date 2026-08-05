package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const terminalStreamInputBufferBytes = 32 << 10

func newTerminalStreamCommand(runtime terminalCommandRuntime) *cobra.Command {
	var forwardStdin bool
	command := &cobra.Command{
		Use:   "stream TARGET",
		Short: "Stream exact live PTY bytes from a terminal",
		Long:  "Stream exact future PTY output bytes from a terminal on its owning Endpoint. Output is written directly to stdout without text or ANSI decoding.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTerminalStream(cmd, runtime, args[0], forwardStdin)
		},
	}
	command.Flags().BoolVar(&forwardStdin, "stdin", false, "forward stdin bytes to the terminal until EOF")
	return command
}

func runTerminalStream(cmd *cobra.Command, runtime terminalCommandRuntime, value string, forwardStdin bool) error {
	ctx, stopSignals := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	target, err := openTerminalAutomationTarget(ctx, cmd, runtime, value)
	if err != nil {
		return err
	}
	defer target.Close()

	attached, _, detach, err := attachTerminalAutomation(ctx, target.Client, target.Ref, apipb.ResizePolicy_RESIZE_POLICY_FOLLOWER, "stream")
	if err != nil {
		return classifyCLIError(err)
	}
	defer detach()
	stream, err := target.Client.OpenResourceStream(attached.GetAttachment().GetResource())
	if err != nil {
		return classifyCLIError(err)
	}
	defer stream.Close()

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	outputDone := make(chan error, 1)
	go func() {
		outputDone <- receiveTerminalPTYStream(streamCtx, stream, cmd.OutOrStdout())
	}()

	var inputDone <-chan error
	if forwardStdin {
		restoreInput, err := makeTerminalStreamInputRaw(cmd.InOrStdin())
		if err != nil {
			return classifyCLIError(err)
		}
		defer restoreInput()
		result := make(chan error, 1)
		inputDone = result
		go func() {
			result <- forwardTerminalStreamInput(streamCtx, cmd.InOrStdin(), target.Client, attached.GetAttachment().GetResource())
		}()
	}

	for {
		select {
		case err := <-outputDone:
			cancel()
			if err != nil {
				return classifyCLIError(err)
			}
			return nil
		case err := <-inputDone:
			if err != nil {
				cancel()
				return classifyCLIError(err)
			}
			// stdin EOF does not close a shared PTY. Continue receiving until the
			// terminal exits, the caller cancels, or the Endpoint disconnects.
			inputDone = nil
		case <-ctx.Done():
			cancel()
			return classifyCLIError(ctx.Err())
		}
	}
}

func receiveTerminalPTYStream(ctx context.Context, stream clientruntime.ResourceStream, output io.Writer) error {
	for {
		typ, payload, err := stream.Receive(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		switch typ {
		case wire.TypePTYOutput:
			if err := writeTerminalStreamBytes(output, payload); err != nil {
				return err
			}
		case wire.TypeSyncLost:
			dropped, err := wire.DecodeSyncLostPayload(payload)
			if err != nil {
				return fmt.Errorf("decode PTY sync loss: %w", err)
			}
			return &cliError{code: 8, message: fmt.Sprintf("raw PTY stream lost synchronization after dropping %d bytes", dropped)}
		case wire.TypeClosed:
			if _, err := wire.DecodeClosedPayload(payload); err != nil {
				return fmt.Errorf("decode PTY close: %w", err)
			}
			return nil
		default:
			return fmt.Errorf("raw PTY stream received unsupported frame type %d", typ)
		}
	}
}

func writeTerminalStreamBytes(output io.Writer, payload []byte) error {
	for len(payload) > 0 {
		written, err := output.Write(payload)
		if err != nil {
			return err
		}
		if written <= 0 || written > len(payload) {
			return io.ErrShortWrite
		}
		payload = payload[written:]
	}
	return nil
}

func forwardTerminalStreamInput(ctx context.Context, input io.Reader, client terminalProtocolClient, attachment *apipb.ResourceHandle) error {
	buffer := make([]byte, terminalStreamInputBufferBytes)
	for {
		count, readErr := input.Read(buffer)
		if count > 0 {
			data := append([]byte(nil), buffer[:count]...)
			if err := client.TerminalInput(ctx, &apipb.TerminalInputCommand{Attachment: attachment, Data: data}); err != nil {
				return err
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
	}
}

func makeTerminalStreamInputRaw(input io.Reader) (func(), error) {
	file, ok := input.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return func() {}, nil
	}
	state, err := term.MakeRaw(int(file.Fd()))
	if err != nil {
		return nil, fmt.Errorf("enter raw stdin mode: %w", err)
	}
	return func() { _ = term.Restore(int(file.Fd()), state) }, nil
}
