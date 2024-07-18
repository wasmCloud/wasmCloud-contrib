package secrets

import (
	"fmt"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/nats-io/nkeys"
)

func TestSubjectMapper(t *testing.T) {
	// Given prefix, version, servicename
	// queue group name must be '<prefix>.<servicename>'
	// nats subscription subject must be '<prefix>.<version>.<servicename>'
	// wildcard subscription must be the subscription subject above + '.>'
	// operation name comes as '<prefix>.<version>.<servicename>.<operation>'

	serviceName := "kube"
	version := "v0"
	prefix := "wasmcloud.secrets"

	s := SubjectMapper{
		Prefix:      prefix,
		ServiceName: serviceName,
		Version:     version,
	}

	if want, got := fmt.Sprintf("%s.%s", prefix, serviceName), s.QueueGroupName(); got != want {
		t.Errorf("QueueGroupName: want %#v, got %#v", want, got)
	}

	if want, got := fmt.Sprintf("%s.%s.%s", prefix, version, serviceName), s.SecretsSubject(); got != want {
		t.Errorf("SecretsSubject: want %#v, got %#v", want, got)
	}

	if want, got := fmt.Sprintf("%s.%s.%s.>", prefix, version, serviceName), s.SecretWildcardSubject(); got != want {
		t.Errorf("SecretsSubject: want %#v, got %#v", want, got)
	}

	if want, got := "get", s.ParseOperation(fmt.Sprintf("%s.%s.%s.get", prefix, version, serviceName)); got != want {
		t.Errorf("ParseOperation: want %#v, got %#v", want, got)
	}

	if want, got := "", s.ParseOperation(fmt.Sprintf("%s.%s.malformed_subject", prefix, version)); got != want {
		t.Errorf("ParseOperation: want %#v, got %#v", want, got)
	}

	if want, got := "", s.ParseOperation("malformed_subject"); got != want {
		t.Errorf("ParseOperation: want %#v, got %#v", want, got)
	}
}

func TestJWTClaims(t *testing.T) {
	claims := jwt.RegisteredClaims{
		// A usual scenario is to set the expiration time relative to the current time
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Issuer:    "test",
		Subject:   "somebody",
		ID:        "1",
		Audience:  []string{"somebody_else"},
	}

	kp, err := nkeys.CreateAccount()
	if err != nil {
		t.Fatal(err)
	}

	token := jwt.NewWithClaims(SigningMethodEd25519, claims)
	_, err = token.SignedString(kp)
	if err != nil {
		t.Fatal(err)
	}

	validJWT := "eyJ0eXAiOiJqd3QiLCJhbGciOiJFZDI1NTE5In0.eyJqdGkiOiJTakI1Zm05NzRTanU5V01nVFVjaHNiIiwiaWF0IjoxNjQ0ODQzNzQzLCJpc3MiOiJBQ09KSk42V1VQNE9ERDc1WEVCS0tUQ0NVSkpDWTVaS1E1NlhWS1lLNEJFSldHVkFPT1FIWk1DVyIsInN1YiI6Ik1CQ0ZPUE02SlcyQVBKTFhKRDNaNU80Q043Q1BZSjJCNEZUS0xKVVI1WVI1TUlUSVU3SEQzV0Q1Iiwid2FzY2FwIjp7Im5hbWUiOiJFY2hvIiwiaGFzaCI6IjRDRUM2NzNBN0RDQ0VBNkE0MTY1QkIxOTU4MzJDNzkzNjQ3MUNGN0FCNDUwMUY4MzdGOEQ2NzlGNDQwMEJDOTciLCJ0YWdzIjpbXSwiY2FwcyI6WyJ3YXNtY2xvdWQ6aHR0cHNlcnZlciJdLCJyZXYiOjQsInZlciI6IjAuMy40IiwicHJvdiI6ZmFsc2V9fQ.ZWyD6VQqzaYM1beD2x9Fdw4o_Bavy3ZG703Eg4cjhyJwUKLDUiVPVhqHFE6IXdV4cW6j93YbMT6VGq5iBDWmAg"
	t.Run("ParseWithClaims", func(t *testing.T) {
		_, err := jwt.ParseWithClaims(validJWT, &jwt.RegisteredClaims{}, KeyPairFromIssuer())
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("ComponentClaims", func(t *testing.T) {
		token, err := jwt.ParseWithClaims(validJWT, &WasCap{}, KeyPairFromIssuer())
		if err != nil {
			t.Fatal(err)
		}

		var componentClaims ComponentClaims
		wasCap := token.Claims.(*WasCap)
		err = wasCap.ParseCapability(&componentClaims)
		if err != nil {
			t.Error(err)
		}
	})
}
