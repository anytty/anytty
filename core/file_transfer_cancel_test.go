package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUploadCancelAtomicallyRemovesResumeOwnerBeforeStreamCleanup(t *testing.T) {
	server := NewServer()
	defer func() { _ = server.Shutdown(t.Context()) }()
	tempPath := filepath.Join(t.TempDir(), "upload.part")
	if err := os.WriteFile(tempPath, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	owner := &protocolSession{server: server, scope: TransportScope{PrincipalID: "principal"}, sessionID: 1, fileChannels: make(map[uint16]*sessionFileTransfer), fileIDs: make(map[string]uint16)}
	canceller := &protocolSession{server: server, scope: TransportScope{PrincipalID: "principal"}, sessionID: 2, fileChannels: make(map[uint16]*sessionFileTransfer), fileIDs: make(map[string]uint16)}
	resumer := &protocolSession{server: server, scope: TransportScope{PrincipalID: "principal"}, sessionID: 3, fileChannels: make(map[uint16]*sessionFileTransfer), fileIDs: make(map[string]uint16)}
	id := "0123456789abcdef0123456789abcdef"
	record := &uploadTransferRecord{ID: id, PrincipalID: "principal", TargetPath: filepath.Join(t.TempDir(), "target.bin"), TempPath: tempPath, Size: 7, ExpiresAt: time.Now().Add(time.Minute), AttachedSessionID: owner.sessionID, attachedSession: owner}
	server.fileUploads[id] = record

	taken, ok := canceller.takeOwnedUploadForCancel(id)
	if !ok || taken != record {
		t.Fatalf("cancel did not acquire upload owner: ok=%v record=%p", ok, taken)
	}
	if _, err := resumer.openFileUpload(FileUploadOpenRequest{Path: record.TargetPath, Size: record.Size, Overwrite: true, ResumeTransferToken: fileUploadResumeToken(id)}); err == nil {
		t.Fatal("resume takeover succeeded after cancel atomically acquired the owner")
	}
	if _, exists := server.fileUploads[id]; exists {
		t.Fatal("cancelled upload remained in the resumable registry")
	}
}
