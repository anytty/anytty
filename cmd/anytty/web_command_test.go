package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/spf13/cobra"
)

func TestLocalWebStatusIncludesPasswordProtectionWithoutSecret(t *testing.T) {
	status := localWebStatusFromProto(&apipb.RemoteLocalStatusResult{
		Enabled: true, HttpUrl: "http://127.0.0.1:4321", PasswordProtected: true,
	})
	if !status.PasswordProtected || status.URL != "http://127.0.0.1:4321" {
		t.Fatalf("status = %#v", status)
	}
}

func TestPromptLocalWebPasswordRejectsNonInteractiveInput(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(bytes.NewBufferString("password\n"))
	if _, err := promptLocalWebPassword(command); err == nil || !strings.Contains(err.Error(), "interactive terminal") {
		t.Fatalf("prompt error = %v", err)
	}
}
