package localweb

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"runtime"
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
}

type Server struct {
	address    string
	httpURL    string
	httpServer *http.Server
	listener   net.Listener
	bridge     *loopback.Server
	registry   *binding.Registry
	done       chan struct{}
	stopOnce   sync.Once
}

type bootstrapResponse struct {
	Bridge  bridgeBootstrap  `json:"bridge"`
	Machine machineBootstrap `json:"machine"`
}

type bridgeBootstrap struct {
	Port  uint16 `json:"port"`
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
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		return nil, fmt.Errorf("listen for local web: %w", err)
	}
	actualAddress := listener.Addr().String()
	httpURL := "http://" + actualAddress

	token, err := randomToken()
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	registry := binding.NewRegistry()
	engineHandle, err := registry.CreateEngine(&bindingHost{core: options.Core})
	if err != nil {
		_ = listener.Close()
		_ = registry.Close()
		return nil, err
	}
	bridge, err := loopback.Start(loopback.RegistryEngine{Registry: registry, Handle: engineHandle}, token, httpURL)
	if err != nil {
		_ = listener.Close()
		_ = registry.Close()
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
		Bridge:  bridgeBootstrap{Port: bridge.Port(), Token: token},
		Machine: machineBootstrap{ID: localEndpointID, Name: machineName, Platform: runtime.GOOS},
	}

	server := &Server{
		address: actualAddress, httpURL: httpURL, listener: listener, bridge: bridge,
		registry: registry, done: make(chan struct{}),
	}
	server.httpServer = &http.Server{
		Handler:           secureHandler(actualAddress, assets, bootstrap),
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

func secureHandler(expectedHost string, assets map[string]webAsset, bootstrap bootstrapResponse) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob:; media-src 'self' data: blob:; worker-src 'self'; child-src 'self'; connect-src 'self' ws://127.0.0.1:*; object-src 'none'; base-uri 'none'; frame-src 'none'; frame-ancestors 'none'; form-action 'none'; manifest-src 'none'")
		writer.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		if request.Host != expectedHost {
			http.Error(writer, "invalid local web host", http.StatusForbidden)
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
