package localweb

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateAddressRequiresIPv4Loopback(t *testing.T) {
	for _, address := range []string{"127.0.0.1:0", "127.10.20.30:4321"} {
		if err := validateAddress(address); err != nil {
			t.Fatalf("validateAddress(%q): %v", address, err)
		}
	}
	for _, address := range []string{"0.0.0.0:1234", "192.168.1.2:1234", "localhost:1234", "[::1]:1234", "127.0.0.1"} {
		if err := validateAddress(address); err == nil {
			t.Fatalf("validateAddress(%q) accepted a non-IPv4-loopback address", address)
		}
	}
}

func TestSecureHandlerProtectsBootstrapAndAssets(t *testing.T) {
	bootstrap := bootstrapResponse{
		Bridge:  bridgeBootstrap{Path: "/api/bridge", Token: "abcdefghijklmnopqrstuvwxyz0123456789_-ABCDE"},
		Machine: machineBootstrap{ID: "local", Name: "Studio", Platform: "darwin"},
	}
	handler := secureHandler("127.0.0.1:4321", map[string]webAsset{
		"index.html": {contentType: "text/html; charset=utf-8", data: []byte("<main>AnyTTY</main>")},
	}, bootstrap, nil, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:4321/api/bootstrap", nil)
	request.Host = "127.0.0.1:4321"
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("bootstrap response = %d headers=%v", recorder.Code, recorder.Header())
	}
	var decoded bootstrapResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil || decoded != bootstrap {
		t.Fatalf("bootstrap body = %#v, %v", decoded, err)
	}
	if !strings.Contains(recorder.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") ||
		recorder.Header().Get("X-Content-Type-Options") != "nosniff" || recorder.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("security headers are missing")
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "http://evil.example/api/bootstrap", nil)
	request.Host = "evil.example"
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected Host response = %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "http://127.0.0.1:4321/", nil)
	request.Host = "127.0.0.1:4321"
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "<main>AnyTTY</main>" {
		t.Fatalf("index response = %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestSecureHandlerRequiresPasswordBeforeBootstrap(t *testing.T) {
	auth, err := newWebAuthenticator([]byte("correct horse battery staple"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(auth.close)
	handler := secureHandler("127.0.0.1:4321", map[string]webAsset{
		"index.html": {contentType: "text/html; charset=utf-8", data: []byte("<main>AnyTTY</main>")},
	}, bootstrapResponse{
		Bridge:  bridgeBootstrap{Path: "/api/bridge", Token: "abcdefghijklmnopqrstuvwxyz0123456789_-ABCDE"},
		Machine: machineBootstrap{ID: "local", Name: "Studio", Platform: "darwin"},
	}, auth, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "https://terminal.example/api/bootstrap", nil)
	request.Host = "terminal.example"
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated bootstrap response = %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "https://terminal.example/api/auth/login", strings.NewReader(`{"password":"correct horse battery staple"}`))
	request.Host = "terminal.example"
	request.Header.Set("Origin", "https://terminal.example")
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("login response = %d %q", recorder.Code, recorder.Body.String())
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("login cookies = %#v", cookies)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "https://terminal.example/api/bootstrap", nil)
	request.Host = "terminal.example"
	request.AddCookie(cookies[0])
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("authenticated bootstrap response = %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestSecureHandlerRejectsCrossOriginLogin(t *testing.T) {
	auth, err := newWebAuthenticator([]byte("correct horse battery staple"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(auth.close)
	handler := secureHandler("127.0.0.1:4321", map[string]webAsset{}, bootstrapResponse{}, auth, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://terminal.example/api/auth/login", strings.NewReader(`{"password":"correct horse battery staple"}`))
	request.Host = "terminal.example"
	request.Header.Set("Origin", "https://evil.example")
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cross-origin login response = %d", recorder.Code)
	}
}

func TestSecureHandlerRequiresHTTPSForPublicLogin(t *testing.T) {
	auth, err := newWebAuthenticator([]byte("correct horse battery staple"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(auth.close)
	handler := secureHandler("127.0.0.1:4321", map[string]webAsset{}, bootstrapResponse{}, auth, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://terminal.example/api/auth/login", strings.NewReader(`{"password":"correct horse battery staple"}`))
	request.Host = "terminal.example"
	request.Header.Set("Origin", "http://terminal.example")
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUpgradeRequired {
		t.Fatalf("insecure public login response = %d", recorder.Code)
	}
}

func TestRequestOriginAllowsForwardedHostOnlyBehindLoopback(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:4321/api/auth/login", nil)
	request.Host = "127.0.0.1:4321"
	request.Header.Set("Origin", "https://terminal.example")
	request.Header.Set("X-Forwarded-Host", "terminal.example")
	if !requestOriginMatchesHost(request) {
		t.Fatal("trusted loopback proxy host was rejected")
	}

	request.Host = "terminal.example"
	request.Header.Set("Origin", "https://evil.example")
	request.Header.Set("X-Forwarded-Host", "evil.example")
	if requestOriginMatchesHost(request) {
		t.Fatal("forwarded host overrode a public request Host")
	}
}
