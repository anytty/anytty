package localweb

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	accesscontract "github.com/anytty/anytty/access/contract"
	daemonprovider "github.com/anytty/anytty/access/provider/daemon"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	accessserver "github.com/anytty/anytty/access/server"
	corev2 "github.com/anytty/anytty/daemon/core"
	providercore "github.com/anytty/anytty/daemon/provider"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/bindingpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/proto"
)

func TestServerOpensAuthenticatedBindingSession(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := remoteauth.NewIdentity("local-web-test", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	providerSocket := filepath.Join(t.TempDir(), "provider.sock")
	coreServer := corev2.NewServer(
		corev2.WithSocketPath(providerSocket),
		corev2.WithHistoryDisabled(),
	)
	coreCtx, coreCancel := context.WithCancel(context.Background())
	coreDone := make(chan error, 1)
	providerServer, err := providercore.New(coreServer, providercore.Config{Socket: providerSocket})
	if err != nil {
		coreCancel()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = providerServer.Shutdown(context.Background()) })
	go func() { coreDone <- providerServer.ListenAndServe(coreCtx) }()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, statErr := os.Stat(providerSocket); statErr == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	accessCore, err := accessserver.New(accessserver.Config{
		Socket: filepath.Join(t.TempDir(), "access.sock"),
		Auth:   &accessserver.AuthServices{Access: &localWebTestAccessService{identity: identity}},
		Provider: func(dialCtx context.Context) (terminalprovider.Provider, error) {
			return daemonprovider.DialTerminal(dialCtx, providerSocket)
		},
	})
	if err != nil {
		coreCancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = accessCore.Close()
		coreCancel()
		_ = coreServer.Shutdown(context.Background())
		select {
		case <-coreDone:
		case <-time.After(2 * time.Second):
		}
	})
	password := "correct horse battery staple"
	server, err := Start(Options{Core: accessCore, Address: DefaultAddress, MachineName: "Test machine", Password: []byte(password)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Stop(context.Background()) })

	response, err := http.Get(server.URL() + "/api/bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated bootstrap response = %d", response.StatusCode)
	}
	loginRequest, err := http.NewRequest(http.MethodPost, server.URL()+"/api/auth/login", strings.NewReader(fmt.Sprintf(`{"password":%q}`, password)))
	if err != nil {
		t.Fatal(err)
	}
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRequest.Header.Set("Origin", server.URL())
	loginResponse, err := http.DefaultClient.Do(loginRequest)
	if err != nil {
		t.Fatal(err)
	}
	_ = loginResponse.Body.Close()
	if loginResponse.StatusCode != http.StatusNoContent || len(loginResponse.Cookies()) != 1 {
		t.Fatalf("login response = %d cookies=%v", loginResponse.StatusCode, loginResponse.Cookies())
	}
	cookie := loginResponse.Cookies()[0]
	bootstrapRequest, err := http.NewRequest(http.MethodGet, server.URL()+"/api/bootstrap", nil)
	if err != nil {
		t.Fatal(err)
	}
	bootstrapRequest.AddCookie(cookie)
	response, err = http.DefaultClient.Do(bootstrapRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var bootstrap bootstrapResponse
	if err := json.NewDecoder(response.Body).Decode(&bootstrap); err != nil {
		t.Fatal(err)
	}
	config, err := websocket.NewConfig(strings.Replace(server.URL(), "http://", "ws://", 1)+bootstrap.Bridge.Path, server.URL())
	if err != nil {
		t.Fatal(err)
	}
	config.Protocol = []string{"anytty.binding.v1"}
	config.Header.Set("Cookie", cookie.String())
	connection, err := websocket.DialConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := websocket.Message.Send(connection, append([]byte{0x01}, []byte(bootstrap.Bridge.Token)...)); err != nil {
		t.Fatal(err)
	}
	if frame := receiveLocalWebFrame(t, connection); frame[0] != 0x21 {
		t.Fatalf("authentication response operation = %#x", frame[0])
	}
	payload, err := proto.Marshal(&bindingpb.OpenSessionRequest{RequestId: "integration", EndpointId: localEndpointID, Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE})
	if err != nil {
		t.Fatal(err)
	}
	request := make([]byte, 9+len(payload))
	request[0] = 0x10
	binary.BigEndian.PutUint64(request[1:9], 1)
	copy(request[9:], payload)
	if err := websocket.Message.Send(connection, request); err != nil {
		t.Fatal(err)
	}
	eventFrame := receiveLocalWebAcceptedEvent(t, connection, 1)
	var event bindingpb.EventEnvelope
	if err := proto.Unmarshal(eventFrame[21:], &event); err != nil {
		t.Fatal(err)
	}
	result := event.GetOpenSession()
	if result == nil || result.GetError() != nil || result.GetSessionHandle() == 0 {
		t.Fatalf("open session result = %#v", result)
	}
	command, err := proto.Marshal(&apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
	if err != nil {
		t.Fatal(err)
	}
	execute := make([]byte, 17+len(command))
	execute[0] = 0x11
	binary.BigEndian.PutUint64(execute[1:9], 2)
	binary.BigEndian.PutUint64(execute[9:17], result.GetSessionHandle())
	copy(execute[17:], command)
	if err := websocket.Message.Send(connection, execute); err != nil {
		t.Fatal(err)
	}
	executeFrame := receiveLocalWebAcceptedEvent(t, connection, 2)
	var executeEvent bindingpb.EventEnvelope
	if err := proto.Unmarshal(executeFrame[21:], &executeEvent); err != nil {
		t.Fatal(err)
	}
	if executeEvent.GetExecute().GetError() != nil || executeEvent.GetExecute().GetResult().GetTerminalList() == nil {
		t.Fatalf("terminal list result = %#v", executeEvent.GetExecute())
	}
}

func receiveLocalWebFrame(t *testing.T, connection *websocket.Conn) []byte {
	t.Helper()
	_ = connection.SetReadDeadline(time.Now().Add(5 * time.Second))
	var frame []byte
	if err := websocket.Message.Receive(connection, &frame); err != nil {
		t.Fatal(err)
	}
	if len(frame) < 21 || int(binary.BigEndian.Uint32(frame[17:21])) != len(frame)-21 {
		t.Fatalf("invalid bridge frame (%d bytes)", len(frame))
	}
	return frame
}

func receiveLocalWebAcceptedEvent(t *testing.T, connection *websocket.Conn, requestID uint64) []byte {
	t.Helper()
	var (
		accepted bool
		event    []byte
	)
	for range 2 {
		frame := receiveLocalWebFrame(t, connection)
		switch frame[0] {
		case 0x20:
			if accepted {
				t.Fatal("received duplicate acceptance frame")
			}
			if got := binary.BigEndian.Uint64(frame[1:9]); got != requestID {
				t.Fatalf("acceptance request ID = %d, want %d", got, requestID)
			}
			accepted = true
		case 0x30:
			if event != nil {
				t.Fatal("received duplicate event frame")
			}
			event = frame
		default:
			t.Fatalf("unexpected bridge operation = %#x", frame[0])
		}
	}
	if !accepted || event == nil {
		t.Fatalf("accepted = %t, event received = %t", accepted, event != nil)
	}
	return event
}

type localWebTestAccessService struct {
	identity remoteauth.Identity
}

func (service *localWebTestAccessService) Identity(_ context.Context, challenge []byte) (accesscontract.ClientAccessIdentity, error) {
	proof, err := remoteauth.SignDeviceIdentityProof(service.identity, challenge)
	return accesscontract.ClientAccessIdentity{
		DeviceID: service.identity.DeviceID, DeviceFingerprint: service.identity.Fingerprint,
		DevicePublicKey: append([]byte(nil), service.identity.PublicKey...), Challenge: append([]byte(nil), challenge...), Proof: proof,
	}, err
}

func (*localWebTestAccessService) CreateTicket(context.Context, accesscontract.ClientAccessTicketRequest) (accesscontract.ClientAccessTicket, error) {
	return accesscontract.ClientAccessTicket{}, nil
}
func (*localWebTestAccessService) List(context.Context) ([]accesscontract.ClientAccessRecord, error) {
	return nil, nil
}
func (*localWebTestAccessService) GrantActive(context.Context, string, time.Time, time.Time) bool {
	return false
}
func (*localWebTestAccessService) Revoke(context.Context, string) (accesscontract.ClientAccessRecord, error) {
	return accesscontract.ClientAccessRecord{}, nil
}
