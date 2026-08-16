package webrtc

import (
	"bytes"
	"errors"
	"sync"
	"testing"

	"github.com/anytty/anytty/shared/transport/datachannel"
	pionwebrtc "github.com/pion/webrtc/v4"
)

func TestChannelBuffersMessagesUntilTransportInstallsHandler(t *testing.T) {
	dataChannel := &recordingDataChannel{}
	channel := newChannel(dataChannel)
	first := []byte("device-hello")
	dataChannel.receive(first)
	first[0] = 'X'
	dataChannel.receive([]byte("capability-accepted"))

	var received [][]byte
	channel.SetMessageHandler(func(payload []byte) {
		received = append(received, append([]byte(nil), payload...))
	})
	dataChannel.receive([]byte("protocol-result"))

	want := [][]byte{[]byte("device-hello"), []byte("capability-accepted"), []byte("protocol-result")}
	if len(received) != len(want) {
		t.Fatalf("received %d messages, want %d", len(received), len(want))
	}
	for index := range want {
		if !bytes.Equal(received[index], want[index]) {
			t.Fatalf("message %d = %q, want %q", index, received[index], want[index])
		}
	}
}

func TestChannelClosesWhenPreHandlerBufferIsExhausted(t *testing.T) {
	dataChannel := &recordingDataChannel{}
	channel := newChannel(dataChannel)
	closed := 0
	channel.SetCloseHandler(func() { closed++ })
	for index := 0; index <= preHandlerMessageLimit; index++ {
		dataChannel.receive([]byte("early-frame"))
	}
	if dataChannel.closeCalls != 1 || closed != 1 {
		t.Fatalf("overflow close calls = (%d, %d), want (1, 1)", dataChannel.closeCalls, closed)
	}
}

func TestChannelSendFailurePreservesTransportCause(t *testing.T) {
	sendErr := errors.New("data channel is closed")
	dataChannel := &failingDataChannel{sendErr: sendErr}
	channel := newChannel(dataChannel)
	transport := datachannel.New(channel)

	if err := transport.Send([]byte("request")); !errors.Is(err, sendErr) {
		t.Fatalf("send error = %v, want %v", err, sendErr)
	}
	if _, err := transport.Recv(); !errors.Is(err, sendErr) {
		t.Fatalf("recv error = %v, want %v", err, sendErr)
	}
	if dataChannel.closeCalls != 1 {
		t.Fatalf("data channel close calls = %d, want 1", dataChannel.closeCalls)
	}
}

func TestChannelCloseNotifiesTransportWhenPionDoesNotCallback(t *testing.T) {
	dataChannel := &silentCloseDataChannel{}
	channel := newChannel(dataChannel)
	transport := datachannel.New(channel)

	if err := channel.Close(); err != nil {
		t.Fatalf("close channel: %v", err)
	}
	select {
	case <-transport.Done():
	default:
		t.Fatal("local data channel close left transport lifecycle open")
	}
	if dataChannel.closeCalls != 1 {
		t.Fatalf("data channel close calls = %d, want 1", dataChannel.closeCalls)
	}
}

type failingDataChannel struct {
	sendErr      error
	closeHandler func()
	closeOnce    sync.Once
	closeCalls   int
}

func (channel *failingDataChannel) OnClose(handler func())                { channel.closeHandler = handler }
func (*failingDataChannel) OnMessage(func(pionwebrtc.DataChannelMessage)) {}
func (*failingDataChannel) OnBufferedAmountLow(func())                    {}
func (*failingDataChannel) BufferedAmount() uint64                        { return 0 }
func (*failingDataChannel) SetBufferedAmountLowThreshold(uint64)          {}
func (channel *failingDataChannel) Send([]byte) error                     { return channel.sendErr }
func (channel *failingDataChannel) Close() error {
	channel.closeCalls++
	channel.closeOnce.Do(func() {
		if channel.closeHandler != nil {
			channel.closeHandler()
		}
	})
	return nil
}

type recordingDataChannel struct {
	messageHandler func(pionwebrtc.DataChannelMessage)
	closeHandler   func()
	closeCalls     int
}

type silentCloseDataChannel struct {
	closeCalls int
}

func (*silentCloseDataChannel) OnClose(func())                                {}
func (*silentCloseDataChannel) OnMessage(func(pionwebrtc.DataChannelMessage)) {}
func (*silentCloseDataChannel) OnBufferedAmountLow(func())                    {}
func (*silentCloseDataChannel) BufferedAmount() uint64                        { return 0 }
func (*silentCloseDataChannel) SetBufferedAmountLowThreshold(uint64)          {}
func (*silentCloseDataChannel) Send([]byte) error                             { return nil }
func (channel *silentCloseDataChannel) Close() error {
	channel.closeCalls++
	return nil
}

func (channel *recordingDataChannel) OnClose(handler func()) { channel.closeHandler = handler }
func (channel *recordingDataChannel) OnMessage(handler func(pionwebrtc.DataChannelMessage)) {
	channel.messageHandler = handler
}
func (*recordingDataChannel) OnBufferedAmountLow(func())           {}
func (*recordingDataChannel) BufferedAmount() uint64               { return 0 }
func (*recordingDataChannel) SetBufferedAmountLowThreshold(uint64) {}
func (*recordingDataChannel) Send([]byte) error                    { return nil }
func (channel *recordingDataChannel) Close() error {
	channel.closeCalls++
	if channel.closeHandler != nil {
		channel.closeHandler()
	}
	return nil
}
func (channel *recordingDataChannel) receive(payload []byte) {
	channel.messageHandler(pionwebrtc.DataChannelMessage{Data: payload})
}
