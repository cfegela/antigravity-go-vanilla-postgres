package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go-vanilla-crud/auth"
	"go-vanilla-crud/db"
	"go-vanilla-crud/middleware"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterHandler registers a new user
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Username == "" || req.Email == "" || len(req.Password) < 6 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Username, valid email, and password (min 6 chars) are required"})
		return
	}

	// Check existing user
	var exists int
	err := db.DB.QueryRow("SELECT id FROM users WHERE email = $1 OR username = $2", req.Email, req.Username).Scan(&exists)
	if err == nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "Email or username already exists"})
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to process password"})
		return
	}

	var user UserResponse
	err = db.DB.QueryRow(
		"INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id, username, email, created_at",
		req.Username, req.Email, hashedPassword,
	).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create user"})
		return
	}

	// Issue Tokens
	accessToken, err := auth.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate token"})
		return
	}

	rawRefreshToken, _ := auth.GenerateRandomString(32)
	refreshTokenHash := auth.HashToken(rawRefreshToken)
	expiresAt := time.Now().Add(auth.RefreshTokenTTL)

	_, _ = db.DB.Exec("INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
		user.ID, refreshTokenHash, expiresAt)

	auth.SetTokenCookies(w, accessToken, rawRefreshToken)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Registration successful",
		"user":    user,
	})
}

// LoginHandler authenticates a user
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var user struct {
		UserResponse
		PasswordHash string
	}

	err := db.DB.QueryRow(
		"SELECT id, username, email, password_hash, created_at FROM users WHERE email = $1", req.Email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt)

	if err == sql.ErrNoRows || !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid email or password"})
		return
	}

	// Issue Tokens
	accessToken, err := auth.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate token"})
		return
	}

	rawRefreshToken, _ := auth.GenerateRandomString(32)
	refreshTokenHash := auth.HashToken(rawRefreshToken)
	expiresAt := time.Now().Add(auth.RefreshTokenTTL)

	// Clean up old refresh tokens for user and save new one
	_, _ = db.DB.Exec("DELETE FROM refresh_tokens WHERE user_id = $1 OR expires_at < NOW()", user.ID)
	_, _ = db.DB.Exec("INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
		user.ID, refreshTokenHash, expiresAt)

	auth.SetTokenCookies(w, accessToken, rawRefreshToken)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Login successful",
		"user":    user.UserResponse,
	})
}

// RefreshTokenHandler exchanges valid refresh token cookie for a new access token
func RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing refresh token"})
		return
	}

	rawToken := cookie.Value
	tokenHash := auth.HashToken(rawToken)

	var userID int
	var username string
	var expiresAt time.Time

	err = db.DB.QueryRow(`
		SELECT rt.user_id, u.username, rt.expires_at 
		FROM refresh_tokens rt
		JOIN users u ON u.id = rt.user_id
		WHERE rt.token_hash = $1`, tokenHash,
	).Scan(&userID, &username, &expiresAt)

	if err != nil || time.Now().After(expiresAt) {
		auth.ClearTokenCookies(w)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or expired refresh token"})
		return
	}

	// Issue new Access Token
	newAccessToken, err := auth.GenerateAccessToken(userID, username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to refresh token"})
		return
	}

	auth.SetTokenCookies(w, newAccessToken, "")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Token refreshed successfully",
	})
}

// LogoutHandler revokes user session and clears cookies
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		tokenHash := auth.HashToken(cookie.Value)
		_, _ = db.DB.Exec("DELETE FROM refresh_tokens WHERE token_hash = $1", tokenHash)
	}

	auth.ClearTokenCookies(w)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}

// MeHandler retrieves the profile of the currently logged in user
func MeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	var user UserResponse
	err := db.DB.QueryRow("SELECT id, username, email, created_at FROM users WHERE id = $1", claims.UserID).
		Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		return
	}

	json.NewEncoder(w).Encode(user)
}
