package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	jwtSecret       []byte
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "super-secret-jwt-key-change-in-production-12345"
	}
	jwtSecret = []byte(secret)
}

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// HashPassword hashes plain text password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPasswordHash compares password hash with plain text
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateAccessToken creates a short-lived JWT access token
func GenerateAccessToken(userID int, username string) (string, error) {
	expirationTime := time.Now().Add(AccessTokenTTL)
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateAccessToken parses and validates a JWT token string
func ValidateAccessToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	return claims, nil
}

// GenerateRandomString generates a secure random string for refresh tokens
func GenerateRandomString(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken hashes a raw token for storage in DB
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// SetTokenCookies attaches HttpOnly cookies to HTTP response
func SetTokenCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		Expires:  time.Now().Add(AccessTokenTTL),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // Set true in HTTPS production
	})

	if refreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Path:     "/api/auth/refresh",
			Expires:  time.Now().Add(RefreshTokenTTL),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false, // Set true in HTTPS production
		})
	}
}

// ClearTokenCookies clears auth cookies from client
func ClearTokenCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/auth/refresh",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
