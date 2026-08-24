package localweb

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/client/binding"
	"github.com/anytty/anytty/client/binding/loopback"
)

const DefaultAddress = "127.0.0.1:0"

type Options struct {
	Core        Core
	Address     string
	MachineName string
	Password    []byte
}

type Server struct {
	address    string
	httpURL    string
	httpServer *http.Server
	listener   net.Listener
	bridge     *loopback.Server
	registry   *binding.Registry
	auth       *webAuthenticator
	done       chan struct{}
	stopOnce   sync.Once
}

type bootstrapResponse struct {
	Bridge  bridgeBootstrap  `json:"bridge"`
	Machine machineBootstrap `json:"machine"`
}

type bridgeBootstrap struct {
	Path  string `json:"path"`
	Token string `json:"token"`
}

type machineBootstrap struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

func Start(options Options) (*Server, error) {
	if options.Core == nil {
		return nil, fmt.Errorf("local web core is required")
	}
	address := strings.TrimSpace(options.Address)
	if address == "" {
		address = DefaultAddress
	}
	if err := validateAddress(address); err != nil {
		return nil, err
	}
	assets, err := loadBundledAssets()
	if err != nil {
		return nil, err
	}
	auth, err := newWebAuthenticator(options.Password)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		auth.close()
		return nil, fmt.Errorf("listen for local web: %w", err)
	}
	actualAddress := listener.Addr().String()
	httpURL := "http://" + actualAddress

	token, err := randomToken()
	if err != nil {
		_ = listener.Close()
		auth.close()
		return nil, err
	}
	registry := binding.NewRegistry()
	engineHandle, err := registry.CreateEngine(&bindingHost{core: options.Core})
	if err != nil {
		_ = listener.Close()
		_ = registry.Close()
		auth.close()
		return nil, err
	}
	bridge, err := loopback.Start(loopback.RegistryEngine{Registry: registry, Handle: engineHandle}, token, httpURL)
	if err != nil {
		_ = listener.Close()
		_ = registry.Close()
		auth.close()
		return nil, err
	}
	machineName := strings.TrimSpace(options.MachineName)
	if machineName == "" {
		machineName, _ = os.Hostname()
	}
	if machineName == "" {
		machineName = "This machine"
	}
	bootstrap := bootstrapResponse{
		Bridge:  bridgeBootstrap{Path: "/api/bridge", Token: token},
		Machine: machineBootstrap{ID: localEndpointID, Name: machineName, Platform: runtime.GOOS},
	}
	bridgeProxy := newBridgeProxy(bridge.Port(), httpURL)

	server := &Server{
		address: actualAddress, httpURL: httpURL, listener: listener, bridge: bridge,
		registry: registry, auth: auth, done: make(chan struct{}),
	}
	server.httpServer = &http.Server{
		Handler:           secureHandler(actualAddress, assets, bootstrap, auth, bridgeProxy),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	go func() {
		defer close(server.done)
		_ = server.httpServer.Serve(listener)
	}()
	return server, nil
}

func (server *Server) Address() string { return server.address }
func (server *Server) URL() string     { return server.httpURL }
func (server *Server) PasswordProtected() bool {
	return server != nil && server.auth != nil
}

func (server *Server) Stop(ctx context.Context) error {
	if server == nil {
		return nil
	}
	var stopErr error
	server.stopOnce.Do(func() {
		stopErr = server.httpServer.Shutdown(ctx)
		server.bridge.Stop()
		if err := server.registry.Close(); stopErr == nil && err != nil {
			stopErr = err
		}
		server.auth.close()
		select {
		case <-server.done:
		case <-ctx.Done():
			if stopErr == nil {
				stopErr = ctx.Err()
			}
		}
	})
	return stopErr
}

func validateAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("local web address %q must be HOST:PORT", address)
	}
	if strings.TrimSpace(port) == "" {
		return fmt.Errorf("local web address %q is missing a port", address)
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil || !ip.IsLoopback() || ip.To4() == nil {
		return fmt.Errorf("local web address must use an IPv4 loopback host")
	}
	return nil
}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate local web credential: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func secureHandler(expectedHost string, assets map[string]webAsset, bootstrap bootstrapResponse, auth *webAuthenticator, bridgeProxy http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob:; media-src 'self' data: blob:; worker-src 'self'; child-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-src 'none'; frame-ancestors 'none'; form-action 'none'; manifest-src 'none'")
		writer.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		if request.Host != expectedHost && auth == nil {
			http.Error(writer, "invalid local web host", http.StatusForbidden)
			return
		}
		if request.URL.Path == "/api/auth/login" {
			handleLogin(writer, request, auth)
			return
		}
		if request.URL.Path == "/api/bridge" {
			handleBridge(writer, request, auth, bridgeProxy)
			return
		}
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writer.Header().Set("Allow", "GET, HEAD")
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if request.URL.Path == "/api/bootstrap" {
			writer.Header().Set("Cache-Control", "no-store")
			writer.Header().Set("Content-Type", "application/json; charset=utf-8")
			if auth != nil && !auth.authenticated(request) {
				writeJSONError(writer, http.StatusUnauthorized, "authentication_required")
				return
			}
			if request.Method == http.MethodGet {
				_ = json.NewEncoder(writer).Encode(bootstrap)
			}
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+request.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		asset, ok := assets[name]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", asset.contentType)
		if name == "index.html" {
			writer.Header().Set("Cache-Control", "no-store")
		} else {
			writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(asset.data)))
		if request.Method == http.MethodGet {
			_, _ = writer.Write(asset.data)
		}
	})
}

func newBridgeProxy(port uint16, localOrigin string) http.Handler {
	target := &url.URL{Scheme: "http", Host: net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port)))}
	proxy := httputil.NewSingleHostReverseProxy(target)
	direct := proxy.Director
	proxy.Director = func(request *http.Request) {
		direct(request)
		request.URL.Path = "/"
		request.URL.RawPath = ""
		request.URL.RawQuery = ""
		request.Host = target.Host
		request.Header.Set("Origin", localOrigin)
		request.Header.Del("Cookie")
		request.Header.Del("Authorization")
	}
	proxy.ErrorHandler = func(writer http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(writer, "local Web bridge is unavailable", http.StatusBadGateway)
	}
	return proxy
}

func handleBridge(writer http.ResponseWriter, request *http.Request, auth *webAuthenticator, bridgeProxy http.Handler) {
	writer.Header().Set("Cache-Control", "no-store")
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", "GET")
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if request.URL.RawQuery != "" {
		http.Error(writer, "invalid WebSocket path", http.StatusBadRequest)
		return
	}
	if auth != nil && !auth.authenticated(request) {
		http.Error(writer, "authentication required", http.StatusUnauthorized)
		return
	}
	if !requestOriginMatchesHost(request) {
		http.Error(writer, "invalid WebSocket origin", http.StatusForbidden)
		return
	}
	if bridgeProxy == nil {
		http.Error(writer, "local Web bridge is unavailable", http.StatusServiceUnavailable)
		return
	}
	bridgeProxy.ServeHTTP(writer, request)
}

func handleLogin(writer http.ResponseWriter, request *http.Request, auth *webAuthenticator) {
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	if auth == nil {
		http.NotFound(writer, request)
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", "POST")
		writeJSONError(writer, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if !requestOriginMatchesHost(request) {
		writeJSONError(writer, http.StatusForbidden, "invalid_origin")
		return
	}
	if !requestUsesHTTPS(request) && !originUsesLoopback(request) {
		writeJSONError(writer, http.StatusUpgradeRequired, "https_required")
		return
	}
	if mediaType := strings.ToLower(strings.TrimSpace(strings.Split(request.Header.Get("Content-Type"), ";")[0])); mediaType != "application/json" {
		writeJSONError(writer, http.StatusUnsupportedMediaType, "content_type_required")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, 2*maximumPasswordBytes))
	if err != nil {
		writeJSONError(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	defer clear(body)
	var payload struct {
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || len(payload.Password) > maximumPasswordBytes {
		writeJSONError(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSONError(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	password := []byte(payload.Password)
	payload.Password = ""
	defer clear(password)
	token, retryAfter, matched, err := auth.authenticate(loginClientKey(request), password)
	if err != nil {
		writeJSONError(writer, http.StatusInternalServerError, "authentication_unavailable")
		return
	}
	if retryAfter > 0 {
		seconds := max(1, int((retryAfter+time.Second-1)/time.Second))
		writer.Header().Set("Retry-After", strconv.Itoa(seconds))
		writeJSONError(writer, http.StatusTooManyRequests, "too_many_attempts")
		return
	}
	if !matched {
		writeJSONError(writer, http.StatusUnauthorized, "invalid_password")
		return
	}
	http.SetCookie(writer, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", HttpOnly: true,
		Secure: requestUsesHTTPS(request), SameSite: http.SameSiteStrictMode,
	})
	writer.WriteHeader(http.StatusNoContent)
}

func writeJSONError(writer http.ResponseWriter, status int, code string) {
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{"error": code})
}

func requestOriginMatchesHost(request *http.Request) bool {
	origin, err := url.Parse(strings.TrimSpace(request.Header.Get("Origin")))
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") || origin.Host == "" || origin.User != nil || (origin.Path != "" && origin.Path != "/") || origin.RawQuery != "" || origin.Fragment != "" {
		return false
	}
	if strings.EqualFold(origin.Host, request.Host) {
		return true
	}
	if !hostUsesLoopback(request.Host) {
		return false
	}
	return strings.EqualFold(origin.Host, forwardedRequestHost(request))
}

func forwardedRequestHost(request *http.Request) string {
	forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-Host"), ",")[0])
	return forwarded
}

func requestUsesHTTPS(request *http.Request) bool {
	if request.TLS != nil {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-Proto"), ",")[0]), "https") {
		return true
	}
	origin, err := url.Parse(strings.TrimSpace(request.Header.Get("Origin")))
	return err == nil && origin.Scheme == "https"
}

func originUsesLoopback(request *http.Request) bool {
	origin, err := url.Parse(strings.TrimSpace(request.Header.Get("Origin")))
	return err == nil && hostUsesLoopback(origin.Host)
}

func hostUsesLoopback(value string) bool {
	host := value
	if parsed, _, err := net.SplitHostPort(value); err == nil {
		host = parsed
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}
