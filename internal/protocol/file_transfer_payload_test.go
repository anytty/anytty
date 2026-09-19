package protocol

import (
	"bytes"
	"testing"

	"github.com/anytty/anytty/proto/access/wirepb"
	"google.golang.org/protobuf/proto"
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

// TestFileTransferOptionalFieldsRoundTrip 覆盖 Phase 4 的可选扩展字段。
func TestFileTransferOptionalFieldsRoundTrip(t *testing.T) {
	dataPayload, err := EncodeFileTransferData(FileTransferData{Offset: 12, Data: []byte("chunk"), Encoding: "zstd"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := DecodeFileTransferData(dataPayload)
	if err != nil || data.Encoding != "zstd" || string(data.Data) != "chunk" {
		t.Fatalf("data %#v %v", data, err)
	}
	ackPayload, err := EncodeFileTransferAck(FileTransferAck{
		Offset: 17, WindowBytes: 65536, TransferredBytes: 17, TotalBytes: 100, ElapsedMillis: 250,
	})
	if err != nil {
		t.Fatal(err)
	}
	ack, err := DecodeFileTransferAck(ackPayload)
	if err != nil || ack.TransferredBytes != 17 || ack.TotalBytes != 100 || ack.ElapsedMillis != 250 {
		t.Fatalf("ack %#v %v", ack, err)
	}
	digest := bytes.Repeat([]byte{0x11}, 32)
	finishPayload, err := EncodeFileTransferFinish(FileTransferFinish{Size: 17, SHA256: digest, ElapsedMillis: 99})
	if err != nil {
		t.Fatal(err)
	}
	finish, err := DecodeFileTransferFinish(finishPayload)
	if err != nil || finish.ElapsedMillis != 99 {
		t.Fatalf("finish %#v %v", finish, err)
	}
	if _, err := EncodeFileTransferFinish(FileTransferFinish{Size: 17, SHA256: digest, ElapsedMillis: 0}); err != nil {
		t.Fatal(err)
	}
}

// TestFileTransferOptionalFieldsDefaultIsByteIdentical 证明未启用新字段时
// payload 与旧协议完全一致（proto3 零值不编码）。
func TestFileTransferOptionalFieldsDefaultIsByteIdentical(t *testing.T) {
	dataPayload, err := EncodeFileTransferData(FileTransferData{Offset: 7, Data: []byte("abc")})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(dataPayload, []byte("zstd")) {
		t.Fatalf("identity data frame unexpectedly contains encoding: %q", dataPayload)
	}
	ackPayload, err := EncodeFileTransferAck(FileTransferAck{Offset: 7, WindowBytes: 9})
	if err != nil {
		t.Fatal(err)
	}
	legacyAck, err := proto.Marshal(&wirepb.FileTransferAck{Offset: 7, WindowBytes: 9})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ackPayload, legacyAck) {
		t.Fatalf("zero-value ack drifted from legacy encoding: %q vs %q", ackPayload, legacyAck)
	}
	digest := bytes.Repeat([]byte{0x22}, 32)
	finishPayload, err := EncodeFileTransferFinish(FileTransferFinish{Size: 3, SHA256: digest})
	if err != nil {
		t.Fatal(err)
	}
	legacyFinish, err := proto.Marshal(&wirepb.FileTransferFinish{Size: 3, Sha256: digest})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(finishPayload, legacyFinish) {
		t.Fatalf("zero-value finish drifted from legacy encoding")
	}
}
