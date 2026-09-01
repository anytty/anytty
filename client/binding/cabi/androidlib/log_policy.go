//go:build cgo

package main

import (
	"bytes"
	"strings"

	clientruntime "github.com/anytty/anytty/client/runtime"
)

var (
	cloudTimingPrefix        = []byte("anytty cloud connect ")
	cloudFailurePrefix       = []byte("anytty cloud failure ")
	cloudPresencePrefix      = []byte("anytty cloud presence ")
	cloudSessionPrefix       = []byte("anytty cloud session ")
	directTimingPrefix       = []byte("anytty direct connect ")
	directFailurePrefix      = []byte("anytty direct failure ")
	webRTCDiagnosticPrefix   = []byte("anytty webrtc ")
	endpointSupervisorPrefix = []byte("anytty endpoint_supervisor ")
)

func androidDiagnosticAllowed(payload []byte) bool {
	if bytes.HasPrefix(payload, cloudSessionPrefix) {
		return len(sanitizeCloudSessionDiagnostic(bytes.TrimSpace(payload))) != 0
	}
	return bytes.HasPrefix(payload, cloudTimingPrefix) ||
		bytes.HasPrefix(payload, cloudFailurePrefix) ||
		bytes.HasPrefix(payload, cloudPresencePrefix) ||
		bytes.HasPrefix(payload, directTimingPrefix) ||
		bytes.HasPrefix(payload, directFailurePrefix) ||
		bytes.HasPrefix(payload, webRTCDiagnosticPrefix) ||
		bytes.HasPrefix(payload, endpointSupervisorPrefix)
}

func sanitizeAndroidDiagnostic(payload []byte) []byte {
	trimmed := bytes.TrimSpace(payload)
	if bytes.HasPrefix(payload, cloudSessionPrefix) {
		return sanitizeCloudSessionDiagnostic(trimmed)
	}
	if !bytes.HasPrefix(payload, endpointSupervisorPrefix) {
		return bytes.Join(bytes.Fields(trimmed), []byte(" "))
	}
	if index := bytes.Index(trimmed, []byte(" invalidate_error=")); index >= 0 {
		trimmed = append(bytes.Clone(trimmed[:index]), []byte(" invalidate_failed=true")...)
	}
	fields := bytes.Fields(trimmed)
	filtered := fields[:0]
	for _, field := range fields {
		if bytes.HasPrefix(field, []byte("endpoint=")) {
			continue
		}
		filtered = append(filtered, field)
	}
	return bytes.Join(filtered, []byte(" "))
}

func sanitizeCloudSessionDiagnostic(payload []byte) []byte {
	fields := bytes.Fields(payload)
	if len(fields) < 9 || !bytes.Equal(fields[0], []byte("anytty")) || !bytes.Equal(fields[1], []byte("cloud")) || !bytes.Equal(fields[2], []byte("session")) {
		return nil
	}
	values := make(map[string]string, 6)
	for _, field := range fields[3:] {
		key, value, found := strings.Cut(string(field), "=")
		if !found || !cloudSessionDiagnosticKey(key) {
			continue
		}
		if _, duplicate := values[key]; duplicate || !validCloudSessionDiagnosticValue(key, value) {
			return nil
		}
		values[key] = value
	}
	keys := []string{"stage", "origin", "cause", "code", "rpc_code", "retryable"}
	canonical := make([]string, 0, len(keys)+3)
	canonical = append(canonical, "anytty", "cloud", "session")
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			return nil
		}
		canonical = append(canonical, key+"="+value)
	}
	return []byte(strings.Join(canonical, " "))
}

func cloudSessionDiagnosticKey(key string) bool {
	switch key {
	case "stage", "origin", "cause", "code", "rpc_code", "retryable":
		return true
	default:
		return false
	}
}

func validCloudSessionDiagnosticValue(key, value string) bool {
	switch key {
	case "stage":
		return value == "closed"
	case "origin":
		return value == "application" || value == "signaling" || value == "local"
	case "cause":
		switch value {
		case "local_close", "ended", "admin_disconnect", "eof", "context_canceled", "deadline_exceeded", "grpc_error", "error":
			return true
		default:
			return false
		}
	case "code":
		return validCloudSessionRuntimeCode(value)
	case "rpc_code":
		return validCloudSessionRPCCode(value)
	case "retryable":
		return value == "true" || value == "false"
	default:
		return false
	}
}

func validCloudSessionRuntimeCode(value string) bool {
	switch clientruntime.ErrorCode(value) {
	case clientruntime.ErrorCode("none"),
		clientruntime.ErrorInvalidRequest,
		clientruntime.ErrorUnsupportedRoute,
		clientruntime.ErrorIdentity,
		clientruntime.ErrorAuthorization,
		clientruntime.ErrorNotFound,
		clientruntime.ErrorUnavailable,
		clientruntime.ErrorCanceled,
		clientruntime.ErrorUserStopped,
		clientruntime.ErrorStaleSession,
		clientruntime.ErrorStaleResource,
		clientruntime.ErrorResourceExhausted,
		clientruntime.ErrorEntitlement,
		clientruntime.ErrorDaemonBlocked,
		clientruntime.ErrorDaemonDeleted,
		clientruntime.ErrorRelayNotInPlan,
		clientruntime.ErrorRelayQuotaExhausted,
		clientruntime.ErrorRelayConcurrencyExhausted,
		clientruntime.ErrorSubscriptionInactive,
		clientruntime.ErrorRelayRegionUnavailable:
		return true
	default:
		return false
	}
}

func validCloudSessionRPCCode(value string) bool {
	switch value {
	case "none", "OK", "Canceled", "Unknown", "InvalidArgument", "DeadlineExceeded", "NotFound", "AlreadyExists",
		"PermissionDenied", "ResourceExhausted", "FailedPrecondition", "Aborted", "OutOfRange", "Unimplemented",
		"Internal", "Unavailable", "DataLoss", "Unauthenticated":
		return true
	default:
		return false
	}
}
