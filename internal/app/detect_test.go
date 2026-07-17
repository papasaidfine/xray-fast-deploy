package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// An endpoint that serves an HTML page (as ifconfig.me does for non-CLI user
// agents) must be skipped, and detection must fall through to an endpoint that
// returns a bare IPv4 address.
func TestDetectPublicIPv4SkipsHTMLBody(t *testing.T) {
	htmlServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<!DOCTYPE html>\n<html><body>What Is My IP Address?</body></html>")
	}))
	defer htmlServer.Close()

	ipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "203.0.113.7\n")
	}))
	defer ipServer.Close()

	got, err := detectPublicIPv4From([]string{htmlServer.URL, ipServer.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "203.0.113.7" {
		t.Fatalf("got %q, want %q", got, "203.0.113.7")
	}
}

// When no endpoint returns a valid IPv4 address, detection must fail rather
// than return a garbage body.
func TestDetectPublicIPv4AllInvalid(t *testing.T) {
	htmlServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<!DOCTYPE html><html></html>")
	}))
	defer htmlServer.Close()

	if _, err := detectPublicIPv4From([]string{htmlServer.URL}); err == nil {
		t.Fatal("expected error when no endpoint returns a valid IPv4 address")
	}
}
