// Package auth menangani autentikasi operator (bcrypt + JWT) dan middleware HTTP.
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword menghasilkan hash bcrypt dari password mentah.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword mengembalikan true bila password cocok dengan hash bcrypt.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
