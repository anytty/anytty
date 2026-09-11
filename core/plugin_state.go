package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

const pluginStateValueLimit = 64 * 1024

func (r *pluginRouter) stateKey(plugin, collection string) string { return plugin + "/" + collection }
func (r *pluginRouter) statePath(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(r.stateDir, hex.EncodeToString(sum[:])+".pb")
}
func (r *pluginRouter) loadState(key, collection string) (*apipb.PluginStateSnapshot, error) {
	if snapshot := r.states[key]; snapshot != nil {
		return snapshot, nil
	}
	snapshot := &apipb.PluginStateSnapshot{Collection: collection, BootEpoch: r.boot}
	data, err := os.ReadFile(r.statePath(key))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		if len(data) > pluginStateValueLimit+2048 {
			return nil, errors.New("plugin state exceeds size limit")
		}
		if err := proto.Unmarshal(data, snapshot); err != nil {
			return nil, err
		}
		if snapshot.Collection != collection {
			return nil, errors.New("plugin state collection mismatch")
		}
		snapshot.BootEpoch = r.boot
	}
	if len(r.states) >= 1024 {
		return nil, ErrProtocolResourceExhausted
	}
	r.states[key] = snapshot
	return snapshot, nil
}
func (r *pluginRouter) persistState(key string, snapshot *apipb.PluginStateSnapshot) error {
	data, err := proto.Marshal(snapshot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(r.stateDir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(r.stateDir, ".state-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, r.statePath(key))
}
func (r *pluginRouter) state(s *protocolSession, request *apipb.PluginStateRequest) *apipb.PluginResult {
	reg := r.authorized(s, request.GetSourceLease())
	if reg == nil {
		return pluginError("STALE_CONTEXT", "registration is not current")
	}
	if !pluginName(request.GetCollection()) || request.GetOperation() == nil {
		return pluginError("INVALID_REQUEST", "state collection and operation required")
	}
	key := r.stateKey(reg.address.GetPluginId(), request.GetCollection())
	current, err := r.loadState(key, request.GetCollection())
	if err != nil {
		return pluginError("UNAVAILABLE", "cannot load plugin state: "+err.Error())
	}
	switch operation := request.GetOperation().(type) {
	case *apipb.PluginStateRequest_Get:
	case *apipb.PluginStateRequest_Watch:
		if operation.Watch.GetCancel() {
			delete(reg.watches, key)
		} else {
			if len(reg.watches) >= 64 && !reg.watches[key] {
				return pluginError("RESOURCE_EXHAUSTED", "watch capacity reached")
			}
			reg.watches[key] = true
		}
	case *apipb.PluginStateRequest_Put:
		if !reg.service {
			return pluginError("PERMISSION_DENIED", "only daemon plugin service may write state")
		}
		put := operation.Put
		if put.GetExpectedRevision() != current.GetRevision() {
			return pluginError("CONFLICT", "state revision changed")
		}
		if put.GetValue() == nil || !pluginName(put.GetValue().GetSchema()) || put.GetValue().GetVersion() == 0 || proto.Size(put.GetValue()) > pluginStateValueLimit {
			return pluginError("INVALID_REQUEST", "bounded versioned Protobuf value required")
		}
		next := &apipb.PluginStateSnapshot{Collection: current.Collection, Revision: current.Revision + 1, Value: proto.Clone(put.Value).(*apipb.PluginPayload), BootEpoch: r.boot}
		if err := r.persistState(key, next); err != nil {
			return pluginError("UNAVAILABLE", "cannot persist plugin state: "+err.Error())
		}
		r.states[key] = next
		current = next
		for _, watcher := range r.registrations {
			if watcher.watches[key] {
				message := &apipb.PluginMessage{Source: cloneAddress(reg.address), Destination: cloneAddress(watcher.address), Body: &apipb.PluginMessage_StateChanged{StateChanged: proto.Clone(next).(*apipb.PluginStateSnapshot)}}
				if !r.enqueue(watcher, message) {
					delete(watcher.watches, key)
					watcher.resync = true
					r.signal(watcher)
				}
			}
		}
	default:
		return pluginError("UNSUPPORTED", "state operation unsupported")
	}
	return &apipb.PluginResult{Result: &apipb.PluginResult_State{State: proto.Clone(current).(*apipb.PluginStateSnapshot)}}
}
