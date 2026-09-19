package server_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	daemonprovider "github.com/anytty/anytty/access/provider/daemon"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	accessserver "github.com/anytty/anytty/access/server"
	corev2 "github.com/anytty/anytty/daemon/core"
	providercore "github.com/anytty/anytty/daemon/provider"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

// TestStorageCommandsTerminateInAccess 证明 storage.* 不再依赖 terminal provider：
// provider 工厂直接失败，storage 命令仍然成功。
func TestStorageCommandsTerminateInAccess(t *testing.T) {
	t.Parallel()
	accessSocket := filepath.Join(t.TempDir(), "access.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	access, err := accessserver.New(accessserver.Config{
		Socket: accessSocket,
		Files:  accessserverTestFiles(t),
		Provider: func(context.Context) (terminalprovider.Provider, error) {
			return nil, errors.New("provider must not be dialed for storage commands")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = access.Close() }()
	go func() { _ = access.Serve(ctx) }()
	waitForSocket(t, accessSocket)

	application, closeClient := dialApplication(t, ctx, accessSocket)
	defer closeClient()
	key := &apipb.StorageKey{AppId: "app-e2e", Scope: apipb.StorageScope_STORAGE_SCOPE_PRIVATE, OwnerId: "owner", Key: "theme"}
	put, err := application.StoragePut(ctx, &apipb.StoragePutCommand{Key: key, Value: []byte("dark")})
	if err != nil {
		t.Fatalf("storage put: %v", err)
	}
	if put.GetEntry().GetVersion() != 1 || string(put.GetEntry().GetValue()) != "dark" {
		t.Fatalf("put result = %#v", put.GetEntry())
	}
	get, err := application.StorageGet(ctx, &apipb.StorageGetCommand{Key: key})
	if err != nil {
		t.Fatalf("storage get: %v", err)
	}
	if string(get.GetEntry().GetValue()) != "dark" {
		t.Fatalf("get result = %#v", get.GetEntry())
	}
	list, err := application.StorageList(ctx, &apipb.StorageListCommand{AppId: "app-e2e", Scope: apipb.StorageScope_STORAGE_SCOPE_PRIVATE, OwnerId: "owner"})
	if err != nil {
		t.Fatalf("storage list: %v", err)
	}
	if len(list.GetEntries()) != 1 {
		t.Fatalf("list result = %#v", list.GetEntries())
	}
	if _, err := application.StorageDelete(ctx, &apipb.StorageDeleteCommand{Key: key}); err != nil {
		t.Fatalf("storage delete: %v", err)
	}
	if _, err := application.StorageGet(ctx, &apipb.StorageGetCommand{Key: key}); err == nil {
		t.Fatal("storage get after delete unexpectedly succeeded")
	}
}

// TestStorageEventsFanOutThroughAccessSubscription 证明 access-local storage 变更
// 通过 daemon 返回的既有订阅 token 广播，客户端不需要新 wire。
func TestStorageEventsFanOutThroughAccessSubscription(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	daemonSocket := filepath.Join(dir, "anyttyd.sock")
	accessSocket := filepath.Join(dir, "anyttyd.sock.access")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := corev2.NewServer(
		corev2.WithSocketPath(daemonSocket),
		corev2.WithHistoryDisabled(),
	)
	defer func() { _ = server.Shutdown(context.Background()) }()
	providerServer, err := providercore.New(server, providercore.Config{Socket: daemonSocket})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = providerServer.Shutdown(context.Background()) }()
	go func() { _ = providerServer.ListenAndServe(ctx) }()
	waitForSocket(t, daemonSocket)

	access, err := accessserver.New(accessserver.Config{
		Socket: accessSocket,
		Files:  accessserverTestFiles(t),
		Provider: func(dialCtx context.Context) (terminalprovider.Provider, error) {
			return daemonprovider.DialTerminal(dialCtx, daemonSocket)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = access.Close() }()
	go func() { _ = access.Serve(ctx) }()
	waitForSocket(t, accessSocket)

	application, closeClient := dialApplication(t, ctx, accessSocket)
	defer closeClient()
	subscription, events, err := application.EventSubscribe(ctx, &apipb.EventSubscribeCommand{
		Types: []apipb.ApplicationEventType{apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_STORAGE_CHANGED},
	})
	if err != nil {
		t.Fatalf("event subscribe: %v", err)
	}
	if _, err := application.StoragePut(ctx, &apipb.StoragePutCommand{
		Key:   &apipb.StorageKey{AppId: "app-events", Scope: apipb.StorageScope_STORAGE_SCOPE_PUBLIC, Key: "k"},
		Value: []byte("v"),
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.GetStorageChanged().GetKey().GetAppId() != "app-events" ||
			event.GetStorageChanged().GetOperation() != "put" ||
			string(event.GetSubscription().GetOpaqueToken()) != string(subscription.GetSubscription().GetOpaqueToken()) {
			t.Fatalf("storage event = %#v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for storage event through access")
	}
	if err := application.ReleaseResource(ctx, &apipb.ReleaseResourceCommand{Resource: subscription.GetSubscription()}); err != nil {
		t.Fatalf("release subscription: %v", err)
	}
	if _, err := application.StoragePut(ctx, &apipb.StoragePutCommand{
		Key:   &apipb.StorageKey{AppId: "app-events", Scope: apipb.StorageScope_STORAGE_SCOPE_PUBLIC, Key: "k2"},
		Value: []byte("v"),
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		t.Fatalf("released subscription received event %#v", event)
	case <-time.After(200 * time.Millisecond):
	}
}

func dialApplication(t *testing.T, ctx context.Context, socket string) (*clientruntime.ApplicationSession, func()) {
	t.Helper()
	connection, err := unixtransport.DialContext(ctx, socket)
	if err != nil {
		t.Fatal(err)
	}
	client := internalprotocol.NewClient(connection)
	if err := client.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: "storage-e2e"}); err != nil {
		_ = client.Close()
		t.Fatal(err)
	}
	application, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{
		EndpointID: clientendpoint.EndpointID("local"),
		RouteID:    clientendpoint.RouteID("storage-e2e"),
		Generation: 1,
	}, client)
	if err != nil {
		_ = client.Close()
		t.Fatal(err)
	}
	return application, func() { _ = client.Close() }
}
