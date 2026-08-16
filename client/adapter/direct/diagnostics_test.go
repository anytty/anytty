package direct

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
)

func TestReportDirectFailureEmitsOnlyStableClassifications(t *testing.T) {
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

	handshake := &remoteauth.HandshakeError{
		Code:   remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_PROOF_INVALID,
		Detail: "secret credential and address",
		Cause:  errors.New("secret SDP"),
	}
	failure := &clientruntime.Error{
		Code:      clientruntime.ErrorUnavailable,
		Message:   "secret user-facing message",
		Cause:     handshake,
		Retryable: false,
	}
	if got := reportDirectFailure(23, directFailureDataChannelAuth, failure); got != failure {
		t.Fatalf("reported error = %v, want original failure", got)
	}

	message := output.String()
	for _, want := range []string{
		"anytty direct failure generation=23",
		"stage=datachannel_auth",
		"code=unavailable",
		"signaling_code=none",
		"auth_code=AUTH_ERROR_CODE_CAPABILITY_PROOF_INVALID",
		"retryable=false",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("diagnostic log %q does not contain %q", message, want)
		}
	}
	for _, secret := range []string{"secret", "credential", "address", "SDP", "user-facing"} {
		if strings.Contains(message, secret) {
			t.Fatalf("diagnostic log leaked %q: %q", secret, message)
		}
	}
}

func TestReportDirectFailureIncludesSignalingCode(t *testing.T) {
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

	failure := &SignalingError{
		Code:    remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_AUTHORIZATION,
		Message: "secret daemon response",
	}
	reportDirectFailure(29, directFailurePeerSetup, failure)
	if message := output.String(); !strings.Contains(message, "signaling_code=DIRECT_SIGNALING_ERROR_CODE_AUTHORIZATION") || strings.Contains(message, "secret") {
		t.Fatalf("unexpected diagnostic log %q", message)
	}
}
