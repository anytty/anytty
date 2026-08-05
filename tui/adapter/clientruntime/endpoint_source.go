package clientruntimeadapter

import (
	"context"
	"fmt"
	"sync"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

// EndpointEventSource 把共享 ClientRuntime 的 endpoint mailbox 投影为 TUI application event source。
// 它不建立连接、不保存 generation，也不解释 terminal lifecycle；关闭由调用 context 统一驱动。
type EndpointEventSource struct {
	Runtime     clientruntime.Runtime
	EndpointID  state.EndpointID
	EndpointIDs []state.EndpointID
}

// WatchEndpointEvents 订阅一个 Endpoint 的 runtime event，并在独立 bounded channel 中做纯枚举/展示映射。
func (source EndpointEventSource) WatchEndpointEvents(ctx context.Context) (<-chan port.EndpointRuntimeEvent, error) {
	endpointIDs := normalizeEndpointEventIDs(source.EndpointID, source.EndpointIDs)
	if source.Runtime == nil || ctx == nil || len(endpointIDs) == 0 {
		return nil, fmt.Errorf("endpoint event source requires runtime, context, and endpoint_id")
	}
	streams := make([]<-chan clientruntime.EndpointEvent, 0, len(endpointIDs))
	for _, endpointID := range endpointIDs {
		events, err := source.Runtime.WatchEndpoint(ctx, clientruntimeEndpointID(endpointID))
		if err != nil {
			return nil, err
		}
		streams = append(streams, events)
	}
	out := make(chan port.EndpointRuntimeEvent, 16)
	merged := make(chan clientruntime.EndpointEvent, 16)
	var waitGroup sync.WaitGroup
	for _, events := range streams {
		waitGroup.Add(1)
		go func(events <-chan clientruntime.EndpointEvent) {
			defer waitGroup.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-events:
					if !ok {
						return
					}
					select {
					case merged <- event:
					case <-ctx.Done():
						return
					}
				}
			}
		}(events)
	}
	go func() {
		waitGroup.Wait()
		close(merged)
	}()
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-merged:
				if !ok {
					return
				}
				projected := ProjectEndpointEvent(event)
				select {
				case out <- projected:
				default:
					select {
					case <-out:
					default:
					}
					select {
					case out <- projected:
					default:
					}
				}
			}
		}
	}()
	return out, nil
}

func normalizeEndpointEventIDs(initial state.EndpointID, values []state.EndpointID) []state.EndpointID {
	seen := make(map[state.EndpointID]struct{}, len(values)+1)
	result := make([]state.EndpointID, 0, len(values)+1)
	appendID := func(value state.EndpointID) {
		value = state.NormalizeEndpointID(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	appendID(initial)
	for _, value := range values {
		appendID(value)
	}
	return result
}

func clientruntimeEndpointID(endpointID state.EndpointID) endpoint.EndpointID {
	return endpoint.EndpointID(state.NormalizeEndpointID(endpointID))
}

var _ port.EndpointEventSource = EndpointEventSource{}
