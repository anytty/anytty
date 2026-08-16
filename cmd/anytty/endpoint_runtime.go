package main

import (
	"fmt"
	"strings"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
)

type resolvedTerminalRef struct {
	EndpointID endpointdomain.EndpointID
	TerminalID string
}

func (ref resolvedTerminalRef) String() string { return string(ref.EndpointID) + ":" + ref.TerminalID }

func loadNormalizedConnectionRegistry() (endpointdomain.Registry, error) {
	registry, err := endpointdomain.Load("")
	if err != nil {
		return endpointdomain.Registry{}, &cliError{code: 2, message: err.Error(), cause: err}
	}
	registry, err = registry.Normalize()
	if err != nil {
		return endpointdomain.Registry{}, &cliError{code: 2, message: err.Error(), cause: err}
	}
	return registry, nil
}

func resolveTerminalRef(target, requestedEndpoint string, registry endpointdomain.Registry) (resolvedTerminalRef, error) {
	target = strings.TrimSpace(target)
	requestedEndpoint = strings.TrimSpace(requestedEndpoint)
	if target == "" {
		return resolvedTerminalRef{}, usageCLIError("terminal target cannot be empty")
	}
	endpointID := endpointdomain.EndpointID(requestedEndpoint)
	terminalID := target
	if endpoint, id, found := strings.Cut(target, ":"); found {
		if endpoint == "" {
			return resolvedTerminalRef{}, usageCLIError("terminal target must be ENDPOINT_ID:TERMINAL_ID")
		}
		if requestedEndpoint != "" && endpoint != requestedEndpoint {
			return resolvedTerminalRef{}, usageCLIError("TARGET endpoint conflicts with --endpoint")
		}
		endpointID = endpointdomain.EndpointID(endpoint)
		terminalID = id
	}
	if endpointID == "" {
		endpointID = registry.Default
	}
	if terminalID == "" || strings.Contains(terminalID, ":") {
		return resolvedTerminalRef{}, usageCLIError("terminal target must be ENDPOINT_ID:TERMINAL_ID")
	}
	cfg, ok := registry.Endpoints[endpointID]
	if !ok {
		return resolvedTerminalRef{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", endpointID)}
	}
	if !cfg.Enabled {
		return resolvedTerminalRef{}, &cliError{code: 4, message: fmt.Sprintf("endpoint %s is disabled", endpointID)}
	}
	return resolvedTerminalRef{EndpointID: endpointID, TerminalID: terminalID}, nil
}

func resolveEndpointConfig(requested string, registry endpointdomain.Registry) (endpointdomain.Endpoint, error) {
	id := endpointdomain.EndpointID(strings.TrimSpace(requested))
	if id == "" {
		id = registry.Default
	}
	cfg, ok := registry.Endpoints[id]
	if !ok {
		return endpointdomain.Endpoint{}, &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", id)}
	}
	if !cfg.Enabled {
		return endpointdomain.Endpoint{}, &cliError{code: 4, message: fmt.Sprintf("endpoint %s is disabled", id)}
	}
	return cfg, nil
}
