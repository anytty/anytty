package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLicensesCommandPrintsEmbeddedThirdPartyNotices(t *testing.T) {
	var output bytes.Buffer
	command := newRootCmd()
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"licenses"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "AnyTTY CLI Third-Party Notices") || !strings.Contains(text, "github.com/spf13/cobra") {
		t.Fatalf("licenses output does not contain embedded notices: %q", text)
	}
}
