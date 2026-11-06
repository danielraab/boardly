package offline

import (
	_ "embed"
	"strings"
)

// consentHostList is the list of consent-management provider hosts whose
// requests the browser fails while it takes a screenshot.
//
//go:embed consent_hosts.txt
var consentHostList string

// consentStyles hides the cookie banners of common consent tools and undoes
// the scroll lock they put on the page.
//
//go:embed consent.css
var consentStyles string

// hostMatcher reports whether a host is one of hosts or a subdomain of one.
func hostMatcher(hosts []string) func(host string) bool {
	set := make(map[string]bool, len(hosts))
	for _, h := range hosts {
		set[strings.ToLower(h)] = true
	}
	return func(host string) bool {
		host = strings.TrimSuffix(strings.ToLower(host), ".")
		for host != "" {
			if set[host] {
				return true
			}
			i := strings.IndexByte(host, '.')
			if i < 0 {
				return false
			}
			host = host[i+1:]
		}
		return false
	}
}

// parseHostList returns the hosts of a list with one host per line, without
// blank lines and # comments.
func parseHostList(list string) []string {
	var hosts []string
	for line := range strings.Lines(list) {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		if line = strings.TrimSpace(line); line != "" {
			hosts = append(hosts, line)
		}
	}
	return hosts
}

// isConsentHost reports whether host belongs to a consent-management
// provider of consent_hosts.txt.
var isConsentHost = hostMatcher(parseHostList(consentHostList))
