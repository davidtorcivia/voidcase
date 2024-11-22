// internal/handlers/pages.go
package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"voidcase/internal/models"
)

type PageHandler struct {
	db *sql.DB
}

func NewPageHandler(db *sql.DB) *PageHandler {
	return &PageHandler{db: db}
}

func (h *PageHandler) isAdmin(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}
	var exists bool
	err = h.db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM sessions 
            WHERE id = ? AND expires_at > datetime('now')
        )`, cookie.Value).Scan(&exists)
	return err == nil && exists
}

func (h *PageHandler) getConfig() (*models.SiteConfig, error) {
	config := &models.SiteConfig{}
	err := h.db.QueryRow(`
        SELECT about_text, contact_info, tracking_code, theme_name, updated_at 
        FROM site_config WHERE id = 1
    `).Scan(&config.AboutText, &config.ContactInfo, &config.TrackingCode,
		&config.ThemeName, &config.UpdatedAt)
	if err == sql.ErrNoRows {
		return &models.SiteConfig{ThemeName: "default"}, nil
	}
	return config, err
}

func (h *PageHandler) getNavigation() ([]string, error) {
	// Reuse existing NavigationHandler
	nav := NewNavigationHandler(h.db)
	return nav.GetNavigation()
}

func (h *PageHandler) AboutHandler(w http.ResponseWriter, r *http.Request) {
	config, err := h.getConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nav, err := h.getNavigation()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Use loadTemplates helper instead of direct template parsing
	tmpl, err := loadTemplates(config.ThemeName, "base.html", "about.html")
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:      "About",
		About:      config.AboutText,
		Contact:    config.ContactInfo,
		Navigation: nav,
		Theme:      config.ThemeName,
		IsAdmin:    h.isAdmin(r),
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ContactHandler(w http.ResponseWriter, r *http.Request) {
	config, err := h.getConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles(
		filepath.Join("templates", "themes", config.ThemeName, "base.html"),
		filepath.Join("templates", "contact.html"),
	))

	data := PageData{
		Title:        "Contact",
		Theme:        config.ThemeName,
		Contact:      config.ContactInfo,
		TrackingCode: template.HTML(config.TrackingCode),
		IsAdmin:      h.isAdmin(r),
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
