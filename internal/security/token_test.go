package security

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("0123456789abcdef0123456789abcdef-test")

func newManager(now time.Time) *tokenManager {
	return &tokenManager{secret: secret, issuer: "portfolio-api", ttl: time.Hour, now: func() time.Time { return now }}
}

func TestIssueAndVerify(t *testing.T) {
	m := newManager(time.Now())
	token, exp, err := m.Issue(42, "pwv1")
	if err != nil {
		t.Fatal(err)
	}
	if time.Until(exp) <= 0 {
		t.Fatalf("expiry %v is not in the future", exp)
	}
	claims, err := m.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if id, _ := claims.AdminID(); id != 42 || claims.PasswordVersion != "pwv1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerifyRejects(t *testing.T) {
	now := time.Now()
	m := newManager(now)
	valid, _, _ := m.Issue(1, "v")

	expired, _, _ := newManager(now.Add(-2*time.Hour)).Issue(1, "v")

	otherKey := &tokenManager{secret: []byte("another-secret-another-secret-123"), issuer: "portfolio-api", ttl: time.Hour, now: time.Now}
	wrongKey, _, _ := otherKey.Issue(1, "v")

	otherIssuer := &tokenManager{secret: secret, issuer: "someone-else", ttl: time.Hour, now: time.Now}
	wrongIssuer, _, _ := otherIssuer.Issue(1, "v")

	// alg=none must never be accepted.
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "1", "iss": "portfolio-api", "aud": audience, "exp": now.Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)

	// HS512 with the right key is still the wrong algorithm.
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"sub": "1", "iss": "portfolio-api", "aud": audience, "exp": now.Add(time.Hour).Unix(),
	}).SignedString(secret)

	parts := strings.Split(valid, ".")
	tampered := parts[0] + "." + strings.TrimRight(parts[1], "=") + "e30." + parts[2]

	cases := map[string]string{
		"expired":      expired,
		"wrong key":    wrongKey,
		"wrong issuer": wrongIssuer,
		"alg none":     none,
		"alg HS512":    hs512,
		"tampered":     tampered,
		"garbage":      "not-a-jwt",
		"empty":        "",
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := m.Verify(token); err == nil {
				t.Fatal("expected verification to fail")
			}
		})
	}
}

func TestComparePassword(t *testing.T) {
	hash := string(dummyHash) // hash of "timing-equalizer"
	if !ComparePassword(hash, "timing-equalizer") {
		t.Fatal("expected match")
	}
	if ComparePassword(hash, "wrong") {
		t.Fatal("expected mismatch")
	}
	if ComparePassword("", "anything") {
		t.Fatal("empty hash must never match")
	}
	if PasswordVersion("a") == PasswordVersion("b") {
		t.Fatal("password versions must differ for different hashes")
	}
}
