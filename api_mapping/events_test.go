package apimapping

import (
	"testing"

	"github.com/anytty/anytty/proto/apipb"
)

func TestEventFilterPreservesUnspecifiedStorageScope(t *testing.T) {
	filter := EventFilterFromProto(&apipb.EventSubscribeCommand{Types: []apipb.ApplicationEventType{
		apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE,
	}})
	if filter.StorageScope != "" || filter.StorageAppID != "" || filter.StorageOwnerID != "" || filter.StorageKeyPrefix != "" {
		t.Fatalf("terminal lifecycle filter gained storage constraints: %#v", filter)
	}
}
