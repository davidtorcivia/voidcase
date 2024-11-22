// internal/handlers/tags.go
package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"voidcase/internal/models"

	"github.com/gorilla/mux"
)

type TagHandler struct {
	db *sql.DB
}

func NewTagHandler(db *sql.DB) *TagHandler {
	return &TagHandler{db: db}
}

// isAdmin checks if the current request is from an admin user
func (h *TagHandler) isAdmin(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}
	// Verify session in database
	var exists bool
	err = h.db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM sessions 
            WHERE id = ? AND expires_at > datetime('now')
        )`, cookie.Value).Scan(&exists)
	return err == nil && exists
}

func (h *TagHandler) TagHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tag := vars["tag"]

	projects, err := h.getProjectsByTag(tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config, err := h.getConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Use theme-specific template path
	themePath := filepath.Join("templates", "themes", config.ThemeName)
	tmpl, err := template.ParseFiles(
		filepath.Join(themePath, "base.html"),
		filepath.Join(themePath, "tag.html"),
	)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:        "Projects - " + tag,
		Projects:     projects,
		Navigation:   []string{tag},
		Theme:        config.ThemeName,
		TrackingCode: template.HTML(config.TrackingCode),
		IsAdmin:      h.isAdmin(r),
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

func (h *TagHandler) getConfig() (*models.SiteConfig, error) {
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

func (h *TagHandler) getProjectsByTag(tag string) ([]models.Project, error) {
	// Get projects with basic info
	rows, err := h.db.Query(`
        SELECT DISTINCT p.id, p.title, p.description, p.video_embed, p.date, 
               p.created_at, p.updated_at
        FROM projects p
        JOIN project_tags pt ON p.id = pt.project_id
        JOIN tags t ON pt.tag_id = t.id
        WHERE t.name = ?
        ORDER BY p.date DESC`, tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.VideoEmbed,
			&p.Date, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}

		// Get images for each project
		imgRows, err := h.db.Query(`
            SELECT id, hash, path, created_at 
            FROM images 
            WHERE project_id = ?
            ORDER BY created_at`, p.ID)
		if err != nil {
			return nil, err
		}
		defer imgRows.Close()

		var images []models.Image
		for imgRows.Next() {
			var img models.Image
			if err := imgRows.Scan(&img.ID, &img.Hash, &img.Path, &img.CreatedAt); err != nil {
				return nil, err
			}
			images = append(images, img)
		}
		p.Images = images

		projects = append(projects, p)
	}
	return projects, nil
}
