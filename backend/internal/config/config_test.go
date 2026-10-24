package config

import (
	"reflect"
	"strings"
	"testing"
)

func env(vars map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := vars[k]
		return v, ok
	}
}

const testBaseURL = "http://localhost:3000"

// withBaseURL returns vars plus the required BOARDLY_BASE_URL, unless vars sets it.
func withBaseURL(vars map[string]string) map[string]string {
	out := map[string]string{"BOARDLY_BASE_URL": testBaseURL}
	for k, v := range vars {
		out[k] = v
	}
	return out
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(withBaseURL(nil)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8080 || cfg.DBPath != "./data/boardly.db" || cfg.LogFormat != "json" {
		t.Fatalf("got %+v", cfg)
	}
	if got := cfg.BaseURL.String(); got != "http://localhost:3000" {
		t.Fatalf("BaseURL = %q", got)
	}
	wantSMTP := SMTP{Port: 587, From: "Boardly <noreply@localhost>", TLS: "starttls"}
	if cfg.SMTP != wantSMTP {
		t.Fatalf("SMTP = %+v, want %+v", cfg.SMTP, wantSMTP)
	}
	if cfg.TrustProxyHeaders {
		t.Fatal("TrustProxyHeaders should default to false")
	}
	if cfg.DataDir != "./data" || cfg.FetchAllowPrivate {
		t.Fatalf("DataDir = %q, FetchAllowPrivate = %v; want ./data, false", cfg.DataDir, cfg.FetchAllowPrivate)
	}
	if cfg.SecureCookies() {
		t.Fatal("http base URL must not use secure cookies")
	}
	if want := (Previews{MaxBytes: 10 << 20, MaxPixels: 25_000_000, Width: 800}); cfg.Previews != want {
		t.Fatalf("Previews = %+v, want %+v", cfg.Previews, want)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"BOARDLY_PORT":                "9000",
		"BOARDLY_DB_PATH":             "/tmp/x.db",
		"BOARDLY_LOG_FORMAT":          "text",
		"BOARDLY_BASE_URL":            "https://dash.example.org/",
		"BOARDLY_SMTP_HOST":           "smtp.example.org",
		"BOARDLY_SMTP_PORT":           "465",
		"BOARDLY_SMTP_USERNAME":       "user",
		"BOARDLY_SMTP_PASSWORD":       "secret",
		"BOARDLY_SMTP_FROM":           "Dash <dash@example.org>",
		"BOARDLY_SMTP_TLS":            "tls",
		"BOARDLY_TRUST_PROXY_HEADERS": "true",
		"BOARDLY_DATA_DIR":            "/srv/boardly",
		"BOARDLY_FETCH_ALLOW_PRIVATE": "true",
		"BOARDLY_PREVIEW_MAX_SIZE_MB": "2",
		"BOARDLY_PREVIEW_MAX_PIXELS":  "1000000",
		"BOARDLY_PREVIEW_WIDTH":       "640",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 9000 || cfg.DBPath != "/tmp/x.db" || cfg.LogFormat != "text" {
		t.Fatalf("got %+v", cfg)
	}
	if got := cfg.BaseURL.String(); got != "https://dash.example.org" {
		t.Fatalf("BaseURL = %q (trailing slash should be removed)", got)
	}
	if cfg.Origin() != "https://dash.example.org" || !cfg.SecureCookies() {
		t.Fatalf("Origin = %q, SecureCookies = %v", cfg.Origin(), cfg.SecureCookies())
	}
	wantSMTP := SMTP{Host: "smtp.example.org", Port: 465, Username: "user", Password: "secret", From: "Dash <dash@example.org>", TLS: "tls"}
	if !reflect.DeepEqual(cfg.SMTP, wantSMTP) {
		t.Fatalf("SMTP = %+v, want %+v", cfg.SMTP, wantSMTP)
	}
	if !cfg.TrustProxyHeaders {
		t.Fatal("TrustProxyHeaders should be true")
	}
	if cfg.DataDir != "/srv/boardly" || !cfg.FetchAllowPrivate {
		t.Fatalf("DataDir = %q, FetchAllowPrivate = %v", cfg.DataDir, cfg.FetchAllowPrivate)
	}
	if want := (Previews{MaxBytes: 2 << 20, MaxPixels: 1_000_000, Width: 640}); cfg.Previews != want {
		t.Fatalf("Previews = %+v, want %+v", cfg.Previews, want)
	}
}

func TestLoadBaseURLRequired(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
	}{
		{"unset", nil},
		{"empty", map[string]string{"BOARDLY_BASE_URL": ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(env(tt.vars))
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), "BOARDLY_BASE_URL") {
				t.Fatalf("error %q does not name BOARDLY_BASE_URL", err)
			}
		})
	}
}

func TestLoadPort(t *testing.T) {
	tests := []struct {
		name    string
		vars    map[string]string
		want    int
		wantErr bool
	}{
		{"default", nil, 8080, false},
		{"empty uses default", map[string]string{"BOARDLY_PORT": ""}, 8080, false},
		{"set", map[string]string{"BOARDLY_PORT": "9000"}, 9000, false},
		{"not a number", map[string]string{"BOARDLY_PORT": "abc"}, 0, true},
		{"out of range", map[string]string{"BOARDLY_PORT": "0"}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadPort(env(tt.vars))
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "BOARDLY_PORT") {
					t.Fatalf("err = %v, want an error naming BOARDLY_PORT", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("port = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestOriginIgnoresPath(t *testing.T) {
	cfg, err := Load(env(map[string]string{"BOARDLY_BASE_URL": "https://example.org:8443/dash"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Origin() != "https://example.org:8443" {
		t.Fatalf("Origin = %q", cfg.Origin())
	}
}

func TestLoadInvalid(t *testing.T) {
	tests := []struct {
		name, key, value string
	}{
		{"port not a number", "BOARDLY_PORT", "abc"},
		{"port out of range", "BOARDLY_PORT", "70000"},
		{"unknown log format", "BOARDLY_LOG_FORMAT", "xml"},
		{"base URL relative", "BOARDLY_BASE_URL", "/dash"},
		{"base URL without host", "BOARDLY_BASE_URL", "http://"},
		{"base URL other scheme", "BOARDLY_BASE_URL", "ftp://example.org"},
		{"base URL with query", "BOARDLY_BASE_URL", "https://example.org?x=1"},
		{"smtp port not a number", "BOARDLY_SMTP_PORT", "smtp"},
		{"unknown TLS mode", "BOARDLY_SMTP_TLS", "ssl"},
		{"proxy flag not a bool", "BOARDLY_TRUST_PROXY_HEADERS", "maybe"},
		{"private fetch flag not a bool", "BOARDLY_FETCH_ALLOW_PRIVATE", "sometimes"},
		{"RP ID of another host", "BOARDLY_WEBAUTHN_RP_ID", "example.org"},
		{"AAGUID list URL relative", "BOARDLY_AAGUID_LIST_URL", "/aaguids.json"},
		{"preview size zero", "BOARDLY_PREVIEW_MAX_SIZE_MB", "0"},
		{"preview size negative", "BOARDLY_PREVIEW_MAX_SIZE_MB", "-1"},
		{"preview pixels not a number", "BOARDLY_PREVIEW_MAX_PIXELS", "many"},
		{"preview width not a number", "BOARDLY_PREVIEW_WIDTH", "wide"},
		{"preview width fraction", "BOARDLY_PREVIEW_WIDTH", "800.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(env(withBaseURL(map[string]string{tt.key: tt.value})))
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("error %q does not name %s", err, tt.key)
			}
		})
	}
}

func TestWebAuthnRPID(t *testing.T) {
	tests := []struct {
		name, baseURL, rpID, want string
	}{
		{"default dev", testBaseURL, "", "localhost"},
		{"default from base URL host", "https://home.example.org:8443/dash", "", "home.example.org"},
		{"override with parent domain", "https://home.example.org", "example.org", "example.org"},
		{"override with same host", "https://home.example.org", "HOME.example.org", "HOME.example.org"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{"BOARDLY_BASE_URL": tt.baseURL}
			if tt.rpID != "" {
				vars["BOARDLY_WEBAUTHN_RP_ID"] = tt.rpID
			}
			cfg, err := Load(env(vars))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.WebAuthnRPID != tt.want {
				t.Fatalf("WebAuthnRPID = %q, want %q", cfg.WebAuthnRPID, tt.want)
			}
		})
	}
}

func TestAAGUIDListURL(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want string
	}{
		{"unset uses upstream", nil, DefaultAAGUIDListURL},
		{"self-hosted", map[string]string{"BOARDLY_AAGUID_LIST_URL": "https://files.example.org/aaguids.json"}, "https://files.example.org/aaguids.json"},
		{"empty disables", map[string]string{"BOARDLY_AAGUID_LIST_URL": ""}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(env(withBaseURL(tt.vars)))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.AAGUIDListURL != tt.want {
				t.Fatalf("AAGUIDListURL = %q, want %q", cfg.AAGUIDListURL, tt.want)
			}
		})
	}
}
