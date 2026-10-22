package images

import (
	"errors"
	"net"
	"net/netip"
	"syscall"
)

// ErrBlocked is returned when a fetch would connect to a non-public address.
var ErrBlocked = errors.New("address not allowed")

// blockedPrefixes are non-public ranges that the netip methods don't cover.
var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),     // "this network"
	netip.MustParsePrefix("100.64.0.0/10"), // carrier-grade NAT
	netip.MustParsePrefix("192.0.0.0/24"),  // IETF protocol assignments
	netip.MustParsePrefix("198.18.0.0/15"), // benchmarking
	netip.MustParsePrefix("240.0.0.0/4"),   // reserved, includes broadcast
	netip.MustParsePrefix("64:ff9b::/96"),  // NAT64, may reach private IPv4
}

// IsBlocked reports whether ip is loopback, private, link-local, unique-local,
// carrier-grade NAT, multicast, unspecified or otherwise not public.
// IPv4-mapped IPv6 addresses are checked as IPv4.
func IsBlocked(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return true
	}
	for _, p := range blockedPrefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// guardedDialer returns a dialer that refuses blocked addresses, unless
// allowPrivate is set. The check runs in Control, after DNS resolution, on
// the address actually connected to, so DNS rebinding can't bypass it.
func guardedDialer(allowPrivate bool) *net.Dialer {
	d := &net.Dialer{}
	if allowPrivate {
		return d
	}
	d.Control = func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		ip, err := netip.ParseAddr(host)
		if err != nil || IsBlocked(ip) {
			return ErrBlocked
		}
		return nil
	}
	return d
}
