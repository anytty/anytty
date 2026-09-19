package providerproto

import (
	"bytes"
	"testing"

	providerv1 "github.com/anytty/anytty/proto/provider/v1"
)

func TestFrameRoundTrip(t *testing.T) {
	frame, err := EncodeFrame(7, TypeRequest, []byte("payload"))
	if err != nil {
		t.Fatal(err)
	}
	channel, typ, payload, err := DecodeFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	if channel != 7 || typ != TypeRequest || !bytes.Equal(payload, []byte("payload")) {
		t.Fatalf("decoded frame = %d/%d/%q", channel, typ, payload)
	}
}

func TestControlPayloadRoundTrips(t *testing.T) {
	helloPayload, err := EncodeHelloPayload(&providerv1.Hello{Version: Version, Client: "access"})
	if err != nil {
		t.Fatal(err)
	}
	hello, err := DecodeHelloPayload(helloPayload)
	if err != nil || hello.GetVersion() != Version || hello.GetClient() != "access" {
		t.Fatalf("hello = %#v err=%v", hello, err)
	}

	requestPayload, err := EncodeRequestPayload(&providerv1.Request{
		Id: 3, Command: &providerv1.Request_TerminalGet{TerminalGet: &providerv1.TerminalRef{TerminalId: "t1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := DecodeRequestPayload(requestPayload)
	if err != nil || request.GetId() != 3 || request.GetTerminalGet().GetTerminalId() != "t1" {
		t.Fatalf("request = %#v err=%v", request, err)
	}

	errorPayload, err := EncodeErrorPayload(9, ErrorNotFound, "terminal not found")
	if err != nil {
		t.Fatal(err)
	}
	response, err := DecodeResponsePayload(errorPayload)
	if err != nil || response.GetId() != 9 || response.GetError().GetCode() != ErrorNotFound || response.GetError().GetMessage() != "terminal not found" {
		t.Fatalf("error response = %#v err=%v", response, err)
	}
}

func TestDecodeMalformedPayloadsFailClosed(t *testing.T) {
	if _, err := DecodeHelloPayload([]byte{0xff, 0xff}); err == nil {
		t.Fatal("malformed hello accepted")
	}
	if _, err := DecodeRequestPayload([]byte{0xff, 0xff}); err == nil {
		t.Fatal("malformed request accepted")
	}
	if _, err := DecodeResponsePayload([]byte{0xff, 0xff}); err == nil {
		t.Fatal("malformed response accepted")
	}
}
