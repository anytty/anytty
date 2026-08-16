//go:build cgo

package main

import (
	"context"
	pionadapter "github.com/anytty/anytty/client/adapter/webrtc/pion"
	"github.com/anytty/anytty/client/binding"
	"github.com/anytty/anytty/client/binding/enginehost"
	"github.com/anytty/anytty/client/mobileconfig"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/bindingpb"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

var iosSessionAuthority = clientruntime.NewSessionGenerationAuthority()

type iosProductionHost struct {
	*enginehost.Host
	broker *binding.PlatformBroker
}

func newIOSProductionHost() (*iosProductionHost, error) {
	configureIOSLogging()
	broker := binding.NewPlatformBroker()
	host, err := enginehost.New(enginehost.Options{
		Broker:           broker,
		DirectPeers:      newIOSPeerFactory(),
		ClientName:       "anytty-ios",
		CredentialPrefix: "ios-access-",
		SessionAuthority: iosSessionAuthority,
		CloudProduct:     cloudv1.ClientProduct_CLIENT_PRODUCT_IOS,
		CloudProfileResolve: func(_ context.Context, reference string) (*bindingpb.CloudProfileRecord, error) {
			return mobileconfig.ResolveCloudProfile(reference)
		},
		EnableLocalDiscovery: true,
	})
	if err != nil {
		_ = broker.Close()
		return nil, err
	}
	return &iosProductionHost{Host: host, broker: broker}, nil
}

func newIOSPeerFactory() pionadapter.Factory {
	return pionadapter.Factory{Logger: nil}
}

func (host *iosProductionHost) close() error { return host.Close() }
