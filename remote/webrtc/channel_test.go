package webrtc

import (
	"bytes"
	"errors"
	"testing"

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

func TestChannelSendFailureNotifiesLifecycleOnce(t *testing.T) {
	sendErr := errors.New("data channel is closed")
	dataChannel := &failingDataChannel{sendErr: sendErr}
	channel := newChannel(dataChannel)
	closed := 0
	channel.SetCloseHandler(func() { closed++ })

	if err := channel.Send([]byte("request")); !errors.Is(err, sendErr) {
		t.Fatalf("send error = %v, want %v", err, sendErr)
	}
	if err := channel.Send([]byte("retry")); !errors.Is(err, sendErr) {
		t.Fatalf("second send error = %v, want %v", err, sendErr)
	}
	if closed != 1 {
		t.Fatalf("close notifications = %d, want 1", closed)
	}
}

type failingDataChannel struct {
	sendErr error
}

func (*failingDataChannel) OnClose(func())                                {}
func (*failingDataChannel) OnMessage(func(pionwebrtc.DataChannelMessage)) {}
func (*failingDataChannel) OnBufferedAmountLow(func())                    {}
func (*failingDataChannel) BufferedAmount() uint64                        { return 0 }
func (*failingDataChannel) SetBufferedAmountLowThreshold(uint64)          {}
func (channel *failingDataChannel) Send([]byte) error                     { return channel.sendErr }
func (*failingDataChannel) Close() error                                  { return nil }

type recordingDataChannel struct {
	messageHandler func(pionwebrtc.DataChannelMessage)
	closeHandler   func()
	closeCalls     int
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
