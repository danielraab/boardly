package ratelimit

import (
	"sync"
	"time"
)

// Window limits events per key over two sliding windows: at most perHour
// events in the last hour and at most perDay in the last 24 hours. Unlike
// Limiter it counts exactly, without a burst refilling in between. A limit of
// 0 turns its window off. Counts live in memory only.
type Window struct {
	perHour, perDay int

	mu        sync.Mutex
	events    map[string][]time.Time // ascending, only the last 24 hours
	lastSweep time.Time
}

const day = 24 * time.Hour

// NewWindow returns a limiter with the given limits; 0 turns a limit off.
func NewWindow(perHour, perDay int) *Window {
	return &Window{perHour: perHour, perDay: perDay, events: map[string][]time.Time{}}
}

// Allow records an event for key at now if both limits allow it. Otherwise it
// records nothing and returns false with the time until the event would be allowed.
func (w *Window) Allow(key string, now time.Time) (ok bool, retryAfter time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.sweep(now)
	times := prune(w.events[key], now.Add(-day))

	if wait := waitFor(times, w.perHour, time.Hour, now); wait > retryAfter {
		retryAfter = wait
	}
	if wait := waitFor(times, w.perDay, day, now); wait > retryAfter {
		retryAfter = wait
	}
	if retryAfter > 0 {
		w.set(key, times)
		return false, retryAfter
	}
	w.set(key, append(times, now))
	return true, 0
}

func (w *Window) set(key string, times []time.Time) {
	if len(times) == 0 {
		delete(w.events, key)
		return
	}
	w.events[key] = times
}

// sweep drops keys whose events are all older than a day, at most once an
// hour, so keys that stop sending events don't stay in memory.
func (w *Window) sweep(now time.Time) {
	if now.Sub(w.lastSweep) < time.Hour {
		return
	}
	w.lastSweep = now
	for key, times := range w.events {
		w.set(key, prune(times, now.Add(-day)))
	}
}

// prune drops the events at or before cutoff.
func prune(times []time.Time, cutoff time.Time) []time.Time {
	i := 0
	for i < len(times) && !times[i].After(cutoff) {
		i++
	}
	return times[i:]
}

// waitFor returns how long until fewer than limit events lie within the last
// span, or 0 if that is already the case or limit is 0.
func waitFor(times []time.Time, limit int, span time.Duration, now time.Time) time.Duration {
	if limit <= 0 {
		return 0
	}
	recent := prune(times, now.Add(-span))
	if len(recent) < limit {
		return 0
	}
	// The event that has to leave the window for one more to fit.
	return recent[len(recent)-limit].Add(span).Sub(now)
}
