package mail

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func checkGolden(t *testing.T, name string, write bool, m Message) {
	t.Helper()
	got := "To: " + m.To + "\nSubject: " + m.Subject + "\n\n" + m.Text + "\n--- html ---\n" + m.HTML
	path := filepath.Join("testdata", name+".golden")
	if *update && write {
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
}

var invitationData = InvitationData{
	InviterName: "Ann", Title: "Work", ValidDays: 14,
	Link: "https://dash.example.org/invitation?token=abc",
}

func TestRenderInvitationGolden(t *testing.T) {
	for _, tt := range []struct {
		locale, golden string
		group          bool
	}{
		{"en", "invitation_en", false},
		{"de", "invitation_de", false},
		{"fr", "invitation_en", false}, // unknown locale falls back to English
		{"en", "invitation_group_en", true},
		{"de", "invitation_group_de", true},
	} {
		t.Run(tt.golden+"/"+tt.locale, func(t *testing.T) {
			data := invitationData
			data.IsGroup = tt.group
			m, err := RenderInvitation(tt.locale, "bob@example.org", data)
			if err != nil {
				t.Fatal(err)
			}
			checkGolden(t, tt.golden, strings.HasSuffix(tt.golden, tt.locale), m)
		})
	}
}

func TestRenderInvitationKeepsUserTextOnOneLineAndEscaped(t *testing.T) {
	data := invitationData
	data.Title = "Work\r\nBcc: evil@example.org"
	data.InviterName = "<b>Ann</b>"
	m, err := RenderInvitation("en", "bob@example.org", data)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(m.Subject, "\r\n") {
		t.Errorf("subject has a line break: %q", m.Subject)
	}
	if strings.Contains(m.HTML, "<b>Ann</b>") {
		t.Errorf("name isn't escaped in %s", m.HTML)
	}
}

func TestRenderInvitationNoticeGolden(t *testing.T) {
	for _, tt := range []struct {
		locale, golden string
		open           bool
	}{
		{"en", "invitation_notice_en", true},
		{"de", "invitation_notice_de", true},
		{"fr", "invitation_notice_en", true},
		{"en", "invitation_notice_closed_en", false},
		{"de", "invitation_notice_closed_de", false},
	} {
		t.Run(tt.golden+"/"+tt.locale, func(t *testing.T) {
			m, err := RenderInvitationNotice(tt.locale, "new@example.org", InvitationNoticeData{RegistrationOpen: tt.open, ValidDays: 14})
			if err != nil {
				t.Fatal(err)
			}
			checkGolden(t, tt.golden, strings.HasSuffix(tt.golden, tt.locale), m)
		})
	}
}

func TestInvitationNoticeContent(t *testing.T) {
	open, err := RenderInvitationNotice("en", "new@example.org", InvitationNoticeData{RegistrationOpen: true, ValidDays: 14})
	if err != nil {
		t.Fatal(err)
	}
	closed, err := RenderInvitationNotice("en", "new@example.org", InvitationNoticeData{ValidDays: 14})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []Message{open, closed} {
		for _, part := range []string{m.Text, m.HTML} {
			if strings.Contains(part, "http") && !strings.Contains(part, "<!doctype") || strings.Contains(part, "href=") {
				t.Errorf("notice has a link: %s", part)
			}
		}
	}
	if !strings.Contains(open.Text, "within 14 days") || !strings.Contains(open.HTML, "within 14 days") {
		t.Errorf("open notice doesn't name the 14 days: %s", open.Text)
	}
	if strings.Contains(closed.Text, "sign up") || strings.Contains(closed.HTML, "sign up") {
		t.Errorf("closed notice suggests signing up: %s", closed.Text)
	}
}
