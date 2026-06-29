// Package auth implements djanGO's authentication system.
// Mirrors Django's django.contrib.auth — same developer-facing API.
//
// Django reference: django/contrib/auth/hashers.py
//
// Django uses PBKDF2 with SHA256, 870000 iterations (Django 5.x default).
// Format stored in the database: algorithm$iterations$salt$hash
// Example: pbkdf2_sha256$870000$abcdef123456$<base64-hash>
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// DefaultIterations matches Django 5.x default for PBKDF2SHA256.
	// Django: PBKDF2PasswordHasher.iterations = 870000
	DefaultIterations = 870_000
	saltLen           = 16
)

// MakePassword hashes a plain-text password using PBKDF2-SHA256 —
// mirrors Django's make_password(password).
//
// Django:
//
//	from django.contrib.auth.hashers import make_password
//	make_password("mypassword")
//	# → "pbkdf2_sha256$870000$<salt>$<hash>"
func MakePassword(password string) (string, error) {
	salt, err := randomSalt()
	if err != nil {
		return "", err
	}
	return encode(password, salt, DefaultIterations), nil
}

// CheckPassword verifies a plain-text password against a stored hash —
// mirrors Django's check_password(password, encoded).
//
// Django:
//
//	from django.contrib.auth.hashers import check_password
//	check_password("mypassword", encoded)  # → True/False
func CheckPassword(password, encoded string) bool {
	parts := strings.SplitN(encoded, "$", 4)
	if len(parts) != 4 || parts[0] != "pbkdf2_sha256" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	salt := parts[2]
	candidate := encode(password, salt, iterations)
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(encoded)) == 1
}

func encode(password, salt string, iterations int) string {
	hash := pbkdf2.Key([]byte(password), []byte(salt), iterations, 32, sha256.New)
	b64 := base64.StdEncoding.EncodeToString(hash)
	return fmt.Sprintf("pbkdf2_sha256$%d$%s$%s", iterations, salt, b64)
}

func randomSalt() (string, error) {
	b := make([]byte, saltLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
