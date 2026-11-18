package httpapi

import (
	"net/http"
	"testing"
)

// JSON responses of the API carry nosniff and a policy that runs and loads
// nothing (openspec change add-web-security-headers).
func TestJSONResponsesCarrySecurityHeaders(t *testing.T) {
	s := newServer(t)
	session := s.signIn(t, "ann@example.org")
	for name, get := range map[string]func() *http.Response{
		"success":      func() *http.Response { return s.req("GET", "/api/me", "", withCookie(session)).Result() },
		"error":        func() *http.Response { return s.req("GET", "/api/me", "").Result() },
		"unknown path": func() *http.Response { return s.req("GET", "/api/nope", "").Result() },
		"app API":      func() *http.Response { return s.req("GET", linkwardenPrefix+"/api/v1/config", "").Result() },
		"tRPC":         func() *http.Response { return s.trpcQuery("config.clientConfig", `{"0":{"json":null}}`).Result() },
	} {
		t.Run(name, func(t *testing.T) {
			res := get()
			if res.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("Content-Type = %q, status %d", res.Header.Get("Content-Type"), res.StatusCode)
			}
			if got := res.Header.Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q", got)
			}
			if got := res.Header.Get("Content-Security-Policy"); got != apiPolicy {
				t.Errorf("Content-Security-Policy = %q, want %q", got, apiPolicy)
			}
		})
	}
}
