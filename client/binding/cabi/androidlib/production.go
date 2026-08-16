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

var androidSessionAuthority = clientruntime.NewSessionGenerationAuthority()

// androidProductionHost 只装配 Android generation、Pion peer 与共享 engine host。
// credential/auth/Hello/API 逻辑不得在 JNI 包内形成平台分叉。
type androidProductionHost struct {
	*enginehost.Host
	broker *binding.PlatformBroker
}

func newAndroidProductionHost() (*androidProductionHost, error) {
	return newAndroidProductionHostWithPeers(pionadapter.Factory{
		NetworkFactory:      pionadapter.NewDefaultRouteNet,
		RouteNetworkFactory: newAndroidRouteNetwork,
		Logger:              nil,
	})
}

func newAndroidProductionHostWithPeers(peers pionadapter.Factory) (*androidProductionHost, error) {
	configureAndroidLogging()
	broker := binding.NewPlatformBroker()
	host, err := enginehost.New(enginehost.Options{
		Broker: broker, DirectPeers: peers, ClientName: "anytty-android", CredentialPrefix: "android-access-",
		SessionAuthority: androidSessionAuthority, CloudProduct: cloudv1.ClientProduct_CLIENT_PRODUCT_ANDROID, EnableLocalDiscovery: true,
		CloudProfileResolve: func(_ context.Context, reference string) (*bindingpb.CloudProfileRecord, error) {
			return mobileconfig.ResolveCloudProfile(reference)
		},
	})
	if err != nil {
		_ = broker.Close()
		return nil, err
	}
	return &androidProductionHost{Host: host, broker: broker}, nil
}

func (host *androidProductionHost) close() error { return host.Close() }
