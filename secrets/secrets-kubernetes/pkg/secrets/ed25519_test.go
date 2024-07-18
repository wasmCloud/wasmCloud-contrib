package secrets

import (
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/nats-io/nkeys"
)

func TestEd25519(t *testing.T) {
	t.Run("SignAndVerify", func(t *testing.T) {
		kp, err := nkeys.CreateAccount()
		if err != nil {
			t.Fatal(err)
		}
		pubKey, err := kp.PublicKey()
		if err != nil {
			t.Fatal(err)
		}

		claims := jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    pubKey,
			Subject:   "somebody",
			ID:        "1",
			Audience:  []string{"somebody_else"},
		}

		token := jwt.NewWithClaims(SigningMethodEd25519, claims)
		signedJWT, err := token.SignedString(kp)
		if err != nil {
			t.Fatal(err)
		}

		// try opening the signed jwt
		_, err = jwt.ParseWithClaims(signedJWT, &jwt.RegisteredClaims{}, KeyPairFromIssuer())
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Verify", func(t *testing.T) {
		// jwt from wasmcloud host codebase
		validJWT := "eyJ0eXAiOiJqd3QiLCJhbGciOiJFZDI1NTE5In0.eyJqdGkiOiJTakI1Zm05NzRTanU5V01nVFVjaHNiIiwiaWF0IjoxNjQ0ODQzNzQzLCJpc3MiOiJBQ09KSk42V1VQNE9ERDc1WEVCS0tUQ0NVSkpDWTVaS1E1NlhWS1lLNEJFSldHVkFPT1FIWk1DVyIsInN1YiI6Ik1CQ0ZPUE02SlcyQVBKTFhKRDNaNU80Q043Q1BZSjJCNEZUS0xKVVI1WVI1TUlUSVU3SEQzV0Q1Iiwid2FzY2FwIjp7Im5hbWUiOiJFY2hvIiwiaGFzaCI6IjRDRUM2NzNBN0RDQ0VBNkE0MTY1QkIxOTU4MzJDNzkzNjQ3MUNGN0FCNDUwMUY4MzdGOEQ2NzlGNDQwMEJDOTciLCJ0YWdzIjpbXSwiY2FwcyI6WyJ3YXNtY2xvdWQ6aHR0cHNlcnZlciJdLCJyZXYiOjQsInZlciI6IjAuMy40IiwicHJvdiI6ZmFsc2V9fQ.ZWyD6VQqzaYM1beD2x9Fdw4o_Bavy3ZG703Eg4cjhyJwUKLDUiVPVhqHFE6IXdV4cW6j93YbMT6VGq5iBDWmAg"
		_, err := jwt.ParseWithClaims(validJWT, &jwt.RegisteredClaims{}, KeyPairFromIssuer())
		if err != nil {
			t.Error(err)
		}
	})
}
