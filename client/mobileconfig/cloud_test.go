package mobileconfig

import (
	"encoding/base64"
	"testing"
)

func TestResolveCloudProfileUsesSharedBuildConfiguration(t *testing.T) {
	previousAddress, previousName, previousCA := ControllerAddress, ControllerServerName, ControllerCAPEMBase64
	t.Cleanup(func() {
		ControllerAddress, ControllerServerName, ControllerCAPEMBase64 = previousAddress, previousName, previousCA
	})
	ControllerAddress = " controller.example:443 "
	ControllerServerName = " controller.example "
	ControllerCAPEMBase64 = base64.StdEncoding.EncodeToString([]byte("CA"))

	profile, err := ResolveCloudProfile(" default ")
	if err != nil {
		t.Fatal(err)
	}
	if profile.GetControllerAddress() != "controller.example:443" || profile.GetControllerServerName() != "controller.example" || string(profile.GetControllerCaPem()) != "CA" {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestResolveCloudProfileRejectsInvalidConfiguration(t *testing.T) {
	previous := ControllerCAPEMBase64
	t.Cleanup(func() { ControllerCAPEMBase64 = previous })
	ControllerCAPEMBase64 = "invalid"
	if _, err := ResolveCloudProfile("default"); err == nil {
		t.Fatal("invalid CA encoding was accepted")
	}
	if _, err := ResolveCloudProfile("other"); err == nil {
		t.Fatal("unknown profile was accepted")
	}
}
