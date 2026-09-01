//go:build cgo

package main

import (
	"strings"
	"testing"
)

func TestAndroidCloudSessionDiagnosticIsAllowedAndSanitized(t *testing.T) {
	payload := []byte(`anytty cloud session stage=closed origin=signaling cause=grpc_error code=unavailable rpc_code=Unavailable retryable=true session=private-session error="private backend address" endpoint=edge.example address=203.0.113.7 credential=private-grant`)
	if !androidDiagnosticAllowed(payload) {
		t.Fatal("structured Cloud session diagnostic was rejected")
	}
	want := "anytty cloud session stage=closed origin=signaling cause=grpc_error code=unavailable rpc_code=Unavailable retryable=true"
	got := string(sanitizeAndroidDiagnostic(payload))
	if got != want {
		t.Fatalf("sanitized Cloud session diagnostic = %q, want %q", got, want)
	}
	for _, forbidden := range []string{"private-session", "backend address", "edge.example", "203.0.113.7", "private-grant"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("Cloud session sanitizer retained %q: %q", forbidden, got)
		}
	}
}

func TestAndroidDetailedCloudSignalingDiagnosticRemainsPrivate(t *testing.T) {
	for _, payload := range [][]byte{
		[]byte(`anytty cloud signaling session=private-session stage=stream_ended cause=grpc_error grpc_code=Unavailable error="private backend address"`),
		[]byte("anytty cloud sessionx stage=closed origin=signaling"),
		[]byte("ordinary application log"),
	} {
		if androidDiagnosticAllowed(payload) {
			t.Fatalf("private diagnostic was allowlisted: %q", payload)
		}
	}
}

func TestAndroidCloudSessionDiagnosticRejectsPrivateAllowedFieldValues(t *testing.T) {
	valid := "anytty cloud session stage=closed origin=signaling cause=grpc_error code=unavailable rpc_code=Unavailable retryable=true"
	tests := map[string]string{
		"stage":     strings.Replace(valid, "stage=closed", "stage=private-session-uuid", 1),
		"origin":    strings.Replace(valid, "origin=signaling", "origin=private-device-address", 1),
		"cause":     strings.Replace(valid, "cause=grpc_error", "cause=private-relay-address", 1),
		"code":      strings.Replace(valid, "code=unavailable", "code=private-credential", 1),
		"rpc_code":  strings.Replace(valid, "rpc_code=Unavailable", "rpc_code=private-token", 1),
		"retryable": strings.Replace(valid, "retryable=true", "retryable=private-grant", 1),
	}
	for name, payload := range tests {
		t.Run(name, func(t *testing.T) {
			if androidDiagnosticAllowed([]byte(payload)) {
				t.Fatalf("diagnostic with private %s value was allowlisted: %q", name, payload)
			}
			if got := sanitizeAndroidDiagnostic([]byte(payload)); len(got) != 0 {
				t.Fatalf("diagnostic with private %s value sanitized to %q, want fail closed", name, got)
			}
		})
	}
}

func TestAndroidCloudSessionDiagnosticRequiresEveryFieldOnce(t *testing.T) {
	for name, payload := range map[string]string{
		"missing":   "anytty cloud session stage=closed origin=signaling cause=grpc_error code=unavailable rpc_code=Unavailable",
		"duplicate": "anytty cloud session stage=closed origin=signaling origin=local cause=grpc_error code=unavailable rpc_code=Unavailable retryable=true",
	} {
		t.Run(name, func(t *testing.T) {
			if androidDiagnosticAllowed([]byte(payload)) {
				t.Fatalf("%s-field diagnostic was allowlisted: %q", name, payload)
			}
			if got := sanitizeAndroidDiagnostic([]byte(payload)); len(got) != 0 {
				t.Fatalf("%s-field diagnostic sanitized to %q, want fail closed", name, got)
			}
		})
	}
}
