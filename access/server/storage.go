package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anytty/anytty/access/storage"
	apimapping "github.com/anytty/anytty/api_mapping"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	"google.golang.org/protobuf/proto"
)

// executeStorage 在 access 本地终结 storage.* command（Phase 2 迁移起点）。
// 真值是 server 级 store；事件通过 event.subscribe 的既有订阅 token 广播，
// 客户端 wire 不变。
func (session *session) executeStorage(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	_ = ctx
	if err := apimapping.ValidateFileStorageCommand(command); err != nil {
		return nil, &routeError{apiError: apimapping.ErrorToProto(err, false)}
	}
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_StorageGet:
		key := value.StorageGet.GetKey()
		entry, err := session.server.storage.Get(key.GetAppId(), storageScopeFromProto(key.GetScope()), key.GetOwnerId(), key.GetKey())
		if err != nil {
			return nil, storageRouteError(err)
		}
		envelope := newResultEnvelope(command)
		envelope.Result = &apipb.ResultEnvelope_StorageGet{StorageGet: &apipb.StorageGetResult{
			Entry: storageEntryToProto(entry),
		}}
		return envelope, nil
	case *apipb.CommandEnvelope_StoragePut:
		key := value.StoragePut.GetKey()
		fence := value.StoragePut.GetVersion()
		entry, err := session.server.storage.Put(storage.PutRequest{
			AppID:           key.GetAppId(),
			Scope:           storageScopeFromProto(key.GetScope()),
			OwnerID:         key.GetOwnerId(),
			Key:             key.GetKey(),
			Value:           append([]byte(nil), value.StoragePut.GetValue()...),
			CheckVersion:    fence.GetCheckVersion(),
			ExpectedVersion: fence.GetExpectedVersion(),
		})
		if err != nil {
			return nil, storageRouteError(err)
		}
		envelope := newResultEnvelope(command)
		envelope.Result = &apipb.ResultEnvelope_StoragePut{StoragePut: &apipb.StoragePutResult{
			Entry: storageEntryToProto(entry),
		}}
		return envelope, nil
	case *apipb.CommandEnvelope_StorageDelete:
		key := value.StorageDelete.GetKey()
		fence := value.StorageDelete.GetVersion()
		result, err := session.server.storage.Delete(storage.DeleteRequest{
			AppID:           key.GetAppId(),
			Scope:           storageScopeFromProto(key.GetScope()),
			OwnerID:         key.GetOwnerId(),
			Key:             key.GetKey(),
			CheckVersion:    fence.GetCheckVersion(),
			ExpectedVersion: fence.GetExpectedVersion(),
		})
		if err != nil {
			return nil, storageRouteError(err)
		}
		envelope := newResultEnvelope(command)
		envelope.Result = &apipb.ResultEnvelope_StorageDelete{StorageDelete: &apipb.StorageDeleteResult{
			Key:     proto.Clone(key).(*apipb.StorageKey),
			Deleted: result.Deleted,
			Version: result.Version,
		}}
		return envelope, nil
	case *apipb.CommandEnvelope_StorageList:
		entries := session.server.storage.List(
			value.StorageList.GetAppId(),
			storageScopeFromProto(value.StorageList.GetScope()),
			value.StorageList.GetOwnerId(),
			value.StorageList.GetPrefix(),
		)
		listResult := &apipb.StorageListResult{}
		for _, entry := range entries {
			listResult.Entries = append(listResult.Entries, storageEntryToProto(entry))
		}
		envelope := newResultEnvelope(command)
		envelope.Result = &apipb.ResultEnvelope_StorageList{StorageList: listResult}
		return envelope, nil
	default:
		return nil, errUnsupportedCommand
	}
}

// registerLocalRoutes 在 provider 成功执行 command 后接管 access-local 资源：
// event subscription 需要同时观察 access-local storage 变更；release 需要退订。
func (session *session) registerLocalRoutes(command *apipb.CommandEnvelope, result *apipb.ResultEnvelope) {
	if command == nil || result == nil {
		return
	}
	if command.GetEventSubscribe() != nil {
		session.registerStorageSubscription(command, result)
	}
	if release := command.GetReleaseResource(); release != nil {
		session.releaseLocalSubscription(release.GetResource())
	}
}

func (session *session) registerStorageSubscription(command *apipb.CommandEnvelope, result *apipb.ResultEnvelope) {
	subscription := result.GetEventSubscription().GetSubscription()
	if subscription == nil || len(subscription.GetOpaqueToken()) == 0 {
		return
	}
	subscribe := command.GetEventSubscribe()
	if !storageSubscriptionRequested(subscribe) {
		return
	}
	ctx, cancel := context.WithCancel(session.lifetimeContext())
	changes := session.server.storage.Subscribe(ctx)
	entry := &localSubscription{handle: cloneResourceHandle(subscription), cancel: cancel}
	session.localMu.Lock()
	if session.localSubscriptions == nil {
		session.localSubscriptions = make(map[string]*localSubscription)
	}
	if existing := session.localSubscriptions[string(subscription.GetOpaqueToken())]; existing != nil {
		existing.cancel()
	}
	session.localSubscriptions[string(subscription.GetOpaqueToken())] = entry
	session.localMu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case change := <-changes:
				if !storageChangeMatchesFilter(change, subscribe) {
					continue
				}
				envelope := storageEventEnvelope(entry.handle, change)
				payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(envelope)
				if err != nil {
					continue
				}
				if err := session.sendFrame(0, wire.TypeEvent, payload); err != nil {
					return
				}
			}
		}
	}()
}

func (session *session) releaseLocalSubscription(resource *apipb.ResourceHandle) {
	if resource == nil || len(resource.GetOpaqueToken()) == 0 {
		return
	}
	session.localMu.Lock()
	entry := session.localSubscriptions[string(resource.GetOpaqueToken())]
	delete(session.localSubscriptions, string(resource.GetOpaqueToken()))
	session.localMu.Unlock()
	if entry != nil {
		entry.cancel()
	}
}

func (session *session) releaseAllLocalSubscriptions() {
	session.localMu.Lock()
	entries := make([]*localSubscription, 0, len(session.localSubscriptions))
	for _, entry := range session.localSubscriptions {
		entries = append(entries, entry)
	}
	session.localSubscriptions = nil
	session.localMu.Unlock()
	for _, entry := range entries {
		entry.cancel()
	}
}

type localSubscription struct {
	handle *apipb.ResourceHandle
	cancel context.CancelFunc
}

// storageSubscriptionRequested 报告订阅是否覆盖 storage 变更。
// Types 为空表示订阅全部事件类型（与 daemon EventFilter 语义一致）。
func storageSubscriptionRequested(command *apipb.EventSubscribeCommand) bool {
	if command == nil {
		return false
	}
	if len(command.GetTypes()) == 0 {
		return true
	}
	for _, typ := range command.GetTypes() {
		if typ == apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_STORAGE_CHANGED {
			return true
		}
	}
	return false
}

func storageChangeMatchesFilter(change storage.Change, command *apipb.EventSubscribeCommand) bool {
	if appID := command.GetStorageAppId(); appID != "" && appID != change.AppID {
		return false
	}
	if scope := command.GetStorageScope(); scope != apipb.StorageScope_STORAGE_SCOPE_UNSPECIFIED {
		if storageScopeFromProto(scope) != change.Scope {
			return false
		}
	}
	if ownerID := command.GetStorageOwnerId(); ownerID != "" && ownerID != change.OwnerID {
		return false
	}
	if prefix := command.GetStorageKeyPrefix(); prefix != "" && !strings.HasPrefix(change.Key, prefix) {
		return false
	}
	return true
}

func storageEventEnvelope(handle *apipb.ResourceHandle, change storage.Change) *apipb.EventEnvelope {
	return &apipb.EventEnvelope{
		EventId:           fmt.Sprintf("storage.changed-%s-%d", change.Key, change.Version),
		TimestampUnixNano: time.Now().UTC().UnixNano(),
		ApiVersion:        &apipb.ApiVersion{Major: 1},
		OriginSession:     cloneSessionStamp(handle.GetSession()),
		Subscription:      cloneResourceHandle(handle),
		Event: &apipb.EventEnvelope_StorageChanged{StorageChanged: &apipb.StorageChangedEvent{
			Key: &apipb.StorageKey{
				AppId:   change.AppID,
				Scope:   storageScopeToProto(change.Scope),
				OwnerId: change.OwnerID,
				Key:     change.Key,
			},
			Version:   change.Version,
			Operation: change.Op,
		}},
	}
}

// newResultEnvelope 构造带 client correlation 的空结果信封；
// 调用方填充 Result oneof。access 不重写客户端 request/session stamp。
func newResultEnvelope(command *apipb.CommandEnvelope) *apipb.ResultEnvelope {
	requestContext := command.GetContext()
	return &apipb.ResultEnvelope{
		RequestId:     requestContext.GetRequestId(),
		OriginSession: cloneSessionStamp(requestContext.GetSession()),
	}
}

func storageEntryToProto(entry storage.Entry) *apipb.StorageEntry {
	return &apipb.StorageEntry{
		Key: &apipb.StorageKey{
			AppId:   entry.AppID,
			Scope:   storageScopeToProto(entry.Scope),
			OwnerId: entry.OwnerID,
			Key:     entry.Key,
		},
		Value:             append([]byte(nil), entry.Value...),
		Version:           entry.Version,
		UpdatedAtUnixNano: entry.UpdatedAt.UnixNano(),
	}
}

func storageScopeFromProto(scope apipb.StorageScope) storage.Scope {
	if scope == apipb.StorageScope_STORAGE_SCOPE_PRIVATE {
		return storage.ScopePrivate
	}
	return storage.ScopePublic
}

func storageScopeToProto(scope storage.Scope) apipb.StorageScope {
	if scope == storage.ScopePrivate {
		return apipb.StorageScope_STORAGE_SCOPE_PRIVATE
	}
	return apipb.StorageScope_STORAGE_SCOPE_PUBLIC
}

func storageRouteError(err error) error {
	apiError := &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE, Message: err.Error(), Retryable: true}
	switch {
	case errors.Is(err, storage.ErrEntryNotFound):
		apiError = &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND, Message: err.Error()}
	case errors.Is(err, storage.ErrInvalidKey):
		apiError = &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, Message: err.Error()}
	case errors.Is(err, storage.ErrVersionConflict):
		apiError = &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_CONFLICT, Message: err.Error()}
	}
	return &routeError{apiError: apiError}
}

func cloneResourceHandle(resource *apipb.ResourceHandle) *apipb.ResourceHandle {
	if resource == nil {
		return nil
	}
	return proto.Clone(resource).(*apipb.ResourceHandle)
}
