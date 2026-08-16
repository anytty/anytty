package remoteauth

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/anytty/anytty/shared/securefs"
)

func TestLoadOrCreateIdentityKeepsStableFingerprintAndPrivatePermissions(t *testing.T) {
	dir := t.TempDir()
	first, err := LoadOrCreateIdentity(dir, "device-1")
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	second, err := LoadOrCreateIdentity(dir, "device-1")
	if err != nil {
		t.Fatalf("reload identity: %v", err)
	}
	if first.DeviceID != "device-1" || first.Fingerprint == "" || second.Fingerprint != first.Fingerprint {
		t.Fatalf("identity is not stable: first=%#v second=%#v", first, second)
	}
	path := filepath.Join(dir, identityPrivateKeyFile)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat private key: %v", err)
	}
	if !securefs.IsPrivateFile(path, info) {
		t.Fatal("private key does not satisfy the platform privacy contract")
	}
}

func TestLoadOrCreateIdentityRejectsDifferentDeviceID(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadOrCreateIdentity(dir, "device-1"); err != nil {
		t.Fatalf("create identity: %v", err)
	}
	if _, err := LoadOrCreateIdentity(dir, "device-2"); err == nil {
		t.Fatal("identity store must not silently replace device identity")
	}
}

func TestLoadOrCreateLocalIdentityKeepsGeneratedDeviceID(t *testing.T) {
	dir := t.TempDir()
	first, err := LoadOrCreateLocalIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadOrCreateLocalIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	if first.DeviceID == "" || first.DeviceID != second.DeviceID || first.Fingerprint != second.Fingerprint {
		t.Fatalf("identity changed: first=%#v second=%#v", first, second)
	}
}

func TestLoadOrCreateLocalIdentitySerializesFirstCreationAcrossCallers(t *testing.T) {
	dir := t.TempDir()
	results := make(chan Identity, 2)
	errorsCh := make(chan error, 2)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			identity, err := LoadOrCreateLocalIdentity(dir)
			results <- identity
			errorsCh <- err
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	var first Identity
	for identity := range results {
		if first.DeviceID == "" {
			first = identity
		} else if identity.DeviceID != first.DeviceID || identity.Fingerprint != first.Fingerprint {
			t.Fatalf("concurrent identity creation split truth: first=%#v second=%#v", first, identity)
		}
	}
}

func TestLoadOrCreateIdentityRejectsUnknownAndTrailingState(t *testing.T) {
	for _, mutate := range []func([]byte) []byte{
		func(payload []byte) []byte {
			return bytes.Replace(payload, []byte("\n}"), []byte(",\n  \"legacy_token\": \"forbidden\"\n}"), 1)
		},
		func(payload []byte) []byte { return append(payload, []byte("{}\n")...) },
	} {
		dir := t.TempDir()
		if _, err := LoadOrCreateIdentity(dir, "device-1"); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, identityPrivateKeyFile)
		payload, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, mutate(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadOrCreateLocalIdentity(dir); err == nil || !strings.Contains(err.Error(), "decode remote identity") {
			t.Fatalf("invalid identity state error = %v", err)
		}
	}
}
