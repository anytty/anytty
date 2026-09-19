package gateway

import (
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// ListenerSpec is one parsed --listen value. Supported forms:
//
//	tcp:127.0.0.1:7331   (port 0 picks a free port)
//	tcp:[::1]:7331
//	unix:/tmp/anytty-access.sock
//	127.0.0.1:7331       (bare address, treated as tcp)
type ListenerSpec struct {
	Network string
	Address string
}

// ParseListenerSpec parses one --listen value.
func ParseListenerSpec(value string) (ListenerSpec, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ListenerSpec{}, errors.New("gateway: listener must not be empty")
	}
	if rest, ok := strings.CutPrefix(value, "unix:"); ok {
		spec := ListenerSpec{Network: "unix", Address: strings.TrimSpace(rest)}
		return spec, spec.Validate()
	}
	if rest, ok := strings.CutPrefix(value, "tcp:"); ok {
		spec := ListenerSpec{Network: "tcp", Address: strings.TrimSpace(rest)}
		return spec, spec.Validate()
	}
	spec := ListenerSpec{Network: "tcp", Address: value}
	return spec, spec.Validate()
}

// Validate rejects incomplete or unsupported specs before any bind.
func (spec ListenerSpec) Validate() error {
	switch spec.Network {
	case "unix":
		if strings.TrimSpace(spec.Address) == "" {
			return errors.New("gateway: unix listener requires a socket path")
		}
	case "tcp":
		if strings.TrimSpace(spec.Address) == "" {
			return errors.New("gateway: tcp listener requires HOST:PORT")
		}
		if _, _, err := net.SplitHostPort(spec.Address); err != nil {
			return fmt.Errorf("gateway: tcp listener %q must be HOST:PORT", spec.Address)
		}
	default:
		return fmt.Errorf("gateway: unsupported listener network %q", spec.Network)
	}
	return nil
}

// String renders the spec back to the --listen form.
func (spec ListenerSpec) String() string {
	return spec.Network + ":" + spec.Address
}

// Listen binds the listener. Unix sockets are chmod 0600 and a stale socket
// file is replaced only after confirming no live listener answers on it.
func (spec ListenerSpec) Listen() (net.Listener, error) {
	switch spec.Network {
	case "unix":
		if err := prepareUnixSocket(spec.Address); err != nil {
			return nil, err
		}
		listener, err := net.Listen("unix", spec.Address)
		if err != nil {
			return nil, fmt.Errorf("gateway: listen unix %q: %w", spec.Address, err)
		}
		if err := os.Chmod(spec.Address, 0o600); err != nil {
			_ = listener.Close()
			_ = os.Remove(spec.Address)
			return nil, fmt.Errorf("gateway: secure unix socket %q: %w", spec.Address, err)
		}
		return listener, nil
	case "tcp":
		listener, err := net.Listen("tcp", spec.Address)
		if err != nil {
			return nil, fmt.Errorf("gateway: listen tcp %q: %w", spec.Address, err)
		}
		return listener, nil
	default:
		return nil, fmt.Errorf("gateway: unsupported listener network %q", spec.Network)
	}
}

// prepareUnixSocket refuses to steal a live socket and removes a stale file.
func prepareUnixSocket(path string) error {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("gateway: inspect unix socket %q: %w", path, err)
	}
	conn, err := net.DialTimeout("unix", path, 100*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return fmt.Errorf("gateway: unix socket %q already has an active listener", path)
	}
	if errors.Is(err, syscall.ECONNREFUSED) || (runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(10061))) {
		err = os.Remove(path)
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("gateway: unix socket %q is not dialable and not removable: %w", path, err)
	}
	return nil
}
