// Package mobileconfig owns build-time mobile client configuration. Values
// are overridden with Go linker -X flags by the Android and iOS build scripts.
package mobileconfig

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/anytty/anytty/proto/bindingpb"
)

var (
	ControllerAddress     = "cloud.anytty.com:443"
	ControllerServerName  = "cloud.anytty.com"
	ControllerCAPEMBase64 string
)

func ResolveCloudProfile(reference string) (*bindingpb.CloudProfileRecord, error) {
	reference = strings.TrimSpace(reference)
	address := strings.TrimSpace(ControllerAddress)
	serverName := strings.TrimSpace(ControllerServerName)
	if reference != "default" || address == "" || serverName == "" {
		return nil, fmt.Errorf("AnyTTY Cloud profile is unavailable")
	}
	var caPEM []byte
	if encoded := strings.TrimSpace(ControllerCAPEMBase64); encoded != "" {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode AnyTTY Cloud controller CA: %w", err)
		}
		caPEM = decoded
	}
	return &bindingpb.CloudProfileRecord{
		AccountProfileRef:    reference,
		ControllerAddress:    address,
		ControllerServerName: serverName,
		ControllerCaPem:      append([]byte(nil), caPEM...),
	}, nil
}
