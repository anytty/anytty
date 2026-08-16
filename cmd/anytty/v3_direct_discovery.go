package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/proto/wire"
	"github.com/grandcat/zeroconf"
)

const (
	anyTTYDirectServiceType             = "_anytty._tcp"
	anyTTYDirectDomain                  = "local."
	anyTTYDirectDiscoveryTTL            = 30
	directDiscoveryDefaultRefreshPeriod = 2 * time.Second
)

var directDiscoveryRefreshPeriod = directDiscoveryDefaultRefreshPeriod

type directDiscoveryServer interface {
	Shutdown()
}

var registerDirectDiscovery = func(instance string, port int, text []string, addresses []string) (directDiscoveryServer, error) {
	// RegisterProxy appends the selected domain to the hostname.
	server, err := zeroconf.RegisterProxy(instance, anyTTYDirectServiceType, anyTTYDirectDomain, port, strings.ToLower(instance), addresses, text, nil)
	if err != nil {
		return nil, err
	}
	server.TTL(anyTTYDirectDiscoveryTTL)
	return server, nil
}

// startDirectDiscovery advertises only wildcard listeners. Explicit listener
// addresses are user-managed routes and must not silently broaden LAN exposure.
func startDirectDiscovery(ctx context.Context, deviceID, fingerprint, configuredAddress, actualAddress string) (func(), error) {
	host, _, err := net.SplitHostPort(configuredAddress)
	if err != nil {
		return nil, fmt.Errorf("parse Direct discovery listener %q: %w", configuredAddress, err)
	}
	if !isWildcardHost(host) {
		return func() {}, nil
	}
	_, portValue, err := net.SplitHostPort(actualAddress)
	if err != nil {
		return nil, fmt.Errorf("parse Direct discovery address %q: %w", actualAddress, err)
	}
	port, err := strconv.Atoi(portValue)
	if err != nil || port == 0 {
		return nil, fmt.Errorf("parse Direct discovery port %q", portValue)
	}
	discoveryKey := directDiscoveryKey(deviceID, fingerprint)
	instance := "AnyTTY-" + discoveryKey[:12]
	text := []string{
		"v=1", fmt.Sprintf("p=%d", wire.Version), "k=" + discoveryKey,
	}
	addresses, addressErr := v3PrivateLANAddresses()
	addresses = uniqueSortedStrings(addresses)
	var server directDiscoveryServer
	var initialErr error
	if addressErr != nil {
		initialErr = fmt.Errorf("enumerate Direct LAN addresses: %w", addressErr)
	} else if len(addresses) > 0 {
		server, err = registerDirectDiscovery(instance, port, text, addresses)
		if err != nil {
			initialErr = fmt.Errorf("publish Direct DNS-SD service: %w", err)
		}
	}
	discoveryContext, cancel := context.WithCancel(ctx)
	var mutex sync.Mutex
	current := server
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(directDiscoveryRefreshPeriod)
		defer ticker.Stop()
		last := ""
		if server != nil {
			last = strings.Join(addresses, "\x00")
		}
		for {
			select {
			case <-discoveryContext.Done():
				return
			case <-ticker.C:
				updated, listErr := v3PrivateLANAddresses()
				if listErr != nil {
					continue
				}
				updated = uniqueSortedStrings(updated)
				key := strings.Join(updated, "\x00")
				mutex.Lock()
				unchanged := current != nil && key == last
				mutex.Unlock()
				if unchanged {
					continue
				}
				mutex.Lock()
				previous := current
				current = nil
				last = ""
				mutex.Unlock()
				if previous != nil {
					previous.Shutdown()
				}
				if len(updated) == 0 {
					continue
				}
				next, registerErr := registerDirectDiscovery(instance, port, text, updated)
				if registerErr != nil {
					continue
				}
				mutex.Lock()
				current = next
				last = key
				mutex.Unlock()
			}
		}
	}()
	var closeOnce sync.Once
	return func() {
		closeOnce.Do(func() {
			cancel()
			<-done
			mutex.Lock()
			if current != nil {
				current.Shutdown()
			}
			mutex.Unlock()
		})
	}, initialErr
}

func directDiscoveryKey(deviceID, fingerprint string) string {
	digest := sha256.Sum256([]byte("anytty-lan-discovery-v1\x00" + strings.TrimSpace(deviceID) + "\x00" + strings.TrimSpace(fingerprint)))
	return hex.EncodeToString(digest[:])
}
