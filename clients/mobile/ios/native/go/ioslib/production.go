//go:build cgo

package main

import (
	pionadapter "github.com/anytty/anytty/client/adapter/webrtc/pion"
	"github.com/anytty/anytty/client/binding"
	"github.com/anytty/anytty/client/binding/enginehost"
	clientruntime "github.com/anytty/anytty/client/runtime"
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
	network, err := pionadapter.NewDefaultRouteNet()
	if err != nil {
		_ = broker.Close()
		return nil, err
	}
	host, err := enginehost.New(enginehost.Options{
		Broker:           broker,
		DirectPeers:      pionadapter.Factory{Network: network, Logger: nil},
		ClientName:       "anytty-ios",
		CredentialPrefix: "ios-access-",
		SessionAuthority: iosSessionAuthority,
		CloudProduct:     cloudv1.ClientProduct_CLIENT_PRODUCT_IOS,
	})
	if err != nil {
		_ = broker.Close()
		return nil, err
	}
	return &iosProductionHost{Host: host, broker: broker}, nil
}

func (host *iosProductionHost) close() error { return host.Close() }
