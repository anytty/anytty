package sessions

import (
	"testing"
	"time"

	"github.com/anytty/anytty/shared/transport/memory"
)

func TestCloseGrantClosesOnlyMatchingSessions(t *testing.T) {
	registry := NewRegistry()
	firstClient, firstServer := memory.NewPair()
	secondClient, secondServer := memory.NewPair()
	otherClient, otherServer := memory.NewPair()
	defer firstClient.Close()
	defer secondClient.Close()
	defer otherClient.Close()

	releaseFirst := registry.Track("grant-a", firstServer, time.Time{})
	releaseSecond := registry.Track("grant-a", secondServer, time.Time{})
	releaseOther := registry.Track("grant-b", otherServer, time.Time{})
	defer releaseFirst()
	defer releaseSecond()
	defer releaseOther()

	if registry.ActiveGrant("grant-a") != 2 {
		t.Fatalf("active grant-a = %d", registry.ActiveGrant("grant-a"))
	}
	if closed := registry.CloseGrant("grant-a"); closed != 2 {
		t.Fatalf("closed sessions = %d", closed)
	}
	if _, err := firstClient.Recv(); err == nil {
		t.Fatal("first session survived revoke")
	}
	if _, err := secondClient.Recv(); err == nil {
		t.Fatal("second session survived revoke")
	}
	select {
	case <-otherClient.Done():
		t.Fatal("unrelated session was closed")
	default:
	}
	if registry.ActiveGrant("grant-a") != 0 {
		t.Fatalf("grant-a sessions after revoke = %d", registry.ActiveGrant("grant-a"))
	}
}

func TestTrackAfterRevokeClosesImmediately(t *testing.T) {
	registry := NewRegistry()
	registry.CloseGrant("grant-a")
	client, server := memory.NewPair()
	defer client.Close()
	registry.Track("grant-a", server, time.Time{})
	if _, err := client.Recv(); err == nil {
		t.Fatal("session registered after revoke stayed open")
	}
}

func TestTrackExpiresSession(t *testing.T) {
	registry := NewRegistry()
	client, server := memory.NewPair()
	defer client.Close()
	release := registry.Track("grant-expiring", server, time.Now().Add(20*time.Millisecond))
	defer release()
	if _, err := client.Recv(); err == nil {
		t.Fatal("expired session stayed open")
	}
}

func TestReleaseStopsExpiryTimer(t *testing.T) {
	registry := NewRegistry()
	client, server := memory.NewPair()
	defer client.Close()
	release := registry.Track("grant-release", server, time.Now().Add(30*time.Millisecond))
	release()
	time.Sleep(60 * time.Millisecond)
	if registry.ActiveGrant("grant-release") != 0 {
		t.Fatalf("released session still registered: %d", registry.ActiveGrant("grant-release"))
	}
	_ = server.Close()
}
