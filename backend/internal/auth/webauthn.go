package auth

import (
	"fmt"

	"github.com/go-webauthn/webauthn/webauthn"
)

// NewWebAuthn returns the relying party for passkeys: bound to rpID and
// accepting ceremonies only from origin (the web app's origin).
func NewWebAuthn(rpID, origin string) (*webauthn.WebAuthn, error) {
	w, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: "Famodash",
		RPOrigins:     []string{origin},
	})
	if err != nil {
		return nil, fmt.Errorf("configuring WebAuthn: %w", err)
	}
	return w, nil
}
