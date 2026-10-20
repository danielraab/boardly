package config

import (
	"reflect"
	"strings"
	"testing"
)

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
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
