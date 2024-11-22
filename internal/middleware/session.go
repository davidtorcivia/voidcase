// internal/middleware/session.go
package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

// generateSessionID creates a secure random session identifier
func generateSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// If we can't get random bytes, panic - this is a severe security issue
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

func (am *AuthMiddleware) CreateSession(userID int64) (string, error) {
	sessionID := generateSessionID()
	expiry := time.Now().Add(24 * time.Hour)

	_, err := am.db.Exec(`
        INSERT INTO sessions (id, user_id, created_at, expires_at)
        VALUES (?, ?, ?, ?)
    `, sessionID, userID, time.Now(), expiry)

	return sessionID, err
}

func (am *AuthMiddleware) ValidateSession(sessionID string) bool {
	var exists bool
	err := am.db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM sessions 
            WHERE id = ? AND expires_at > datetime('now')
        )`, sessionID).Scan(&exists)
	return err == nil && exists
}

func (am *AuthMiddleware) DeleteSession(sessionID string) error {
	_, err := am.db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}
