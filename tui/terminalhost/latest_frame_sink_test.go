package terminalhost

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/tui/render"
)

func TestLatestFrameSinkPropagatesUnderlyingWriteError(t *testing.T) {
	want := errors.New("write failed")
	sink := NewLatestFrameSink(frameSinkFunc(func(render.Frame) error { return want }))
	done, err := sink.WriteFrameWithCompletion(render.Frame{Lines: []string{"frame"}})
	if err != nil {
		t.Fatalf("enqueue frame: %v", err)
	}
	completion := <-done
	if completion.Written || !errors.Is(completion.Err, want) {
		t.Fatalf("underlying error must reach runtime completion, got %#v", completion)
	}
	sink.Close()
}

func TestLatestFrameSinkDropsIntermediateFramesWhileWriterBusy(t *testing.T) {
	underlying := newBlockingRecordingFrameSink()
	sink := NewLatestFrameSink(underlying)
	defer sink.Close()

	if err := sink.WriteFrame(render.Frame{Lines: []string{"one"}}); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	select {
	case <-underlying.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first underlying write")
	}
	if err := sink.WriteFrame(render.Frame{Lines: []string{"two"}}); err != nil {
		t.Fatalf("write second frame: %v", err)
	}
	if err := sink.WriteFrame(render.Frame{Lines: []string{"three"}}); err != nil {
		t.Fatalf("write third frame: %v", err)
	}
	close(underlying.release)

	if !underlying.waitFrames(2, time.Second) {
		t.Fatalf("timed out waiting for coalesced writes, frames=%#v", underlying.framesSnapshot())
	}
	got := underlying.framesSnapshot()
	if len(got) != 2 || got[0].Lines[0] != "one" || got[1].Lines[0] != "three" {
		t.Fatalf("expected first and latest frames only, got %#v", got)
	}
}

func TestLatestFrameSinkReportsDroppedCompleteFrameAsNotWritten(t *testing.T) {
	underlying := newBlockingRecordingFrameSink()
	sink := NewLatestFrameSink(underlying)
	defer sink.Close()

	if err := sink.WriteFrame(render.Frame{Lines: []string{"one"}}); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	select {
	case <-underlying.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first underlying write")
	}
	dropped, err := sink.WriteFrameWithCompletion(render.Frame{Lines: []string{"two"}})
	if err != nil {
		t.Fatalf("write dropped frame: %v", err)
	}
	written, err := sink.WriteFrameWithCompletion(render.Frame{Lines: []string{"three"}})
	if err != nil {
		t.Fatalf("write latest frame: %v", err)
	}
	if completion := <-dropped; completion.Written {
		t.Fatalf("overwritten frame must not report written completion: %#v", completion)
	}
	close(underlying.release)
	if completion := <-written; !completion.Written {
		t.Fatalf("latest frame should report written completion: %#v", completion)
	}
}

func TestLatestFrameSinkKeepsIncrementalPatchesInOrder(t *testing.T) {
	underlying := newBlockingRecordingFrameSink()
	sink := NewLatestFrameSink(underlying)
	defer sink.Close()

	if err := sink.WriteFrame(render.Frame{Lines: []string{"base"}}); err != nil {
		t.Fatalf("write base frame: %v", err)
	}
	select {
	case <-underlying.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first underlying write")
	}
	for i := 0; i < 3; i++ {
		if err := sink.WriteFrame(render.Frame{Patch: &render.FramePatch{Rect: render.Rect{W: 10, H: 3}, Dir: render.FramePatchScrollUp, LineY: i, LineANSI: string(rune('a' + i))}}); err != nil {
			t.Fatalf("write patch %d: %v", i, err)
		}
	}
	close(underlying.release)

	if !underlying.waitFrames(4, time.Second) {
		t.Fatalf("timed out waiting for patch writes, frames=%#v", underlying.framesSnapshot())
	}
	got := underlying.framesSnapshot()
	if len(got) != 4 || got[0].Lines[0] != "base" {
		t.Fatalf("expected base plus patches, got %#v", got)
	}
	for i := 1; i < 4; i++ {
		if got[i].Patch == nil || got[i].Patch.LineANSI != string(rune('a'+i-1)) {
			t.Fatalf("patch %d out of order: %#v", i, got[i])
		}
	}
}

func TestLatestFrameSinkCompleteFrameDropsStaleQueuedPatches(t *testing.T) {
	underlying := newBlockingRecordingFrameSink()
	sink := NewLatestFrameSink(underlying)
	defer sink.Close()

	if err := sink.WriteFrame(render.Frame{Lines: []string{"base"}}); err != nil {
		t.Fatalf("write base frame: %v", err)
	}
	select {
	case <-underlying.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first underlying write")
	}
	if err := sink.WriteFrame(render.Frame{Patch: &render.FramePatch{Rect: render.Rect{W: 10, H: 3}, Dir: render.FramePatchScrollUp, LineANSI: "stale"}}); err != nil {
		t.Fatalf("write stale patch: %v", err)
	}
	if err := sink.WriteFrame(render.Frame{Lines: []string{"fresh"}}); err != nil {
		t.Fatalf("write fresh frame: %v", err)
	}
	close(underlying.release)

	if !underlying.waitFrames(2, time.Second) {
		t.Fatalf("timed out waiting for coalesced complete frame, frames=%#v", underlying.framesSnapshot())
	}
	got := underlying.framesSnapshot()
	if len(got) != 2 || got[1].Patch != nil || got[1].Lines[0] != "fresh" {
		t.Fatalf("expected stale patch dropped before fresh complete frame, got %#v", got)
	}
}

func TestLatestFrameSinkNextClearsDequeuedPatchReferences(t *testing.T) {
	sink := &LatestFrameSink{}
	sink.cond = sync.NewCond(&sink.mu)
	largeLine := strings.Repeat("x", 1024)
	sink.patches = []latestFrameSinkItem{
		{frame: render.Frame{Patch: &render.FramePatch{LineANSI: largeLine}}, done: make(chan render.FrameWriteCompletion, 1)},
		{frame: render.Frame{Patch: &render.FramePatch{LineANSI: "tail"}}, done: make(chan render.FrameWriteCompletion, 1)},
	}

	if item, ok := sink.next(); !ok || item.frame.Patch == nil || item.frame.Patch.LineANSI != largeLine {
		t.Fatalf("expected first patch frame, got ok=%v item=%#v", ok, item)
	}
	retained := sink.patches[:cap(sink.patches)]
	if len(sink.patches) != 1 || retained[1].frame.Patch != nil || retained[1].done != nil {
		t.Fatalf("dequeued patch should not remain in backing array, len=%d retained=%#v", len(sink.patches), retained)
	}

	if item, ok := sink.next(); !ok || item.frame.Patch == nil || item.frame.Patch.LineANSI != "tail" {
		t.Fatalf("expected second patch frame, got ok=%v item=%#v", ok, item)
	}
	retained = sink.patches[:cap(sink.patches)]
	for i, item := range retained {
		if item.frame.Patch != nil || item.done != nil {
			t.Fatalf("patch backing array kept item at index %d: %#v", i, item)
		}
	}
}

type blockingRecordingFrameSink struct {
	mu      sync.Mutex
	started chan struct{}
	release chan struct{}
	wrote   chan struct{}
	frames  []render.Frame
	writes  int
}

type frameSinkFunc func(render.Frame) error

func (fn frameSinkFunc) WriteFrame(frame render.Frame) error { return fn(frame) }

func newBlockingRecordingFrameSink() *blockingRecordingFrameSink {
	return &blockingRecordingFrameSink{
		started: make(chan struct{}),
		release: make(chan struct{}),
		wrote:   make(chan struct{}, 8),
	}
}

func (sink *blockingRecordingFrameSink) WriteFrame(frame render.Frame) error {
	sink.mu.Lock()
	sink.writes++
	writes := sink.writes
	if writes == 1 {
		close(sink.started)
	}
	sink.mu.Unlock()
	if writes == 1 {
		<-sink.release
	}
	sink.mu.Lock()
	sink.frames = append(sink.frames, frame.Clone())
	sink.mu.Unlock()
	select {
	case sink.wrote <- struct{}{}:
	default:
	}
	return nil
}

func (sink *blockingRecordingFrameSink) waitFrames(count int, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		sink.mu.Lock()
		ok := len(sink.frames) >= count
		sink.mu.Unlock()
		if ok {
			return true
		}
		select {
		case <-sink.wrote:
		case <-deadline:
			return false
		}
	}
}

func (sink *blockingRecordingFrameSink) framesSnapshot() []render.Frame {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	out := make([]render.Frame, len(sink.frames))
	for i, frame := range sink.frames {
		out[i] = frame.Clone()
	}
	return out
}
