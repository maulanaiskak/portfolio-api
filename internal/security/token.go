package security

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const audience = "portfolio-admin"

type Claims struct {
	PasswordVersion string `json:"pwv"`
	jwt.RegisteredClaims
}

// AdminID returns the subject as an admin_users.id.
func (c Claims) AdminID() (int64, error) {
	return strconv.ParseInt(c.Subject, 10, 64)
}

type TokenManager interface {
	Issue(adminID int64, passwordVersion string) (token string, expiresAt time.Time, err error)
	Verify(token string) (Claims, error)
}

type tokenManager struct {
	secret []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time
}

func (m *tokenManager) Issue(adminID int64, passwordVersion string) (string, time.Time, error) {
	now := m.now()
	exp := now.Add(m.ttl)
	claims := Claims{
		PasswordVersion: passwordVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   strconv.FormatInt(adminID, 10),
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        uuid.NewString(),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	return token, exp, err
}

// Verify checks signature, algorithm (HS256 only, so "none"/RS-confusion
// tokens are rejected), issuer, audience and time claims.
func (m *tokenManager) Verify(token string) (Claims, error) {
	var claims Claims
	parsed, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil {
		return Claims{}, err
	}
	if !parsed.Valid {
		return Claims{}, errors.New("invalid token")
	}
	return claims, nil
}

func NewTokenManager(secret []byte, issuer string, ttl time.Duration) TokenManager {
	return &tokenManager{secret: secret, issuer: issuer, ttl: ttl, now: time.Now}
}
