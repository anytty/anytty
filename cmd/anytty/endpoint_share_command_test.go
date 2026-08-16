package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
)

func TestEndpointShareCLITransfersConfigOnceAndImportsAtomically(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	sourcePath := root + "/source.yaml"
	targetPath := filepath.Join(root, "config", "anytty", endpointdomain.DefaultFileName)
	source := endpointdomain.Endpoint{
		ID: "studio", Label: "Studio", LabelSource: endpointdomain.SourceUser,
		DaemonIdentity: endpointdomain.DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"},
		ConnectMode:    endpointdomain.ConnectOnDemand, Enabled: true,
		Routes: map[endpointdomain.RouteID]endpointdomain.AccessRoute{
			"direct": {
				ID: "direct", Kind: endpointdomain.RouteDirectWebRTCTCP, Enabled: true, CredentialRef: "grant:source",
				Source: endpointdomain.SourceBootstrap, PolicySource: endpointdomain.SourceBootstrap,
				SignalingAddresses: []string{"127.0.0.1:41120"}, ICETCPAddresses: []string{"127.0.0.1:41121"},
			},
		},
	}
	if err := endpointdomain.Save(sourcePath, endpointdomain.Registry{Version: endpointdomain.RegistryVersion, Default: source.ID, Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{source.ID: source}}); err != nil {
		t.Fatal(err)
	}
	if err := endpointdomain.Save(targetPath, endpointdomain.Registry{Version: endpointdomain.RegistryVersion, Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{}}); err != nil {
		t.Fatal(err)
	}

	output := newFirstLineWriter()
	senderRuntime := &endpointCommandRuntime{registryPath: sourcePath}
	sender := newEndpointShareCommand(senderRuntime)
	sender.SetOut(output)
	sender.SetErr(output)
	sender.SetArgs([]string{"studio", "--listen", "127.0.0.1:0", "--raw", "--ttl", "1m"})
	senderDone := make(chan error, 1)
	go func() { senderDone <- sender.Execute() }()
	var uri string
	select {
	case uri = <-output.first:
	case <-time.After(5 * time.Second):
		t.Fatal("sender did not publish share URI")
	}
	uri = strings.TrimSpace(uri)
	if !strings.HasPrefix(uri, "anytty://share?payload=") {
		t.Fatalf("sender output=%q", uri)
	}

	receiverRuntime := &endpointCommandRuntime{}
	receiver := newEndpointShareReceiveCommand(receiverRuntime)
	var receiverOutput bytes.Buffer
	receiver.SetOut(&receiverOutput)
	receiver.SetErr(&receiverOutput)
	receiver.SetArgs([]string{uri, "--yes"})
	if err := receiver.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-senderDone; err != nil {
		t.Fatal(err)
	}
	registry, err := loadNormalizedConnectionRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Endpoints) != 1 {
		t.Fatalf("target registry=%#v", registry)
	}
	for _, imported := range registry.Endpoints {
		route := imported.Routes["direct"]
		if route.CredentialRef != "" || route.Source != endpointdomain.SourceShare || route.PolicySource != endpointdomain.SourceShare {
			t.Fatalf("imported route is not config-only share: %#v", route)
		}
	}
	if strings.Contains(receiverOutput.String(), "\t") ||
		!strings.Contains(receiverOutput.String(), "ACTION") ||
		!strings.Contains(receiverOutput.String(), "direct-webrtc-tcp") ||
		!strings.Contains(receiverOutput.String(), "Status") ||
		!strings.Contains(receiverOutput.String(), "config only") {
		t.Fatalf("receiver output=%q", receiverOutput.String())
	}

	replay := newEndpointShareReceiveCommand(receiverRuntime)
	replay.SetArgs([]string{uri, "--yes"})
	if err := replay.Execute(); err == nil {
		t.Fatal("consumed CLI share offer unexpectedly replayed")
	}
}

type firstLineWriter struct {
	mu     sync.Mutex
	buffer bytes.Buffer
	first  chan string
	once   sync.Once
}

func newFirstLineWriter() *firstLineWriter {
	return &firstLineWriter{first: make(chan string, 1)}
}

func (writer *firstLineWriter) Write(payload []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	written, err := writer.buffer.Write(payload)
	if index := strings.IndexByte(writer.buffer.String(), '\n'); index >= 0 {
		writer.once.Do(func() { writer.first <- writer.buffer.String()[:index+1] })
	}
	return written, err
}
