package config

import (
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
	want := Config{Port: 8080, DBPath: "./data/famodash.db", LogFormat: "json"}
	if cfg != want {
		t.Fatalf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"FAMODASH_PORT":       "9000",
		"FAMODASH_DB_PATH":    "/tmp/x.db",
		"FAMODASH_LOG_FORMAT": "text",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Config{Port: 9000, DBPath: "/tmp/x.db", LogFormat: "text"}
	if cfg != want {
		t.Fatalf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadInvalid(t *testing.T) {
	tests := []struct {
		name, key, value string
	}{
		{"port not a number", "FAMODASH_PORT", "abc"},
		{"port out of range", "FAMODASH_PORT", "70000"},
		{"unknown log format", "FAMODASH_LOG_FORMAT", "xml"},
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
