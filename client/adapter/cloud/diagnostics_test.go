package cloud

import (
	"bytes"
	"errors"
	"io"
	"log"
	"strings"
	"testing"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestReportCloudFailureEmitsOnlyStableClassifications(t *testing.T) {
	var output bytes.Buffer
	previousWriter, previousFlags, previousPrefix := log.Writer(), log.Flags(), log.Prefix()
	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})

	cause := status.Error(codes.PermissionDenied, "secret endpoint, address, and credential")
	failure := &clientruntime.Error{
		Code:      clientruntime.ErrorRelayNotInPlan,
		Message:   "secret user-facing backend message",
		Cause:     cause,
		Retryable: false,
	}
	if got := reportCloudFailure(17, cloudFailureEdgeExchange, failure); got != failure {
		t.Fatalf("reported error = %v, want original failure", got)
	}

	message := output.String()
	for _, want := range []string{
		"anytty cloud failure generation=17",
		"stage=edge_exchange",
		"code=relay_not_in_plan",
		"rpc_code=PermissionDenied",
		"auth_code=none",
		"retryable=false",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("diagnostic log %q does not contain %q", message, want)
		}
	}
	for _, secret := range []string{"secret", "endpoint", "address", "credential", "backend message"} {
		if strings.Contains(message, secret) {
			t.Fatalf("diagnostic log leaked %q: %q", secret, message)
		}
	}
}

func TestReportCloudFailureIncludesRemoteAuthCodeWithoutDetail(t *testing.T) {
	var output bytes.Buffer
	previousWriter, previousFlags, previousPrefix := log.Writer(), log.Flags(), log.Prefix()
	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})

	failure := &remoteauth.HandshakeError{
		Code:   remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL,
		Detail: "secret remote auth frame",
	}
	reportCloudFailure(31, cloudFailurePairingHandshake, failure)
	message := output.String()
	if !strings.Contains(message, "auth_code=AUTH_ERROR_CODE_PROTOCOL") || strings.Contains(message, "secret") || strings.Contains(message, "remote auth frame") {
		t.Fatalf("unexpected diagnostic log %q", message)
	}
}

func TestReportCloudSessionCloseEmitsOnlyStableClassifications(t *testing.T) {
	tests := []struct {
		name        string
		origin      cloudSessionCloseOrigin
		cause       error
		terminalErr error
		want        []string
	}{
		{
			name:        "signaling grpc failure",
			origin:      cloudSessionCloseSignaling,
			terminalErr: status.Error(codes.Unavailable, "private Edge address and session credential"),
			want:        []string{"origin=signaling", "cause=grpc_error", "code=unavailable", "rpc_code=Unavailable", "retryable=true"},
		},
		{
			name:        "application eof",
			origin:      cloudSessionCloseApplication,
			cause:       io.EOF,
			terminalErr: io.EOF,
			want:        []string{"origin=application", "cause=eof", "code=unavailable", "rpc_code=Unknown", "retryable=false"},
		},
		{
			name:   "local close",
			origin: cloudSessionCloseLocal,
			want:   []string{"origin=local", "cause=local_close", "code=none", "rpc_code=none", "retryable=false"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			previousWriter, previousFlags, previousPrefix := log.Writer(), log.Flags(), log.Prefix()
			log.SetOutput(&output)
			log.SetFlags(0)
			log.SetPrefix("")
			t.Cleanup(func() {
				log.SetOutput(previousWriter)
				log.SetFlags(previousFlags)
				log.SetPrefix(previousPrefix)
			})

			cause := test.cause
			if test.origin == cloudSessionCloseSignaling {
				cause = cloudSignalingTermination(test.terminalErr)
			}
			reportCloudSessionClose(test.origin, cause, test.terminalErr)
			message := output.String()
			if !strings.Contains(message, "anytty cloud session stage=closed") {
				t.Fatalf("missing Cloud session close diagnostic: %q", message)
			}
			for _, want := range test.want {
				if !strings.Contains(message, want) {
					t.Fatalf("diagnostic log %q does not contain %q", message, want)
				}
			}
			for _, private := range []string{"private", "Edge address", "session credential", "error="} {
				if strings.Contains(message, private) {
					t.Fatalf("diagnostic log leaked %q: %q", private, message)
				}
			}
		})
	}
}

func TestCloudConnectionErrorMarksTransientRPCFailuresRetryable(t *testing.T) {
	for _, code := range []codes.Code{codes.Unavailable, codes.DeadlineExceeded, codes.Aborted, codes.ResourceExhausted, codes.NotFound} {
		t.Run(code.String(), func(t *testing.T) {
			mapped := cloudConnectionError(status.Error(code, "backend detail"))
			var runtimeError *clientruntime.Error
			if !errors.As(mapped, &runtimeError) {
				t.Fatalf("mapped error type = %T", mapped)
			}
			if runtimeError.Code != clientruntime.ErrorUnavailable || !runtimeError.Retryable || status.Code(mapped) != code {
				t.Fatalf("mapped error = %#v, rpc code = %s", runtimeError, status.Code(mapped))
			}
		})
	}
}
