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
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "10.0.0.1:5555"
	r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.2")

	if got := ClientIP(r, false); got != "10.0.0.1" {
		t.Fatalf("untrusted: got %q", got)
	}
	if got := ClientIP(r, true); got != "203.0.113.7" {
		t.Fatalf("trusted: got %q", got)
	}
	r.Header.Del("X-Forwarded-For")
	if got := ClientIP(r, true); got != "10.0.0.1" {
		t.Fatalf("trusted without header: got %q", got)
	}
}
