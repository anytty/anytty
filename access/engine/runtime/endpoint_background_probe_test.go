package runtime

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

type initialHealthController struct {
	*supervisorController
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
	failure error
}

func (controller *initialHealthController) Probe(ctx context.Context, session ApplicationReadyPeerSession) error {
	if controller.calls.Add(1) == 1 {
		close(controller.started)
		select {
		case <-controller.release:
			return controller.failure
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return controller.supervisorController.Probe(ctx, session)
}

func TestFreshAuthenticatedSessionIsUsableBeforeBackgroundProbe(t *testing.T) {
	controller := &initialHealthController{supervisorController: newSupervisorController("studio", 0), started: make(chan struct{}), release: make(chan struct{}), failure: io.EOF}
	supervisor, err := NewEndpointSupervisor(controller, EndpointSupervisorOptions{BackgroundInitialProbe: true, ProbeTimeout: time.Second, Backoff: []time.Duration{0}, Logf: func(string, ...any) {}})
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Close()
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorTakeover)
	select {
	case <-controller.started:
	case <-time.After(time.Second):
		t.Fatal("probe did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	lease, err := supervisor.Acquire(ctx, "studio")
	if err != nil {
		t.Fatalf("fresh session waited for background health: %v", err)
	}
	if lease.Stamp().Generation != 1 {
		t.Fatal("wrong fresh generation")
	}
	_ = lease.Close()
	close(controller.release)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		projection := onlySupervisorProjection(t, supervisor)
		if projection.Phase == EndpointSupervisorReady && projection.SessionStamp.Generation == 2 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("failed background health did not invalidate and replace the session")
}
