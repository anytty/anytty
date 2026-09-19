package keys

import (
	"strconv"
	"testing"
)

func TestHistoryRetainsLast64(t *testing.T) {
	h := NewHistory(DefaultLimit)
	for i := 1; i <= DefaultLimit+1; i++ {
		id := "ev-" + strconv.Itoa(i)
		if !h.Add(id, Event{Kind: KindKey, Key: "a"}) {
			t.Fatalf("Add(%s) rejected", id)
		}
	}
	if h.Len() != DefaultLimit {
		t.Fatalf("len = %d, want %d", h.Len(), DefaultLimit)
	}
	if _, ok := h.Get("ev-1"); ok {
		t.Fatal("oldest event must have expired")
	}
	ev, ok := h.Get("ev-2")
	if !ok || ev.Key != "a" {
		t.Fatalf("ev-2 = %+v ok=%v, want retained", ev, ok)
	}
	if _, ok := h.Get("ev-" + strconv.Itoa(DefaultLimit+1)); !ok {
		t.Fatal("newest event missing")
	}

	if h.Add("", Event{Kind: KindKey}) {
		t.Fatal("empty id must be rejected")
	}
	if h.Len() != DefaultLimit {
		t.Fatalf("len after rejected add = %d", h.Len())
	}
}

func TestHistoryDefaultLimit(t *testing.T) {
	if got := NewHistory(0); got.limit != DefaultLimit {
		t.Fatalf("limit = %d, want %d", got.limit, DefaultLimit)
	}
	if got := NewHistory(3); got.limit != 3 {
		t.Fatalf("limit = %d, want 3", got.limit)
	}
}
