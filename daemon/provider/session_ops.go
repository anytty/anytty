package provider

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/anytty/anytty/daemon/core"
	"github.com/anytty/anytty/daemon/core/history"
	"github.com/anytty/anytty/internal/providerproto"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	"google.golang.org/protobuf/proto"
)

// ---- history token ownership -------------------------------------------------

func (session *session) reserveHistoryToken() error {
	session.resourceMu.Lock()
	defer session.resourceMu.Unlock()
	if session.historyTokenReservations+len(session.historyTokens) >= 128 {
		return &ProviderError{Code: providerproto.ErrorExhausted, Message: "provider: history token capacity is exhausted"}
	}
	session.historyTokenReservations++
	return nil
}

func (session *session) rollbackHistoryTokenReservation() {
	session.resourceMu.Lock()
	if session.historyTokenReservations > 0 {
		session.historyTokenReservations--
	}
	session.resourceMu.Unlock()
}

func (session *session) commitHistoryToken(terminalID string, token history.HistoryToken) {
	session.resourceMu.Lock()
	if session.historyTokenReservations > 0 {
		session.historyTokenReservations--
	}
	session.historyTokens[token] = terminalID
	session.resourceMu.Unlock()
}

func (session *session) ownsHistoryToken(token history.HistoryToken) bool {
	session.resourceMu.Lock()
	defer session.resourceMu.Unlock()
	_, ok := session.historyTokens[token]
	return ok
}

func (session *session) forgetHistoryToken(token history.HistoryToken) bool {
	session.resourceMu.Lock()
	defer session.resourceMu.Unlock()
	if _, ok := session.historyTokens[token]; !ok {
		return false
	}
	delete(session.historyTokens, token)
	return true
}

func (session *session) releaseOwnedHistoryToken(terminalID string, token history.HistoryToken) {
	if !session.forgetHistoryToken(token) {
		return
	}
	_ = session.server.core.TerminalHistoryRelease(context.Background(), terminalID, token)
}

func (session *session) releaseAllHistorySnapshots() {
	session.resourceMu.Lock()
	type ownedToken struct {
		terminalID string
		token      history.HistoryToken
	}
	owned := make([]ownedToken, 0, len(session.historyTokens))
	for token, terminalID := range session.historyTokens {
		owned = append(owned, ownedToken{terminalID: terminalID, token: token})
	}
	clear(session.historyTokens)
	session.historyTokenReservations = 0
	session.resourceMu.Unlock()
	for _, item := range owned {
		_ = session.server.core.TerminalHistoryRelease(context.Background(), item.terminalID, item.token)
	}
}

func historyTokenInvalidated(err error) bool {
	return errors.Is(err, history.ErrHistoryStaleWindow)
}

// ---- live screen baselines ---------------------------------------------------

const (
	liveScreenBaselineTTL             = 2 * time.Second
	maxLiveScreenBaselineEntries      = 64
	maxLiveScreenBaselineSessionBytes = 1 << 20
	maxLiveScreenBaselineServerBytes  = 64 << 20
)

type cachedLiveScreenBaseline struct {
	screen    *core.NativeScreenBaseline
	bytes     int64
	pins      int
	expiresAt time.Time
}

type sessionLiveScreenBaselines struct {
	confirmed *cachedLiveScreenBaseline
	offered   *cachedLiveScreenBaseline
}

func (session *session) acquireLiveScreenBaseline(terminalID string, observed core.LiveRevision) (*core.NativeScreenBaseline, func()) {
	if session == nil || terminalID == "" || observed == 0 {
		return nil, func() {}
	}
	now := time.Now()
	session.liveBaselineMu.Lock()
	session.pruneLiveScreenBaselinesLocked(now)
	entry := session.liveBaselines[terminalID]
	if entry == nil {
		session.liveBaselineMu.Unlock()
		return nil, func() {}
	}
	if entry.offered != nil && entry.offered.screen.Revision() == observed {
		if entry.confirmed != nil && entry.confirmed.pins > 0 {
			session.liveBaselineMu.Unlock()
			return nil, func() {}
		}
		session.dropCachedLiveScreenBaselineLocked(entry.confirmed)
		entry.confirmed = entry.offered
		entry.offered = nil
	}
	cached := entry.confirmed
	if cached == nil || cached.screen.Revision() != observed {
		session.liveBaselineMu.Unlock()
		return nil, func() {}
	}
	cached.pins++
	cached.expiresAt = time.Time{}
	screen := cached.screen
	session.ensureLiveScreenBaselineTimerLocked()
	session.liveBaselineMu.Unlock()

	var once sync.Once
	return screen, func() {
		once.Do(func() {
			session.liveBaselineMu.Lock()
			if cached.pins > 0 {
				cached.pins--
			}
			if cached.pins == 0 {
				cached.expiresAt = time.Now().Add(liveScreenBaselineTTL)
			}
			session.ensureLiveScreenBaselineTimerLocked()
			session.liveBaselineMu.Unlock()
		})
	}
}

func (session *session) offerLiveScreenBaseline(terminalID string, screen *core.NativeScreenBaseline) {
	if session == nil || terminalID == "" || screen == nil {
		return
	}
	bytes := screen.Bytes() + 64
	if bytes <= 0 || bytes > maxLiveScreenBaselineSessionBytes {
		return
	}
	now := time.Now()
	session.liveBaselineMu.Lock()
	defer session.liveBaselineMu.Unlock()
	if session.liveBaselineClosed {
		return
	}
	session.pruneLiveScreenBaselinesLocked(now)
	entry := session.liveBaselines[terminalID]
	if entry == nil {
		if len(session.liveBaselines) >= maxLiveScreenBaselineEntries {
			return
		}
		entry = &sessionLiveScreenBaselines{}
		session.liveBaselines[terminalID] = entry
	}
	if entry.offered != nil {
		if entry.offered.pins > 0 || entry.offered.screen.Revision() > screen.Revision() {
			return
		}
		session.dropCachedLiveScreenBaselineLocked(entry.offered)
		entry.offered = nil
	}
	if session.liveBaselineBytes+bytes > maxLiveScreenBaselineSessionBytes || !session.server.reserveLiveBaselineBytes(bytes) {
		if entry.confirmed == nil {
			delete(session.liveBaselines, terminalID)
		}
		return
	}
	session.liveBaselineBytes += bytes
	entry.offered = &cachedLiveScreenBaseline{screen: screen, bytes: bytes, expiresAt: now.Add(liveScreenBaselineTTL)}
	session.ensureLiveScreenBaselineTimerLocked()
}

func (session *session) dropCachedLiveScreenBaselineLocked(cached *cachedLiveScreenBaseline) {
	if cached == nil || cached.bytes <= 0 {
		return
	}
	session.liveBaselineBytes -= cached.bytes
	if session.liveBaselineBytes < 0 {
		session.liveBaselineBytes = 0
	}
	session.server.releaseLiveBaselineBytes(cached.bytes)
	cached.bytes = 0
}

func (session *session) pruneLiveScreenBaselinesLocked(now time.Time) {
	for terminalID, entry := range session.liveBaselines {
		if cachedLiveScreenBaselineExpired(entry.confirmed, now) {
			session.dropCachedLiveScreenBaselineLocked(entry.confirmed)
			entry.confirmed = nil
		}
		if cachedLiveScreenBaselineExpired(entry.offered, now) {
			session.dropCachedLiveScreenBaselineLocked(entry.offered)
			entry.offered = nil
		}
		if entry.confirmed == nil && entry.offered == nil {
			delete(session.liveBaselines, terminalID)
		}
	}
}

func cachedLiveScreenBaselineExpired(cached *cachedLiveScreenBaseline, now time.Time) bool {
	return cached != nil && cached.pins == 0 && !cached.expiresAt.IsZero() && !now.Before(cached.expiresAt)
}

func (session *session) ensureLiveScreenBaselineTimerLocked() {
	if session.liveBaselineClosed || session.liveBaselineTimer != nil || len(session.liveBaselines) == 0 {
		return
	}
	session.liveBaselineTimer = time.AfterFunc(liveScreenBaselineTTL, session.expireLiveScreenBaselines)
}

func (session *session) expireLiveScreenBaselines() {
	session.liveBaselineMu.Lock()
	session.liveBaselineTimer = nil
	if !session.liveBaselineClosed {
		session.pruneLiveScreenBaselinesLocked(time.Now())
		session.ensureLiveScreenBaselineTimerLocked()
	}
	session.liveBaselineMu.Unlock()
}

func (session *session) clearLiveScreenBaselines() {
	if session == nil {
		return
	}
	session.liveBaselineMu.Lock()
	session.liveBaselineClosed = true
	if session.liveBaselineTimer != nil {
		session.liveBaselineTimer.Stop()
		session.liveBaselineTimer = nil
	}
	for terminalID, entry := range session.liveBaselines {
		session.dropCachedLiveScreenBaselineLocked(entry.confirmed)
		session.dropCachedLiveScreenBaselineLocked(entry.offered)
		delete(session.liveBaselines, terminalID)
	}
	session.liveBaselineMu.Unlock()
}

// ---- events ------------------------------------------------------------------

type providerEventSubscription struct {
	cancel context.CancelFunc
	filter core.EventFilter
}

func (session *session) reserveEventSubscription() error {
	session.resourceMu.Lock()
	defer session.resourceMu.Unlock()
	if session.eventSubscriptionCount >= 64 {
		return &ProviderError{Code: providerproto.ErrorExhausted, Message: "provider: event subscription capacity is exhausted"}
	}
	session.eventSubscriptionCount++
	return nil
}

func (session *session) releaseEventSubscription() {
	session.resourceMu.Lock()
	if session.eventSubscriptionCount > 0 {
		session.eventSubscriptionCount--
	}
	session.resourceMu.Unlock()
}

func (session *session) stopEvents() {
	session.eventMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(session.eventSubscriptions))
	for id, subscription := range session.eventSubscriptions {
		cancels = append(cancels, subscription.cancel)
		delete(session.eventSubscriptions, id)
	}
	session.eventMu.Unlock()
	for range cancels {
		session.releaseEventSubscription()
	}
	for _, cancel := range cancels {
		cancel()
	}
}

func eventSubscriptionToken(subscriptionID uint64) []byte {
	token := make([]byte, 10)
	copy(token, "ev")
	binary.BigEndian.PutUint64(token[2:], subscriptionID)
	return token
}

func eventSubscriptionID(token []byte) (uint64, bool) {
	if len(token) != 10 || string(token[:2]) != "ev" {
		return 0, false
	}
	return binary.BigEndian.Uint64(token[2:]), true
}

func (session *session) releaseSessionState() {
	session.stopEvents()
	session.releaseAllHistorySnapshots()
	session.clearLiveScreenBaselines()
}

// ---- dispatch ----------------------------------------------------------------

func (session *session) dispatchHistoryWindow(ctx context.Context, command *providerv1.HistoryWindowCommand) (*providerv1.Response, error) {
	request := HistoryWindowRequestFromProto(command)
	coreServer := session.server.core
	if _, err := coreServer.GetTerminal(request.TerminalID); err != nil {
		if request.Token != "" {
			session.forgetHistoryToken(request.Token)
		}
		return nil, mapCoreError(err)
	}
	latest := request.Mode == "" || request.Mode == history.HistoryWindowModeLatest
	if latest {
		if err := session.reserveHistoryToken(); err != nil {
			return nil, err
		}
		snapshot, err := coreServer.TerminalHistoryFreeze(ctx, request.TerminalID, history.FreezeHistoryRequest{
			TerminalID: request.TerminalID, Cols: request.Cols, Limit: request.Limit,
		})
		if err != nil {
			session.rollbackHistoryTokenReservation()
			return nil, mapCoreError(err)
		}
		session.commitHistoryToken(request.TerminalID, snapshot.Token)
		request.Token = snapshot.Token
		if err := ctx.Err(); err != nil {
			session.releaseOwnedHistoryToken(request.TerminalID, request.Token)
			return nil, &ProviderError{Code: providerproto.ErrorUnavailable, Message: err.Error()}
		}
	} else if request.Token == "" {
		return nil, mapCoreError(history.ErrHistoryInvalidMutation)
	} else if !session.ownsHistoryToken(request.Token) {
		return nil, mapCoreError(history.ErrHistoryStaleWindow)
	}
	window, err := coreServer.TerminalHistoryWindow(ctx, request.TerminalID, request)
	if err != nil {
		if latest || historyTokenInvalidated(err) {
			session.releaseOwnedHistoryToken(request.TerminalID, request.Token)
		}
		return nil, mapCoreError(err)
	}
	if latest && ctx.Err() != nil {
		session.releaseOwnedHistoryToken(request.TerminalID, request.Token)
		return nil, &ProviderError{Code: providerproto.ErrorUnavailable, Message: ctx.Err().Error()}
	}
	return &providerv1.Response{Result: &providerv1.Response_HistoryWindow{HistoryWindow: historyWindowToProto("", window)}}, nil
}

func (session *session) dispatchHistoryCopy(ctx context.Context, command *providerv1.HistoryCopyCommand) (*providerv1.Response, error) {
	coreServer := session.server.core
	if command.GetMaxLines() > 0 || command.GetMaxBytes() > 0 {
		request := HistoryCopyChunkRequestFromProto(command)
		if _, err := coreServer.GetTerminal(request.TerminalID); err != nil {
			if request.Token != "" {
				session.forgetHistoryToken(request.Token)
			}
			return nil, mapCoreError(err)
		}
		if request.Token == "" {
			return nil, mapCoreError(history.ErrHistoryInvalidMutation)
		}
		if !session.ownsHistoryToken(request.Token) {
			return nil, mapCoreError(history.ErrHistoryStaleWindow)
		}
		result, err := coreServer.TerminalHistoryCopyChunk(ctx, request.TerminalID, request)
		if historyTokenInvalidated(err) {
			session.releaseOwnedHistoryToken(request.TerminalID, request.Token)
		}
		if err != nil {
			return nil, mapCoreError(err)
		}
		return &providerv1.Response{Result: &providerv1.Response_HistoryCopy{HistoryCopy: HistoryCopyChunkToProto(result)}}, nil
	}
	request := HistoryCopyRequestFromProto(command)
	if _, err := coreServer.GetTerminal(request.TerminalID); err != nil {
		if request.Token != "" {
			session.forgetHistoryToken(request.Token)
		}
		return nil, mapCoreError(err)
	}
	if request.Token == "" {
		return nil, mapCoreError(history.ErrHistoryInvalidMutation)
	}
	if !session.ownsHistoryToken(request.Token) {
		return nil, mapCoreError(history.ErrHistoryStaleWindow)
	}
	text, err := coreServer.TerminalHistoryCopy(ctx, request.TerminalID, request)
	if historyTokenInvalidated(err) {
		session.releaseOwnedHistoryToken(request.TerminalID, request.Token)
	}
	if err != nil {
		return nil, mapCoreError(err)
	}
	return &providerv1.Response{Result: &providerv1.Response_HistoryCopy{HistoryCopy: HistoryCopyToProto(text)}}, nil
}

func (session *session) dispatchHistorySearch(ctx context.Context, command *providerv1.HistorySearchCommand) (*providerv1.Response, error) {
	request := HistorySearchRequestFromProto(command)
	coreServer := session.server.core
	if _, err := coreServer.GetTerminal(request.TerminalID); err != nil {
		if request.Token != "" {
			session.forgetHistoryToken(request.Token)
		}
		return nil, mapCoreError(err)
	}
	if request.Token == "" {
		return nil, mapCoreError(history.ErrHistoryInvalidMutation)
	}
	if !session.ownsHistoryToken(request.Token) {
		return nil, mapCoreError(history.ErrHistoryStaleWindow)
	}
	result, err := coreServer.TerminalHistorySearch(ctx, request.TerminalID, request)
	if historyTokenInvalidated(err) {
		session.releaseOwnedHistoryToken(request.TerminalID, request.Token)
	}
	if err != nil {
		return nil, mapCoreError(err)
	}
	return &providerv1.Response{Result: &providerv1.Response_HistorySearch{HistorySearch: HistorySearchToProto("", result)}}, nil
}

func (session *session) dispatchHistoryRelease(ctx context.Context, command *providerv1.HistoryReleaseCommand) (*providerv1.Response, error) {
	terminalID := command.GetTerminal().GetTerminalId()
	token := history.HistoryToken(command.GetToken())
	if token == "" {
		return nil, mapCoreError(history.ErrHistoryInvalidMutation)
	}
	if !session.forgetHistoryToken(token) {
		return nil, mapCoreError(history.ErrHistoryStaleWindow)
	}
	if _, err := session.server.core.GetTerminal(terminalID); err == nil {
		if err := session.server.core.TerminalHistoryRelease(ctx, terminalID, token); err != nil {
			return nil, mapCoreError(err)
		}
	}
	return &providerv1.Response{Result: &providerv1.Response_HistoryRelease{HistoryRelease: &providerv1.Acknowledge{}}}, nil
}

func (session *session) dispatchHistoryBacklog(command *providerv1.HistoryBacklogStatusCommand) (*providerv1.Response, error) {
	status, err := session.server.core.TerminalHistoryBacklogStatus(command.GetTerminal().GetTerminalId())
	if err != nil {
		return nil, mapCoreError(err)
	}
	return &providerv1.Response{Result: &providerv1.Response_HistoryBacklogStatus{HistoryBacklogStatus: historyBacklogToProto("", status)}}, nil
}

func (session *session) dispatchLiveScreenNext(ctx context.Context, command *providerv1.LiveScreenNextCommand) (*providerv1.Response, error) {
	terminalID := command.GetTerminal().GetTerminalId()
	if _, err := session.server.core.GetTerminal(terminalID); err != nil {
		return nil, mapCoreError(err)
	}
	observed := core.LiveRevision(command.GetObservedRevision())
	base, releaseBase := session.acquireLiveScreenBaseline(terminalID, observed)
	defer releaseBase()
	snapshot, next, err := session.server.core.NextLiveScreenWithBaseline(ctx, terminalID, observed, base)
	if err != nil {
		return nil, mapCoreError(err)
	}
	session.offerLiveScreenBaseline(terminalID, next)
	return &providerv1.Response{Result: &providerv1.Response_LiveScreen{LiveScreen: nativeScreenToProto("", snapshot)}}, nil
}

func (session *session) dispatchEventSubscribe(ctx context.Context, command *providerv1.EventSubscribeCommand) (*providerv1.Response, error) {
	if err := session.reserveEventSubscription(); err != nil {
		return nil, err
	}
	filter := core.EventFilter{TerminalID: command.GetTerminalId(), Types: eventTypesFromProto(command.GetTypes())}
	eventCtx, cancel := context.WithCancel(context.Background())
	session.eventMu.Lock()
	session.nextEventSub++
	subscriptionID := session.nextEventSub
	session.eventSubscriptions[subscriptionID] = &providerEventSubscription{cancel: cancel, filter: filter}
	session.eventMu.Unlock()
	_ = ctx
	token := eventSubscriptionToken(subscriptionID)
	initial := session.attachmentSnapshotEvents(filter)
	events := session.server.core.Events(eventCtx, filter)
	go func() {
		defer session.clearEventSubscription(subscriptionID)
		send := func(event *providerv1.TerminalEvent) bool {
			if event == nil {
				return true
			}
			payload, err := encodeEventPayload(event)
			if err != nil {
				return true
			}
			return session.sendFrame(0, providerproto.TypeEvent, payload) == nil
		}
		for _, event := range initial {
			if !send(event) {
				return
			}
		}
		for {
			select {
			case <-eventCtx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				if !send(terminalEventToProto(event)) {
					return
				}
			}
		}
	}()
	return &providerv1.Response{Result: &providerv1.Response_EventSubscribe{EventSubscribe: &providerv1.EventSubscriptionResult{
		OpaqueToken:   token,
		InitialEvents: initial,
	}}}, nil
}

func (session *session) dispatchEventRelease(command *providerv1.EventSubscriptionReleaseCommand) (*providerv1.Response, error) {
	id, ok := eventSubscriptionID(command.GetOpaqueToken())
	if !ok {
		return nil, &ProviderError{Code: providerproto.ErrorNotFound, Message: "provider: event subscription token is invalid"}
	}
	session.eventMu.Lock()
	subscription := session.eventSubscriptions[id]
	delete(session.eventSubscriptions, id)
	session.eventMu.Unlock()
	if subscription == nil {
		return nil, &ProviderError{Code: providerproto.ErrorNotFound, Message: "provider: event subscription is not active"}
	}
	session.releaseEventSubscription()
	subscription.cancel()
	return &providerv1.Response{Result: &providerv1.Response_EventRelease{EventRelease: &providerv1.Acknowledge{}}}, nil
}

func (session *session) clearEventSubscription(id uint64) {
	session.eventMu.Lock()
	_, exists := session.eventSubscriptions[id]
	if exists {
		delete(session.eventSubscriptions, id)
	}
	session.eventMu.Unlock()
	if exists {
		session.releaseEventSubscription()
	}
}

func (session *session) attachmentSnapshotEvents(filter core.EventFilter) []*providerv1.TerminalEvent {
	terminals := session.server.core.ListTerminals()
	if len(terminals) == 0 {
		return nil
	}
	if !eventTypeMatchesFilter(core.EventTerminalMetadataChanged, filter) {
		return nil
	}
	now := time.Now().UTC()
	events := make([]*providerv1.TerminalEvent, 0, len(terminals))
	for index := range terminals {
		terminal := terminals[index].Clone()
		if filter.TerminalID != "" && filter.TerminalID != terminal.ID {
			continue
		}
		info := terminalInfoToProto(terminal)
		events = append(events, &providerv1.TerminalEvent{
			Type:              providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_METADATA_CHANGED,
			TerminalId:        terminal.ID,
			Terminal:          info,
			Attachment:        session.server.registry.projection(terminal.ID, terminal.Size),
			TimestampUnixNano: now.UnixNano(),
		})
	}
	return events
}

func eventTypeMatchesFilter(typ core.EventType, filter core.EventFilter) bool {
	if len(filter.Types) == 0 {
		return true
	}
	for _, candidate := range filter.Types {
		if candidate == typ {
			return true
		}
	}
	return false
}

func eventTypesFromProto(types []providerv1.TerminalEventType) []core.EventType {
	out := make([]core.EventType, 0, len(types))
	for _, typ := range types {
		switch typ {
		case providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_CREATED:
			out = append(out, core.EventTerminalCreated)
		case providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_EXITED:
			out = append(out, core.EventTerminalExited)
		case providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_RESIZED:
			out = append(out, core.EventTerminalResized)
		case providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_METADATA_CHANGED:
			out = append(out, core.EventTerminalMetadataChanged)
		case providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_REMOVED:
			out = append(out, core.EventTerminalRemoved)
		case providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_CHANGED:
			out = append(out, core.EventTerminalChanged)
		}
	}
	return out
}

func eventTypeToProto(typ core.EventType) providerv1.TerminalEventType {
	switch typ {
	case core.EventTerminalCreated:
		return providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_CREATED
	case core.EventTerminalExited:
		return providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_EXITED
	case core.EventTerminalResized:
		return providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_RESIZED
	case core.EventTerminalMetadataChanged:
		return providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_METADATA_CHANGED
	case core.EventTerminalRemoved:
		return providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_REMOVED
	case core.EventTerminalChanged:
		return providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_CHANGED
	default:
		return providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_UNSPECIFIED
	}
}

func terminalEventToProto(event core.Event) *providerv1.TerminalEvent {
	projection := &providerv1.TerminalEvent{
		Type:              eventTypeToProto(event.Type),
		TerminalId:        event.TerminalID,
		TimestampUnixNano: unixNanoOrZero(event.Timestamp),
	}
	if event.Terminal != nil {
		projection.Terminal = terminalInfoToProto(*event.Terminal)
	}
	if event.Attachment != nil {
		projection.Attachment = &providerv1.TerminalAttachmentProjection{
			AttachmentCount: int32(event.Attachment.AttachmentCount),
			ResizeEpoch:     event.Attachment.ResizeEpoch,
			ResizeControl:   resizeControlToProto(event.Attachment.ResizeControl),
		}
	}
	if projection.GetType() == providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_UNSPECIFIED && projection.GetTerminal() == nil {
		return nil
	}
	return projection
}

func encodeEventPayload(event *providerv1.TerminalEvent) ([]byte, error) {
	return proto.Marshal(&providerv1.Event{TerminalEvent: event})
}

func resizeControlToProto(control *core.TerminalResizeControl) *providerv1.ResizeControl {
	if control == nil {
		return nil
	}
	projection := &providerv1.ResizeControl{
		CanResize:      control.CanResize,
		Reason:         resizeReasonFromCore(control.Reason),
		SizeLocked:     control.SizeLocked,
		SurfaceId:      control.SurfaceID,
		OwnerSurfaceId: control.OwnerSurfaceID,
		OwnerViewId:    control.OwnerViewID,
	}
	if control.ResizeOwnership != nil {
		projection.ResizeOwnership = &providerv1.ResizeOwnership{
			OwnerAttachmentId: control.ResizeOwnership.OwnerAttachmentID,
			OwnerSurfaceId:    control.ResizeOwnership.OwnerSurfaceID,
			OwnerViewId:       control.ResizeOwnership.OwnerViewID,
			Size:              sizeToProto(control.ResizeOwnership.Size),
			SizeLocked:        control.ResizeOwnership.SizeLocked,
			Epoch:             control.ResizeOwnership.Epoch,
		}
	}
	return projection
}

func resizeReasonFromCore(reason core.TerminalResizeReason) providerv1.ResizeReason {
	switch reason {
	case core.TerminalResizeReasonOwner:
		return providerv1.ResizeReason_RESIZE_REASON_OWNER
	case core.TerminalResizeReasonObserver:
		return providerv1.ResizeReason_RESIZE_REASON_OBSERVER
	case core.TerminalResizeReasonSizeLocked:
		return providerv1.ResizeReason_RESIZE_REASON_SIZE_LOCKED
	default:
		return providerv1.ResizeReason_RESIZE_REASON_FOLLOWER
	}
}
