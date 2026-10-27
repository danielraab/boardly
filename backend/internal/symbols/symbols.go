// Package symbols holds the fixed set of symbols that link widgets, notes and
// note categories can show as their icon.
package symbols

import "strings"

// known are the symbol keys. Keep in sync with
// frontend/src/components/icons/symbols.ts.
var known = map[string]bool{}

func init() {
	for _, s := range strings.Fields(`
		book-open bookmark briefcase calendar camera chart-bar cloud code cpu
		database file-text film folder gamepad globe hard-drive heart home image
		inbox key lock mail map-pin message-square monitor music network newspaper
		phone printer router rss search server settings shield shopping-cart star
		terminal tv users wifi wrench`) {
		known[s] = true
	}
}

// Valid reports whether name is a known symbol.
func Valid(name string) bool {
	return known[name]
}
