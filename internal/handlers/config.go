// internal/handlers/config.go
package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"voidcase/internal/db"
	"voidcase/internal/models"

	"github.com/gorilla/csrf"
)

type ConfigHandler struct {
	db *db.DB
}

func NewConfigHandler(sqlDB *sql.DB) *ConfigHandler {
	return &ConfigHandler{db: db.New(sqlDB)}
}

func (h *ConfigHandler) AdminConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		config, err := h.db.GetSiteConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl := template.Must(template.ParseFiles(
			filepath.Join("templates", "admin", "layout.html"),
			filepath.Join("templates", "admin", "config.html"),
		))

		data := PageData{
			Title:      "Site Configuration",
			SiteConfig: config,
			CSRFToken:  csrf.Token(r),
			IsAdmin:    true,
		}

		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Rest of handler remains the same
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config := &models.SiteConfig{
		AboutText:    r.FormValue("about_text"),
		ContactInfo:  r.FormValue("contact_info"),
		TrackingCode: r.FormValue("tracking_code"),
		ThemeName:    r.FormValue("theme_name"),
		UpdatedAt:    time.Now(),
	}

	if err := h.db.UpdateSiteConfig(config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
}
