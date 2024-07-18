package secrets

import (
	"errors"
	"fmt"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/nats-io/nkeys"
)

var ErrEd25519Verification = errors.New("ed25519: verification error")

// SigningMethodNats implements the Ed25519 family.
type SigningMethodNats struct{}

// Specific instance for Ed25519 - nkeys edition
var (
	SigningMethodEd25519 *SigningMethodNats
)

func init() {
	SigningMethodEd25519 = &SigningMethodNats{}
	jwt.RegisterSigningMethod(SigningMethodEd25519.Alg(), func() jwt.SigningMethod {
		return SigningMethodEd25519
	})
}

func (m *SigningMethodNats) Alg() string {
	return "Ed25519"
}

// Verify implements token verification for the SigningMethod.
func (m *SigningMethodNats) Verify(signingString string, sig []byte, key interface{}) error {
	var ed25519Key nkeys.KeyPair
	var ok bool

	if ed25519Key, ok = key.(nkeys.KeyPair); !ok {
		return fmt.Errorf("%w: Ed25519 sign expects nkeys.KeyPair", jwt.ErrInvalidKeyType)
	}

	return ed25519Key.Verify([]byte(signingString), sig)
}

// Sign implements token signing for the SigningMethod.
func (m *SigningMethodNats) Sign(signingString string, key interface{}) ([]byte, error) {
	var ed25519Key nkeys.KeyPair
	var ok bool

	if ed25519Key, ok = key.(nkeys.KeyPair); !ok {
		return nil, fmt.Errorf("%w: Ed25519 sign expects nkeys.KeyPair", jwt.ErrInvalidKeyType)
	}

	return ed25519Key.Sign([]byte(signingString))
}

func KeyPairFromIssuer() func(token *jwt.Token) (interface{}, error) {
	return func(token *jwt.Token) (interface{}, error) {
		iss, err := token.Claims.GetIssuer()
		if err != nil {
			return nil, err
		}
		return nkeys.FromPublicKey(iss)
	}
}
