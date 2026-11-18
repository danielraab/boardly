// Package ratelimit limits requests per client IP and determines the client IP.
package ratelimit

import (
	"container/list"
	"net"
	"net/http"
	"net/netip"
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

// ClientIP returns the client's IP. With trustedProxies > 0 it is taken from
// X-Forwarded-For: all header lines form one list, and the entry that the
// outermost of the trusted proxies added is used, counting trustedProxies
// entries from the right (or the leftmost entry if there are fewer). Entries
// further left were sent by the client and are never used. Without the
// header, with trustedProxies 0, or when the entry isn't an IP address, it
// is the host part of RemoteAddr.
func ClientIP(r *http.Request, trustedProxies int) string {
	if trustedProxies > 0 {
		if ip, ok := forwardedIP(r.Header.Values("X-Forwarded-For"), trustedProxies); ok {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// forwardedIP returns the X-Forwarded-For entry trustedProxies places from the
// right of the combined header lines, if it is a valid IP address.
func forwardedIP(lines []string, trustedProxies int) (string, bool) {
	entries := strings.Split(strings.Join(lines, ","), ",")
	if len(entries) == 1 && strings.TrimSpace(entries[0]) == "" {
		return "", false
	}
	entry := strings.TrimSpace(entries[max(len(entries)-trustedProxies, 0)])
	if host, _, err := net.SplitHostPort(entry); err == nil {
		entry = host // with a port, or an IPv6 address in brackets
	}
	addr, err := netip.ParseAddr(strings.Trim(entry, "[]"))
	if err != nil {
		return "", false
	}
	return addr.Unmap().String(), true
}
