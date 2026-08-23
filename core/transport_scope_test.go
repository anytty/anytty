package core

import (
	"context"
	"strings"
	"testing"
	"time"

	clientendpoint "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestServeScopedTransportRestrictsGeneratedTerminalCommands(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour)
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()), WithClientAccessService(newGrantAccessTestService(map[string]time.Time{"grant-scope": expiresAt})))
	server.cfg.applicationFactory = applicationTestExecutorFactory
	registerScopedTestTerminal(t, server, "term-1")
	registerScopedTestTerminal(t, server, "term-2")
	application, closeClient := newScopedApplicationClient(t, server, TransportScope{GrantID: "grant-scope", GrantExpiresAt: expiresAt, PrincipalID: "subject", TerminalID: "term-1"})
	defer closeClient()

	if _, err := application.TerminalGet(context.Background(), &apipb.TerminalGetCommand{Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-1"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := application.TerminalGet(context.Background(), &apipb.TerminalGetCommand{Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-2"}}); err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("cross-terminal get error = %v", err)
	}
	listed, err := application.TerminalList(context.Background(), &apipb.TerminalListCommand{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.GetTerminals()) != 1 || listed.GetTerminals()[0].GetRef().GetTerminalId() != "term-1" {
		t.Fatalf("scoped list = %#v", listed.GetTerminals())
	}
}

func TestServeScopedTransportRejectsRemoteScopeWithoutGrantIdentity(t *testing.T) {
	server := NewServer()
	_, serverTransport := memory.NewPair()
	defer serverTransport.Close()
	if err := server.ServeScopedTransport(context.Background(), serverTransport, TransportScope{}); err == nil || !strings.Contains(err.Error(), "grant identity") {
		t.Fatalf("zero scope error = %v", err)
	}
}

func newScopedApplicationClient(t *testing.T, server *Server, scope TransportScope) (*clientruntime.ApplicationSession, func()) {
	t.Helper()
	clientTransport, serverTransport := memory.NewPair()
	errCh := make(chan error, 1)
	go func() { errCh <- server.ServeScopedTransport(context.Background(), serverTransport, scope) }()
	client := protocol.NewClient(clientTransport)
	if err := client.Hello(context.Background(), protocol.Hello{Version: wire.Version, Client: "scope-test"}); err != nil {
		t.Fatal(err)
	}
	application, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{
		EndpointID: clientendpoint.EndpointID("local"), RouteID: clientendpoint.RouteID("memory"), Generation: 1,
	}, client)
	if err != nil {
		t.Fatal(err)
	}
	return application, func() {
		_ = client.Close()
		select {
		case err := <-errCh:
			if err != nil && !strings.Contains(err.Error(), "EOF") {
				t.Fatalf("scoped server error: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("scoped server did not stop")
		}
	}
}

func registerScopedTestTerminal(t *testing.T, server *Server, id string) {
	t.Helper()
	if _, err := server.RegisterTerminal(TerminalRecord{ID: id, Name: id, Command: []string{"shell"}, Size: Size{Cols: 12, Rows: 4}}); err != nil {
		t.Fatal(err)
	}
}
