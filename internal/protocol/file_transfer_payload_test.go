package protocol

import (
	"bytes"
	"testing"
)

func TestFileTransferPayloadRoundTrips(t *testing.T) {
	digest := bytes.Repeat([]byte{0x5a}, 32)
	dataPayload, err := EncodeFileTransferData(FileTransferData{Offset: 12, Data: []byte("chunk")})
	if err != nil {
		t.Fatal(err)
	}
	data, err := DecodeFileTransferData(dataPayload)
	if err != nil || data.Offset != 12 || string(data.Data) != "chunk" {
		t.Fatalf("data %#v %v", data, err)
	}
	ackPayload, err := EncodeFileTransferAck(FileTransferAck{Offset: 17, WindowBytes: 65536})
	if err != nil {
		t.Fatal(err)
	}
	ack, err := DecodeFileTransferAck(ackPayload)
	if err != nil || ack.Offset != 17 || ack.WindowBytes != 65536 {
		t.Fatalf("ack %#v %v", ack, err)
	}
	finishPayload, err := EncodeFileTransferFinish(FileTransferFinish{Size: 17, SHA256: digest})
	if err != nil {
		t.Fatal(err)
	}
	finish, err := DecodeFileTransferFinish(finishPayload)
	if err != nil || finish.Size != 17 || !bytes.Equal(finish.SHA256, digest) {
		t.Fatalf("finish %#v %v", finish, err)
	}
	resultPayload, err := EncodeFileTransferResult(FileTransferResult{Path: "/tmp/a", Size: 17, SHA256: digest})
	if err != nil {
		t.Fatal(err)
	}
	result, err := DecodeFileTransferResult(resultPayload)
	if err != nil || result.Path != "/tmp/a" || !bytes.Equal(result.SHA256, digest) {
		t.Fatalf("result %#v %v", result, err)
	}
}

func TestFileTransferFinishRejectsInvalidDigest(t *testing.T) {
	if _, err := EncodeFileTransferFinish(FileTransferFinish{SHA256: []byte("short")}); err == nil {
		t.Fatal("expected digest length error")
	}
}
