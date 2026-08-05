package client

import (
	"testing"

	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

type signalSessionTestStream struct {
	cloudv1.ClientGateway_ConnectClient
	sent []*cloudv1.ClientSignal
}

func (stream *signalSessionTestStream) Send(signal *cloudv1.ClientSignal) error {
	stream.sent = append(stream.sent, signal)
	return nil
}

func TestSignalSessionConfirmsSelectedPathOnce(t *testing.T) {
	stream := &signalSessionTestStream{}
	session := &SignalSession{stream: stream, senderID: "client", bootID: "boot", sessionID: "session"}
	if err := session.ConfirmPath(cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT); err != nil {
		t.Fatal(err)
	}
	if err := session.ConfirmPath(cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_RELAY); err != nil {
		t.Fatal(err)
	}
	if len(stream.sent) != 1 || stream.sent[0].GetStreamSeq() != 3 || stream.sent[0].GetPathSelected().GetPath() != cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT {
		t.Fatalf("path confirmation = %#v", stream.sent)
	}
}
