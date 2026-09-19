package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStoreCASPutDeleteAndList(t *testing.T) {
	store := New()
	entry, err := store.Put(PutRequest{AppID: "app", Scope: ScopePrivate, OwnerID: "owner", Key: "k1", Value: []byte("v1")})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Version != 1 || string(entry.Value) != "v1" {
		t.Fatalf("put entry = %#v", entry)
	}
	conflict, err := store.Put(PutRequest{AppID: "app", Scope: ScopePrivate, OwnerID: "owner", Key: "k1", Value: []byte("v2"), CheckVersion: true, ExpectedVersion: 7})
	if !errors.Is(err, ErrVersionConflict) || conflict.Version != 0 {
		t.Fatalf("conflict put = %#v, err=%v", conflict, err)
	}
	updated, err := store.Put(PutRequest{AppID: "app", Scope: ScopePrivate, OwnerID: "owner", Key: "k1", Value: []byte("v2"), CheckVersion: true, ExpectedVersion: 1})
	if err != nil || updated.Version != 2 {
		t.Fatalf("cas put = %#v, err=%v", updated, err)
	}
	if _, err := store.Get("app", ScopePrivate, "owner", "missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("missing get err = %v", err)
	}
	if _, err := store.Put(PutRequest{Scope: ScopePublic, Key: "k"}); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("invalid key err = %v", err)
	}

	list := store.List("app", ScopePrivate, "owner", "")
	if len(list) != 1 || list[0].Key != "k1" {
		t.Fatalf("list = %#v", list)
	}
	deleted, err := store.Delete(DeleteRequest{AppID: "app", Scope: ScopePrivate, OwnerID: "owner", Key: "k1", CheckVersion: true, ExpectedVersion: 2})
	if err != nil || !deleted.Deleted || deleted.Version != 3 {
		t.Fatalf("delete = %#v, err=%v", deleted, err)
	}
	if list := store.List("app", ScopePrivate, "owner", ""); len(list) != 0 {
		t.Fatalf("list after delete = %#v", list)
	}
}

func TestStoreSubscribeBroadcastsChanges(t *testing.T) {
	store := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := store.Subscribe(ctx)
	if _, err := store.Put(PutRequest{AppID: "app", Key: "k", Value: []byte("v")}); err != nil {
		t.Fatal(err)
	}
	select {
	case change := <-changes:
		if change.Op != OpPut || change.Key != "k" || change.Version != 1 {
			t.Fatalf("change = %#v", change)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for put change")
	}
	if _, err := store.Delete(DeleteRequest{AppID: "app", Key: "k"}); err != nil {
		t.Fatal(err)
	}
	select {
	case change := <-changes:
		if change.Op != OpDelete || change.Version != 2 {
			t.Fatalf("delete change = %#v", change)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delete change")
	}
	cancel()
}
