package symbols

import "testing"

func TestValid(t *testing.T) {
	for _, name := range []string{"home", "shopping-cart", "wrench"} {
		if !Valid(name) {
			t.Errorf("Valid(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"", "Home", "not-a-symbol"} {
		if Valid(name) {
			t.Errorf("Valid(%q) = true, want false", name)
		}
	}
}
