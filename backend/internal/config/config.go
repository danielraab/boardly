// Package config reads the server configuration from FAMODASH_* environment variables.
package config

import (
	"fmt"
	"strconv"
)

// Config holds the server settings.
type Config struct {
	Port      int
	DBPath    string
	LogFormat string // "json" or "text"
}

// Load builds a Config from the given environment lookup (usually os.Getenv).
// Unset variables get their defaults; invalid values return an error naming the variable.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Port:      8080,
		DBPath:    "./data/famodash.db",
		LogFormat: "json",
	}

	if v := getenv("FAMODASH_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("FAMODASH_PORT: %q is not a valid port (1-65535)", v)
		}
		cfg.Port = port
	}

	if v := getenv("FAMODASH_DB_PATH"); v != "" {
		cfg.DBPath = v
	}

	if v := getenv("FAMODASH_LOG_FORMAT"); v != "" {
		if v != "json" && v != "text" {
			return Config{}, fmt.Errorf("FAMODASH_LOG_FORMAT: %q must be \"json\" or \"text\"", v)
		}
		cfg.LogFormat = v
	}

	return cfg, nil
}
