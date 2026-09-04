package apimapping

import (
	"net"
	"strings"

	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/apipb"
)

// ValidateBrowserProxyCommand validates a daemon-side TCP target without doing
// DNS locally or accepting URL syntax in a field that is only a host.
func ValidateBrowserProxyCommand(command *apipb.CommandEnvelope) error {
	if err := ValidateRequestContext(RequestContextForCommand(command)); err != nil {
		return err
	}
	value, ok := command.GetCommand().(*apipb.CommandEnvelope_BrowserProxyOpen)
	if !ok || value.BrowserProxyOpen == nil {
		return validation("browser_proxy_open", "is required")
	}
	host := strings.TrimSpace(value.BrowserProxyOpen.GetHost())
	if host == "" {
		return validation("browser_proxy_open.host", "is required")
	}
	if len(host) > 253 || strings.ContainsAny(host, "\r\n\t /\\@?#") {
		return validation("browser_proxy_open.host", "must be a host name or IP address")
	}
	if strings.HasPrefix(host, "[") || strings.HasSuffix(host, "]") {
		if len(host) < 2 || host[0] != '[' || host[len(host)-1] != ']' {
			return validation("browser_proxy_open.host", "has invalid IPv6 brackets")
		}
		host = host[1 : len(host)-1]
	}
	if net.ParseIP(host) == nil {
		for _, label := range strings.Split(host, ".") {
			if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return validation("browser_proxy_open.host", "must be a host name or IP address")
			}
			for _, char := range label {
				if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
					return validation("browser_proxy_open.host", "must be a host name or IP address")
				}
			}
		}
	}
	if value.BrowserProxyOpen.GetPort() == 0 || value.BrowserProxyOpen.GetPort() > 65535 {
		return validation("browser_proxy_open.port", "must be between 1 and 65535")
	}
	return nil
}

// BrowserProxyToProto creates the public resource handle while keeping the
// daemon connection token opaque to the client API.
func BrowserProxyToProto(origin *apipb.EndpointSessionStamp, proxy corev2.BrowserProxy) *apipb.BrowserProxyOpenResult {
	return &apipb.BrowserProxyOpenResult{Resource: &apipb.ResourceHandle{
		OpaqueToken: cloneBytes(proxy.Token),
		Kind:        apipb.ResourceKind_RESOURCE_KIND_BROWSER_PROXY,
		Session:     cloneSessionStamp(origin),
		Generation:  1,
	}}
}
