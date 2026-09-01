package cloud

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	clientruntime "github.com/anytty/anytty/client/runtime"
	cloudclient "github.com/anytty/anytty/cloud/client"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCloudSignalingTerminationClassifiesAdminAndTransientClose(t *testing.T) {
	admin := cloudSignalingTermination(&cloudclient.SignalSessionCloseError{
		Code:    cloudv1.SignalSessionCloseCode_SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT,
		Message: "maintenance window",
	})
	var runtimeErr *clientruntime.Error
	if !errors.As(admin, &runtimeErr) || runtimeErr.Retryable || runtimeErr.Code != clientruntime.ErrorUserStopped || runtimeErr.Message != "maintenance window" {
		t.Fatalf("admin termination = %#v", admin)
	}
	classified := cloudConnectionError(&cloudclient.SignalSessionCloseError{
		Code:    cloudv1.SignalSessionCloseCode_SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT,
		Message: "account policy changed",
	})
	if !errors.As(classified, &runtimeErr) || runtimeErr.Retryable || runtimeErr.Code != clientruntime.ErrorUserStopped || runtimeErr.Message != "account policy changed" {
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

func TestCloudSessionLifecycleReportsWinningCloseOriginOnce(t *testing.T) {
	tests := []struct {
		name       string
		start      func(*Session, cloudSessionLifecycle, cloudSessionLifecycle)
		wantOrigin string
	}{
		{
			name: "application",
			start: func(session *Session, application, _ cloudSessionLifecycle) {
				go session.watchApplication(application)
				close(application.(*testCloudSignalLifecycle).done)
			},
			wantOrigin: "application",
		},
		{
			name: "signaling",
			start: func(session *Session, _, signaling cloudSessionLifecycle) {
				go session.watchSignaling(signaling)
				close(signaling.(*testCloudSignalLifecycle).done)
			},
			wantOrigin: "signaling",
		},
		{
			name: "local",
			start: func(session *Session, application, signaling cloudSessionLifecycle) {
				go session.watchApplication(application)
				go session.watchSignaling(signaling)
				_ = session.Close()
				close(application.(*testCloudSignalLifecycle).done)
				close(signaling.(*testCloudSignalLifecycle).done)
			},
			wantOrigin: "local",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			previousWriter, previousFlags, previousPrefix := log.Writer(), log.Flags(), log.Prefix()
			log.SetOutput(&output)
			log.SetFlags(0)
			log.SetPrefix("")
			t.Cleanup(func() {
				log.SetOutput(previousWriter)
				log.SetFlags(previousFlags)
				log.SetPrefix(previousPrefix)
			})

			session := &Session{done: make(chan struct{})}
			application := &testCloudSignalLifecycle{done: make(chan struct{}), err: errors.New("private application address")}
			signaling := &testCloudSignalLifecycle{done: make(chan struct{}), err: status.Error(codes.Unavailable, "private signaling address")}
			test.start(session, application, signaling)
			select {
			case <-session.Done():
			case <-time.After(time.Second):
				t.Fatal("Cloud session did not close")
			}
			session.finish(cloudSessionClosure{origin: cloudSessionCloseLocal})

			message := output.String()
			if got := strings.Count(message, "anytty cloud session stage=closed"); got != 1 {
				t.Fatalf("close diagnostic count = %d, log = %q", got, message)
			}
			if !strings.Contains(message, "origin="+test.wantOrigin) {
				t.Fatalf("close diagnostic = %q, want origin %q", message, test.wantOrigin)
			}
			if strings.Contains(message, "private") || strings.Contains(message, "address") {
				t.Fatalf("close diagnostic leaked raw lifecycle error: %q", message)
			}
		})
	}
}

func TestObserveCloudSessionClosureUsesCompletedLifecycleEvidence(t *testing.T) {
	adminErr := &cloudclient.SignalSessionCloseError{
		Code:    cloudv1.SignalSessionCloseCode_SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT,
		Message: "private administrator detail",
	}
	tests := []struct {
		name            string
		applicationDone bool
		signalingDone   bool
		signalingErr    error
		wantOrigin      cloudSessionCloseOrigin
	}{
		{name: "both live", wantOrigin: cloudSessionCloseLocal},
		{name: "application done", applicationDone: true, wantOrigin: cloudSessionCloseApplication},
		{name: "signaling done", signalingDone: true, signalingErr: status.Error(codes.Unavailable, "private Edge detail"), wantOrigin: cloudSessionCloseSignaling},
		{name: "both done transient", applicationDone: true, signalingDone: true, signalingErr: status.Error(codes.Unavailable, "private Edge detail"), wantOrigin: cloudSessionCloseApplication},
		{name: "both done administrative", applicationDone: true, signalingDone: true, signalingErr: adminErr, wantOrigin: cloudSessionCloseSignaling},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			application := &testCloudSignalLifecycle{done: make(chan struct{}), err: errors.New("private application detail")}
			signaling := &testCloudSignalLifecycle{done: make(chan struct{}), err: test.signalingErr}
			if test.applicationDone {
				close(application.done)
			}
			if test.signalingDone {
				close(signaling.done)
			}
			closure := observeCloudSessionClosure(application, signaling)
			if closure.origin != test.wantOrigin {
				t.Fatalf("observed origin = %q, want %q", closure.origin, test.wantOrigin)
			}
			if test.wantOrigin == cloudSessionCloseSignaling && closure.terminalErr != test.signalingErr {
				t.Fatalf("signaling terminal error = %v, want exact lifecycle error", closure.terminalErr)
			}
			if test.wantOrigin == cloudSessionCloseApplication && closure.terminalErr != application.err {
				t.Fatalf("application terminal error = %v, want exact lifecycle error", closure.terminalErr)
			}
		})
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
