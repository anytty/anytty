package endpoint

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type registryDocument struct {
	Version   int                         `yaml:"version"`
	Default   *string                     `yaml:"default"`
	Endpoints map[string]endpointDocument `yaml:"endpoints"`
}

type endpointDocument struct {
	Label             string                   `yaml:"label"`
	LabelSource       string                   `yaml:"label_source,omitempty"`
	DeviceID          string                   `yaml:"device_id,omitempty"`
	DeviceFingerprint string                   `yaml:"device_fingerprint,omitempty"`
	Enabled           *bool                    `yaml:"enabled,omitempty"`
	ConnectMode       string                   `yaml:"connect_mode,omitempty"`
	Selection         selectionPolicyDocument  `yaml:"selection,omitempty"`
	Routes            map[string]routeDocument `yaml:"routes"`
}

type selectionPolicyDocument struct {
	HedgeDelay      string `yaml:"hedge_delay,omitempty"`
	RoutePreference string `yaml:"route_preference,omitempty"`
}

type routeDocument struct {
	Kind             string `yaml:"kind"`
	DisplayName      string `yaml:"display_name,omitempty"`
	Enabled          *bool  `yaml:"enabled,omitempty"`
	ManualOnly       bool   `yaml:"manual_only,omitempty"`
	Priority         *int   `yaml:"priority,omitempty"`
	CredentialRef    string `yaml:"credential_ref,omitempty"`
	SSHCredentialRef string `yaml:"ssh_credential_ref,omitempty"`
	Source           string `yaml:"source,omitempty"`
	PolicySource     string `yaml:"policy_source,omitempty"`

	Socket string `yaml:"socket,omitempty"`

	Host                   string                        `yaml:"host,omitempty"`
	Port                   uint16                        `yaml:"port,omitempty"`
	User                   string                        `yaml:"user,omitempty"`
	ProxyJump              string                        `yaml:"proxy_jump,omitempty"`
	HostKeyFingerprints    []string                      `yaml:"host_key_fingerprints,omitempty"`
	CredentialDescriptor   *credentialDescriptorDocument `yaml:"credential_descriptor,omitempty"`
	RemoteSignalingAddress string                        `yaml:"remote_signaling_address,omitempty"`
	RemoteICETCPAddress    string                        `yaml:"remote_ice_tcp_address,omitempty"`

	SignalingAddresses  []string `yaml:"signaling_addresses,omitempty"`
	ICETCPAddresses     []string `yaml:"ice_tcp_addresses,omitempty"`
	AdvertisedAddresses []string `yaml:"advertised_addresses,omitempty"`
	ServerName          string   `yaml:"server_name,omitempty"`

	TargetDeviceID    string `yaml:"target_device_id,omitempty"`
	AccountProfileRef string `yaml:"account_profile_ref,omitempty"`
	RelayMode         string `yaml:"relay_mode,omitempty"`
	RelayTransport    string `yaml:"relay_transport,omitempty"`
}

type credentialDescriptorDocument struct {
	DescriptorID string `yaml:"descriptor_id"`
	Kind         string `yaml:"kind"`
	Exportable   bool   `yaml:"exportable,omitempty"`
}

func parseRegistry(data []byte) (Registry, error) {
	if len(data) > MaxRegistryBytes {
		return Registry{}, connectionError(ErrorSizeLimit, "endpoint registry exceeds %d bytes", MaxRegistryBytes)
	}
	if strings.TrimSpace(string(data)) == "" {
		return Registry{}, connectionError(ErrorConfig, "endpoint registry is empty")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var document registryDocument
	if err := decoder.Decode(&document); err != nil {
		return Registry{}, connectionError(ErrorConfig, "decode endpoint registry: %v", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Registry{}, connectionError(ErrorConfig, "endpoint registry must contain exactly one YAML document")
		}
		return Registry{}, connectionError(ErrorConfig, "decode trailing endpoint registry data: %v", err)
	}
	if document.Version == 0 {
		return Registry{}, connectionError(ErrorUnsupportedVersion, "endpoint registry version is required")
	}
	if document.Default == nil || document.Endpoints == nil {
		return Registry{}, connectionError(ErrorConfig, "endpoint registry requires default and endpoints fields")
	}
	if *document.Default != strings.TrimSpace(*document.Default) {
		return Registry{}, connectionError(ErrorConfig, "default endpoint id must be canonical")
	}
	registry := Registry{Version: document.Version, Default: EndpointID(*document.Default), Endpoints: make(map[EndpointID]Endpoint, len(document.Endpoints))}
	for key, value := range document.Endpoints {
		if err := validateIdentifier("endpoint", key); err != nil {
			return Registry{}, err
		}
		id := EndpointID(key)
		enabled := true
		if value.Enabled != nil {
			enabled = *value.Enabled
		}
		hedgeDelay := time.Duration(0)
		hedgeDelayConfigured := false
		if value.Selection.HedgeDelay != "" {
			parsed, err := parseHedgeDelay(value.Selection.HedgeDelay)
			if err != nil {
				return Registry{}, connectionError(ErrorConfig, "endpoint %q hedge_delay: %v", id, err)
			}
			hedgeDelay = parsed
			hedgeDelayConfigured = true
		}
		endpoint := Endpoint{
			ID: id, Label: value.Label, LabelSource: EndpointSource(value.LabelSource),
			DaemonIdentity: DaemonIdentity{DeviceID: value.DeviceID, DeviceFingerprint: value.DeviceFingerprint},
			Enabled:        enabled, ConnectMode: ConnectMode(value.ConnectMode),
			SelectionPolicy: SelectionPolicy{HedgeDelay: hedgeDelay, HedgeDelayConfigured: hedgeDelayConfigured, RoutePreference: RoutePreference(value.Selection.RoutePreference)},
			Routes:          make(map[RouteID]AccessRoute, len(value.Routes)),
		}
		for routeKey, routeValue := range value.Routes {
			if err := validateIdentifier("route", routeKey); err != nil {
				return Registry{}, fmt.Errorf("endpoint %q: %w", id, err)
			}
			routeID := RouteID(routeKey)
			routeEnabled := true
			if routeValue.Enabled != nil {
				routeEnabled = *routeValue.Enabled
			}
			var credentialDescriptor *CredentialDescriptor
			if routeValue.CredentialDescriptor != nil {
				credentialDescriptor = &CredentialDescriptor{
					DescriptorID: routeValue.CredentialDescriptor.DescriptorID,
					Kind:         CredentialKind(routeValue.CredentialDescriptor.Kind),
					Exportable:   routeValue.CredentialDescriptor.Exportable,
				}
			}
			endpoint.Routes[routeID] = AccessRoute{
				ID: routeID, DisplayName: routeValue.DisplayName, Kind: RouteKind(routeValue.Kind), Enabled: routeEnabled, ManualOnly: routeValue.ManualOnly,
				Priority: clonePriority(routeValue.Priority), CredentialRef: routeValue.CredentialRef, SSHCredentialRef: routeValue.SSHCredentialRef,
				Source: EndpointSource(routeValue.Source), PolicySource: EndpointSource(routeValue.PolicySource),
				Socket: routeValue.Socket,
				Host:   routeValue.Host, Port: routeValue.Port, User: routeValue.User, ProxyJump: routeValue.ProxyJump,
				HostKeyFingerprints: append([]string(nil), routeValue.HostKeyFingerprints...), CredentialDescriptor: credentialDescriptor,
				RemoteSignalingAddress: routeValue.RemoteSignalingAddress, RemoteICETCPAddress: routeValue.RemoteICETCPAddress,
				SignalingAddresses: append([]string(nil), routeValue.SignalingAddresses...), ICETCPAddresses: append([]string(nil), routeValue.ICETCPAddresses...),
				AdvertisedAddresses: append([]string(nil), routeValue.AdvertisedAddresses...), ServerName: routeValue.ServerName,
				TargetDeviceID: routeValue.TargetDeviceID, AccountProfileRef: routeValue.AccountProfileRef,
				RelayMode: RelayMode(routeValue.RelayMode), RelayTransport: RelayTransport(routeValue.RelayTransport),
			}
		}
		registry.Endpoints[id] = endpoint
	}
	normalized, err := registry.Normalize()
	if err != nil {
		return Registry{}, fmt.Errorf("normalize endpoint registry: %w", err)
	}
	return normalized, nil
}

func clonePriority(priority *int) *int {
	if priority == nil {
		return nil
	}
	value := *priority
	return &value
}

func parseHedgeDelay(value string) (time.Duration, error) {
	unit := time.Millisecond
	number := strings.TrimSuffix(value, "ms")
	if number == value {
		unit = time.Second
		number = strings.TrimSuffix(value, "s")
	}
	if number == value || number == "" {
		return 0, fmt.Errorf("must use an integer ms or s value")
	}
	amount, err := strconv.ParseUint(number, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("must use an integer ms or s value")
	}
	delay := time.Duration(amount) * unit
	if delay > 30*time.Second {
		return 0, fmt.Errorf("must be between 0 and 30s")
	}
	return delay, nil
}
