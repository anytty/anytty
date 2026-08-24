package localweb

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	apilayer "github.com/anytty/anytty/api_layer"
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/bindingpb"
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
	core := corev2.NewServer(
		corev2.WithHistoryDisabled(),
		corev2.WithApplicationExecutorFactory(apilayer.CoreApplicationExecutorFactory),
		corev2.WithClientAccessService(&localWebTestAccessService{identity: identity}),
	)
	t.Cleanup(func() { _ = core.Shutdown(context.Background()) })
	server, err := Start(Options{Core: core, Address: DefaultAddress, MachineName: "Test machine"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Stop(context.Background()) })

	response, err := http.Get(server.URL() + "/api/bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var bootstrap bootstrapResponse
	if err := json.NewDecoder(response.Body).Decode(&bootstrap); err != nil {
		t.Fatal(err)
	}
	config, err := websocket.NewConfig(fmt.Sprintf("ws://127.0.0.1:%d/", bootstrap.Bridge.Port), server.URL())
	if err != nil {
		t.Fatal(err)
	}
	config.Protocol = []string{"anytty.binding.v1"}
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

func (service *localWebTestAccessService) Identity(_ context.Context, challenge []byte) (corev2.ClientAccessIdentity, error) {
	proof, err := remoteauth.SignDeviceIdentityProof(service.identity, challenge)
	return corev2.ClientAccessIdentity{
		DeviceID: service.identity.DeviceID, DeviceFingerprint: service.identity.Fingerprint,
		DevicePublicKey: append([]byte(nil), service.identity.PublicKey...), Challenge: append([]byte(nil), challenge...), Proof: proof,
	}, err
}

func (*localWebTestAccessService) CreateTicket(context.Context, corev2.ClientAccessTicketRequest) (corev2.ClientAccessTicket, error) {
	return corev2.ClientAccessTicket{}, nil
}
func (*localWebTestAccessService) List(context.Context) ([]corev2.ClientAccessRecord, error) {
	return nil, nil
}
func (*localWebTestAccessService) GrantActive(context.Context, string, time.Time, time.Time) bool {
	return false
}
func (*localWebTestAccessService) Revoke(context.Context, string) (corev2.ClientAccessRecord, error) {
	return corev2.ClientAccessRecord{}, nil
}
