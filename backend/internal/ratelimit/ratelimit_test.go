package ratelimit

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestReserveBurstThenRate(t *testing.T) {
	l := New(20, 5, 100)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := range 5 {
		if ok, _ := l.Reserve("1.2.3.4", now); !ok {
			t.Fatalf("request %d rejected within burst", i+1)
		}
	}
	ok, retry := l.Reserve("1.2.3.4", now)
	if ok {
		t.Fatal("sixth request allowed")
	}
	if want := 3 * time.Minute; retry != want { // 20/h = one every 3 minutes
		t.Fatalf("retryAfter = %v, want %v", retry, want)
	}
	// A rejected request takes nothing, so the budget is back after retryAfter.
	if ok, _ := l.Reserve("1.2.3.4", now.Add(retry)); !ok {
		t.Fatal("request after retryAfter rejected")
	}
	if ok, _ := l.Reserve("5.6.7.8", now); !ok {
		t.Fatal("other key affected")
	}
}

func TestEvictsLeastRecentlyUsed(t *testing.T) {
	l := New(20, 1, 2)
	now := time.Now()
	l.Reserve("a", now)
	l.Reserve("b", now)
	l.Reserve("a", now) // a is now most recent
	l.Reserve("c", now) // evicts b
	if len(l.byKey) != 2 {
		t.Fatalf("keys = %d, want 2", len(l.byKey))
	}
	if _, ok := l.byKey["b"]; ok {
		t.Fatal("b should have been evicted")
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name    string
		xff     []string // X-Forwarded-For lines; nil sends none
		proxies int
		want    string
	}{
		{"not trusted", []string{"203.0.113.7"}, 0, "10.0.0.1"},
		{"no header", nil, 1, "10.0.0.1"},
		{"one proxy", []string{"203.0.113.7"}, 1, "203.0.113.7"},
		{"spoofed leftmost entry", []string{"1.2.3.4, 203.0.113.7"}, 1, "203.0.113.7"},
		{"two proxies", []string{"1.2.3.4, 203.0.113.7, 198.51.100.2"}, 2, "203.0.113.7"},
		{"fewer entries than proxies", []string{"203.0.113.7"}, 3, "203.0.113.7"},
		{"several header lines", []string{"1.2.3.4", "203.0.113.7"}, 1, "203.0.113.7"},
		{"invalid entry", []string{"1.2.3.4, unknown"}, 1, "10.0.0.1"},
		{"empty entry", []string{"1.2.3.4, "}, 1, "10.0.0.1"},
		{"empty header", []string{""}, 1, "10.0.0.1"},
		{"entry with port", []string{"203.0.113.7:4711"}, 1, "203.0.113.7"},
		{"IPv6", []string{"2001:db8::1"}, 1, "2001:db8::1"},
		{"IPv6 in brackets with port", []string{"[2001:db8::1]:4711"}, 1, "2001:db8::1"},
		{"IPv4-mapped IPv6", []string{"::ffff:203.0.113.7"}, 1, "203.0.113.7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", nil)
			r.RemoteAddr = "10.0.0.1:5555"
			for _, v := range tt.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if got := ClientIP(r, tt.proxies); got != tt.want {
				t.Fatalf("ClientIP = %q, want %q", got, tt.want)
			}
		})
	}
}
