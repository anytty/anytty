package core

import (
	"context"
	"errors"
	"io"

	"github.com/anytty/anytty/proto/wire"
)

type protocolRawPTYStream struct {
	cancel context.CancelFunc
}

func (session *protocolSession) startRawPTYStream(ctx context.Context, attachment protocolAttachment) error {
	terminal, err := session.server.Terminal(attachment.TerminalID)
	if err != nil {
		return err
	}
	streamCtx, cancel := context.WithCancel(ctx)
	stream := &protocolRawPTYStream{cancel: cancel}
	session.mu.Lock()
	if session.rawPTYStreams[attachment.Channel] != nil {
		session.mu.Unlock()
		cancel()
		return errors.New("raw PTY stream is already open for this attachment")
	}
	session.rawPTYStreams[attachment.Channel] = stream
	session.mu.Unlock()

	subscription := terminal.subscribeRawPTY(streamCtx)
	if err := session.sendFrame(attachment.Channel, wire.TypeStreamReady, nil); err != nil {
		session.clearRawPTYStream(attachment.Channel, stream)
		return err
	}
	go session.forwardRawPTYStream(streamCtx, attachment.Channel, stream, subscription)
	return nil
}

func (session *protocolSession) forwardRawPTYStream(ctx context.Context, channel uint16, stream *protocolRawPTYStream, subscription *rawPTYSubscription) {
	defer session.clearRawPTYStream(channel, stream)
	for {
		raw, err := subscription.receive(ctx)
		if err == nil {
			if ctx.Err() != nil {
				return
			}
			if err := session.sendFrame(channel, wire.TypePTYOutput, raw); err != nil {
				return
			}
			continue
		}
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return
		}
		droppedBytes, exitCode := subscription.termination()
		if errors.Is(err, errRawPTYStreamOverflow) {
			payload := wire.EncodeSyncLostPayload(droppedBytes)
			if sendErr := session.sendFrame(channel, wire.TypeSyncLost, payload); sendErr != nil {
				return
			}
			code := -1
			exitCode = &code
		} else if !errors.Is(err, io.EOF) {
			return
		}
		code := -1
		if exitCode != nil {
			code = *exitCode
		}
		_ = session.sendFrame(channel, wire.TypeClosed, wire.EncodeClosedPayload(code))
		return
	}
}

func (session *protocolSession) stopRawPTYStream(channel uint16) {
	if session == nil || channel == 0 {
		return
	}
	session.mu.Lock()
	stream := session.rawPTYStreams[channel]
	delete(session.rawPTYStreams, channel)
	session.mu.Unlock()
	if stream != nil && stream.cancel != nil {
		stream.cancel()
	}
}

func (session *protocolSession) clearRawPTYStream(channel uint16, stream *protocolRawPTYStream) {
	if session == nil || stream == nil {
		return
	}
	session.mu.Lock()
	if session.rawPTYStreams[channel] == stream {
		delete(session.rawPTYStreams, channel)
	}
	session.mu.Unlock()
	if stream.cancel != nil {
		stream.cancel()
	}
}
