package endpoint

import (
	"context"
	"testing"
)

func TestManagerSubscriptionsFanOutPerView(t *testing.T) {
	m := NewManager(Options{})
	t.Cleanup(func() { _ = m.Close() })
	var changesA, changesB, noticesA, noticesB int
	removeA := m.Subscribe(func() { changesA++ }, func(string, string) { noticesA++ })
	removeB := m.Subscribe(func() { changesB++ }, func(string, string) { noticesB++ })
	m.notifyChange()
	m.notifyNotice("warning", "offline")
	if changesA != 1 || changesB != 1 || noticesA != 1 || noticesB != 1 {
		t.Fatalf("fanout = changes %d/%d notices %d/%d, want 1/1/1/1", changesA, changesB, noticesA, noticesB)
	}
	removeA()
	m.notifyChange()
	m.notifyNotice("info", "connected")
	if changesA != 1 || changesB != 2 || noticesA != 1 || noticesB != 2 {
		t.Fatalf("after unsubscribe = changes %d/%d notices %d/%d, want 1/2/1/2", changesA, changesB, noticesA, noticesB)
	}
	removeB()
}

func TestManagerConfigsReturnsStableCopies(t *testing.T) {
	m := NewManager(Options{Dial: func(context.Context, Config) (sessionConn, error) { return nil, context.Canceled }})
	t.Cleanup(func() { _ = m.Close() })
	if err := m.Register(Config{Name: "z", Kind: KindCommand}); err != nil {
		t.Fatal(err)
	}
	if err := m.Register(Config{Name: "a", Kind: KindCommand}); err != nil {
		t.Fatal(err)
	}
	configs := m.Configs()
	if len(configs) != 2 || configs[0].Name != "a" || configs[1].Name != "z" {
		t.Fatalf("configs = %+v", configs)
	}
	configs[0].Name = "mutated"
	if got, _ := m.Config("a"); got.Name != "a" {
		t.Fatal("Configs returned mutable manager state")
	}
}
