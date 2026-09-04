package core

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"strconv"

	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/proto/wire"
)

type protocolAdmissionLease struct{}

// Release 结束当前 request 的 connection-bound admission lease。
func (protocolAdmissionLease) Release() {}

// AcquireApplication 在当前 protocol request 生命周期内校验 immutable transport scope。
// request goroutine 本身就是 connection-bound lease；session close 会先 cancel，再等待该 goroutine退出。
func (session *protocolSession) AcquireApplication(ctx context.Context, admission ApplicationAdmission) (ApplicationAdmissionLease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !applicationCapabilitySupported(admission.Capability) {
		return nil, ErrApplicationUnsupportedCapability
	}
	scope := session.scope.normalized()
	if admission.Capability == ApplicationCapabilityFile {
		if !applicationFileAllowed(scope, admission.FileOperation) {
			return nil, ErrApplicationForbidden
		}
		return protocolAdmissionLease{}, nil
	}
	if admission.Capability == ApplicationCapabilityClientAccess {
		if !scope.LocalOwner && !scope.ManageClientAccess {
			return nil, ErrApplicationForbidden
		}
		return protocolAdmissionLease{}, nil
	}
	if admission.Capability == ApplicationCapabilityRemoteControl {
		if !scope.LocalOwner {
			return nil, ErrApplicationForbidden
		}
		return protocolAdmissionLease{}, nil
	}
	if scope.AllowDaemon {
		return protocolAdmissionLease{}, nil
	}
	if scope.MachineEventsOnly {
		if admission.Capability == ApplicationCapabilityEventSubscription && admission.MachineLifecycleEventsOnly {
			return protocolAdmissionLease{}, nil
		}
		if admission.Capability == ApplicationCapabilityResourceLifecycle && admission.ResourceKind == ApplicationResourceKindSubscription {
			if subscription, ok := session.eventSubscriptionForToken(admission.ResourceToken); ok && machineLifecycleEventFilter(subscription.filter) {
				return protocolAdmissionLease{}, nil
			}
		}
		return nil, ErrApplicationForbidden
	}
	if admission.Capability == ApplicationCapabilityTerminalInventory {
		if scope.TerminalID == "" {
			return nil, ErrApplicationForbidden
		}
		return protocolAdmissionLease{}, nil
	}
	if admission.Capability == ApplicationCapabilityResourceLifecycle && admission.ResourceKind == ApplicationResourceKindSubscription {
		subscription, ok := session.eventSubscriptionForToken(admission.ResourceToken)
		if !ok || subscription.filter.TerminalID != scope.TerminalID {
			return nil, ErrApplicationForbidden
		}
		return protocolAdmissionLease{}, nil
	}
	target := admission.TerminalID
	if len(admission.ResourceToken) > 0 {
		attachment, err := session.attachmentForToken(admission.ResourceToken)
		if err != nil {
			return nil, ErrApplicationForbidden
		}
		target = attachment.TerminalID
	}
	if target == "" || target != scope.TerminalID {
		return nil, ErrApplicationForbidden
	}
	return protocolAdmissionLease{}, nil
}

func applicationCapabilitySupported(capability ApplicationCapability) bool {
	switch capability {
	case ApplicationCapabilityResourceLifecycle,
		ApplicationCapabilityTerminalLifecycle,
		ApplicationCapabilityTerminalInventory,
		ApplicationCapabilityTerminalAttachment,
		ApplicationCapabilityPathQuery,
		ApplicationCapabilityHistory,
		ApplicationCapabilityLiveScreen,
		ApplicationCapabilityFile,
		ApplicationCapabilityStorage,
		ApplicationCapabilityEventSubscription,
		ApplicationCapabilityClientAccess,
		ApplicationCapabilityRemoteControl,
		ApplicationCapabilityBrowserProxy:
		return true
	default:
		return false
	}
}

func applicationFileAllowed(scope TransportScope, operation string) bool {
	if !scope.AllowDaemon {
		return false
	}
	switch operation {
	case "list", "stat":
		return scope.FileReadMetadata
	case "preview", "download":
		return scope.FileReadContent
	case "upload":
		return scope.FileWriteContent
	case "cancel":
		return scope.FileReadContent || scope.FileWriteContent
	case "mkdir", "rename", "delete", "move":
		return scope.FileMutate
	case "copy":
		return scope.FileMutate && scope.FileReadContent
	default:
		return false
	}
}

// CancelApplicationOperation 当前没有 daemon operation registry，因此始终 fail closed。
func (session *protocolSession) CancelApplicationOperation(context.Context, string) error {
	return ErrApplicationCancellationUnavailable
}

// ReleaseApplicationResource 释放当前 session registry 持有的 opaque attachment token。
func (session *protocolSession) ReleaseApplicationResource(_ context.Context, token []byte) error {
	if proxy := session.browserProxyForToken(token); proxy != nil {
		session.removeBrowserProxy(proxy)
		return nil
	}
	attachment, err := session.attachmentForToken(token)
	if err == nil {
		session.detach(attachmentDetachRequest{Channel: attachment.Channel})
		return nil
	}
	if subscriptionID, ok := applicationEventSubscriptionID(token); ok {
		session.mu.Lock()
		subscription := session.eventSubscriptions[subscriptionID]
		if subscription.cancel != nil {
			delete(session.eventSubscriptions, subscriptionID)
		}
		session.mu.Unlock()
		if subscription.cancel == nil {
			return ErrApplicationForbidden
		}
		session.releaseEventSubscription()
		subscription.cancel()
		return nil
	}
	if transferID, ok := fileTransferIDFromResourceToken(token); ok {
		session.releaseFileTransfer(transferID, false)
		return nil
	}
	return err
}

// ApplicationBrowserProxyOpen dials from the daemon host and publishes the
// resulting bidirectional byte stream only after the connection succeeds.
func (session *protocolSession) ApplicationBrowserProxyOpen(ctx context.Context, host string, port uint16) (BrowserProxy, error) {
	if !session.scope.AllowDaemon {
		return BrowserProxy{}, ErrApplicationForbidden
	}
	if ctx == nil {
		ctx = context.Background()
	}
	address := net.JoinHostPort(host, strconv.Itoa(int(port)))
	session.server.cfg.logger.Info(
		"browser proxy dialing remote target",
		"session_id", session.sessionID,
		"host", host,
		"port", port,
	)
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		session.server.cfg.logger.Warn(
			"browser proxy remote dial failed",
			"session_id", session.sessionID,
			"host", host,
			"port", port,
			"error", err,
		)
		return BrowserProxy{}, err
	}
	channel, err := session.reserveBrowserChannel()
	if err != nil {
		_ = conn.Close()
		return BrowserProxy{}, err
	}
	token := make([]byte, 34)
	binary.BigEndian.PutUint16(token[:2], channel)
	if _, err := rand.Read(token[2:]); err != nil {
		_ = conn.Close()
		session.releaseChannel(channel, protocolChannelBrowserProxy)
		return BrowserProxy{}, err
	}
	proxy := &sessionBrowserProxy{channel: channel, token: token, conn: conn}
	session.browserMu.Lock()
	session.browserChannels[channel] = proxy
	session.browserTokens[string(token)] = channel
	session.browserMu.Unlock()
	session.server.cfg.logger.Info(
		"browser proxy remote target connected",
		"session_id", session.sessionID,
		"channel", channel,
		"host", host,
		"port", port,
	)
	return BrowserProxy{Token: append([]byte(nil), token...)}, nil
}

// ApplicationTerminalDefaults 返回 owning daemon 机器的默认 shell 与 cwd。
func (session *protocolSession) ApplicationTerminalDefaults(context.Context) (TerminalDefaults, error) {
	return pathDefaults(), nil
}

// ApplicationTerminalCreate 把已由 API Mapping 构造的 core record 交给 lifecycle owner。
func (session *protocolSession) ApplicationTerminalCreate(_ context.Context, record TerminalRecord) (TerminalInfo, error) {
	return session.server.RegisterTerminal(record)
}

// ApplicationTerminalList 返回当前 connection scope 可见的 terminal inventory。
func (session *protocolSession) ApplicationTerminalList(context.Context) ([]TerminalInfo, error) {
	scope := session.scope.normalized()
	if scope.AllowDaemon {
		return session.server.ListTerminals(), nil
	}
	if scope.MachineEventsOnly || scope.TerminalID == "" {
		return nil, ErrApplicationForbidden
	}
	terminal, err := session.server.GetTerminal(scope.TerminalID)
	if errors.Is(err, ErrTerminalNotFound) {
		return []TerminalInfo{}, nil
	}
	if err != nil {
		return nil, err
	}
	return []TerminalInfo{terminal}, nil
}

// ApplicationTerminalGet 返回单个 daemon-local terminal lifecycle snapshot。
func (session *protocolSession) ApplicationTerminalGet(_ context.Context, terminalID string) (TerminalInfo, error) {
	return session.server.GetTerminal(terminalID)
}

// ApplicationTerminalAttachmentCount 返回当前 daemon registry 中 terminal 的活动 attachment 数量。
func (session *protocolSession) ApplicationTerminalAttachmentCount(terminalID string) int {
	return session.server.protocolAttachmentCount(terminalID)
}

// ApplicationTerminalRestart 请求 lifecycle owner 按保存的 process specification 重启 terminal。
func (session *protocolSession) ApplicationTerminalRestart(ctx context.Context, terminalID string) error {
	return session.server.RestartTerminal(ctx, terminalID)
}

// ApplicationTerminalKill 终止 terminal process，但保留 record 与 history truth。
func (session *protocolSession) ApplicationTerminalKill(ctx context.Context, terminalID string) error {
	return session.server.KillTerminal(ctx, terminalID)
}

// ApplicationTerminalRemove 删除满足 lifecycle 条件的 terminal record。
func (session *protocolSession) ApplicationTerminalRemove(_ context.Context, terminalID string) error {
	return session.server.RemoveTerminal(terminalID)
}

// ApplicationTerminalSetMetadata 原子更新 terminal name 与 tags。
func (session *protocolSession) ApplicationTerminalSetMetadata(ctx context.Context, terminalID, name string, tags map[string]string) error {
	_, err := session.server.SetMetadata(ctx, terminalID, name, tags)
	return err
}

// ApplicationTerminalSetTags 替换 terminal tags，同时保留 daemon-owned name。
func (session *protocolSession) ApplicationTerminalSetTags(ctx context.Context, terminalID string, tags map[string]string) error {
	info, err := session.server.GetTerminal(terminalID)
	if err != nil {
		return err
	}
	_, err = session.server.SetMetadata(ctx, terminalID, info.Name, tags)
	return err
}

type protocolAttachTransaction struct {
	session    *protocolSession
	attachment protocolAttachment
	result     TerminalAttachment
	committed  bool
}

// Result 返回 pending attachment 的 core-native 不可变投影。
func (transaction *protocolAttachTransaction) Result() TerminalAttachment {
	return transaction.result
}

// Commit 原子发布 pending attachment token。
func (transaction *protocolAttachTransaction) Commit(context.Context) error {
	if transaction == nil || transaction.committed {
		return nil
	}
	if err := transaction.session.publishAttachmentToken(transaction.attachment); err != nil {
		return err
	}
	transaction.committed = true
	return nil
}

// Rollback 释放尚未发布的 attachment，且可以重复调用。
func (transaction *protocolAttachTransaction) Rollback(context.Context) error {
	if transaction == nil || transaction.committed {
		return nil
	}
	transaction.session.detachExact(transaction.attachment)
	return nil
}

// ApplicationTerminalAttach 创建 pending attachment；token 只在 transaction commit 后可查。
func (session *protocolSession) ApplicationTerminalAttach(_ context.Context, request TerminalAttachmentRequest) (TerminalAttachmentTransaction, error) {
	attachment, control, err := session.attach(attachmentRequest{
		TerminalID: request.TerminalID, Mode: string(request.Mode), ResizePolicy: string(request.ResizePolicy),
		SurfaceID: request.SurfaceID, ViewID: request.ViewID,
	})
	if err != nil {
		return nil, err
	}
	info, err := session.server.GetTerminal(attachment.TerminalID)
	if err != nil {
		session.detach(attachmentDetachRequest{Channel: attachment.Channel})
		return nil, err
	}
	result := TerminalAttachment{
		Token: append([]byte(nil), attachment.Token...), TerminalID: attachment.TerminalID,
		Mode: TerminalAttachmentMode(attachment.Mode), ResizePolicy: TerminalResizePolicy(attachment.ResizePolicy), SurfaceID: attachment.SurfaceID,
		ViewID: attachment.ViewID, Size: info.Size, ResizeControl: applicationResizeControl(control),
	}
	return &protocolAttachTransaction{session: session, attachment: attachment, result: result}, nil
}

// ApplicationTerminalDetach 释放 opaque token 对应的 attachment。
func (session *protocolSession) ApplicationTerminalDetach(ctx context.Context, token []byte) error {
	return session.ReleaseApplicationResource(ctx, token)
}

// ApplicationTerminalInput 向 token owning attachment 写入 bytes，不做失败重放。
func (session *protocolSession) ApplicationTerminalInput(ctx context.Context, token, data []byte) error {
	attachment, err := session.attachmentForToken(token)
	if err != nil {
		return err
	}
	return session.input(ctx, attachmentInputRequest{
		TerminalID: attachment.TerminalID, Channel: attachment.Channel, SurfaceID: attachment.SurfaceID,
		ViewID: attachment.ViewID, Data: append([]byte(nil), data...),
	})
}

// ApplicationTerminalResize 协调 attachment resize owner 并返回 daemon 确认后的 size/control。
func (session *protocolSession) ApplicationTerminalResize(ctx context.Context, token []byte, requested Size, policy TerminalResizePolicy, takeOwnership bool, expectedOwnerEpoch uint64) (TerminalResizeResult, error) {
	attachment, err := session.attachmentForToken(token)
	if err != nil {
		return TerminalResizeResult{}, err
	}
	control, canResize, err := session.resizeControlForRequest(attachment, string(policy), attachment.SurfaceID, attachment.ViewID, takeOwnership, expectedOwnerEpoch)
	if err != nil {
		return TerminalResizeResult{}, err
	}
	resized := false
	if canResize && requested != control.ResizeOwnership.Size {
		resized, err = session.resizeTerminalForCurrentOwner(ctx, attachment, requested, control.ResizeOwnership.Epoch)
		if err != nil {
			return TerminalResizeResult{}, err
		}
	}
	info, err := session.server.GetTerminal(attachment.TerminalID)
	if err != nil {
		return TerminalResizeResult{}, err
	}
	control = session.currentResizeControl(attachment, info.Size)
	return TerminalResizeResult{Size: info.Size, Resized: resized, ResizeControl: applicationResizeControl(control)}, nil
}

func (session *protocolSession) resizeTerminalForCurrentOwner(ctx context.Context, attachment protocolAttachment, requested Size, ownerEpoch uint64) (bool, error) {
	session.server.protocolAttachmentMu.Lock()
	defer session.server.protocolAttachmentMu.Unlock()
	key := attachmentKey(attachment)
	current, ok := session.server.protocolAttachments[key]
	if !ok || session.server.protocolResizeOwners[attachment.TerminalID] != key || current.ResizePolicy != attachmentResizePolicyOwner || current.Epoch != ownerEpoch || session.server.protocolSizeLocks[attachment.TerminalID] {
		return false, nil
	}
	if err := session.server.ResizeTerminal(ctx, attachment.TerminalID, requested.Cols, requested.Rows); err != nil {
		return false, err
	}
	return true, nil
}

func (session *protocolSession) currentResizeControl(attachment protocolAttachment, size Size) *attachmentResizeControl {
	session.server.protocolAttachmentMu.Lock()
	defer session.server.protocolAttachmentMu.Unlock()
	if current, ok := session.server.protocolAttachments[attachmentKey(attachment)]; ok {
		attachment = current
	}
	return session.resizeControlForGlobalAttachmentLocked(attachment, size)
}

// ApplicationTerminalResizeLock 修改 attachment owner 的显式 size lock。
func (session *protocolSession) ApplicationTerminalResizeLock(_ context.Context, token []byte, locked bool) (TerminalResizeResult, error) {
	attachment, err := session.attachmentForToken(token)
	if err != nil {
		return TerminalResizeResult{}, err
	}
	control, err := session.setResizeLock(attachmentResizeControlRequest{
		TerminalID: attachment.TerminalID, Channel: attachment.Channel, ResizePolicy: attachment.ResizePolicy,
		SurfaceID: attachment.SurfaceID, ViewID: attachment.ViewID,
	}, locked)
	if err != nil {
		return TerminalResizeResult{}, err
	}
	info, err := session.server.GetTerminal(attachment.TerminalID)
	if err != nil {
		return TerminalResizeResult{}, err
	}
	return TerminalResizeResult{Size: info.Size, ResizeControl: applicationResizeControl(control)}, nil
}

// ApplicationPathListDirectories 查询 owning daemon 文件系统的目录候选。
func (session *protocolSession) ApplicationPathListDirectories(_ context.Context, prefix string, limit int) (PathDirectories, error) {
	return listPathDirectories(prefix, limit)
}

// ApplicationHistoryWindow 查询 authoritative history；latest 会先建立 frozen token，
// 后续分页只能沿 core 返回的 token/cursor 继续，不能从客户端 rows 重建边界。
func (session *protocolSession) ApplicationHistoryWindow(ctx context.Context, request history.HistoryWindowRequest) (history.HistoryWindow, error) {
	latest := request.Mode == "" || request.Mode == history.HistoryWindowModeLatest
	if _, err := session.server.GetTerminal(request.TerminalID); err != nil {
		if !latest && request.Token != "" {
			session.forgetHistoryToken(request.TerminalID, request.Token)
		}
		return history.HistoryWindow{}, err
	}
	if latest {
		if err := session.reserveHistoryToken(); err != nil {
			return history.HistoryWindow{}, err
		}
		snapshot, err := session.server.TerminalHistoryFreeze(ctx, request.TerminalID, history.FreezeHistoryRequest{
			TerminalID: request.TerminalID,
			Cols:       request.Cols,
			Limit:      request.Limit,
		})
		if err != nil {
			session.rollbackHistoryTokenReservation()
			return history.HistoryWindow{}, err
		}
		session.commitHistoryToken(request.TerminalID, snapshot.Token)
		request.Token = snapshot.Token
		if err := ctx.Err(); err != nil {
			session.releaseOwnedHistoryToken(request.TerminalID, request.Token)
			return history.HistoryWindow{}, err
		}
	} else if request.Token == "" {
		return history.HistoryWindow{}, history.ErrHistoryInvalidMutation
	} else if !session.ownsHistoryToken(request.TerminalID, request.Token) {
		return history.HistoryWindow{}, history.ErrHistoryStaleWindow
	}
	window, err := session.server.TerminalHistoryWindow(ctx, request.TerminalID, request)
	if err != nil {
		if latest || historyTokenInvalidated(err) {
			session.releaseOwnedHistoryToken(request.TerminalID, request.Token)
		}
		return history.HistoryWindow{}, err
	}
	if latest && ctx.Err() != nil {
		session.releaseOwnedHistoryToken(request.TerminalID, request.Token)
		return history.HistoryWindow{}, ctx.Err()
	}
	return window, nil
}

// ApplicationHistoryCopy 从 core frozen history token 复制文本。
func (session *protocolSession) ApplicationHistoryCopy(ctx context.Context, request history.HistoryCopyRequest) (string, error) {
	if _, err := session.server.GetTerminal(request.TerminalID); err != nil {
		if request.Token != "" {
			session.forgetHistoryToken(request.TerminalID, request.Token)
		}
		return "", err
	}
	if request.Token == "" {
		return "", history.ErrHistoryInvalidMutation
	}
	if !session.ownsHistoryToken(request.TerminalID, request.Token) {
		return "", history.ErrHistoryStaleWindow
	}
	text, err := session.server.TerminalHistoryCopy(ctx, request.TerminalID, request)
	if historyTokenInvalidated(err) {
		session.releaseOwnedHistoryToken(request.TerminalID, request.Token)
	}
	return text, err
}

func (session *protocolSession) ApplicationHistoryCopyChunk(ctx context.Context, request history.HistoryCopyChunkRequest) (history.HistoryCopyChunkResult, error) {
	if _, err := session.server.GetTerminal(request.TerminalID); err != nil {
		if request.Token != "" {
			session.forgetHistoryToken(request.TerminalID, request.Token)
		}
		return history.HistoryCopyChunkResult{}, err
	}
	if request.Token == "" {
		return history.HistoryCopyChunkResult{}, history.ErrHistoryInvalidMutation
	}
	if !session.ownsHistoryToken(request.TerminalID, request.Token) {
		return history.HistoryCopyChunkResult{}, history.ErrHistoryStaleWindow
	}
	result, err := session.server.TerminalHistoryCopyChunk(ctx, request.TerminalID, request)
	if historyTokenInvalidated(err) {
		session.releaseOwnedHistoryToken(request.TerminalID, request.Token)
	}
	return result, err
}

func (session *protocolSession) ApplicationHistorySearch(ctx context.Context, request history.HistorySearchRequest) (history.HistorySearchResult, error) {
	if _, err := session.server.GetTerminal(request.TerminalID); err != nil {
		if request.Token != "" {
			session.forgetHistoryToken(request.TerminalID, request.Token)
		}
		return history.HistorySearchResult{}, err
	}
	if request.Token == "" {
		return history.HistorySearchResult{}, history.ErrHistoryInvalidMutation
	}
	if !session.ownsHistoryToken(request.TerminalID, request.Token) {
		return history.HistorySearchResult{}, history.ErrHistoryStaleWindow
	}
	result, err := session.server.TerminalHistorySearch(ctx, request.TerminalID, request)
	if historyTokenInvalidated(err) {
		session.releaseOwnedHistoryToken(request.TerminalID, request.Token)
	}
	return result, err
}

// ApplicationHistoryRelease 释放 core-owned frozen history token。
func (session *protocolSession) ApplicationHistoryRelease(ctx context.Context, terminalID string, token history.HistoryToken) error {
	if token == "" {
		return history.ErrHistoryInvalidMutation
	}
	if !session.forgetHistoryToken(terminalID, token) {
		return history.ErrHistoryStaleWindow
	}
	if _, err := session.server.GetTerminal(terminalID); err != nil {
		return nil
	}
	return session.server.TerminalHistoryRelease(ctx, terminalID, token)
}

func historyTokenInvalidated(err error) bool {
	return errors.Is(err, history.ErrHistoryStaleWindow)
}

// ApplicationHistoryBacklogStatus 返回 history output consumer 的诊断投影。
func (session *protocolSession) ApplicationHistoryBacklogStatus(_ context.Context, terminalID string) (HistoryBacklogStatus, error) {
	return session.server.TerminalHistoryBacklogStatus(terminalID)
}

// ApplicationLiveScreenNext 返回或等待 latest-only native screen，不把 live surface 当作 history truth。
func (session *protocolSession) ApplicationLiveScreenNext(ctx context.Context, terminalID string, observed LiveRevision) (NativeScreenSnapshot, error) {
	if _, err := session.server.GetTerminal(terminalID); err != nil {
		return NativeScreenSnapshot{}, err
	}
	terminal, err := session.server.Terminal(terminalID)
	if err != nil {
		return NativeScreenSnapshot{}, err
	}
	terminal.queueMu.Lock()
	liveErr := terminal.liveOutputError
	if liveErr == nil && terminal.outputBuffer != nil {
		liveErr = terminal.outputBuffer.ConsumerError(terminalOutputConsumerLive)
	}
	terminal.queueMu.Unlock()
	if liveErr != nil {
		return NativeScreenSnapshot{}, liveErr
	}
	base, releaseBase := session.acquireLiveScreenBaseline(terminalID, observed)
	defer releaseBase()
	snapshot, currentBase, err := session.server.nextLiveScreenWithBaseline(ctx, terminalID, observed, base)
	if err != nil {
		return NativeScreenSnapshot{}, err
	}
	session.offerLiveScreenBaseline(terminalID, currentBase)
	return snapshot, nil
}

// ApplicationEventSubscribe 建立当前 protocol session owning 的异步事件订阅。
// encoder 来自 API Mapping；编码失败只丢弃该通知，不改变 core event truth。
func (session *protocolSession) ApplicationEventSubscribe(ctx context.Context, filter EventFilter, encoder ApplicationEventEncoder) ([]byte, error) {
	if encoder == nil {
		return nil, ErrApplicationForbidden
	}
	if err := session.reserveEventSubscription(); err != nil {
		return nil, err
	}
	eventCtx, cancel := context.WithCancel(session.lifetimeContext(ctx))
	session.mu.Lock()
	session.nextEventSub++
	subscriptionID := session.nextEventSub
	session.eventSubscriptions[subscriptionID] = applicationEventSubscription{cancel: cancel, filter: filter}
	session.mu.Unlock()
	token := applicationEventSubscriptionToken(subscriptionID)
	events := session.server.Events(eventCtx, filter)
	initial := session.protocolAttachmentSnapshotEvents(filter)
	go func() {
		defer session.clearEventSubscription(subscriptionID)
		send := func(event Event) bool {
			payload, err := encoder(event, token)
			if err != nil {
				return true
			}
			return session.sendFrame(0, wire.TypeEvent, payload) == nil
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
				if !send(event) {
					return
				}
			}
		}
	}()
	return token, nil
}

func applicationEventSubscriptionToken(subscriptionID uint64) []byte {
	token := make([]byte, 10)
	copy(token, "ev")
	binary.BigEndian.PutUint64(token[2:], subscriptionID)
	return token
}

func applicationEventSubscriptionID(token []byte) (uint64, bool) {
	if len(token) != 10 || string(token[:2]) != "ev" {
		return 0, false
	}
	return binary.BigEndian.Uint64(token[2:]), true
}

func (session *protocolSession) eventSubscriptionForToken(token []byte) (applicationEventSubscription, bool) {
	id, ok := applicationEventSubscriptionID(token)
	if !ok {
		return applicationEventSubscription{}, false
	}
	session.mu.RLock()
	defer session.mu.RUnlock()
	subscription, ok := session.eventSubscriptions[id]
	return subscription, ok
}

func machineLifecycleEventFilter(filter EventFilter) bool {
	if filter.TerminalID != "" || filter.StorageAppID != "" || filter.StorageScope != "" || filter.StorageOwnerID != "" || filter.StorageKeyPrefix != "" || len(filter.Types) == 0 {
		return false
	}
	for _, eventType := range filter.Types {
		switch eventType {
		case EventTerminalCreated, EventTerminalExited, EventTerminalMetadataChanged, EventTerminalRemoved, EventTerminalChanged:
		default:
			return false
		}
	}
	return true
}

// ApplicationFileList 返回 daemon-owned directory window。
func (session *protocolSession) ApplicationFileList(_ context.Context, request FileListRequest) (FileListResult, error) {
	return fileList(request)
}

// ApplicationFileStat 返回 daemon-owned path metadata。
func (session *protocolSession) ApplicationFileStat(_ context.Context, request FilePathRequest) (FileEntry, error) {
	return fileStat(request)
}

// ApplicationFilePreview 返回有界文件内容预览。
func (session *protocolSession) ApplicationFilePreview(_ context.Context, request FilePreviewRequest) (FilePreviewResult, error) {
	return filePreview(request)
}

// ApplicationFileMkdir 创建 daemon-owned directory。
func (session *protocolSession) ApplicationFileMkdir(_ context.Context, request FilePathRequest) FileOperationResult {
	return fileMkdir(request)
}

// ApplicationFileRename 原子重命名 daemon-owned path。
func (session *protocolSession) ApplicationFileRename(_ context.Context, request FileRenameRequest) FileOperationResult {
	return fileRename(request)
}

// ApplicationFileDelete 删除 daemon-owned path。
func (session *protocolSession) ApplicationFileDelete(_ context.Context, request FilePathRequest) FileOperationResult {
	return fileDelete(request)
}

// ApplicationFileCopy 批量复制 daemon-owned paths。
func (session *protocolSession) ApplicationFileCopy(_ context.Context, request FileCopyMoveRequest) FileBatchResult {
	return fileCopyMove(request, false)
}

// ApplicationFileMove 批量移动 daemon-owned paths。
func (session *protocolSession) ApplicationFileMove(_ context.Context, request FileCopyMoveRequest) FileBatchResult {
	return fileCopyMove(request, true)
}

// ApplicationFileDownloadOpen 创建 session-bound download transfer。
func (session *protocolSession) ApplicationFileDownloadOpen(ctx context.Context, request FileDownloadOpenRequest) (FileTransfer, error) {
	return session.openFileDownload(session.lifetimeContext(ctx), request)
}

// ApplicationFileUploadOpen 创建或恢复 session-bound upload transfer。
func (session *protocolSession) ApplicationFileUploadOpen(_ context.Context, request FileUploadOpenRequest) (FileTransfer, error) {
	return session.openFileUpload(request)
}

// ApplicationFileTransferCancel 按 current-session resource 或 principal-bound upload resume 凭据取消 transfer。
func (session *protocolSession) ApplicationFileTransferCancel(_ context.Context, request FileTransferCancelRequest) (FileTransferCancelResult, error) {
	if len(request.UploadResumeToken) > 0 {
		transferID, ok := fileTransferIDFromResumeToken(request.UploadResumeToken)
		if !ok {
			return FileTransferCancelResult{}, ErrTerminalNotFound
		}
		return FileTransferCancelResult{Cancelled: session.cancelOwnedUpload(transferID)}, nil
	}
	transferID, ok := fileTransferIDFromResourceToken(request.ResourceToken)
	if !ok {
		return FileTransferCancelResult{}, ErrTerminalNotFound
	}
	return FileTransferCancelResult{Cancelled: session.cancelCurrentFileTransfer(transferID)}, nil
}

// ApplicationStorageGet 返回 daemon opaque storage entry。
func (session *protocolSession) ApplicationStorageGet(ctx context.Context, appID string, scope StorageScope, ownerID, key string) (StorageEntry, error) {
	return session.server.StorageGet(ctx, appID, scope, ownerID, key)
}

// ApplicationStoragePut 执行 opaque value CAS put。
func (session *protocolSession) ApplicationStoragePut(ctx context.Context, request StoragePutRequest) (StorageEntry, error) {
	return session.server.StoragePut(ctx, request)
}

// ApplicationStorageDelete 执行 opaque value CAS delete。
func (session *protocolSession) ApplicationStorageDelete(ctx context.Context, request StorageDeleteRequest) (StorageDeleteResult, error) {
	return session.server.StorageDelete(ctx, request)
}

// ApplicationStorageList 返回稳定 storage key window。
func (session *protocolSession) ApplicationStorageList(ctx context.Context, appID string, scope StorageScope, ownerID, prefix string) []StorageEntry {
	return session.server.StorageList(ctx, appID, scope, ownerID, prefix)
}

func (session *protocolSession) ApplicationClientAccessIdentity(ctx context.Context, challenge []byte) (ClientAccessIdentity, error) {
	service, err := session.clientAccessService()
	if err != nil {
		return ClientAccessIdentity{}, err
	}
	return service.Identity(ctx, append([]byte(nil), challenge...))
}

func (session *protocolSession) ApplicationClientAccessList(ctx context.Context) ([]ClientAccessRecord, error) {
	service, err := session.clientAccessService()
	if err != nil {
		return nil, err
	}
	return service.List(ctx)
}

func (session *protocolSession) ApplicationClientAccessCreateTicket(ctx context.Context, request ClientAccessTicketRequest) (ClientAccessTicket, error) {
	service, err := session.clientAccessService()
	if err != nil {
		return ClientAccessTicket{}, err
	}
	return service.CreateTicket(ctx, request)
}

func (session *protocolSession) ApplicationClientAccessRevoke(ctx context.Context, grantID string) (ClientAccessRecord, error) {
	return session.server.revokeClientAccess(ctx, grantID)
}

func (session *protocolSession) ApplicationRemoteStatus(ctx context.Context) (RemoteStatus, error) {
	service, err := session.remoteService()
	if err != nil {
		return RemoteStatus{}, err
	}
	return service.Status(ctx)
}

func (session *protocolSession) ApplicationRemotePairStart(ctx context.Context, request RemotePairStartRequest) (RemotePairStartResult, error) {
	service, err := session.remoteService()
	if err != nil {
		return RemotePairStartResult{}, err
	}
	return service.PairStart(ctx, request)
}

func (session *protocolSession) ApplicationRemoteLocalEnable(ctx context.Context, request RemoteLocalEnableRequest) (RemoteLocalStatus, error) {
	service, err := session.remoteService()
	if err != nil {
		return RemoteLocalStatus{}, err
	}
	return service.LocalEnable(ctx, request)
}

func (session *protocolSession) ApplicationRemoteLocalStatus(ctx context.Context) (RemoteLocalStatus, error) {
	service, err := session.remoteService()
	if err != nil {
		return RemoteLocalStatus{}, err
	}
	return service.LocalStatus(ctx)
}

func (session *protocolSession) ApplicationRemoteLocalDisable(ctx context.Context) (RemoteLocalStatus, error) {
	service, err := session.remoteService()
	if err != nil {
		return RemoteLocalStatus{}, err
	}
	return service.LocalDisable(ctx)
}

func (session *protocolSession) ApplicationRemoteCloudStatus(ctx context.Context) (RemoteCloudStatus, error) {
	service, err := session.remoteService()
	if err != nil {
		return RemoteCloudStatus{}, err
	}
	return service.CloudStatus(ctx)
}

func (session *protocolSession) ApplicationRemoteCloudEnable(ctx context.Context) (RemoteCloudStatus, error) {
	service, err := session.remoteService()
	if err != nil {
		return RemoteCloudStatus{}, err
	}
	return service.CloudEnable(ctx)
}

func (session *protocolSession) ApplicationRemoteCloudDisable(ctx context.Context) (RemoteCloudStatus, error) {
	service, err := session.remoteService()
	if err != nil {
		return RemoteCloudStatus{}, err
	}
	return service.CloudDisable(ctx)
}

func (session *protocolSession) ApplicationRemoteCloudEdges(ctx context.Context) (RemoteCloudEdgeSelection, error) {
	service, err := session.remoteService()
	if err != nil {
		return RemoteCloudEdgeSelection{}, err
	}
	return service.CloudEdges(ctx)
}

func (session *protocolSession) ApplicationRemoteCloudPreferEdge(ctx context.Context, edgeID string, expectedRevision uint64) (RemoteCloudEdgeSelection, error) {
	service, err := session.remoteService()
	if err != nil {
		return RemoteCloudEdgeSelection{}, err
	}
	return service.CloudPreferEdge(ctx, edgeID, expectedRevision)
}

func (session *protocolSession) ApplicationRemoteCloudReselectEdge(ctx context.Context) (RemoteCloudEdgeSelection, error) {
	service, err := session.remoteService()
	if err != nil {
		return RemoteCloudEdgeSelection{}, err
	}
	return service.CloudReselectEdge(ctx)
}

func applicationResizeControl(control *attachmentResizeControl) *TerminalResizeControl {
	if control == nil {
		return nil
	}
	out := &TerminalResizeControl{
		CanResize: control.CanResize, Reason: TerminalResizeReason(control.Reason), SizeLocked: control.SizeLocked,
		SurfaceID: control.SurfaceID, OwnerSurfaceID: control.OwnerSurfaceID, OwnerViewID: control.OwnerViewID,
	}
	if ownership := control.ResizeOwnership; ownership != nil {
		out.ResizeOwnership = &TerminalResizeOwnership{
			OwnerAttachmentID: ownership.OwnerAttachmentID, OwnerSurfaceID: ownership.OwnerSurfaceID,
			OwnerViewID: ownership.OwnerViewID, Size: ownership.Size, SizeLocked: ownership.SizeLocked, Epoch: ownership.Epoch,
		}
	}
	return out
}
