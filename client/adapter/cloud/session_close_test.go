package cloud

import (
	"context"
	"errors"
	"testing"
	"time"

	clientruntime "github.com/anytty/anytty/client/runtime"
	cloudclient "github.com/anytty/anytty/cloud/client"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

func TestCloudSignalingTerminationClassifiesAdminAndTransientClose(t *testing.T) {
	admin := cloudSignalingTermination(&cloudclient.SignalSessionCloseError{
		Code:    cloudv1.SignalSessionCloseCode_SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT,
		Message: "maintenance window",
	})
	var runtimeErr *clientruntime.Error
	if !errors.As(admin, &runtimeErr) || runtimeErr.Retryable || runtimeErr.Code != clientruntime.ErrorUnavailable || runtimeErr.Message != "maintenance window" {
		t.Fatalf("admin termination = %#v", admin)
	}
	classified := cloudConnectionError(&cloudclient.SignalSessionCloseError{
		Code:    cloudv1.SignalSessionCloseCode_SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT,
		Message: "account policy changed",
	})
	if !errors.As(classified, &runtimeErr) || runtimeErr.Retryable || runtimeErr.Message != "account policy changed" {
		t.Fatalf("pre-ready admin classification = %#v", classified)
	}

	transient := cloudSignalingTermination(errors.New("edge unavailable"))
	if !errors.As(transient, &runtimeErr) || !runtimeErr.Retryable || runtimeErr.Code != clientruntime.ErrorUnavailable {
		t.Fatalf("transient termination = %#v", transient)
	}
}

func TestAdministrativeCloseCancelsPostAnswerPeerReadyWithTypedCause(t *testing.T) {
	signaling := &testCloudSignalLifecycle{done: make(chan struct{})}
	waiter := &testCloudPeerReadyWaiter{started: make(chan struct{})}
	ctx, stop := cloudPeerSetupContext(context.Background(), signaling)
	defer stop()
	result := make(chan error, 1)
	go func() { result <- waitCloudPeerReady(ctx, waiter, signaling) }()
	<-waiter.started
	signaling.err = &cloudclient.SignalSessionCloseError{
		Code:    cloudv1.SignalSessionCloseCode_SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT,
		Message: "device access revoked",
	}
	close(signaling.done)

	select {
	case err := <-result:
		if !cloudclient.IsAdminDisconnect(err) || err.Error() != "device access revoked" {
			t.Fatalf("post-answer peer-ready error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("post-answer peer-ready wait ignored administrative close")
	}
}

type testCloudSignalLifecycle struct {
	done chan struct{}
	err  error
}

func (lifecycle *testCloudSignalLifecycle) Done() <-chan struct{} { return lifecycle.done }
func (lifecycle *testCloudSignalLifecycle) Err() error            { return lifecycle.err }

type testCloudPeerReadyWaiter struct{ started chan struct{} }

func (waiter *testCloudPeerReadyWaiter) WaitReady(ctx context.Context) error {
	close(waiter.started)
	<-ctx.Done()
	return context.Cause(ctx)
}
