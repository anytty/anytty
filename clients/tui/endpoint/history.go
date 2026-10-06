package endpoint

import (
	"context"
	"errors"
	"time"

	"github.com/anytty/anytty/clients/tui/history"
	"github.com/anytty/anytty/proto/access/apipb"
)

// HistoryBackend binds the terminal's history to its current attachment
// connection. Frozen tokens must not migrate to a reconnected generation.
func (p *RemotePTY) HistoryBackend(ctx context.Context) (history.Backend, *apipb.TerminalRef, error) {
	p.mu.Lock()
	att, closed := p.att, p.closed
	p.mu.Unlock()
	if closed || att == nil {
		return nil, nil, errors.New("terminal history: attachment is offline")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	return &historyBackend{conn: att.client, timeout: p.mgr.opts.CallTimeout},
		&apipb.TerminalRef{EndpointId: p.cfg.Name, TerminalId: p.id}, nil
}

type historyBackend struct {
	conn    sessionConn
	timeout time.Duration
}

func (b *historyBackend) call(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	select {
	case <-b.conn.Done():
		return nil, errors.New("terminal history: connection changed; reopen copy mode")
	default:
	}
	if b.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, b.timeout)
		defer cancel()
	}
	result, err := b.conn.execute(ctx, command)
	if err != nil {
		return nil, err
	}
	if err := apiErrorFromProto(result.GetError()); err != nil {
		return nil, err
	}
	return result, nil
}

func (b *historyBackend) Window(ctx context.Context, req *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error) {
	r, err := b.call(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_HistoryWindow{HistoryWindow: req}})
	if err != nil {
		return nil, err
	}
	if r.GetHistoryWindow() == nil {
		return nil, errors.New("terminal history: missing window result")
	}
	return r.GetHistoryWindow(), nil
}

func (b *historyBackend) Copy(ctx context.Context, req *apipb.HistoryCopyCommand) (*apipb.HistoryCopyResult, error) {
	r, err := b.call(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_HistoryCopy{HistoryCopy: req}})
	if err != nil {
		return nil, err
	}
	if r.GetHistoryCopy() == nil {
		return nil, errors.New("terminal history: missing copy result")
	}
	return r.GetHistoryCopy(), nil
}

func (b *historyBackend) Search(ctx context.Context, req *apipb.HistorySearchCommand) (*apipb.HistorySearchResult, error) {
	r, err := b.call(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_HistorySearch{HistorySearch: req}})
	if err != nil {
		return nil, err
	}
	if r.GetHistorySearch() == nil {
		return nil, errors.New("terminal history: missing search result")
	}
	return r.GetHistorySearch(), nil
}

func (b *historyBackend) Release(ctx context.Context, req *apipb.HistoryReleaseCommand) error {
	_, err := b.call(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_HistoryRelease{HistoryRelease: req}})
	return err
}

var _ history.Source = (*RemotePTY)(nil)
