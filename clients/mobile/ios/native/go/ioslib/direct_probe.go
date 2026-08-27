//go:build cgo

package main

import (
	"context"

	"github.com/anytty/anytty/client/binding/directprobe"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"google.golang.org/protobuf/proto"
)

func directRouteReachable(payload []byte) (bool, error) {
	route := &remoteauthpb.DirectWebRTCTCPRouteConfig{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, route); err != nil {
		return false, err
	}
	return directprobe.Reachable(context.Background(), route, nil), nil
}
