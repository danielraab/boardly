// Package config reads the server configuration from BOARDLY_* environment variables.
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

	// DataDir holds the files the server writes itself, such as images.
	DataDir string

	// AppTitle is the product name the web app and the sign-in email show,
	// so an instance such as a staging system can be told apart.
	AppTitle string

	// BaseURL is the public URL of the web app, without a trailing slash.
	// It is used for sign-in links, the Origin check and the cookie's Secure flag.
	BaseURL *url.URL

	SMTP SMTP

	// AllowRegistration lets sign-in create accounts for new addresses.
	AllowRegistration bool

	// TrustProxyHeaders makes the server take the client IP from X-Forwarded-For.
	TrustProxyHeaders bool

	// WebAuthnRPID is the relying party ID passkeys are bound to. It defaults
	// to the host name of BaseURL.
	WebAuthnRPID string

	// AAGUIDListURL is where the passkey provider list is downloaded from.
	// Empty disables provider detection.
	AAGUIDListURL string

	// FetchAllowPrivate lets the favicon fetcher connect to private,
	// loopback and link-local addresses.
	FetchAllowPrivate bool

	Previews Previews
}

// Previews holds the limits of link preview images.
type Previews struct {
	// MaxBytes is the largest image accepted before conversion.
	MaxBytes int64
	// MaxPixels is the largest width times height accepted before decoding.
	MaxPixels int64
	// Width is the largest width of a converted preview.
	Width int
}

// DefaultAppTitle is the app title when BOARDLY_APP_TITLE is unset or blank.
const DefaultAppTitle = "Boardly"

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
// Unset optional variables get their defaults; a missing BOARDLY_BASE_URL or an
// invalid value returns an error naming the variable.
func Load(lookupEnv func(string) (string, bool)) (Config, error) {
	getenv := func(key string) string {
		v, _ := lookupEnv(key)
		return v
	}
	cfg := Config{
		DBPath:    "./data/boardly.db",
		LogFormat: "json",
		DataDir:   "./data",
		AppTitle:  DefaultAppTitle,
		SMTP: SMTP{
			Port: 587,
			From: "Boardly <noreply@localhost>",
			TLS:  "starttls",
		},
		AllowRegistration: true,
		Previews:          Previews{MaxBytes: 10 << 20, MaxPixels: 25_000_000, Width: 800},
	}

	port, err := LoadPort(lookupEnv)
	if err != nil {
		return Config{}, err
	}
	cfg.Port = port

	if v := getenv("BOARDLY_DB_PATH"); v != "" {
		cfg.DBPath = v
	}

	if v := getenv("BOARDLY_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}

	if v := strings.TrimSpace(getenv("BOARDLY_APP_TITLE")); v != "" {
		cfg.AppTitle = v
	}

	if v := getenv("BOARDLY_LOG_FORMAT"); v != "" {
		if v != "json" && v != "text" {
			return Config{}, fmt.Errorf("BOARDLY_LOG_FORMAT: %q must be \"json\" or \"text\"", v)
		}
		cfg.LogFormat = v
	}

	baseURL := getenv("BOARDLY_BASE_URL")
	if baseURL == "" {
		return Config{}, fmt.Errorf("BOARDLY_BASE_URL: required, set it to the public URL of the web app (e.g. https://dash.example.org)")
	}
	u, err := parseBaseURL(baseURL)
	if err != nil {
		return Config{}, fmt.Errorf("BOARDLY_BASE_URL: %w", err)
	}
	cfg.BaseURL = u

	cfg.SMTP.Host = getenv("BOARDLY_SMTP_HOST")
	if v := getenv("BOARDLY_SMTP_PORT"); v != "" {
		port, err := parsePort(v)
		if err != nil {
			return Config{}, fmt.Errorf("BOARDLY_SMTP_PORT: %w", err)
		}
		cfg.SMTP.Port = port
	}
	cfg.SMTP.Username = getenv("BOARDLY_SMTP_USERNAME")
	cfg.SMTP.Password = getenv("BOARDLY_SMTP_PASSWORD")
	if v := getenv("BOARDLY_SMTP_FROM"); v != "" {
		cfg.SMTP.From = v
	}
	if v := getenv("BOARDLY_SMTP_TLS"); v != "" {
		if v != "starttls" && v != "tls" && v != "none" {
			return Config{}, fmt.Errorf("BOARDLY_SMTP_TLS: %q must be \"starttls\", \"tls\" or \"none\"", v)
		}
		cfg.SMTP.TLS = v
	}

	if v := getenv("BOARDLY_ALLOW_REGISTRATION"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("BOARDLY_ALLOW_REGISTRATION: %q must be \"true\" or \"false\"", v)
		}
		cfg.AllowRegistration = b
	}

	if v := getenv("BOARDLY_TRUST_PROXY_HEADERS"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("BOARDLY_TRUST_PROXY_HEADERS: %q must be \"true\" or \"false\"", v)
		}
		cfg.TrustProxyHeaders = b
	}

	if v := getenv("BOARDLY_FETCH_ALLOW_PRIVATE"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("BOARDLY_FETCH_ALLOW_PRIVATE: %q must be \"true\" or \"false\"", v)
		}
		cfg.FetchAllowPrivate = b
	}

	for _, p := range []struct {
		name string
		set  func(int64)
	}{
		{"BOARDLY_PREVIEW_MAX_SIZE_MB", func(n int64) { cfg.Previews.MaxBytes = n << 20 }},
		{"BOARDLY_PREVIEW_MAX_PIXELS", func(n int64) { cfg.Previews.MaxPixels = n }},
		{"BOARDLY_PREVIEW_WIDTH", func(n int64) { cfg.Previews.Width = int(n) }},
	} {
		if v := getenv(p.name); v != "" {
			n, err := strconv.ParseInt(v, 10, 32)
			if err != nil || n < 1 {
				return Config{}, fmt.Errorf("%s: %q must be a positive whole number", p.name, v)
			}
			p.set(n)
		}
	}

	cfg.WebAuthnRPID = cfg.BaseURL.Hostname()
	if v := getenv("BOARDLY_WEBAUTHN_RP_ID"); v != "" {
		if !isRegistrableSuffix(cfg.BaseURL.Hostname(), v) {
			return Config{}, fmt.Errorf("BOARDLY_WEBAUTHN_RP_ID: %q must be the host name of BOARDLY_BASE_URL or a parent domain of it", v)
		}
		cfg.WebAuthnRPID = v
	}

	cfg.AAGUIDListURL = DefaultAAGUIDListURL
	if v, ok := lookupEnv("BOARDLY_AAGUID_LIST_URL"); ok {
		if v != "" {
			if u, err := url.Parse(v); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return Config{}, fmt.Errorf("BOARDLY_AAGUID_LIST_URL: %q must be an absolute http(s) URL or empty", v)
			}
		}
		cfg.AAGUIDListURL = v
	}

	return cfg, nil
}

// LoadPort reads only BOARDLY_PORT (default 8080), for commands that don't
// need the full configuration, such as the health check.
func LoadPort(lookupEnv func(string) (string, bool)) (int, error) {
	v, _ := lookupEnv("BOARDLY_PORT")
	if v == "" {
		return 8080, nil
	}
	port, err := parsePort(v)
	if err != nil {
		return 0, fmt.Errorf("BOARDLY_PORT: %w", err)
	}
	return port, nil
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
