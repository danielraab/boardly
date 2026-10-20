// Package webauthntest provides a software passkey authenticator for tests. It
// answers the JSON options the server sends with the JSON credentials a browser
// (via @simplewebauthn/browser) posts back, using "none" attestation and ES256.
package webauthntest

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
)

// Authenticator flags.
const (
	flagUP = 0x01
	flagUV = 0x04
	flagBE = 0x08
	flagBS = 0x10
	flagAT = 0x40
)

// ErrExcluded is returned by Create when the options exclude a credential the
// authenticator holds (the browser's InvalidStateError).
var ErrExcluded = errors.New("authenticator already registered")

// Credential is a passkey the authenticator holds.
type Credential struct {
	ID         []byte
	UserHandle []byte
	key        *ecdsa.PrivateKey
	signCount  uint32
}

// Authenticator is a synced (backup-eligible) platform authenticator.
type Authenticator struct {
	Origin string
	AAGUID [16]byte
	// Credentials are the passkeys created so far, oldest first.
	Credentials []*Credential
	// BadSignature makes the next assertions carry an invalid signature.
	BadSignature bool
}

// New returns an authenticator that acts on pages of origin.
func New(origin string, aaguid [16]byte) *Authenticator {
	return &Authenticator{Origin: origin, AAGUID: aaguid}
}

var b64 = base64.RawURLEncoding

type creationOptions struct {
	Challenge string `json:"challenge"`
	RP        struct {
		ID string `json:"id"`
	} `json:"rp"`
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	ExcludeCredentials []struct {
		ID string `json:"id"`
	} `json:"excludeCredentials"`
}

// Create answers PublicKeyCredentialCreationOptionsJSON with a new passkey and
// returns the RegistrationResponseJSON.
func (a *Authenticator) Create(optionsJSON []byte) ([]byte, error) {
	var opts creationOptions
	if err := json.Unmarshal(optionsJSON, &opts); err != nil {
		return nil, fmt.Errorf("creation options: %w", err)
	}
	for _, ex := range opts.ExcludeCredentials {
		for _, c := range a.Credentials {
			if ex.ID == b64.EncodeToString(c.ID) {
				return nil, ErrExcluded
			}
		}
	}
	userHandle, err := b64.DecodeString(opts.User.ID)
	if err != nil {
		return nil, fmt.Errorf("user id: %w", err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	cred := &Credential{ID: make([]byte, 32), UserHandle: userHandle, key: key}
	if _, err := rand.Read(cred.ID); err != nil {
		return nil, err
	}

	ecdh, err := key.PublicKey.ECDH()
	if err != nil {
		return nil, err
	}
	point := ecdh.Bytes() // 0x04 || X || Y
	coseKey, err := webauthncbor.Marshal(map[int]any{
		1:  2,  // kty: EC2
		3:  -7, // alg: ES256
		-1: 1,  // crv: P-256
		-2: point[1:33],
		-3: point[33:],
	})
	if err != nil {
		return nil, err
	}

	var authData bytes.Buffer
	authData.Write(rpIDHash(opts.RP.ID))
	authData.WriteByte(flagUP | flagUV | flagBE | flagBS | flagAT)
	_ = binary.Write(&authData, binary.BigEndian, cred.signCount)
	authData.Write(a.AAGUID[:])
	_ = binary.Write(&authData, binary.BigEndian, uint16(len(cred.ID)))
	authData.Write(cred.ID)
	authData.Write(coseKey)

	attObj, err := webauthncbor.Marshal(map[string]any{
		"fmt":      "none",
		"attStmt":  map[string]any{},
		"authData": authData.Bytes(),
	})
	if err != nil {
		return nil, err
	}
	clientData, err := a.clientData("webauthn.create", opts.Challenge)
	if err != nil {
		return nil, err
	}
	a.Credentials = append(a.Credentials, cred)

	return json.Marshal(map[string]any{
		"id":    b64.EncodeToString(cred.ID),
		"rawId": b64.EncodeToString(cred.ID),
		"type":  "public-key",
		"response": map[string]any{
			"clientDataJSON":    b64.EncodeToString(clientData),
			"attestationObject": b64.EncodeToString(attObj),
			"transports":        []string{"internal", "hybrid"},
		},
		"authenticatorAttachment": "platform",
		"clientExtensionResults":  map[string]any{},
	})
}

type requestOptions struct {
	Challenge string `json:"challenge"`
	RPID      string `json:"rpId"`
}

// Get answers PublicKeyCredentialRequestOptionsJSON with an assertion by cred
// (nil: the newest passkey) and returns the AuthenticationResponseJSON.
func (a *Authenticator) Get(optionsJSON []byte, cred *Credential) ([]byte, error) {
	var opts requestOptions
	if err := json.Unmarshal(optionsJSON, &opts); err != nil {
		return nil, fmt.Errorf("request options: %w", err)
	}
	if cred == nil {
		if len(a.Credentials) == 0 {
			return nil, errors.New("no passkeys")
		}
		cred = a.Credentials[len(a.Credentials)-1]
	}
	cred.signCount++

	var authData bytes.Buffer
	authData.Write(rpIDHash(opts.RPID))
	authData.WriteByte(flagUP | flagUV | flagBE | flagBS)
	_ = binary.Write(&authData, binary.BigEndian, cred.signCount)

	clientData, err := a.clientData("webauthn.get", opts.Challenge)
	if err != nil {
		return nil, err
	}
	clientHash := sha256.Sum256(clientData)
	signed := sha256.Sum256(append(authData.Bytes(), clientHash[:]...))
	sig, err := ecdsa.SignASN1(rand.Reader, cred.key, signed[:])
	if err != nil {
		return nil, err
	}
	if a.BadSignature {
		sig[len(sig)-1] ^= 0xff
	}

	return json.Marshal(map[string]any{
		"id":    b64.EncodeToString(cred.ID),
		"rawId": b64.EncodeToString(cred.ID),
		"type":  "public-key",
		"response": map[string]any{
			"clientDataJSON":    b64.EncodeToString(clientData),
			"authenticatorData": b64.EncodeToString(authData.Bytes()),
			"signature":         b64.EncodeToString(sig),
			"userHandle":        b64.EncodeToString(cred.UserHandle),
		},
		"authenticatorAttachment": "platform",
		"clientExtensionResults":  map[string]any{},
	})
}

func (a *Authenticator) clientData(typ, challenge string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":        typ,
		"challenge":   challenge,
		"origin":      a.Origin,
		"crossOrigin": false,
	})
}

func rpIDHash(rpID string) []byte {
	h := sha256.Sum256([]byte(rpID))
	return h[:]
}
