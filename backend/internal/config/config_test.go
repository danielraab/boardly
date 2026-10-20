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

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8080 || cfg.DBPath != "./data/famodash.db" || cfg.LogFormat != "json" {
		t.Fatalf("got %+v", cfg)
	}
	if got := cfg.BaseURL.String(); got != "http://localhost:3000" {
		t.Fatalf("BaseURL = %q", got)
	}
	wantSMTP := SMTP{Port: 587, From: "Famodash <noreply@localhost>", TLS: "starttls"}
	if cfg.SMTP != wantSMTP {
		t.Fatalf("SMTP = %+v, want %+v", cfg.SMTP, wantSMTP)
	}
	if cfg.TrustProxyHeaders {
		t.Fatal("TrustProxyHeaders should default to false")
	}
	if cfg.SecureCookies() {
		t.Fatal("http base URL must not use secure cookies")
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"FAMODASH_PORT":                "9000",
		"FAMODASH_DB_PATH":             "/tmp/x.db",
		"FAMODASH_LOG_FORMAT":          "text",
		"FAMODASH_BASE_URL":            "https://dash.example.org/",
		"FAMODASH_SMTP_HOST":           "smtp.example.org",
		"FAMODASH_SMTP_PORT":           "465",
		"FAMODASH_SMTP_USERNAME":       "user",
		"FAMODASH_SMTP_PASSWORD":       "secret",
		"FAMODASH_SMTP_FROM":           "Dash <dash@example.org>",
		"FAMODASH_SMTP_TLS":            "tls",
		"FAMODASH_TRUST_PROXY_HEADERS": "true",
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
}

func TestOriginIgnoresPath(t *testing.T) {
	cfg, err := Load(env(map[string]string{"FAMODASH_BASE_URL": "https://example.org:8443/dash"}))
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
		{"port not a number", "FAMODASH_PORT", "abc"},
		{"port out of range", "FAMODASH_PORT", "70000"},
		{"unknown log format", "FAMODASH_LOG_FORMAT", "xml"},
		{"base URL relative", "FAMODASH_BASE_URL", "/dash"},
		{"base URL without host", "FAMODASH_BASE_URL", "http://"},
		{"base URL other scheme", "FAMODASH_BASE_URL", "ftp://example.org"},
		{"base URL with query", "FAMODASH_BASE_URL", "https://example.org?x=1"},
		{"smtp port not a number", "FAMODASH_SMTP_PORT", "smtp"},
		{"unknown TLS mode", "FAMODASH_SMTP_TLS", "ssl"},
		{"proxy flag not a bool", "FAMODASH_TRUST_PROXY_HEADERS", "maybe"},
		{"RP ID of another host", "FAMODASH_WEBAUTHN_RP_ID", "example.org"},
		{"AAGUID list URL relative", "FAMODASH_AAGUID_LIST_URL", "/aaguids.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(env(map[string]string{tt.key: tt.value}))
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
		{"default dev", "", "", "localhost"},
		{"default from base URL host", "https://home.example.org:8443/dash", "", "home.example.org"},
		{"override with parent domain", "https://home.example.org", "example.org", "example.org"},
		{"override with same host", "https://home.example.org", "HOME.example.org", "HOME.example.org"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{}
			if tt.baseURL != "" {
				vars["FAMODASH_BASE_URL"] = tt.baseURL
			}
			if tt.rpID != "" {
				vars["FAMODASH_WEBAUTHN_RP_ID"] = tt.rpID
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
		{"self-hosted", map[string]string{"FAMODASH_AAGUID_LIST_URL": "https://files.example.org/aaguids.json"}, "https://files.example.org/aaguids.json"},
		{"empty disables", map[string]string{"FAMODASH_AAGUID_LIST_URL": ""}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(env(tt.vars))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.AAGUIDListURL != tt.want {
				t.Fatalf("AAGUIDListURL = %q, want %q", cfg.AAGUIDListURL, tt.want)
			}
		})
	}
}
