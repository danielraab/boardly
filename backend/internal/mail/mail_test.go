package mail

import (
	"bytes"
	"context"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"famodash/internal/config"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func TestNewWithoutHostLogsWarningAndReturnsLogSender(t *testing.T) {
	buf := captureLog(t)
	s := New(config.SMTP{})
	if _, ok := s.(LogSender); !ok {
		t.Fatalf("got %T, want LogSender", s)
	}
	if !strings.Contains(buf.String(), "level=WARN") || !strings.Contains(buf.String(), "mail delivery is disabled") {
		t.Fatalf("missing startup warning in %q", buf.String())
	}
}

func TestNewWithHostReturnsSMTPSender(t *testing.T) {
	if _, ok := New(config.SMTP{Host: "smtp.example.org"}).(*SMTPSender); !ok {
		t.Fatal("want *SMTPSender")
	}
}

func TestLogSenderWritesLinkAndCode(t *testing.T) {
	buf := captureLog(t)
	link := "http://localhost:3000/login/verify#token=abc_DEF-123"
	m, err := RenderMagicLink("en", "ann@example.org", MagicLinkData{Link: link, Code: "K7Q-4MZ", ValidMinutes: 15})
	if err != nil {
		t.Fatal(err)
	}
	if err := (LogSender{}).Send(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{link, "K7Q-4MZ", "ann@example.org"} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("log %q does not contain %q", buf.String(), want)
		}
	}
}

func TestRenderMagicLinkGolden(t *testing.T) {
	data := MagicLinkData{Link: "https://dash.example.org/login/verify#token=TOKEN", Code: "K7Q-4MZ", ValidMinutes: 15}
	for _, tt := range []struct{ locale, golden string }{
		{"en", "en"},
		{"de", "de"},
		{"fr", "en"}, // unknown locale falls back to English
		{"", "en"},
		{"../en", "en"},
	} {
		t.Run(tt.locale, func(t *testing.T) {
			m, err := RenderMagicLink(tt.locale, "ann@example.org", data)
			if err != nil {
				t.Fatal(err)
			}
			got := "To: " + m.To + "\nSubject: " + m.Subject + "\n\n" + m.Text + "\n--- html ---\n" + m.HTML
			path := filepath.Join("testdata", "magic_link_"+tt.golden+".golden")
			if *update && tt.locale == tt.golden {
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run with -update to create it)", err)
			}
			if got != string(want) {
				t.Fatalf("output differs from %s:\n%s", path, got)
			}
		})
	}
}

func TestRenderMagicLinkMentionsValidity(t *testing.T) {
	m, err := RenderMagicLink("en", "a@example.org", MagicLinkData{Link: "x", ValidMinutes: 15})
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{m.Subject, m.Text, m.HTML} {
		if part == "" {
			t.Fatal("empty part")
		}
	}
	if !strings.Contains(m.Text, "15 minutes") || !strings.Contains(m.HTML, "15 minutes") {
		t.Fatal("validity not mentioned")
	}
	if !strings.Contains(m.Text, "ignore") {
		t.Fatal("ignore note missing")
	}
}

func TestRenderMagicLinkOffersLinkThenCode(t *testing.T) {
	for _, locale := range []string{"en", "de"} {
		m, err := RenderMagicLink(locale, "a@example.org", MagicLinkData{Link: "https://x/login/verify#token=T", Code: "K7Q-4MZ", ValidMinutes: 15})
		if err != nil {
			t.Fatal(err)
		}
		for name, part := range map[string]string{"text": m.Text, "html": m.HTML} {
			link, code := strings.Index(part, "#token=T"), strings.Index(part, "K7Q-4MZ")
			if link < 0 || code < 0 || code < link {
				t.Errorf("%s %s: link at %d, code at %d; want the link before the code", locale, name, link, code)
			}
		}
	}
}

// TestSMTPSenderIntegration sends a real message. It runs only when
// FAMODASH_TEST_SMTP_HOST is set, e.g. against the Mailpit dev service:
//
//	FAMODASH_TEST_SMTP_HOST=localhost FAMODASH_TEST_SMTP_PORT=1025 go test ./internal/mail -run Integration
func TestSMTPSenderIntegration(t *testing.T) {
	host := os.Getenv("FAMODASH_TEST_SMTP_HOST")
	if host == "" {
		t.Skip("FAMODASH_TEST_SMTP_HOST not set")
	}
	cfg, err := config.Load(func(k string) (string, bool) {
		switch k {
		case "FAMODASH_BASE_URL":
			return "http://localhost:3000", true
		case "FAMODASH_SMTP_HOST":
			return host, true
		case "FAMODASH_SMTP_PORT":
			return os.LookupEnv("FAMODASH_TEST_SMTP_PORT")
		case "FAMODASH_SMTP_TLS":
			return "none", true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := RenderMagicLink("de", "integration@example.org", MagicLinkData{Link: "http://localhost:3000/login/verify#token=INTEGRATION", ValidMinutes: 15})
	if err != nil {
		t.Fatal(err)
	}
	if err := New(cfg.SMTP).Send(context.Background(), m); err != nil {
		t.Fatalf("Send: %v", err)
	}
}
