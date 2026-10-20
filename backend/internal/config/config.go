// Package config reads the server configuration from FAMODASH_* environment variables.
package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Config holds the server settings.
type Config struct {
	Port      int
	DBPath    string
	LogFormat string // "json" or "text"

	// BaseURL is the public URL of the web app, without a trailing slash.
	// It is used for sign-in links, the Origin check and the cookie's Secure flag.
	BaseURL *url.URL

	SMTP SMTP

	// TrustProxyHeaders makes the server take the client IP from X-Forwarded-For.
	TrustProxyHeaders bool

	// WebAuthnRPID is the relying party ID passkeys are bound to. It defaults
	// to the host name of BaseURL.
	WebAuthnRPID string

	// AAGUIDListURL is where the passkey provider list is downloaded from.
	// Empty disables provider detection.
	AAGUIDListURL string
}

// DefaultAAGUIDListURL is the community-maintained passkey provider list.
const DefaultAAGUIDListURL = "https://raw.githubusercontent.com/passkeydeveloper/passkey-authenticator-aaguids/main/combined_aaguid.json"

// SMTP holds the mail delivery settings. An empty Host disables delivery:
// sign-in links are then written to the log.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      string // "starttls", "tls" or "none"
}

// Load builds a Config from the given environment lookup (usually os.LookupEnv).
// Unset variables get their defaults; invalid values return an error naming the variable.
func Load(lookupEnv func(string) (string, bool)) (Config, error) {
	getenv := func(key string) string {
		v, _ := lookupEnv(key)
		return v
	}
	cfg := Config{
		Port:      8080,
		DBPath:    "./data/famodash.db",
		LogFormat: "json",
		SMTP: SMTP{
			Port: 587,
			From: "Famodash <noreply@localhost>",
			TLS:  "starttls",
		},
	}

	if v := getenv("FAMODASH_PORT"); v != "" {
		port, err := parsePort(v)
		if err != nil {
			return Config{}, fmt.Errorf("FAMODASH_PORT: %w", err)
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

	baseURL := "http://localhost:3000"
	if v := getenv("FAMODASH_BASE_URL"); v != "" {
		baseURL = v
	}
	u, err := parseBaseURL(baseURL)
	if err != nil {
		return Config{}, fmt.Errorf("FAMODASH_BASE_URL: %w", err)
	}
	cfg.BaseURL = u

	cfg.SMTP.Host = getenv("FAMODASH_SMTP_HOST")
	if v := getenv("FAMODASH_SMTP_PORT"); v != "" {
		port, err := parsePort(v)
		if err != nil {
			return Config{}, fmt.Errorf("FAMODASH_SMTP_PORT: %w", err)
		}
		cfg.SMTP.Port = port
	}
	cfg.SMTP.Username = getenv("FAMODASH_SMTP_USERNAME")
	cfg.SMTP.Password = getenv("FAMODASH_SMTP_PASSWORD")
	if v := getenv("FAMODASH_SMTP_FROM"); v != "" {
		cfg.SMTP.From = v
	}
	if v := getenv("FAMODASH_SMTP_TLS"); v != "" {
		if v != "starttls" && v != "tls" && v != "none" {
			return Config{}, fmt.Errorf("FAMODASH_SMTP_TLS: %q must be \"starttls\", \"tls\" or \"none\"", v)
		}
		cfg.SMTP.TLS = v
	}

	if v := getenv("FAMODASH_TRUST_PROXY_HEADERS"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("FAMODASH_TRUST_PROXY_HEADERS: %q must be \"true\" or \"false\"", v)
		}
		cfg.TrustProxyHeaders = b
	}

	cfg.WebAuthnRPID = cfg.BaseURL.Hostname()
	if v := getenv("FAMODASH_WEBAUTHN_RP_ID"); v != "" {
		if !isRegistrableSuffix(cfg.BaseURL.Hostname(), v) {
			return Config{}, fmt.Errorf("FAMODASH_WEBAUTHN_RP_ID: %q must be the host name of FAMODASH_BASE_URL or a parent domain of it", v)
		}
		cfg.WebAuthnRPID = v
	}

	cfg.AAGUIDListURL = DefaultAAGUIDListURL
	if v, ok := lookupEnv("FAMODASH_AAGUID_LIST_URL"); ok {
		if v != "" {
			if u, err := url.Parse(v); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return Config{}, fmt.Errorf("FAMODASH_AAGUID_LIST_URL: %q must be an absolute http(s) URL or empty", v)
			}
		}
		cfg.AAGUIDListURL = v
	}

	return cfg, nil
}

func parsePort(v string) (int, error) {
	port, err := strconv.Atoi(v)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("%q is not a valid port (1-65535)", v)
	}
	return port, nil
}

func parseBaseURL(v string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSuffix(v, "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("%q must be an absolute http(s) URL", v)
	}
	if u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return nil, fmt.Errorf("%q must not contain credentials, a query or a fragment", v)
	}
	return u, nil
}

// isRegistrableSuffix reports whether rpID is host or a parent domain of it,
// which is what browsers accept as relying party ID for pages on host.
func isRegistrableSuffix(host, rpID string) bool {
	host, rpID = strings.ToLower(host), strings.ToLower(rpID)
	return host == rpID || strings.HasSuffix(host, "."+rpID)
}

// Origin returns the scheme and host of the base URL, as browsers send it in the Origin header.
func (c Config) Origin() string {
	return c.BaseURL.Scheme + "://" + c.BaseURL.Host
}

// SecureCookies reports whether cookies must carry the Secure attribute.
func (c Config) SecureCookies() bool {
	return c.BaseURL.Scheme == "https"
}
