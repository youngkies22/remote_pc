package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"remote_pc/internal/model"
)

// Claims adalah payload token JWT operator.
type Claims struct {
	UserID   string     `json:"uid"`
	Username string     `json:"username"`
	Role     model.Role `json:"role"`
	jwt.RegisteredClaims
}

// TokenManager menerbitkan dan memverifikasi token JWT.
type TokenManager struct {
	secret []byte
	expiry time.Duration
}

// NewTokenManager membuat TokenManager dengan secret dan masa berlaku token.
func NewTokenManager(secret string, expiryHours int) *TokenManager {
	if expiryHours <= 0 {
		expiryHours = 24
	}
	return &TokenManager{
		secret: []byte(secret),
		expiry: time.Duration(expiryHours) * time.Hour,
	}
}

// Issue menerbitkan token JWT yang ditandatangani untuk user tertentu.
func (m *TokenManager) Issue(u model.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.expiry)
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

// Verify memvalidasi token dan mengembalikan klaimnya bila valid.
func (m *TokenManager) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode signing token tidak valid")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token tidak valid")
	}
	return claims, nil
}
