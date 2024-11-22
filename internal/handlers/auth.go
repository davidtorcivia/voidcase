// internal/handlers/auth.go
package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"

	"voidcase/internal/models"

	"github.com/gorilla/csrf"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles user authentication
type AuthHandler struct {
	db       *sql.DB
	shutdown chan struct{}
}

// NewAuthHandler creates a new auth handler instance
func NewAuthHandler(db *sql.DB) *AuthHandler {
	h := &AuthHandler{
		db:       db,
		shutdown: make(chan struct{}),
	}

	// Start cleanup goroutine
	go h.cleanupRoutine()

	return h
}

// Add cleanup routine
func (h *AuthHandler) cleanupRoutine() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.cleanupExpiredSessions()
		case <-h.shutdown:
			return
		}
	}
}

// Add shutdown method
func (h *AuthHandler) Shutdown() {
	close(h.shutdown)
}

// LoginHandler handles user login requests
func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, err := loadTemplates("", "admin/layout.html", "admin/login.html")
		if err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}

		data := PageData{
			Title:     "Login",
			CSRFToken: csrf.Token(r),
			IsAdmin:   false, // Not admin on login page
		}

		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		return
	}

	username := strings.ToLower(r.FormValue("username"))
	password := r.FormValue("password")

	var user models.User
	var hashedPw string
	err := h.db.QueryRow(`
        SELECT id, password_hash FROM users 
        WHERE LOWER(username) = ?
    `, username).Scan(&user.ID, &hashedPw)

	if err != nil || bcrypt.CompareHashAndPassword(
		[]byte(hashedPw), []byte(password)) != nil {
		http.Redirect(w, r, "/admin/login?error=1", http.StatusSeeOther)
		return
	}

	sessionID := make([]byte, 32)
	if _, err := rand.Read(sessionID); err != nil {
		log.Printf("Session generation error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	session := base64.URLEncoding.EncodeToString(sessionID)

	// Begin transaction for session management
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Transaction error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Clear old sessions for this user
	_, err = tx.Exec("DELETE FROM sessions WHERE user_id = ?", user.ID)
	if err != nil {
		log.Printf("Session cleanup error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create new session
	_, err = tx.Exec(`
        INSERT INTO sessions (id, user_id, created_at, expires_at)
        VALUES (?, ?, ?, ?)
    `, session, user.ID, time.Now(),
		time.Now().Add(24*time.Hour))

	if err != nil {
		log.Printf("Session creation error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Transaction commit error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// LogoutHandler handles user logout requests
func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil {
		if _, err := h.db.Exec("DELETE FROM sessions WHERE id = ?", cookie.Value); err != nil {
			log.Printf("Session deletion error: %v", err)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ValidateSession checks if a session is valid
func (h *AuthHandler) ValidateSession(session string) bool {
	var exists bool
	err := h.db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM sessions 
            WHERE id = ? AND expires_at > datetime('now')
        )`, session).Scan(&exists)
	return err == nil && exists
}

func (h *AuthHandler) cleanupExpiredSessions() {
	_, err := h.db.Exec("DELETE FROM sessions WHERE expires_at < datetime('now')")
	if err != nil {
		log.Printf("Session cleanup error: %v", err)
	}
}
