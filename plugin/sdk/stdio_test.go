package sdk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/apipb"
)

func TestStdioMultiplexesReceiveAndSend(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	receiving := make(chan struct{})
	sent := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- ServeStdio(ctx, right, right, func(ctx context.Context, cmd *apipb.PluginCommand) (*apipb.PluginResult, error) {
			if cmd.GetRegister() != nil {
				return &apipb.PluginResult{Result: &apipb.PluginResult_Registration{Registration: &apipb.PluginRegistration{Address: &apipb.PluginAddress{TuiInstanceId: "a"}, SourceLease: []byte("lease")}}}, nil
			}
			if cmd.GetReceive() != nil {
				close(receiving)
				select {
				case <-sent:
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				return &apipb.PluginResult{Result: &apipb.PluginResult_Batch{Batch: &apipb.PluginBatch{}}}, nil
			}
			if cmd.GetSend() != nil {
				close(sent)
				return &apipb.PluginResult{Result: &apipb.PluginResult_Ack{Ack: &apipb.PluginAck{Delivered: 1}}}, nil
			}
			return nil, errors.New("unexpected")
		})
	}()
	client := NewStdioClient(left, left)
	defer client.Close()
	if _, err := client.Register(ctx, &apipb.PluginRegisterRequest{}); err != nil {
		t.Fatal(err)
	}
	receiveDone := make(chan error, 1)
	go func() { _, err := client.Receive(ctx, time.Second); receiveDone <- err }()
	<-receiving
	sendCtx, stop := context.WithTimeout(ctx, time.Second)
	defer stop()
	if err := client.Send(sendCtx, &apipb.PluginMessage{RequestId: "a"}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-receiveDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-sendCtx.Done():
		t.Fatal("blocked send behind receive")
	}
	client.Close()
	right.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("bridge leaked")
	}
}
func TestFrameBoundsAndTruncation(t *testing.T) {
	var prefix [10]byte
	n := binary.PutUvarint(prefix[:], MaxFrameBytes+1)
	if err := ReadFrame(bufio.NewReader(bytes.NewReader(prefix[:n])), &apipb.PluginMessage{}); err == nil {
		t.Fatal("oversize accepted")
	}
	if err := ReadFrame(bufio.NewReader(bytes.NewReader([]byte{5, 1})), &apipb.PluginMessage{}); err == nil {
		t.Fatal("truncated accepted")
	}
}

func TestBridgeCancellationBypassesSaturatedWorkers(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	observed := make(chan struct{}, 32)
	stopped := make(chan struct{}, 32)
	done := make(chan error, 1)
	go func() {
		done <- ServeStdio(context.Background(), right, right, func(ctx context.Context, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
			if command.GetReceive() != nil {
				observed <- struct{}{}
				<-ctx.Done()
				stopped <- struct{}{}
				return nil, ctx.Err()
			}
			return &apipb.PluginResult{Result: &apipb.PluginResult_Ack{Ack: &apipb.PluginAck{Delivered: 1}}}, nil
		})
	}()
	client := NewStdioClient(left, left)
	defer client.Close()
	cancelled := make([]context.CancelFunc, 32)
	var calls sync.WaitGroup
	for i := range cancelled {
		ctx, cancel := context.WithCancel(context.Background())
		cancelled[i] = cancel
		calls.Add(1)
		go func() {
			defer calls.Done()
			_, _ = client.Execute(ctx, &apipb.PluginCommand{Command: &apipb.PluginCommand_Receive{Receive: &apipb.PluginReceiveRequest{WaitMillis: 25000}}})
		}()
	}
	for range cancelled {
		select {
		case <-observed:
		case <-time.After(time.Second):
			t.Fatal("workers did not start")
		}
	}
	_, err := client.Execute(context.Background(), &apipb.PluginCommand{Command: &apipb.PluginCommand_Send{Send: &apipb.PluginSendRequest{}}})
	var remote *Error
	if !errors.As(err, &remote) || remote.Code != "RESOURCE_EXHAUSTED" {
		t.Fatalf("saturated call=%v", err)
	}
	cancelled[0]()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("cancel was blocked behind worker admission")
	}
	// The cancelled worker releases its admission slot before sending its reply.
	deadline := time.Now().Add(time.Second)
	for {
		_, err = client.Execute(context.Background(), &apipb.PluginCommand{Command: &apipb.PluginCommand_Send{Send: &apipb.PluginSendRequest{}}})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		runtime.Gosched()
	}
	for _, cancel := range cancelled {
		cancel()
	}
	calls.Wait()
	client.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("bridge leaked after cancellation")
	}
}

func TestBridgeValidatesAndEnforcesDeadlines(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeStdio(context.Background(), right, right, func(ctx context.Context, _ *apipb.PluginCommand) (*apipb.PluginResult, error) {
			calls.Add(1)
			<-ctx.Done()
			return nil, ctx.Err()
		})
	}()
	reader := bufio.NewReader(left)
	for i, tc := range []struct {
		deadline time.Time
		code     string
	}{{time.Now().Add(-time.Second), "DEADLINE_EXCEEDED"}, {time.Now().Add(3 * time.Minute), "INVALID_REQUEST"}, {time.Now().Add(100 * time.Millisecond), "DEADLINE_EXCEEDED"}} {
		request := &apipb.PluginBridgeFrame{RequestId: fmt.Sprint(i), DeadlineUnixMillis: tc.deadline.UnixMilli(), Payload: &apipb.PluginBridgeFrame_Command{Command: &apipb.PluginCommand{Command: &apipb.PluginCommand_Receive{Receive: &apipb.PluginReceiveRequest{}}}}}
		if err := WriteFrame(left, request); err != nil {
			t.Fatal(err)
		}
		reply := &apipb.PluginBridgeFrame{}
		if err := ReadFrame(reader, reply); err != nil {
			t.Fatal(err)
		}
		if got := reply.GetResult().GetError().GetCode(); got != tc.code {
			t.Fatalf("got %s want %s", got, tc.code)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("expired/invalid requests executed: %d", calls.Load())
	}
	left.Close()
	<-done
}

func TestStdioCancellationReturnsWhileOutputIsBlocked(t *testing.T) {
	left, right := net.Pipe()
	defer right.Close()
	client := NewStdioClient(left, left)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := client.Execute(ctx, &apipb.PluginCommand{Command: &apipb.PluginCommand_Receive{Receive: &apipb.PluginReceiveRequest{}}})
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked writer prevented deadline")
	}
}
