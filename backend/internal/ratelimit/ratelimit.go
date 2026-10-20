// Package ratelimit limits requests per client IP and determines the client IP.
package ratelimit

import (
	"container/list"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter is an in-memory token bucket per key (usually a client IP).
// It keeps at most maxKeys buckets and evicts the least recently used one.
type Limiter struct {
	limit   rate.Limit
	burst   int
	maxKeys int

	mu    sync.Mutex
	order *list.List // front = most recently used; values are *entry
	byKey map[string]*list.Element
}

type entry struct {
	key string
	lim *rate.Limiter
}

// New returns a limiter that allows `perHour` events per hour per key, with the given burst.
func New(perHour, burst, maxKeys int) *Limiter {
	return &Limiter{
		limit:   rate.Limit(float64(perHour) / time.Hour.Seconds()),
		burst:   burst,
		maxKeys: maxKeys,
		order:   list.New(),
		byKey:   map[string]*list.Element{},
	}
}

// Reserve takes one event for key at time now. If the key has no budget left,
// it takes nothing and returns false with the time until the event would be allowed.
func (l *Limiter) Reserve(key string, now time.Time) (ok bool, retryAfter time.Duration) {
	r := l.get(key).ReserveN(now, 1)
	if !r.OK() {
		return false, time.Hour
	}
	if d := r.DelayFrom(now); d > 0 {
		r.CancelAt(now)
		return false, d
	}
	return true, 0
}

func (l *Limiter) get(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	if el, ok := l.byKey[key]; ok {
		l.order.MoveToFront(el)
		return el.Value.(*entry).lim
	}
	e := &entry{key: key, lim: rate.NewLimiter(l.limit, l.burst)}
	l.byKey[key] = l.order.PushFront(e)
	for l.order.Len() > l.maxKeys {
		oldest := l.order.Back()
		l.order.Remove(oldest)
		delete(l.byKey, oldest.Value.(*entry).key)
	}
	return e.lim
}

// ClientIP returns the client's IP: the first X-Forwarded-For hop when trustProxy
// is set and the header is present, otherwise the host part of RemoteAddr.
func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			first, _, _ := strings.Cut(xff, ",")
			if ip := strings.TrimSpace(first); ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
