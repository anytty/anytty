// Package sdk implements the public daemon-routed Protobuf plugin client.
package sdk

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

type Executor func(context.Context, *apipb.PluginCommand) (*apipb.PluginResult, error)

type Error struct{ Code, Message string }

func (e *Error) Error() string { return e.Code + ": " + e.Message }

type Client struct {
	execute         Executor
	endpointFactory func(string) Executor
	mu              sync.RWMutex
	registration    *apipb.PluginRegistration
	closeTransport  func() error
}

func NewClient(execute Executor) *Client { return &Client{execute: execute} }

// ForEndpoint shares transport but owns a separate daemon registration.
func (c *Client) ForEndpoint(endpointID string) *Client {
	if c.endpointFactory == nil {
		return NewClient(func(context.Context, *apipb.PluginCommand) (*apipb.PluginResult, error) {
			return nil, errors.New("transport does not support endpoint selection")
		})
	}
	child := NewClient(c.endpointFactory(endpointID))
	child.endpointFactory = c.endpointFactory
	return child
}

// Execute always reaches the selected daemon, including self-addressed calls.
func (c *Client) Execute(ctx context.Context, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
	if c == nil || c.execute == nil || command == nil {
		return nil, errors.New("plugin executor and command required")
	}
	result, err := c.execute(ctx, proto.Clone(command).(*apipb.PluginCommand))
	if err != nil {
		return nil, err
	}
	if result == nil || result.GetResult() == nil {
		return nil, errors.New("missing plugin result")
	}
	if failure := result.GetError(); failure != nil {
		return nil, &Error{Code: failure.Code, Message: failure.Message}
	}
	return result, nil
}

func (c *Client) Register(ctx context.Context, request *apipb.PluginRegisterRequest) (*apipb.PluginRegistration, error) {
	if request == nil {
		return nil, errors.New("registration required")
	}
	result, err := c.Execute(ctx, &apipb.PluginCommand{Command: &apipb.PluginCommand_Register{Register: request}})
	if err != nil {
		return nil, err
	}
	reg := result.GetRegistration()
	if reg == nil || len(reg.SourceLease) == 0 || reg.Address == nil {
		return nil, errors.New("invalid daemon registration")
	}
	c.mu.Lock()
	c.registration = proto.Clone(reg).(*apipb.PluginRegistration)
	c.mu.Unlock()
	return proto.Clone(reg).(*apipb.PluginRegistration), nil
}

func (c *Client) Registration() *apipb.PluginRegistration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.registration == nil {
		return nil
	}
	return proto.Clone(c.registration).(*apipb.PluginRegistration)
}
func (c *Client) lease() ([]byte, error) {
	reg := c.Registration()
	if reg == nil {
		return nil, errors.New("plugin is not registered")
	}
	return reg.SourceLease, nil
}

// Send acknowledges daemon delivery only. Business completion is the matching
// PluginReply consumed from Receive; delivery is never an execution result.
func (c *Client) Send(ctx context.Context, message *apipb.PluginMessage) error {
	lease, err := c.lease()
	if err != nil {
		return err
	}
	if message == nil {
		return errors.New("message required")
	}
	result, err := c.Execute(ctx, &apipb.PluginCommand{Command: &apipb.PluginCommand_Send{Send: &apipb.PluginSendRequest{SourceLease: lease, Message: message}}})
	if err == nil && result.GetAck() == nil {
		return errors.New("missing daemon delivery acknowledgement")
	}
	return err
}
func (c *Client) Receive(ctx context.Context, wait time.Duration) (*apipb.PluginBatch, error) {
	lease, err := c.lease()
	if err != nil {
		return nil, err
	}
	if wait < 0 || wait > 30*time.Second {
		return nil, fmt.Errorf("receive wait must be between zero and 30 seconds")
	}
	result, err := c.Execute(ctx, &apipb.PluginCommand{Command: &apipb.PluginCommand_Receive{Receive: &apipb.PluginReceiveRequest{SourceLease: lease, WaitMillis: uint32(wait / time.Millisecond), MaxMessages: 64}}})
	if err != nil {
		return nil, err
	}
	if result.GetBatch() == nil {
		return nil, errors.New("missing message batch")
	}
	return result.GetBatch(), nil
}
func (c *Client) State(ctx context.Context, request *apipb.PluginStateRequest) (*apipb.PluginStateSnapshot, error) {
	lease, err := c.lease()
	if err != nil {
		return nil, err
	}
	if request == nil {
		return nil, errors.New("state request required")
	}
	request = proto.Clone(request).(*apipb.PluginStateRequest)
	request.SourceLease = lease
	result, err := c.Execute(ctx, &apipb.PluginCommand{Command: &apipb.PluginCommand_State{State: request}})
	if err != nil {
		return nil, err
	}
	if result.GetState() == nil {
		return nil, errors.New("missing state snapshot")
	}
	return result.GetState(), nil
}
func (c *Client) Unregister(ctx context.Context) error {
	lease, err := c.lease()
	if err != nil {
		return nil
	}
	_, err = c.Execute(ctx, &apipb.PluginCommand{Command: &apipb.PluginCommand_Unregister{Unregister: &apipb.PluginUnregisterRequest{SourceLease: lease}}})
	c.mu.Lock()
	c.registration = nil
	c.mu.Unlock()
	return err
}
func (c *Client) Close() error {
	if c.closeTransport != nil {
		return c.closeTransport()
	}
	return nil
}
