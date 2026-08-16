package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	systemadapter "github.com/anytty/anytty/client/adapter/system"
	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/remoteauthpb"
	remotev2daemon "github.com/anytty/anytty/remote/daemon"
	remotev2webrtc "github.com/anytty/anytty/remote/webrtc"
)

var v3DirectRuntimeAddresses struct {
	sync.RWMutex
	signaling string
	ice       string
}

const (
	defaultDirectSignalingAddress = "0.0.0.0:41120"
)

var v3PrivateLANAddresses = systemadapter.PrivateLANIPv4Addresses

type v3DirectPairingRouteOptions struct {
	DefaultListen      string
	DirectAddresses    []string
	SignalingAddresses []string
	ICETCPAddresses    []string
	ServerName         string
}

type v3RemoteDaemonCore = remotev2daemon.ScopedTransportServer

// startV3DirectDaemon 启动当前 daemon 唯一的 embedded signaling 与共享 ICE-TCP mux。
// listener 使用同一 DeviceIdentity/AccessStore/Core；Direct 失败必须阻止 daemon 发布不可兑换的 Route hint。
func startV3DirectDaemon(ctx context.Context, core v3RemoteDaemonCore, clientAccess v3ClientAccessRuntime, logger *slog.Logger) (func(), error) {
	if core == nil || clientAccess.Store == nil {
		return nil, fmt.Errorf("Direct daemon requires core-v2 and client access store")
	}
	signalingAddress, iceAddress := v3DirectAddresses()
	rootListener, err := net.Listen("tcp", signalingAddress)
	if err != nil {
		return nil, fmt.Errorf("listen Direct signaling %q: %w", signalingAddress, err)
	}
	signalingListener := rootListener
	var iceListener net.Listener
	if signalingAddress == iceAddress {
		signalingListener, iceListener, err = remotev2webrtc.SplitDirectListener(rootListener)
		if err != nil {
			_ = rootListener.Close()
			return nil, fmt.Errorf("split Direct signaling and ICE-TCP listener: %w", err)
		}
	} else {
		iceListener, err = net.Listen("tcp", iceAddress)
		if err != nil {
			_ = signalingListener.Close()
			return nil, fmt.Errorf("listen Direct ICE-TCP %q: %w", iceAddress, err)
		}
	}
	server, err := remotev2webrtc.NewDirectServer(clientAccess.Identity, remotev2daemon.SessionAcceptor{
		Core: core, Identity: clientAccess.Identity, AccessStore: clientAccess.Store,
	}, signalingListener, iceListener, nil, remotev2webrtc.WithPionLogger(logger))
	if err != nil {
		_ = signalingListener.Close()
		_ = iceListener.Close()
		return nil, err
	}
	actualSignaling := signalingListener.Addr().String()
	actualICE := iceListener.Addr().String()
	closeDiscovery := func() {}
	if signalingAddress == iceAddress {
		var discoveryClose func()
		discoveryClose, err = startDirectDiscovery(ctx, clientAccess.Identity.DeviceID, clientAccess.Identity.Fingerprint, signalingAddress, actualSignaling)
		if discoveryClose != nil {
			closeDiscovery = discoveryClose
		}
		if err != nil {
			logger.Warn("Direct LAN discovery is unavailable", "error", err)
		}
	}
	v3DirectRuntimeAddresses.Lock()
	v3DirectRuntimeAddresses.signaling = actualSignaling
	v3DirectRuntimeAddresses.ice = actualICE
	v3DirectRuntimeAddresses.Unlock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		if serveErr := server.Serve(ctx); serveErr != nil && ctx.Err() == nil {
			logger.Error("Direct WebRTC server stopped", "error", serveErr)
		}
	}()
	logger.Info("Direct WebRTC server listening", "signaling", actualSignaling, "ice_tcp", actualICE)
	return func() {
		closeDiscovery()
		_ = server.Close()
		<-done
		v3DirectRuntimeAddresses.Lock()
		if v3DirectRuntimeAddresses.signaling == actualSignaling && v3DirectRuntimeAddresses.ice == actualICE {
			v3DirectRuntimeAddresses.signaling = ""
			v3DirectRuntimeAddresses.ice = ""
		}
		v3DirectRuntimeAddresses.Unlock()
	}, nil
}

// v3DirectPairingRoute 返回 pair create 签名进 bootstrap 的唯一 Direct Route Proto。
// 显式地址用于 FRP/TCP mapping 并完全替代 LAN seed；默认 wildcard listener 只投影为可预览的 RFC1918 locator。
func v3DirectPairingRoute(options v3DirectPairingRouteOptions) (*remoteauthpb.EndpointRouteConfigV1, error) {
	signalingAddress, iceAddress := v3DirectAddresses()
	if value := strings.TrimSpace(options.DefaultListen); value != "" {
		signalingAddress, iceAddress = value, value
	}
	directOverrides, err := normalizeDirectPairingAddresses(options.DirectAddresses)
	if err != nil {
		return nil, fmt.Errorf("Direct address override: %w", err)
	}
	signalingOverrides, err := normalizeDirectPairingAddresses(options.SignalingAddresses)
	if err != nil {
		return nil, fmt.Errorf("Direct signaling address override: %w", err)
	}
	iceOverrides, err := normalizeDirectPairingAddresses(options.ICETCPAddresses)
	if err != nil {
		return nil, fmt.Errorf("Direct ICE-TCP address override: %w", err)
	}
	if len(signalingOverrides) == 0 != (len(iceOverrides) == 0) {
		return nil, fmt.Errorf("Direct signaling and ICE-TCP overrides must be provided together")
	}
	if len(directOverrides) > 0 && (len(signalingOverrides) > 0 || len(iceOverrides) > 0) {
		return nil, fmt.Errorf("Direct address cannot be combined with separate signaling or ICE-TCP overrides")
	}
	if len(directOverrides) > 0 {
		signalingOverrides = append([]string(nil), directOverrides...)
		iceOverrides = append([]string(nil), directOverrides...)
	} else if len(signalingOverrides) == 0 {
		signalingOverrides, iceOverrides, err = directListenerSeeds(signalingAddress, iceAddress)
		if err != nil {
			return nil, err
		}
	}
	advertised := append(append([]string(nil), signalingOverrides...), iceOverrides...)
	advertised = uniqueSortedStrings(advertised)
	serverName := strings.TrimSpace(options.ServerName)
	if strings.ContainsAny(serverName, "\r\n\t ") {
		return nil, fmt.Errorf("Direct server name must not contain whitespace")
	}
	return &remoteauthpb.EndpointRouteConfigV1{
		SchemaVersion: endpoint.RouteConfigVersion, RouteId: "direct", Enabled: true,
		Route: &remoteauthpb.EndpointRouteConfigV1_DirectWebrtcTcp{DirectWebrtcTcp: &remoteauthpb.DirectWebRTCTCPRouteConfig{
			SignalingAddresses: signalingOverrides, IceTcpAddresses: iceOverrides, AdvertisedAddresses: advertised, ServerName: serverName,
		}},
	}, nil
}

func directListenerSeeds(signalingAddress, iceAddress string) ([]string, []string, error) {
	signalingHost, signalingPort, err := net.SplitHostPort(signalingAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("parse Direct signaling listener %q: %w", signalingAddress, err)
	}
	iceHost, icePort, err := net.SplitHostPort(iceAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("parse Direct ICE-TCP listener %q: %w", iceAddress, err)
	}
	if !isWildcardHost(signalingHost) || !isWildcardHost(iceHost) {
		signaling, err := normalizeDirectPairingAddresses([]string{signalingAddress})
		if err != nil {
			return nil, nil, err
		}
		ice, err := normalizeDirectPairingAddresses([]string{iceAddress})
		return signaling, ice, err
	}
	hosts, err := v3PrivateLANAddresses()
	if err != nil {
		return nil, nil, err
	}
	if len(hosts) == 0 {
		return nil, nil, fmt.Errorf("no private LAN address is available; provide explicit Direct signaling and ICE-TCP addresses")
	}
	hosts = uniqueSortedStrings(hosts)
	signaling := make([]string, 0, len(hosts))
	ice := make([]string, 0, len(hosts))
	for _, host := range hosts {
		signaling = append(signaling, net.JoinHostPort(host, signalingPort))
		ice = append(ice, net.JoinHostPort(host, icePort))
	}
	return signaling, ice, nil
}

func normalizeDirectPairingAddresses(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("address must not be empty")
		}
		host, port, err := net.SplitHostPort(value)
		portNumber, portErr := strconv.ParseUint(strings.TrimSpace(port), 10, 16)
		if err != nil || portErr != nil || portNumber == 0 || strings.TrimSpace(host) == "" || isWildcardHost(host) {
			return nil, fmt.Errorf("address %q must be a reachable HOST:PORT", value)
		}
		result = append(result, net.JoinHostPort(strings.TrimSpace(host), strings.TrimSpace(port)))
	}
	return uniqueSortedStrings(result), nil
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func isWildcardHost(host string) bool {
	return host == "" || host == "0.0.0.0" || host == "::"
}

func v3DirectAddresses() (string, string) {
	v3DirectRuntimeAddresses.RLock()
	activeSignaling := v3DirectRuntimeAddresses.signaling
	activeICE := v3DirectRuntimeAddresses.ice
	v3DirectRuntimeAddresses.RUnlock()
	if activeSignaling != "" && activeICE != "" {
		return activeSignaling, activeICE
	}
	shared := strings.TrimSpace(os.Getenv("ANYTTY_DIRECT_LISTEN"))
	signaling := strings.TrimSpace(os.Getenv("ANYTTY_DIRECT_SIGNALING_LISTEN"))
	if signaling == "" {
		signaling = shared
	}
	if signaling == "" {
		signaling = defaultDirectSignalingAddress
	}
	ice := strings.TrimSpace(os.Getenv("ANYTTY_DIRECT_ICE_TCP_LISTEN"))
	if ice == "" {
		if shared != "" {
			ice = shared
		} else {
			ice = signaling
		}
	}
	return signaling, ice
}
