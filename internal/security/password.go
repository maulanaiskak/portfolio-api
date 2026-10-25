package security

import (
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

// dummyHash is compared against when the email is unknown, so a login for a
// missing user costs the same bcrypt time as a wrong password (no user
// enumeration via timing).
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)

// ComparePassword reports whether password matches hash. A nil/empty hash
// still burns one bcrypt comparison.
func ComparePassword(hash, password string) bool {
	if hash == "" {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// PasswordVersion is a short fingerprint of the stored hash, embedded in
// tokens so rotating the admin password invalidates every issued token.
func PasswordVersion(hash string) string {
	sum := sha256.Sum256([]byte(hash))
	return hex.EncodeToString(sum[:8])
}
