package core

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

func TestBulkFileFrameReservesOutboundQueueForControlTraffic(t *testing.T) {
	connection := &bufferReportingTransport{sent: make(chan struct{}, 1), done: make(chan struct{})}
	connection.buffered.Store(fileTransferOutboundQueueTarget + 1)
	session := &protocolSession{conn: connection}
	result := make(chan error, 1)
	go func() { result <- session.sendBulkFileFrame(context.Background(), 3, 1, []byte("chunk")) }()

	select {
	case <-connection.sent:
		t.Fatal("bulk file frame consumed the reserved control queue")
	case <-time.After(20 * time.Millisecond):
	}
	connection.buffered.Store(0)
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("bulk file frame did not resume after the queue drained")
	}
}

type bufferReportingTransport struct {
	buffered atomic.Uint64
	sent     chan struct{}
	done     chan struct{}
}

func (transport *bufferReportingTransport) OutboundBufferedAmount() uint64 {
	return transport.buffered.Load()
}
func (transport *bufferReportingTransport) Send([]byte) error {
	transport.sent <- struct{}{}
	return nil
}
func (*bufferReportingTransport) Recv() ([]byte, error)           { return nil, io.EOF }
func (*bufferReportingTransport) Close() error                    { return nil }
func (transport *bufferReportingTransport) Done() <-chan struct{} { return transport.done }
