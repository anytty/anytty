package core

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
)

var errApplicationExecutorPanic = errors.New("internal server error")

func (session *protocolSession) dispatchApplicationPayload(ctx context.Context, payload []byte) ([]byte, bool, int, error) {
	if session.application == nil {
		return nil, false, protocolErrorUnavailable, fmt.Errorf("application executor is unavailable")
	}
	command, err := protocol.DecodeApplicationCommand(payload)
	if err != nil {
		return nil, false, protocolErrorBadRequest, err
	}
	result, err := session.executeApplication(ctx, command)
	if err != nil {
		return nil, false, protocolErrorInternal, err
	}
	payload, err = protocol.EncodeApplicationResult(result)
	if err != nil {
		if errors.Is(err, protocol.ErrApplicationResultTooLarge) {
			return nil, false, protocolErrorExhausted, err
		}
		return nil, false, protocolErrorInternal, err
	}
	return payload, false, 0, nil
}

func (session *protocolSession) executeApplication(ctx context.Context, command *apipb.CommandEnvelope) (result *apipb.ResultEnvelope, err error) {
	defer func() {
		if recover() == nil {
			return
		}
		session.server.cfg.logger.Error("application executor panicked", "stack", string(debug.Stack()))
		result = nil
		err = errApplicationExecutorPanic
	}()
	return session.application.Execute(ctx, command), nil
}
