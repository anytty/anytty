package provider

import (
	"errors"
	"fmt"
	"time"

	"github.com/anytty/anytty/internal/providerproto"
	"github.com/anytty/anytty/pool/core"
	"github.com/anytty/anytty/pool/core/history"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
)

// ProviderError 是 provider typed 错误；Code 语义与 clientruntime/access 对齐。
type ProviderError struct {
	Code    uint32
	Message string
}

func (err *ProviderError) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

func terminalRecordFromSpec(spec *providerv1.TerminalCreateSpec) (core.TerminalRecord, error) {
	if spec == nil || spec.GetTerminalId() == "" {
		return core.TerminalRecord{}, &ProviderError{Code: providerproto.ErrorBadRequest, Message: "terminal id is required"}
	}
	record := core.TerminalRecord{
		ID:      spec.GetTerminalId(),
		Name:    spec.GetName(),
		Command: append([]string(nil), spec.GetCommand()...),
		Tags:    cloneStringMap(spec.GetTags()),
		Size: core.Size{
			Cols: uint16(spec.GetSize().GetCols()),
			Rows: uint16(spec.GetSize().GetRows()),
		},
		Options: core.TerminalCreateOptions{
			Dir:                spec.GetCwd(),
			Env:                append([]string(nil), spec.GetEnv()...),
			ScrollbackSize:     int(spec.GetScrollback().GetSize()),
			ScrollbackMaxBytes: spec.GetScrollback().GetMaxBytes(),
			ScrollbackMaxAge:   time.Duration(spec.GetScrollback().GetMaxAgeMillis()) * time.Millisecond,
		},
	}
	return record, nil
}

func terminalInfoToProto(info core.TerminalInfo) *providerv1.TerminalInfo {
	return terminalInfoToProtoWithAttachments(info, 0)
}

func terminalInfoToProtoWithAttachments(info core.TerminalInfo, attachmentCount int) *providerv1.TerminalInfo {
	projection := &providerv1.TerminalInfo{
		TerminalId:           info.ID,
		Name:                 info.Name,
		Command:              append([]string(nil), info.Command...),
		Tags:                 cloneStringMap(info.Tags),
		Size:                 &providerv1.Size{Cols: uint32(info.Size.Cols), Rows: uint32(info.Size.Rows)},
		State:                string(info.State),
		Cwd:                  info.CWD,
		LiveCwd:              info.LiveCWD,
		CreatedAtUnixNano:    unixNanoOrZero(info.CreatedAt),
		ExitedAtUnixNano:     unixNanoOrZero(info.ExitedAt),
		ForegroundProcess:    info.ForegroundProcess,
		ForegroundCwd:        info.ForegroundCWD,
		LastOutputAtUnixNano: unixNanoOrZero(info.LastOutputAt),
		AttachmentCount:      int32(attachmentCount),
	}
	if info.ExitCode != nil {
		projection.ExitCode = int32(*info.ExitCode)
	}
	return projection
}

func unixNanoOrZero(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UnixNano()
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func mapCoreError(err error) error {
	if err == nil {
		return nil
	}
	code := providerproto.ErrorInternal
	switch {
	case errors.Is(err, core.ErrTerminalNotFound):
		code = providerproto.ErrorNotFound
	case errors.Is(err, core.ErrDuplicateTerminal):
		code = providerproto.ErrorConflict
	case errors.Is(err, core.ErrInvalidTerminalID), errors.Is(err, core.ErrInvalidCommand), errors.Is(err, core.ErrInvalidServerSize):
		code = providerproto.ErrorBadRequest
	case errors.Is(err, core.ErrProtocolResourceExhausted):
		code = providerproto.ErrorExhausted
	case errors.Is(err, core.ErrServerClosed), errors.Is(err, core.ErrHistoryDisabled), errors.Is(err, core.ErrTerminalOutputUnavailable):
		code = providerproto.ErrorUnavailable
	case errors.Is(err, history.ErrHistoryStaleWindow):
		code = providerproto.ErrorStaleResource
	case errors.Is(err, history.ErrHistoryInvalidMutation), errors.Is(err, history.ErrHistoryInvalidSearchPattern):
		code = providerproto.ErrorBadRequest
	case errors.Is(err, history.ErrHistoryWindowLimit), errors.Is(err, history.ErrHistoryCopyTooLarge), errors.Is(err, history.ErrHistoryWindowTooLarge):
		code = providerproto.ErrorExhausted
	}
	return &ProviderError{Code: code, Message: fmt.Sprintf("%s", err)}
}
