//go:build cgo

package main

import (
	"context"

	"github.com/anytty/anytty/client/binding/localprobe"
	"github.com/anytty/anytty/proto/bindingpb"
	"google.golang.org/protobuf/proto"
)

func localDiscoveryReachable(payload []byte) (bool, error) {
	result := &bindingpb.LocalDiscoveryLookupResult{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, result); err != nil {
		return false, err
	}
	return localprobe.Reachable(context.Background(), result, localprobe.DefaultDialCandidate), nil
}
