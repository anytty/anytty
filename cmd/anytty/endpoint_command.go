package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/spf13/cobra"
)

type endpointCommandRuntime struct {
	registryPath string
	socket       *string
	logFile      *string
}

type endpointRouteView struct {
	ID                     string   `json:"id"`
	Kind                   string   `json:"kind"`
	Enabled                bool     `json:"enabled"`
	ManualOnly             bool     `json:"manual_only"`
	Priority               *int     `json:"priority,omitempty"`
	CredentialRef          string   `json:"credential_ref,omitempty"`
	Socket                 string   `json:"socket,omitempty"`
	Host                   string   `json:"host,omitempty"`
	Port                   uint16   `json:"port,omitempty"`
	User                   string   `json:"user,omitempty"`
	ProxyJump              string   `json:"proxy_jump,omitempty"`
	HostKeyFingerprints    []string `json:"host_key_fingerprints,omitempty"`
	RemoteSignalingAddress string   `json:"remote_signaling_address,omitempty"`
	RemoteICETCPAddress    string   `json:"remote_ice_tcp_address,omitempty"`
	SignalingAddresses     []string `json:"signaling_addresses,omitempty"`
	ICETCPAddresses        []string `json:"ice_tcp_addresses,omitempty"`
	AdvertisedAddresses    []string `json:"advertised_addresses,omitempty"`
	ServerName             string   `json:"server_name,omitempty"`
	TargetDeviceID         string   `json:"target_device_id,omitempty"`
	AccountProfileRef      string   `json:"account_profile_ref,omitempty"`
	RelayMode              string   `json:"relay_mode,omitempty"`
	RelayTransport         string   `json:"relay_transport,omitempty"`
}

type endpointView struct {
	ID                string              `json:"id"`
	Label             string              `json:"label"`
	DeviceID          string              `json:"device_id,omitempty"`
	DeviceFingerprint string              `json:"device_fingerprint,omitempty"`
	Enabled           bool                `json:"enabled"`
	Default           bool                `json:"default"`
	ConnectMode       string              `json:"connect_mode"`
	HedgeDelay        string              `json:"hedge_delay,omitempty"`
	Routes            []endpointRouteView `json:"routes"`
}

type endpointTestView struct {
	SchemaVersion        int     `json:"schema_version"`
	Kind                 string  `json:"kind"`
	ID                   string  `json:"id"`
	RouteID              string  `json:"route_id"`
	RouteKind            string  `json:"route_kind"`
	State                string  `json:"state"`
	ObservedPath         string  `json:"observed_path,omitempty"`
	RouteSelectionReason string  `json:"route_selection_reason,omitempty"`
	SnapshotAvailable    bool    `json:"snapshot_available"`
	SampledAt            string  `json:"sampled_at,omitempty"`
	RoundTripMillis      float64 `json:"round_trip_ms,omitempty"`
	LocalIP              string  `json:"local_ip,omitempty"`
	RemoteIP             string  `json:"remote_ip,omitempty"`
	LocalPort            uint16  `json:"local_port,omitempty"`
	RemotePort           uint16  `json:"remote_port,omitempty"`
	LocalCandidateType   string  `json:"local_candidate_type,omitempty"`
	RemoteCandidateType  string  `json:"remote_candidate_type,omitempty"`
	LocalProtocol        string  `json:"local_protocol,omitempty"`
	RemoteProtocol       string  `json:"remote_protocol,omitempty"`
	RelayTransport       string  `json:"relay_transport,omitempty"`
	NetworkClass         string  `json:"network_class,omitempty"`
	BytesSent            uint64  `json:"bytes_sent,omitempty"`
	BytesReceived        uint64  `json:"bytes_received,omitempty"`
	PacketsSent          uint64  `json:"packets_sent,omitempty"`
	LossEvents           uint64  `json:"loss_events,omitempty"`
	Connected            bool    `json:"connected"`
}

type endpointPolicyView struct {
	SchemaVersion  int    `json:"schema_version"`
	Kind           string `json:"kind"`
	EndpointID     string `json:"endpoint_id"`
	Route          string `json:"route"`
	CloudPath      string `json:"cloud_path"`
	RelayTransport string `json:"relay_transport"`
}

func newEndpointCommand(socket, logFile *string) *cobra.Command {
	runtime := &endpointCommandRuntime{socket: socket, logFile: logFile}
	command := &cobra.Command{Use: "endpoint", Short: "Manage daemon endpoints and their routes"}
	command.PersistentFlags().StringVar(&runtime.registryPath, "registry", "", "endpoint registry path (default: XDG config dir endpoints.yaml)")
	command.AddCommand(
		newEndpointListCommand(runtime),
		newEndpointShowCommand(runtime),
		newEndpointAddCommand(runtime),
		newEndpointUpdateCommand(runtime),
		newEndpointRemoveCommand(runtime),
		newEndpointToggleCommand(runtime, true),
		newEndpointToggleCommand(runtime, false),
		newEndpointSetDefaultCommand(runtime),
		newEndpointRouteCommand(runtime),
		newEndpointPolicyCommand(runtime),
		newEndpointTestCommand(runtime),
		newEndpointShareCommand(runtime),
	)
	return command
}

func (runtime *endpointCommandRuntime) load() (endpointdomain.Registry, error) {
	registry, err := endpointdomain.Load(runtime.registryPath)
	if err != nil {
		return endpointdomain.Registry{}, &cliError{code: 2, message: err.Error(), cause: err}
	}
	registry, err = registry.Normalize()
	if err != nil {
		return endpointdomain.Registry{}, &cliError{code: 2, message: err.Error(), cause: err}
	}
	return registry, nil
}

func (runtime *endpointCommandRuntime) update(ctx context.Context, createIfMissing bool, mutate func(endpointdomain.Registry) (endpointdomain.Registry, error)) error {
	_, err := endpointdomain.UpdateContext(ctx, runtime.registryPath, createIfMissing, mutate)
	if err == nil {
		return nil
	}
	var existing *cliError
	if errors.As(err, &existing) {
		return err
	}
	return &cliError{code: 2, message: err.Error(), cause: err}
}

func newEndpointListCommand(runtime *endpointCommandRuntime) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "list", Short: "List configured endpoints without dialing them", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			registry, err := runtime.load()
			if err != nil {
				return err
			}
			views := endpointViews(registry)
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
					SchemaVersion int            `json:"schema_version"`
					Kind          string         `json:"kind"`
					Items         []endpointView `json:"items"`
				}{2, "endpoint_list", views})
			}
			rows := make([][]string, 0, len(views))
			for _, view := range views {
				kinds := make([]string, 0, len(view.Routes))
				for _, route := range view.Routes {
					kinds = append(kinds, route.ID+":"+route.Kind)
				}
				status := "disabled"
				if view.Enabled {
					status = "enabled"
				}
				isDefault := "-"
				if view.Default {
					isDefault = "yes"
				}
				rows = append(rows, []string{view.ID, view.Label, status, isDefault, strings.Join(kinds, ", ")})
			}
			return writeCLITable(cmd.OutOrStdout(), []string{"ID", "LABEL", "STATUS", "DEFAULT", "ROUTES"}, rows)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newEndpointShowCommand(runtime *endpointCommandRuntime) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "show ID", Short: "Show one configured endpoint and all routes", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, err := runtime.load()
			if err != nil {
				return err
			}
			endpoint, ok := registry.Endpoints[endpointdomain.EndpointID(args[0])]
			if !ok {
				return &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", args[0])}
			}
			view := endpointConfigView(endpoint, registry.Default == endpoint.ID)
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
					SchemaVersion int          `json:"schema_version"`
					Kind          string       `json:"kind"`
					Item          endpointView `json:"item"`
				}{2, "endpoint", view})
			}
			fields := []cliField{
				{Label: "ID", Value: view.ID},
				{Label: "Label", Value: view.Label},
				{Label: "Status", Value: map[bool]string{true: "enabled", false: "disabled"}[view.Enabled]},
				{Label: "Default", Value: map[bool]string{true: "yes", false: "no"}[view.Default]},
				{Label: "Connect mode", Value: view.ConnectMode},
			}
			if view.DeviceFingerprint != "" {
				fields = append(fields, cliField{Label: "Device", Value: view.DeviceID}, cliField{Label: "Fingerprint", Value: view.DeviceFingerprint})
			}
			if err := writeCLIFields(cmd.OutOrStdout(), fields...); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "\nRoutes")
			routeRows := make([][]string, 0, len(view.Routes))
			for _, route := range view.Routes {
				status := "disabled"
				if route.Enabled {
					status = "enabled"
				}
				mode := "automatic"
				if route.ManualOnly {
					mode = "manual"
				}
				priority := "full race"
				if route.Priority != nil {
					priority = strconv.Itoa(*route.Priority)
				}
				relay := strings.Trim(strings.Join([]string{route.RelayMode, route.RelayTransport}, "/"), "/")
				routeRows = append(routeRows, []string{route.ID, route.Kind, status, mode, priority, relay})
			}
			return writeCLITable(cmd.OutOrStdout(), []string{"ID", "KIND", "STATUS", "MODE", "PRIORITY", "RELAY"}, routeRows)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

type endpointEditFlags struct {
	label, connectMode, deviceID, deviceFingerprint string
	hedgeDelay                                      time.Duration
}

type routeEditFlags struct {
	routeID, credentialRef, socket, host, user, proxyJump                    string
	remoteSignalingAddress, remoteICETCPAddress                              string
	serverName, targetDeviceID, accountProfileRef, relayMode, relayTransport string
	port                                                                     uint16
	priority                                                                 int
	manualOnly                                                               bool
	hostKeyFingerprints, signalingAddresses, iceTCPAddresses                 []string
	advertisedAddresses                                                      []string
}

func newEndpointAddCommand(runtime *endpointCommandRuntime) *cobra.Command {
	command := &cobra.Command{Use: "add", Short: "Add an endpoint with its first route"}
	for _, kind := range []endpointdomain.RouteKind{endpointdomain.RouteLocalUnix, endpointdomain.RouteSSHWebRTCTCP, endpointdomain.RouteDirectWebRTCTCP, endpointdomain.RouteManagedWebRTC} {
		endpointFlags := &endpointEditFlags{}
		routeFlags := &routeEditFlags{}
		name := routeCommandName(kind)
		child := &cobra.Command{
			Use: name + " ID", Short: "Add an endpoint with a " + string(kind) + " route", Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				id := endpointdomain.EndpointID(strings.TrimSpace(args[0]))
				route := routeFromFlags(kind, *routeFlags)
				endpoint := endpointFromFlags(id, *endpointFlags, route)
				if cmd.Flags().Changed("hedge-delay") {
					endpoint.SelectionPolicy = endpointdomain.SelectionPolicy{HedgeDelay: endpointFlags.hedgeDelay, HedgeDelayConfigured: true}
				}
				if _, err := (endpointdomain.Registry{Version: endpointdomain.RegistryVersion, Default: id, Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{id: endpoint}}).Normalize(); err != nil {
					return usageCLIError(err.Error())
				}
				if err := runtime.update(cmd.Context(), true, func(registry endpointdomain.Registry) (endpointdomain.Registry, error) {
					if _, exists := registry.Endpoints[id]; exists {
						return endpointdomain.Registry{}, &cliError{code: 4, message: fmt.Sprintf("endpoint %s already exists", id)}
					}
					if registry.Endpoints == nil {
						registry.Endpoints = map[endpointdomain.EndpointID]endpointdomain.Endpoint{}
					}
					registry.Version = endpointdomain.RegistryVersion
					registry.Endpoints[id] = endpoint
					if registry.Default == "" {
						registry.Default = id
					}
					return registry, nil
				}); err != nil {
					return err
				}
				return writeCLIFields(cmd.OutOrStdout(),
					cliField{Label: "Endpoint", Value: string(id)},
					cliField{Label: "Route", Value: string(route.ID)},
					cliField{Label: "Status", Value: "added"},
				)
			},
		}
		bindEndpointPolicyFlags(child, endpointFlags)
		bindEndpointIdentityFlags(child, endpointFlags)
		bindRouteEditFlags(child, routeFlags, kind, true)
		command.AddCommand(child)
	}
	return command
}

func newEndpointUpdateCommand(runtime *endpointCommandRuntime) *cobra.Command {
	flags := &endpointEditFlags{}
	command := &cobra.Command{
		Use: "update ID", Short: "Update endpoint identity, label, or selection policy", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := endpointdomain.EndpointID(args[0])
			if err := runtime.update(cmd.Context(), false, func(registry endpointdomain.Registry) (endpointdomain.Registry, error) {
				endpoint, ok := registry.Endpoints[id]
				if !ok {
					return endpointdomain.Registry{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", id)}
				}
				if cmd.Flags().Changed("label") {
					endpoint.Label, endpoint.LabelSource = strings.TrimSpace(flags.label), endpointdomain.SourceUser
				}
				if cmd.Flags().Changed("connect-mode") {
					endpoint.ConnectMode = endpointdomain.ConnectMode(flags.connectMode)
				}
				if cmd.Flags().Changed("hedge-delay") {
					endpoint.SelectionPolicy.HedgeDelay = flags.hedgeDelay
					endpoint.SelectionPolicy.HedgeDelayConfigured = true
				}
				registry.Endpoints[id] = endpoint
				return registry, nil
			}); err != nil {
				return err
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Endpoint", Value: string(id)},
				cliField{Label: "Status", Value: "updated"},
			)
		},
	}
	bindEndpointPolicyFlags(command, flags)
	return command
}

func newEndpointRemoveCommand(runtime *endpointCommandRuntime) *cobra.Command {
	return &cobra.Command{
		Use: "remove ID", Short: "Remove an endpoint from the registry", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := endpointdomain.EndpointID(args[0])
			if err := runtime.update(cmd.Context(), false, func(registry endpointdomain.Registry) (endpointdomain.Registry, error) {
				if _, ok := registry.Endpoints[id]; !ok {
					return endpointdomain.Registry{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", id)}
				}
				if len(registry.Endpoints) == 1 {
					return endpointdomain.Registry{}, &cliError{code: 4, message: "cannot remove the last endpoint"}
				}
				delete(registry.Endpoints, id)
				if registry.Default == id {
					if err := reassignEndpointDefault(&registry); err != nil {
						return endpointdomain.Registry{}, err
					}
				}
				return registry, nil
			}); err != nil {
				return err
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Endpoint", Value: string(id)},
				cliField{Label: "Status", Value: "removed"},
			)
		},
	}
}

func newEndpointToggleCommand(runtime *endpointCommandRuntime, enabled bool) *cobra.Command {
	verb := "disable"
	if enabled {
		verb = "enable"
	}
	return &cobra.Command{
		Use: verb + " ID", Short: endpointVerbTitle(verb) + " an endpoint", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := endpointdomain.EndpointID(args[0])
			if err := runtime.update(cmd.Context(), false, func(registry endpointdomain.Registry) (endpointdomain.Registry, error) {
				if _, ok := registry.Endpoints[id]; !ok {
					return endpointdomain.Registry{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", id)}
				}
				updated, err := endpointdomain.SetEndpointEnabled(registry, id, enabled)
				if err != nil {
					return endpointdomain.Registry{}, &cliError{code: 4, message: err.Error(), cause: err}
				}
				return updated, nil
			}); err != nil {
				return err
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Endpoint", Value: string(id)},
				cliField{Label: "Status", Value: verb + "d"},
			)
		},
	}
}

func reassignEndpointDefault(registry *endpointdomain.Registry) error {
	var firstEnabled endpointdomain.EndpointID
	for _, endpoint := range registry.List() {
		if !endpoint.Enabled {
			continue
		}
		if firstEnabled == "" {
			firstEnabled = endpoint.ID
		}
		for _, route := range endpoint.RouteList() {
			if route.Enabled && route.Kind == endpointdomain.RouteLocalUnix {
				registry.Default = endpoint.ID
				return nil
			}
		}
	}
	if firstEnabled == "" {
		return &cliError{code: 4, message: "endpoint registry requires at least one enabled default endpoint"}
	}
	registry.Default = firstEnabled
	return nil
}

func endpointVerbTitle(verb string) string {
	if verb == "" {
		return ""
	}
	return strings.ToUpper(verb[:1]) + verb[1:]
}

func newEndpointSetDefaultCommand(runtime *endpointCommandRuntime) *cobra.Command {
	return &cobra.Command{
		Use: "set-default ID", Short: "Set the default endpoint", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := endpointdomain.EndpointID(args[0])
			if err := runtime.update(cmd.Context(), false, func(registry endpointdomain.Registry) (endpointdomain.Registry, error) {
				endpoint, ok := registry.Endpoints[id]
				if !ok {
					return endpointdomain.Registry{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", id)}
				}
				if !endpoint.Enabled {
					return endpointdomain.Registry{}, &cliError{code: 4, message: fmt.Sprintf("endpoint %s is disabled", id)}
				}
				registry.Default = id
				return registry, nil
			}); err != nil {
				return err
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Endpoint", Value: string(id)},
				cliField{Label: "Default", Value: "yes"},
			)
		},
	}
}

func newEndpointRouteCommand(runtime *endpointCommandRuntime) *cobra.Command {
	command := &cobra.Command{Use: "route", Short: "Manage routes within an endpoint"}
	command.AddCommand(newEndpointRouteAddCommand(runtime), newEndpointRouteUpdateCommand(runtime), newEndpointRouteRemoveCommand(runtime), newEndpointRouteToggleCommand(runtime, true), newEndpointRouteToggleCommand(runtime, false))
	return command
}

func newEndpointRouteAddCommand(runtime *endpointCommandRuntime) *cobra.Command {
	command := &cobra.Command{Use: "add", Short: "Add a route to an existing endpoint"}
	for _, kind := range []endpointdomain.RouteKind{endpointdomain.RouteLocalUnix, endpointdomain.RouteSSHWebRTCTCP, endpointdomain.RouteDirectWebRTCTCP, endpointdomain.RouteManagedWebRTC} {
		flags := &routeEditFlags{}
		child := &cobra.Command{
			Use: routeCommandName(kind) + " ENDPOINT_ID ROUTE_ID", Args: cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				endpointID, routeID := endpointdomain.EndpointID(args[0]), endpointdomain.RouteID(args[1])
				flags.routeID = string(routeID)
				route := routeFromFlags(kind, *flags)
				if err := runtime.update(cmd.Context(), false, func(registry endpointdomain.Registry) (endpointdomain.Registry, error) {
					endpoint, ok := registry.Endpoints[endpointID]
					if !ok {
						return endpointdomain.Registry{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", endpointID)}
					}
					if _, exists := endpoint.Routes[routeID]; exists {
						return endpointdomain.Registry{}, &cliError{code: 4, message: fmt.Sprintf("endpoint %s route %s already exists", endpointID, routeID)}
					}
					endpoint.Routes[routeID] = route
					registry.Endpoints[endpointID] = endpoint
					return registry, nil
				}); err != nil {
					return err
				}
				return writeCLIFields(cmd.OutOrStdout(),
					cliField{Label: "Endpoint", Value: string(endpointID)},
					cliField{Label: "Route", Value: string(routeID)},
					cliField{Label: "Status", Value: "added"},
				)
			},
		}
		bindRouteEditFlags(child, flags, kind, false)
		command.AddCommand(child)
	}
	return command
}

func newEndpointRouteUpdateCommand(runtime *endpointCommandRuntime) *cobra.Command {
	flags := &routeEditFlags{}
	command := &cobra.Command{
		Use: "update ENDPOINT_ID ROUTE_ID", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			endpointID, routeID := endpointdomain.EndpointID(args[0]), endpointdomain.RouteID(args[1])
			if err := runtime.update(cmd.Context(), false, func(registry endpointdomain.Registry) (endpointdomain.Registry, error) {
				endpoint, ok := registry.Endpoints[endpointID]
				if !ok {
					return endpointdomain.Registry{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", endpointID)}
				}
				route, ok := endpoint.Routes[routeID]
				if !ok {
					return endpointdomain.Registry{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s route %s was not found", endpointID, routeID)}
				}
				applyRouteFlagChanges(cmd, &route, *flags)
				route.PolicySource = endpointdomain.SourceUser
				endpoint.Routes[routeID] = route
				registry.Endpoints[endpointID] = endpoint
				return registry, nil
			}); err != nil {
				return err
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Endpoint", Value: string(endpointID)},
				cliField{Label: "Route", Value: string(routeID)},
				cliField{Label: "Status", Value: "updated"},
			)
		},
	}
	bindRouteEditFlags(command, flags, "", false)
	return command
}

func newEndpointRouteRemoveCommand(runtime *endpointCommandRuntime) *cobra.Command {
	return &cobra.Command{
		Use: "remove ENDPOINT_ID ROUTE_ID", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			endpointID, routeID := endpointdomain.EndpointID(args[0]), endpointdomain.RouteID(args[1])
			if err := runtime.update(cmd.Context(), false, func(registry endpointdomain.Registry) (endpointdomain.Registry, error) {
				endpoint, ok := registry.Endpoints[endpointID]
				if !ok {
					return endpointdomain.Registry{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", endpointID)}
				}
				if _, ok := endpoint.Routes[routeID]; !ok {
					return endpointdomain.Registry{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s route %s was not found", endpointID, routeID)}
				}
				if len(endpoint.Routes) == 1 {
					return endpointdomain.Registry{}, &cliError{code: 4, message: "cannot remove the last route from an endpoint"}
				}
				delete(endpoint.Routes, routeID)
				registry.Endpoints[endpointID] = endpoint
				return registry, nil
			}); err != nil {
				return err
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Endpoint", Value: string(endpointID)},
				cliField{Label: "Route", Value: string(routeID)},
				cliField{Label: "Status", Value: "removed"},
			)
		},
	}
}

func newEndpointRouteToggleCommand(runtime *endpointCommandRuntime, enabled bool) *cobra.Command {
	verb := "disable"
	if enabled {
		verb = "enable"
	}
	return &cobra.Command{
		Use: verb + " ENDPOINT_ID ROUTE_ID", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			endpointID, routeID := endpointdomain.EndpointID(args[0]), endpointdomain.RouteID(args[1])
			if err := runtime.update(cmd.Context(), false, func(registry endpointdomain.Registry) (endpointdomain.Registry, error) {
				endpoint, ok := registry.Endpoints[endpointID]
				if !ok {
					return endpointdomain.Registry{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", endpointID)}
				}
				route, ok := endpoint.Routes[routeID]
				if !ok {
					return endpointdomain.Registry{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s route %s was not found", endpointID, routeID)}
				}
				route.Enabled, route.PolicySource = enabled, endpointdomain.SourceUser
				endpoint.Routes[routeID] = route
				registry.Endpoints[endpointID] = endpoint
				return registry, nil
			}); err != nil {
				return err
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Endpoint", Value: string(endpointID)},
				cliField{Label: "Route", Value: string(routeID)},
				cliField{Label: "Status", Value: verb + "d"},
			)
		},
	}
}

func newEndpointTestCommand(runtime *endpointCommandRuntime) *cobra.Command {
	var jsonOutput bool
	var routeValue string
	command := &cobra.Command{
		Use: "test ID", Short: "Dial one route and verify anytty protocol reachability", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, err := runtime.load()
			if err != nil {
				return err
			}
			id := endpointdomain.EndpointID(args[0])
			endpoint, ok := registry.Endpoints[id]
			if !ok {
				return &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", id)}
			}
			if !endpoint.Enabled {
				return &cliError{code: 4, message: fmt.Sprintf("endpoint %s is disabled", id)}
			}
			cmd.Root().SilenceUsage = true
			routeID, observedPath, selectionReason, snapshot, snapshotAvailable, closeClient, err := probeEndpointProtocolClient(cmd.Context(), endpoint, endpointdomain.RouteID(routeValue), *runtime.socket, *runtime.logFile)
			if err != nil {
				return classifyCLIError(err)
			}
			defer closeClient()
			route, _ := endpoint.Route(routeID)
			view := endpointTestViewFromSnapshot(id, route, observedPath, selectionReason, snapshot, snapshotAvailable)
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(view)
			}
			return printEndpointTestView(cmd, view)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	command.Flags().StringVar(&routeValue, "route", "", "explicit route ID (required when multiple routes are eligible before CONN003)")
	return command
}

func endpointTestViewFromSnapshot(id endpointdomain.EndpointID, route endpointdomain.AccessRoute, observedPath, selectionReason string, snapshot clientruntime.ConnectionSnapshot, valid bool) endpointTestView {
	view := endpointTestView{
		SchemaVersion: 3, Kind: "endpoint_test", ID: string(id), RouteID: string(route.ID), RouteKind: string(route.Kind),
		State: "reachable", ObservedPath: observedPath, RouteSelectionReason: selectionReason, SnapshotAvailable: valid,
	}
	if !valid {
		return view
	}
	view.SampledAt = snapshot.SampledAt.UTC().Format(time.RFC3339Nano)
	view.RoundTripMillis = float64(snapshot.RoundTrip) / float64(time.Millisecond)
	view.LocalIP, view.RemoteIP, view.LocalPort, view.RemotePort = snapshot.LocalAddress, snapshot.RemoteAddress, snapshot.LocalPort, snapshot.RemotePort
	view.LocalCandidateType, view.RemoteCandidateType = snapshot.LocalCandidateType, snapshot.RemoteCandidateType
	view.LocalProtocol, view.RemoteProtocol, view.RelayTransport = snapshot.LocalProtocol, snapshot.RemoteProtocol, snapshot.RelayTransport
	view.NetworkClass = snapshot.NetworkClass
	view.BytesSent, view.BytesReceived, view.PacketsSent, view.LossEvents, view.Connected = snapshot.BytesSent, snapshot.BytesReceived, snapshot.PacketsSent, snapshot.LossEvents, snapshot.Connected
	if snapshot.ObservedPath != "" {
		view.ObservedPath = snapshot.ObservedPath
	}
	return view
}

func printEndpointTestView(cmd *cobra.Command, view endpointTestView) error {
	fields := []cliField{
		{Label: "Endpoint", Value: view.ID},
		{Label: "State", Value: view.State},
		{Label: "Route", Value: fmt.Sprintf("%s (%s)", view.RouteID, view.RouteKind)},
	}
	if view.ObservedPath != "" {
		fields = append(fields, cliField{Label: "Path", Value: view.ObservedPath})
	}
	if !view.SnapshotAvailable {
		fields = append(fields, cliField{Label: "Network snapshot", Value: "unavailable"})
		return writeCLIFields(cmd.OutOrStdout(), fields...)
	}
	fields = append(fields,
		cliField{Label: "Local", Value: formatCLIEndpoint(view.LocalIP, view.LocalPort)},
		cliField{Label: "Remote", Value: formatCLIEndpoint(view.RemoteIP, view.RemotePort)},
	)
	if view.RoundTripMillis > 0 {
		fields = append(fields, cliField{Label: "RTT", Value: fmt.Sprintf("%.1f ms", view.RoundTripMillis)})
	}
	fields = append(fields,
		cliField{Label: "Candidates", Value: fmt.Sprintf("%s / %s", emptyCLIValue(view.LocalCandidateType), emptyCLIValue(view.RemoteCandidateType))},
		cliField{Label: "ICE transport", Value: fmt.Sprintf("%s / %s", emptyCLIValue(view.LocalProtocol), emptyCLIValue(view.RemoteProtocol))},
	)
	if view.RelayTransport != "" {
		fields = append(fields, cliField{Label: "Relay transport", Value: view.RelayTransport})
	}
	if view.NetworkClass != "" {
		fields = append(fields, cliField{Label: "Network", Value: view.NetworkClass})
	}
	fields = append(fields, cliField{Label: "Traffic", Value: fmt.Sprintf("%d sent / %d received bytes", view.BytesSent, view.BytesReceived)})
	return writeCLIFields(cmd.OutOrStdout(), fields...)
}

func formatCLIEndpoint(address string, port uint16) string {
	address = strings.TrimSpace(address)
	if address == "" {
		return "-"
	}
	if port == 0 {
		return address
	}
	if strings.Contains(address, ":") {
		return fmt.Sprintf("[%s]:%d", address, port)
	}
	return fmt.Sprintf("%s:%d", address, port)
}

func emptyCLIValue(value string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return "-"
}

func newEndpointPolicyCommand(runtime *endpointCommandRuntime) *cobra.Command {
	command := &cobra.Command{Use: "policy", Short: "Show or change the connection policy used by the next session"}
	command.AddCommand(newEndpointPolicyShowCommand(runtime), newEndpointPolicySetCommand(runtime))
	return command
}

func newEndpointPolicyShowCommand(runtime *endpointCommandRuntime) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "show ID", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		registry, err := runtime.load()
		if err != nil {
			return err
		}
		target, ok := registry.Endpoints[endpointdomain.EndpointID(args[0])]
		if !ok {
			return &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", args[0])}
		}
		return writeEndpointPolicy(cmd, endpointPolicyViewFromEndpoint(target), jsonOutput)
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newEndpointPolicySetCommand(runtime *endpointCommandRuntime) *cobra.Command {
	var route, cloudPath, relayTransport string
	var jsonOutput bool
	command := &cobra.Command{Use: "set ID", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("route") && !cmd.Flags().Changed("cloud-path") && !cmd.Flags().Changed("relay-transport") {
			return usageCLIError("endpoint policy set requires at least one policy flag")
		}
		id := endpointdomain.EndpointID(args[0])
		var updated endpointdomain.Endpoint
		err := runtime.update(cmd.Context(), false, func(registry endpointdomain.Registry) (endpointdomain.Registry, error) {
			target, ok := registry.Endpoints[id]
			if !ok {
				return endpointdomain.Registry{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", id)}
			}
			current := endpointPolicyViewFromEndpoint(target)
			if cmd.Flags().Changed("route") {
				current.Route = route
			}
			if cmd.Flags().Changed("cloud-path") {
				current.CloudPath = cloudPath
			}
			if cmd.Flags().Changed("relay-transport") {
				current.RelayTransport = relayTransport
			}
			policy, err := endpointPolicyFromView(current)
			if err != nil {
				return endpointdomain.Registry{}, err
			}
			next, err := endpointdomain.SetConnectionPolicy(registry, id, policy)
			if err == nil {
				updated = next.Endpoints[id]
			}
			return next, err
		})
		if err != nil {
			return err
		}
		return writeEndpointPolicy(cmd, endpointPolicyViewFromEndpoint(updated), jsonOutput)
	}}
	command.Flags().StringVar(&route, "route", "", "auto, direct, ssh, or cloud")
	command.Flags().StringVar(&cloudPath, "cloud-path", "", "auto, p2p, relay, or smart_route")
	command.Flags().StringVar(&relayTransport, "relay-transport", "", "auto, udp, or tcp")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func endpointPolicyViewFromEndpoint(target endpointdomain.Endpoint) endpointPolicyView {
	view := endpointPolicyView{SchemaVersion: 1, Kind: "endpoint_policy", EndpointID: string(target.ID), Route: "auto", CloudPath: "auto", RelayTransport: "auto"}
	switch target.SelectionPolicy.RoutePreference {
	case endpointdomain.RoutePreferenceDirect:
		view.Route = "direct"
	case endpointdomain.RoutePreferenceSSH:
		view.Route = "ssh"
	case endpointdomain.RoutePreferenceManagedCloud:
		view.Route = "cloud"
	}
	for _, item := range target.RouteList() {
		if item.Kind != endpointdomain.RouteManagedWebRTC {
			continue
		}
		switch item.RelayMode {
		case endpointdomain.RelayDirect:
			view.CloudPath = "p2p"
		case endpointdomain.RelayOnly:
			view.CloudPath = "relay"
		case endpointdomain.RelaySmart:
			view.CloudPath = "smart_route"
		}
		if item.RelayTransport != "" {
			view.RelayTransport = string(item.RelayTransport)
		}
		break
	}
	return view
}

func endpointPolicyFromView(view endpointPolicyView) (endpointdomain.ConnectionPolicy, error) {
	policy := endpointdomain.ConnectionPolicy{CloudRelayMode: endpointdomain.RelayAuto, RelayTransport: endpointdomain.RelayTransport(view.RelayTransport)}
	switch view.Route {
	case "auto":
		policy.RoutePreference = endpointdomain.RoutePreferenceAuto
	case "direct":
		policy.RoutePreference = endpointdomain.RoutePreferenceDirect
	case "ssh":
		policy.RoutePreference = endpointdomain.RoutePreferenceSSH
	case "cloud":
		policy.RoutePreference = endpointdomain.RoutePreferenceManagedCloud
	default:
		return endpointdomain.ConnectionPolicy{}, usageCLIError("route must be auto, direct, ssh, or cloud")
	}
	switch view.CloudPath {
	case "auto":
		policy.CloudRelayMode = endpointdomain.RelayAuto
	case "p2p":
		policy.CloudRelayMode = endpointdomain.RelayDirect
	case "relay":
		policy.CloudRelayMode = endpointdomain.RelayOnly
	case "smart_route":
		policy.CloudRelayMode = endpointdomain.RelaySmart
	default:
		return endpointdomain.ConnectionPolicy{}, usageCLIError("cloud-path must be auto, p2p, relay, or smart_route")
	}
	if policy.RelayTransport != endpointdomain.RelayTransportAuto && policy.RelayTransport != endpointdomain.RelayTransportUDP && policy.RelayTransport != endpointdomain.RelayTransportTCP {
		return endpointdomain.ConnectionPolicy{}, usageCLIError("relay-transport must be auto, udp, or tcp")
	}
	return policy, nil
}

func writeEndpointPolicy(cmd *cobra.Command, view endpointPolicyView, jsonOutput bool) error {
	if jsonOutput {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(view)
	}
	return writeCLIFields(cmd.OutOrStdout(),
		cliField{Label: "Endpoint", Value: view.EndpointID},
		cliField{Label: "Route", Value: view.Route},
		cliField{Label: "Cloud path", Value: view.CloudPath},
		cliField{Label: "Relay transport", Value: view.RelayTransport},
	)
}

func bindEndpointPolicyFlags(command *cobra.Command, flags *endpointEditFlags) {
	command.Flags().StringVar(&flags.label, "label", "", "display label")
	command.Flags().StringVar(&flags.connectMode, "connect-mode", string(endpointdomain.ConnectOnDemand), "auto, on_demand, or manual")
	command.Flags().DurationVar(&flags.hedgeDelay, "hedge-delay", 300*time.Millisecond, "delay before starting the next priority group")
}

func bindEndpointIdentityFlags(command *cobra.Command, flags *endpointEditFlags) {
	command.Flags().StringVar(&flags.deviceID, "device-id", "", "daemon directory identity")
	command.Flags().StringVar(&flags.deviceFingerprint, "device-fingerprint", "", "pinned daemon identity fingerprint")
}

func bindRouteEditFlags(command *cobra.Command, flags *routeEditFlags, kind endpointdomain.RouteKind, includeRouteID bool) {
	if includeRouteID {
		command.Flags().StringVar(&flags.routeID, "route", routeCommandName(kind), "route ID")
	}
	command.Flags().BoolVar(&flags.manualOnly, "manual-only", false, "exclude this route from automatic selection")
	command.Flags().IntVar(&flags.priority, "priority", -1, "route priority; lower starts earlier, -1 means unset")
	if kind == "" || kind == endpointdomain.RouteSSHWebRTCTCP || kind == endpointdomain.RouteDirectWebRTCTCP || kind == endpointdomain.RouteManagedWebRTC {
		command.Flags().StringVar(&flags.credentialRef, "credential-ref", "", "local secure credential reference")
	}
	if kind == "" || kind == endpointdomain.RouteLocalUnix {
		command.Flags().StringVar(&flags.socket, "socket", "auto", "local daemon socket")
	}
	if kind == "" || kind == endpointdomain.RouteSSHWebRTCTCP {
		command.Flags().StringVar(&flags.host, "host", "", "OpenSSH host or alias")
		command.Flags().Uint16Var(&flags.port, "port", 22, "SSH port")
		command.Flags().StringVar(&flags.user, "user", "", "SSH user hint")
		command.Flags().StringVar(&flags.proxyJump, "proxy-jump", "", "OpenSSH ProxyJump target")
		command.Flags().StringVar(&flags.remoteSignalingAddress, "remote-signaling-address", "127.0.0.1:41120", "daemon loopback signaling address reached through SSH")
		command.Flags().StringVar(&flags.remoteICETCPAddress, "remote-ice-tcp-address", "127.0.0.1:41121", "daemon loopback ICE-TCP address reached through SSH")
		command.Flags().StringSliceVar(&flags.hostKeyFingerprints, "host-key", nil, "accepted SSH host-key fingerprint")
	}
	if kind == "" || kind == endpointdomain.RouteDirectWebRTCTCP {
		command.Flags().StringSliceVar(&flags.signalingAddresses, "signaling-address", nil, "daemon embedded signaling address (repeatable)")
		command.Flags().StringSliceVar(&flags.iceTCPAddresses, "ice-tcp-address", nil, "daemon ICE-TCP address (repeatable)")
		command.Flags().StringSliceVar(&flags.advertisedAddresses, "advertised-address", nil, "explicit LAN or TCP-mapped address override (repeatable)")
		command.Flags().StringVar(&flags.serverName, "server-name", "", "TLS routing server name; not a trust anchor")
	}
	if kind == "" || kind == endpointdomain.RouteManagedWebRTC {
		command.Flags().StringVar(&flags.targetDeviceID, "target-device-id", "", "managed target device ID")
		command.Flags().StringVar(&flags.accountProfileRef, "account-profile-ref", "", "local Cloud account profile reference")
		command.Flags().StringVar(&flags.relayMode, "relay", string(endpointdomain.RelayAuto), "auto, direct, relay_only, or smart_route")
		command.Flags().StringVar(&flags.relayTransport, "relay-transport", string(endpointdomain.RelayTransportAuto), "auto, udp, or tcp")
	}
}

func endpointFromFlags(id endpointdomain.EndpointID, flags endpointEditFlags, route endpointdomain.AccessRoute) endpointdomain.Endpoint {
	label := strings.TrimSpace(flags.label)
	if label == "" {
		label = string(id)
	}
	endpoint := endpointdomain.Endpoint{
		ID: id, Label: label, LabelSource: endpointdomain.SourceUser, Enabled: true, ConnectMode: endpointdomain.ConnectMode(flags.connectMode),
		DaemonIdentity: endpointdomain.DaemonIdentity{DeviceID: strings.TrimSpace(flags.deviceID), DeviceFingerprint: strings.TrimSpace(flags.deviceFingerprint)},
		Routes:         map[endpointdomain.RouteID]endpointdomain.AccessRoute{route.ID: route},
	}
	if flags.hedgeDelay != 300*time.Millisecond {
		endpoint.SelectionPolicy = endpointdomain.SelectionPolicy{HedgeDelay: flags.hedgeDelay, HedgeDelayConfigured: true}
	}
	return endpoint
}

func routeFromFlags(kind endpointdomain.RouteKind, flags routeEditFlags) endpointdomain.AccessRoute {
	routeID := endpointdomain.RouteID(strings.TrimSpace(flags.routeID))
	if routeID == "" {
		routeID = endpointdomain.RouteID(routeCommandName(kind))
	}
	route := endpointdomain.AccessRoute{
		ID: routeID, Kind: kind, Enabled: true, ManualOnly: flags.manualOnly, CredentialRef: strings.TrimSpace(flags.credentialRef),
		Source: endpointdomain.SourceManual, PolicySource: endpointdomain.SourceUser, Socket: strings.TrimSpace(flags.socket),
		Host: strings.TrimSpace(flags.host), Port: flags.port, User: strings.TrimSpace(flags.user), ProxyJump: strings.TrimSpace(flags.proxyJump),
		HostKeyFingerprints:    append([]string(nil), flags.hostKeyFingerprints...),
		RemoteSignalingAddress: strings.TrimSpace(flags.remoteSignalingAddress), RemoteICETCPAddress: strings.TrimSpace(flags.remoteICETCPAddress),
		SignalingAddresses: append([]string(nil), flags.signalingAddresses...), ICETCPAddresses: append([]string(nil), flags.iceTCPAddresses...),
		AdvertisedAddresses: append([]string(nil), flags.advertisedAddresses...), ServerName: strings.TrimSpace(flags.serverName),
		TargetDeviceID: strings.TrimSpace(flags.targetDeviceID), AccountProfileRef: strings.TrimSpace(flags.accountProfileRef),
		RelayMode: endpointdomain.RelayMode(flags.relayMode), RelayTransport: endpointdomain.RelayTransport(flags.relayTransport),
	}
	if flags.priority >= 0 {
		priority := flags.priority
		route.Priority = &priority
	}
	return route
}

func applyRouteFlagChanges(command *cobra.Command, route *endpointdomain.AccessRoute, flags routeEditFlags) {
	if command.Flags().Changed("manual-only") {
		route.ManualOnly = flags.manualOnly
	}
	if command.Flags().Changed("priority") {
		if flags.priority < 0 {
			route.Priority = nil
		} else {
			priority := flags.priority
			route.Priority = &priority
		}
	}
	if command.Flags().Changed("credential-ref") {
		route.CredentialRef = strings.TrimSpace(flags.credentialRef)
	}
	if command.Flags().Changed("socket") {
		route.Socket = strings.TrimSpace(flags.socket)
	}
	if command.Flags().Changed("host") {
		route.Host = strings.TrimSpace(flags.host)
	}
	if command.Flags().Changed("port") {
		route.Port = flags.port
	}
	if command.Flags().Changed("user") {
		route.User = strings.TrimSpace(flags.user)
	}
	if command.Flags().Changed("proxy-jump") {
		route.ProxyJump = strings.TrimSpace(flags.proxyJump)
	}
	if command.Flags().Changed("remote-signaling-address") {
		route.RemoteSignalingAddress = strings.TrimSpace(flags.remoteSignalingAddress)
	}
	if command.Flags().Changed("remote-ice-tcp-address") {
		route.RemoteICETCPAddress = strings.TrimSpace(flags.remoteICETCPAddress)
	}
	if command.Flags().Changed("host-key") {
		route.HostKeyFingerprints = append([]string(nil), flags.hostKeyFingerprints...)
	}
	if command.Flags().Changed("signaling-address") {
		route.SignalingAddresses = append([]string(nil), flags.signalingAddresses...)
	}
	if command.Flags().Changed("ice-tcp-address") {
		route.ICETCPAddresses = append([]string(nil), flags.iceTCPAddresses...)
	}
	if command.Flags().Changed("advertised-address") {
		route.AdvertisedAddresses = append([]string(nil), flags.advertisedAddresses...)
	}
	if command.Flags().Changed("server-name") {
		route.ServerName = strings.TrimSpace(flags.serverName)
	}
	if command.Flags().Changed("target-device-id") {
		route.TargetDeviceID = strings.TrimSpace(flags.targetDeviceID)
	}
	if command.Flags().Changed("account-profile-ref") {
		route.AccountProfileRef = strings.TrimSpace(flags.accountProfileRef)
	}
	if command.Flags().Changed("relay") {
		route.RelayMode = endpointdomain.RelayMode(flags.relayMode)
	}
	if command.Flags().Changed("relay-transport") {
		route.RelayTransport = endpointdomain.RelayTransport(flags.relayTransport)
	}
}

func routeCommandName(kind endpointdomain.RouteKind) string {
	switch kind {
	case endpointdomain.RouteLocalUnix:
		return "local"
	case endpointdomain.RouteSSHWebRTCTCP:
		return "ssh"
	case endpointdomain.RouteDirectWebRTCTCP:
		return "direct"
	case endpointdomain.RouteManagedWebRTC:
		return "cloud"
	default:
		return string(kind)
	}
}

func endpointViews(registry endpointdomain.Registry) []endpointView {
	views := make([]endpointView, 0, len(registry.Endpoints))
	for _, endpoint := range registry.List() {
		views = append(views, endpointConfigView(endpoint, registry.Default == endpoint.ID))
	}
	return views
}

func endpointConfigView(endpoint endpointdomain.Endpoint, isDefault bool) endpointView {
	view := endpointView{
		ID: string(endpoint.ID), Label: endpoint.Label, DeviceID: endpoint.DaemonIdentity.DeviceID,
		DeviceFingerprint: endpoint.DaemonIdentity.DeviceFingerprint, Enabled: endpoint.Enabled, Default: isDefault,
		ConnectMode: string(endpoint.ConnectMode), Routes: []endpointRouteView{},
	}
	if endpoint.SelectionPolicy.HedgeDelayConfigured {
		view.HedgeDelay = endpoint.SelectionPolicy.HedgeDelay.String()
	}
	for _, route := range endpoint.RouteList() {
		view.Routes = append(view.Routes, endpointRouteView{
			ID: string(route.ID), Kind: string(route.Kind), Enabled: route.Enabled, ManualOnly: route.ManualOnly,
			Priority: cloneCLIInt(route.Priority), CredentialRef: route.CredentialRef, Socket: route.Socket,
			Host: route.Host, Port: route.Port, User: route.User, ProxyJump: route.ProxyJump,
			HostKeyFingerprints: append([]string(nil), route.HostKeyFingerprints...), RemoteSignalingAddress: route.RemoteSignalingAddress, RemoteICETCPAddress: route.RemoteICETCPAddress,
			SignalingAddresses: append([]string(nil), route.SignalingAddresses...), ICETCPAddresses: append([]string(nil), route.ICETCPAddresses...),
			AdvertisedAddresses: append([]string(nil), route.AdvertisedAddresses...), ServerName: route.ServerName,
			TargetDeviceID: route.TargetDeviceID, AccountProfileRef: route.AccountProfileRef,
			RelayMode: string(route.RelayMode), RelayTransport: string(route.RelayTransport),
		})
	}
	return view
}

func cloneCLIInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
