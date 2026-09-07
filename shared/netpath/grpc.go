package netpath

import (
	"context"
	"fmt"
	"net"

	"github.com/anytty/anytty/shared/connecttrace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type authenticatedConn struct {
	net.Conn
	auth credentials.AuthInfo
}
type racedCredentials struct {
	credentials.TransportCredentials
}

func NewGRPCClient(address string, creds credentials.TransportCredentials) (*grpc.ClientConn, error) {
	return grpc.NewClient("passthrough:///"+address, GRPCOptions(address, creds)...)
}

func (creds racedCredentials) ClientHandshake(_ context.Context, _ string, conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	verified, ok := conn.(*authenticatedConn)
	if !ok {
		return nil, nil, fmt.Errorf("network race returned an unauthenticated transport")
	}
	return verified.Conn, verified.auth, nil
}

func (creds racedCredentials) Clone() credentials.TransportCredentials {
	return racedCredentials{creds.TransportCredentials.Clone()}
}

// GRPCOptions preserves TLS verification and ALPN while racing network-specific
// DNS + TCP + TLS. passthrough keeps gRPC from resolving on the default network.
func GRPCOptions(address string, creds credentials.TransportCredentials, traceID ...string) []grpc.DialOption {
	authority := creds.Info().ServerName
	if authority == "" {
		authority = address
	}
	return []grpc.DialOption{
		grpc.WithNoProxy(),
		grpc.WithTransportCredentials(racedCredentials{creds}),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			if len(traceID) > 0 {
				ctx = connecttrace.Attach(ctx, traceID[0])
			}
			return Default.Connect(ctx, "tcp", address, func(ctx context.Context, raw net.Conn) (net.Conn, error) {
				conn, auth, err := creds.Clone().ClientHandshake(ctx, authority, raw)
				if err != nil {
					return nil, err
				}
				return &authenticatedConn{Conn: conn, auth: auth}, nil
			})
		}),
	}
}
