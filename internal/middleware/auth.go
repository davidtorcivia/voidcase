// internal/middleware/auth.go
package middleware

import (
	"database/sql"
	"net/http"
)

type AuthMiddleware struct {
	db *sql.DB
}

func NewAuthMiddleware(db *sql.DB) *AuthMiddleware {
	return &AuthMiddleware{db: db}
}

func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		var userID int64
		err = am.db.QueryRow(`
            SELECT user_id FROM sessions 
            WHERE id = ? AND expires_at > datetime('now')
        `, session.Value).Scan(&userID)

		if err != nil {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
