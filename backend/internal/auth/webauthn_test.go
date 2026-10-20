package auth

import "testing"

func TestNewWebAuthn(t *testing.T) {
	w, err := NewWebAuthn("home.example.org", "https://home.example.org")
	if err != nil {
		t.Fatal(err)
	}
	if w.Config.RPID != "home.example.org" || len(w.Config.RPOrigins) != 1 || w.Config.RPOrigins[0] != "https://home.example.org" {
		t.Fatalf("config = %+v", w.Config)
	}
}
