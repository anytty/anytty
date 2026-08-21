//go:build cgo

// Package main 生成 iOS 使用的 AnyTTY Client Engine C archive。
// 稳定 C ABI 只传递 serialized Proto 与 opaque handle；网络与授权状态机由 Go Client Engine 持有。
package main

/*
#include <stdlib.h>
#include <stddef.h>
#include <stdint.h>

typedef uint64_t anytty_handle_t;
typedef enum anytty_status_v1 {
  ANYTTY_STATUS_OK = 0,
  ANYTTY_STATUS_INVALID_ARGUMENT = 1,
  ANYTTY_STATUS_INVALID_HANDLE = 2,
  ANYTTY_STATUS_CLOSED = 3,
  ANYTTY_STATUS_CAPACITY = 4,
  ANYTTY_STATUS_INTERNAL = 5
} anytty_status_v1;
typedef struct anytty_buffer_v1 {
  anytty_handle_t buffer_handle;
  const uint8_t *data;
  size_t length;
} anytty_buffer_v1;
*/
import "C"

import (
	"context"
	"errors"
	"sync"
	"time"
	"unsafe"

	"github.com/anytty/anytty/client/binding"
	"github.com/anytty/anytty/client/binding/loopback"
)

var iosLibrary = struct {
	sync.Mutex
	registry   *binding.Registry
	hosts      map[uint64]iosHost
	platforms  map[uint64]*binding.PlatformBroker
	bridges    map[uint64]*loopback.Server
	buffers    map[uint64]unsafe.Pointer
	nextBuffer uint64
}{
	registry: binding.NewRegistry(), hosts: make(map[uint64]iosHost),
	platforms: make(map[uint64]*binding.PlatformBroker), bridges: make(map[uint64]*loopback.Server), buffers: make(map[uint64]unsafe.Pointer),
}

type iosHost interface {
	binding.Host
	close() error
}

//export anytty_client_abi_version
func anytty_client_abi_version() C.uint32_t { return C.uint32_t(binding.ABIVersion) }

//export anytty_engine_create
func anytty_engine_create(out *C.anytty_handle_t) C.anytty_status_v1 {
	if out == nil {
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	host, err := newIOSProductionHost()
	if err != nil {
		return status(err)
	}
	return createIOSEngine(host, host.broker, out)
}

//export anytty_bridge_start
func anytty_bridge_start(engine C.anytty_handle_t, tokenData *C.uint8_t, tokenLength C.size_t, outPort *C.uint16_t) C.anytty_status_v1 {
	if tokenData == nil || tokenLength != C.size_t(43) || outPort == nil {
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	handle := uint64(engine)
	token := string(C.GoBytes(unsafe.Pointer(tokenData), C.int(tokenLength)))
	iosLibrary.Lock()
	if iosLibrary.hosts[handle] == nil {
		iosLibrary.Unlock()
		return C.ANYTTY_STATUS_INVALID_HANDLE
	}
	if iosLibrary.bridges[handle] != nil {
		iosLibrary.Unlock()
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	bridge, err := loopback.Start(loopback.RegistryEngine{Registry: iosLibrary.registry, Handle: handle}, token)
	if err == nil {
		iosLibrary.bridges[handle] = bridge
		*outPort = C.uint16_t(bridge.Port())
	}
	iosLibrary.Unlock()
	return status(err)
}

//export anytty_local_probe
func anytty_local_probe(data *C.uint8_t, length C.size_t, outReachable *C.uint8_t) C.anytty_status_v1 {
	if data == nil || length == 0 || uint64(length) > uint64(binding.MaxPayloadBytes) || outReachable == nil {
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	reachable, err := localDiscoveryReachable(C.GoBytes(unsafe.Pointer(data), C.int(length)))
	if err != nil {
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	if reachable {
		*outReachable = 1
	} else {
		*outReachable = 0
	}
	return C.ANYTTY_STATUS_OK
}

//export anytty_supervisor_replace_demand
func anytty_supervisor_replace_demand(engine C.anytty_handle_t, data *C.uint8_t, length C.size_t) C.anytty_status_v1 {
	if data == nil || length == 0 || uint64(length) > uint64(binding.MaxPayloadBytes) {
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	host := iosSupervisorHost(uint64(engine))
	if host == nil {
		return C.ANYTTY_STATUS_INVALID_HANDLE
	}
	return status(binding.ReplaceEndpointSupervisorDemand(host, C.GoBytes(unsafe.Pointer(data), C.int(length))))
}

//export anytty_supervisor_signal
func anytty_supervisor_signal(engine C.anytty_handle_t, data *C.uint8_t, length C.size_t) C.anytty_status_v1 {
	if data == nil || length == 0 || uint64(length) > uint64(binding.MaxPayloadBytes) {
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	host := iosSupervisorHost(uint64(engine))
	if host == nil {
		return C.ANYTTY_STATUS_INVALID_HANDLE
	}
	return status(binding.SignalEndpointSupervisor(host, C.GoBytes(unsafe.Pointer(data), C.int(length))))
}

//export anytty_supervisor_wait_ready
func anytty_supervisor_wait_ready(engine C.anytty_handle_t, timeoutMillis C.uint32_t) C.anytty_status_v1 {
	host := iosSupervisorHost(uint64(engine))
	if host == nil {
		return C.ANYTTY_STATUS_INVALID_HANDLE
	}
	ctx := context.Background()
	cancel := func() {}
	if timeoutMillis > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMillis)*time.Millisecond)
	}
	defer cancel()
	err := host.WaitEndpointDemandReady(ctx)
	if errors.Is(err, context.DeadlineExceeded) {
		return C.ANYTTY_STATUS_OK
	}
	return status(err)
}

//export anytty_supervisor_snapshot
func anytty_supervisor_snapshot(engine C.anytty_handle_t, out *C.anytty_buffer_v1) C.anytty_status_v1 {
	if out == nil {
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	host := iosSupervisorHost(uint64(engine))
	if host == nil {
		return C.ANYTTY_STATUS_INVALID_HANDLE
	}
	payload, err := binding.EncodeEndpointSupervisorSnapshot(host)
	if err != nil {
		return status(err)
	}
	return allocateBuffer(payload, out)
}

func iosSupervisorHost(handle uint64) binding.EndpointSupervisorHost {
	iosLibrary.Lock()
	host, _ := iosLibrary.hosts[handle].(binding.EndpointSupervisorHost)
	iosLibrary.Unlock()
	return host
}

func createIOSEngine(host iosHost, broker *binding.PlatformBroker, out *C.anytty_handle_t) C.anytty_status_v1 {
	handle, err := iosLibrary.registry.CreateEngine(host)
	if err != nil {
		_ = host.close()
		return status(err)
	}
	iosLibrary.Lock()
	iosLibrary.hosts[handle] = host
	if broker != nil {
		iosLibrary.platforms[handle] = broker
	}
	iosLibrary.Unlock()
	*out = C.anytty_handle_t(handle)
	return C.ANYTTY_STATUS_OK
}

//export anytty_engine_open_session
func anytty_engine_open_session(engine C.anytty_handle_t, data *C.uint8_t, length C.size_t, out *C.anytty_handle_t) C.anytty_status_v1 {
	return operation(data, length, out, func(payload []byte) (uint64, error) {
		return iosLibrary.registry.OpenSession(uint64(engine), payload)
	})
}

//export anytty_engine_execute
func anytty_engine_execute(engine, session C.anytty_handle_t, data *C.uint8_t, length C.size_t, out *C.anytty_handle_t) C.anytty_status_v1 {
	return operation(data, length, out, func(payload []byte) (uint64, error) {
		return iosLibrary.registry.Execute(uint64(engine), uint64(session), payload)
	})
}

//export anytty_engine_open_resource_stream
func anytty_engine_open_resource_stream(engine, session C.anytty_handle_t, data *C.uint8_t, length C.size_t, out *C.anytty_handle_t) C.anytty_status_v1 {
	return operation(data, length, out, func(payload []byte) (uint64, error) {
		return iosLibrary.registry.OpenResourceStream(uint64(engine), uint64(session), payload)
	})
}

//export anytty_engine_send_resource_stream_frame
func anytty_engine_send_resource_stream_frame(engine, stream C.anytty_handle_t, data *C.uint8_t, length C.size_t) C.anytty_status_v1 {
	if data == nil || length == 0 || uint64(length) > uint64(binding.MaxPayloadBytes) {
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	payload := C.GoBytes(unsafe.Pointer(data), C.int(length))
	return status(iosLibrary.registry.SendResourceStreamFrame(uint64(engine), uint64(stream), payload))
}

//export anytty_engine_close_resource_stream
func anytty_engine_close_resource_stream(engine, stream C.anytty_handle_t) C.anytty_status_v1 {
	return status(iosLibrary.registry.CloseResourceStream(uint64(engine), uint64(stream)))
}

//export anytty_engine_command
func anytty_engine_command(engine C.anytty_handle_t, data *C.uint8_t, length C.size_t, out *C.anytty_handle_t) C.anytty_status_v1 {
	return operation(data, length, out, func(payload []byte) (uint64, error) {
		return iosLibrary.registry.EngineCommand(uint64(engine), payload)
	})
}

//export anytty_engine_next_event
func anytty_engine_next_event(engine C.anytty_handle_t, timeoutMillis C.uint32_t, out *C.anytty_buffer_v1) C.anytty_status_v1 {
	if out == nil {
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	ctx := context.Background()
	cancel := func() {}
	if timeoutMillis > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMillis)*time.Millisecond)
	}
	defer cancel()
	payload, err := iosLibrary.registry.NextEvent(ctx, uint64(engine))
	if err != nil {
		return status(err)
	}
	return allocateBuffer(payload, out)
}

//export anytty_platform_next_request
func anytty_platform_next_request(engine C.anytty_handle_t, timeoutMillis C.uint32_t, out *C.anytty_buffer_v1) C.anytty_status_v1 {
	if out == nil {
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	iosLibrary.Lock()
	broker := iosLibrary.platforms[uint64(engine)]
	iosLibrary.Unlock()
	if broker == nil {
		return C.ANYTTY_STATUS_INVALID_HANDLE
	}
	ctx := context.Background()
	cancel := func() {}
	if timeoutMillis > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMillis)*time.Millisecond)
	}
	defer cancel()
	payload, err := broker.NextRequest(ctx)
	if err != nil {
		return status(err)
	}
	return allocateBuffer(payload, out)
}

//export anytty_platform_complete
func anytty_platform_complete(engine C.anytty_handle_t, data *C.uint8_t, length C.size_t) C.anytty_status_v1 {
	if data == nil || length == 0 || uint64(length) > uint64(binding.MaxPayloadBytes) {
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	iosLibrary.Lock()
	broker := iosLibrary.platforms[uint64(engine)]
	iosLibrary.Unlock()
	if broker == nil {
		return C.ANYTTY_STATUS_INVALID_HANDLE
	}
	err := broker.Complete(C.GoBytes(unsafe.Pointer(data), C.int(length)))
	if errors.Is(err, binding.ErrInvalidHandle) {
		return C.ANYTTY_STATUS_OK
	}
	return status(err)
}

func allocateBuffer(payload []byte, out *C.anytty_buffer_v1) C.anytty_status_v1 {
	allocationLength := len(payload)
	if allocationLength == 0 {
		allocationLength = 1
	}
	pointer := C.malloc(C.size_t(allocationLength))
	if pointer == nil {
		return C.ANYTTY_STATUS_CAPACITY
	}
	if len(payload) > 0 {
		copy(unsafe.Slice((*byte)(pointer), len(payload)), payload)
	}
	iosLibrary.Lock()
	iosLibrary.nextBuffer++
	handle := iosLibrary.nextBuffer
	if handle == 0 {
		iosLibrary.Unlock()
		C.free(pointer)
		return C.ANYTTY_STATUS_CAPACITY
	}
	iosLibrary.buffers[handle] = pointer
	iosLibrary.Unlock()
	out.buffer_handle = C.anytty_handle_t(handle)
	out.data = (*C.uint8_t)(pointer)
	out.length = C.size_t(len(payload))
	return C.ANYTTY_STATUS_OK
}

//export anytty_engine_cancel
func anytty_engine_cancel(engine, operationHandle C.anytty_handle_t) C.anytty_status_v1 {
	return status(iosLibrary.registry.Cancel(uint64(engine), uint64(operationHandle)))
}

//export anytty_engine_close_session
func anytty_engine_close_session(engine, session C.anytty_handle_t) C.anytty_status_v1 {
	return status(iosLibrary.registry.CloseSession(uint64(engine), uint64(session)))
}

//export anytty_engine_release
func anytty_engine_release(engine, handle C.anytty_handle_t) C.anytty_status_v1 {
	return status(iosLibrary.registry.Release(uint64(engine), uint64(handle)))
}

//export anytty_engine_close
func anytty_engine_close(engine C.anytty_handle_t) C.anytty_status_v1 {
	handle := uint64(engine)
	iosLibrary.Lock()
	bridge := iosLibrary.bridges[handle]
	delete(iosLibrary.bridges, handle)
	host := iosLibrary.hosts[handle]
	delete(iosLibrary.hosts, handle)
	broker := iosLibrary.platforms[handle]
	delete(iosLibrary.platforms, handle)
	iosLibrary.Unlock()
	if bridge != nil {
		bridge.Stop()
	}
	err := iosLibrary.registry.CloseEngine(handle)
	if broker != nil {
		_ = broker.Close()
	}
	if host != nil {
		if closeErr := host.close(); err == nil {
			err = closeErr
		}
	}
	return status(err)
}

//export anytty_buffer_free
func anytty_buffer_free(buffer C.anytty_handle_t) C.anytty_status_v1 {
	handle := uint64(buffer)
	iosLibrary.Lock()
	pointer := iosLibrary.buffers[handle]
	if pointer != nil {
		delete(iosLibrary.buffers, handle)
	}
	iosLibrary.Unlock()
	if pointer == nil {
		return C.ANYTTY_STATUS_INVALID_HANDLE
	}
	C.free(pointer)
	return C.ANYTTY_STATUS_OK
}

func operation(data *C.uint8_t, length C.size_t, out *C.anytty_handle_t, invoke func([]byte) (uint64, error)) C.anytty_status_v1 {
	if data == nil || length == 0 || uint64(length) > uint64(binding.MaxPayloadBytes) || out == nil {
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	}
	payload := C.GoBytes(unsafe.Pointer(data), C.int(length))
	handle, err := invoke(payload)
	if err != nil {
		return status(err)
	}
	*out = C.anytty_handle_t(handle)
	return C.ANYTTY_STATUS_OK
}

func status(err error) C.anytty_status_v1 {
	if err == nil {
		return C.ANYTTY_STATUS_OK
	}
	switch {
	case errors.Is(err, binding.ErrInvalidHandle):
		return C.ANYTTY_STATUS_INVALID_HANDLE
	case errors.Is(err, binding.ErrClosed):
		return C.ANYTTY_STATUS_CLOSED
	case errors.Is(err, binding.ErrHandleActive):
		return C.ANYTTY_STATUS_INVALID_ARGUMENT
	default:
		return C.ANYTTY_STATUS_INTERNAL
	}
}

func main() {}
