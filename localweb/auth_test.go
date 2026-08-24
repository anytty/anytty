package localweb

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword([]byte("twelve chars!")); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePassword([]byte("too short")); err == nil {
		t.Fatal("short password was accepted")
	}
	if err := ValidatePassword([]byte{0xff}); err == nil {
		t.Fatal("invalid UTF-8 password was accepted")
	}
	if err := ValidatePassword([]byte("                ")); err == nil {
		t.Fatal("blank password was accepted")
	}
}

func TestPasswordLoginRateLimit(t *testing.T) {
	auth, err := newWebAuthenticator([]byte("correct horse battery staple"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(auth.close)
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	auth.now = func() time.Time { return now }
	for range maxClientLoginFailures {
		_, _, matched, loginErr := auth.authenticate("203.0.113.7", []byte("incorrect password"))
		if loginErr != nil || matched {
			t.Fatalf("failed login matched=%t err=%v", matched, loginErr)
		}
	}
	if _, retryAfter, _, err := auth.authenticate("203.0.113.7", []byte("correct horse battery staple")); err != nil || retryAfter <= 0 {
		t.Fatalf("limited login retry=%s err=%v", retryAfter, err)
	}
	now = now.Add(loginFailureWindow)
	token, retryAfter, matched, err := auth.authenticate("203.0.113.7", []byte("correct horse battery staple"))
	if err != nil || !matched || retryAfter != 0 || token == "" {
		t.Fatalf("login after window token=%q retry=%s matched=%t err=%v", token, retryAfter, matched, err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	if !auth.authenticated(request) {
		t.Fatal("new session was not authenticated")
	}
}
