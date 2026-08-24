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
		Bridge:  bridgeBootstrap{Port: 12345, Token: "abcdefghijklmnopqrstuvwxyz0123456789_-ABCDE"},
		Machine: machineBootstrap{ID: "local", Name: "Studio", Platform: "darwin"},
	}
	handler := secureHandler("127.0.0.1:4321", map[string]webAsset{
		"index.html": {contentType: "text/html; charset=utf-8", data: []byte("<main>AnyTTY</main>")},
	}, bootstrap)

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
