package runtime

import (
	"errors"
	"fmt"
	"strings"

	"github.com/anytty/anytty/client/endpoint"
)

// NewAllRoutesUnavailableError formats the failures from a route race in the
// same order as the attempted routes while preserving each underlying cause.
func NewAllRoutesUnavailableError(attempts []AttemptRequest, failures []error) error {
	details := make([]string, 0, len(attempts))
	causes := make([]error, 0, len(failures))
	for index, attempt := range attempts {
		if index >= len(failures) || failures[index] == nil {
			continue
		}
		reason := strings.TrimRight(routeFailureReason(failures[index]), ".")
		details = append(details, fmt.Sprintf("%s: %s.", routeLabel(attempt.Route()), reason))
		causes = append(causes, failures[index])
	}
	message := "All configured routes are unavailable."
	if len(details) > 0 {
		message += " " + strings.Join(details, " ")
	}
	return &Error{
		Code:      ErrorUnavailable,
		Message:   message,
		Cause:     errors.Join(causes...),
		Attempted: true,
		Retryable: true,
	}
}

func routeLabel(route endpoint.AccessRoute) string {
	if name := strings.TrimSpace(route.DisplayName); name != "" {
		return name
	}
	switch route.Kind {
	case endpoint.RouteDirectWebRTCTCP:
		return "Direct"
	case endpoint.RouteManagedWebRTC:
		return "Cloud"
	case endpoint.RouteSSHWebRTCTCP:
		return "SSH"
	default:
		return string(route.Kind)
	}
}

func routeFailureReason(err error) string {
	if err == nil {
		return "unavailable"
	}
	var runtimeErr *Error
	if errors.As(err, &runtimeErr) && strings.TrimSpace(runtimeErr.Message) != "" {
		return strings.TrimSpace(runtimeErr.Message)
	}
	if reason := strings.Join(strings.Fields(err.Error()), " "); reason != "" {
		return reason
	}
	return "unavailable"
}
