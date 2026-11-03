package ratelimit

import (
	"testing"
	"time"
)

var base = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)

func at(h, m int) time.Time {
	return base.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
}

func mustAllow(t *testing.T, w *Window, key string, now time.Time) {
	t.Helper()
	if ok, wait := w.Allow(key, now); !ok {
		t.Fatalf("Allow(%q, %s) refused, retry after %s", key, now.Format("15:04"), wait)
	}
}

func TestWindowHourlyLimit(t *testing.T) {
	w := NewWindow(3, 10)
	mustAllow(t, w, "ann", at(0, 0))
	mustAllow(t, w, "ann", at(0, 2))
	mustAllow(t, w, "ann", at(0, 5))
	ok, wait := w.Allow("ann", at(0, 30))
	if ok {
		t.Fatal("fourth export within an hour allowed")
	}
	if wait != 30*time.Minute {
		t.Errorf("retry after %s, want 30m (until 10:00 is an hour old)", wait)
	}
	// The refused attempt doesn't count.
	mustAllow(t, w, "ann", at(1, 1))
}

func TestWindowPasses(t *testing.T) {
	w := NewWindow(3, 10)
	mustAllow(t, w, "ann", at(0, 0))
	mustAllow(t, w, "ann", at(0, 3))
	mustAllow(t, w, "ann", at(0, 5))
	mustAllow(t, w, "ann", at(1, 1))
	if ok, wait := w.Allow("ann", at(1, 2)); ok || wait != time.Minute {
		t.Errorf("Allow at 11:02 = %v, %s; want refused for 1m (until 10:03 is an hour old)", ok, wait)
	}
}

func TestWindowDailyLimit(t *testing.T) {
	w := NewWindow(3, 10)
	for i := range 10 {
		mustAllow(t, w, "ann", base.Add(time.Duration(i)*time.Hour))
	}
	ok, wait := w.Allow("ann", at(10, 0))
	if ok {
		t.Fatal("eleventh export within 24 hours allowed")
	}
	if wait != 14*time.Hour {
		t.Errorf("retry after %s, want 14h (until the first export is a day old)", wait)
	}
	mustAllow(t, w, "ann", at(24, 0))
}

func TestWindowPerKey(t *testing.T) {
	w := NewWindow(1, 10)
	mustAllow(t, w, "ann", base)
	if ok, _ := w.Allow("ann", base); ok {
		t.Fatal("second export for ann allowed")
	}
	mustAllow(t, w, "bob", base)
}

func TestWindowDisabled(t *testing.T) {
	w := NewWindow(0, 5)
	for range 5 {
		mustAllow(t, w, "ann", base)
	}
	if ok, _ := w.Allow("ann", base); ok {
		t.Fatal("sixth export within a day allowed with PerDay 5")
	}

	off := NewWindow(0, 0)
	for range 100 {
		mustAllow(t, off, "ann", base)
	}
}

func TestWindowForgetsIdleKeys(t *testing.T) {
	w := NewWindow(3, 10)
	mustAllow(t, w, "ann", base)
	mustAllow(t, w, "bob", at(25, 0))
	if _, ok := w.events["ann"]; ok {
		t.Error("ann's events are kept after a day")
	}
}
