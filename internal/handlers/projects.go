// internal/handlers/projects.go
package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"voidcase/internal/models"
	"voidcase/internal/utils"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
)

type ProjectHandler struct {
	db *sql.DB
}

func NewProjectHandler(db *sql.DB) *ProjectHandler {
	return &ProjectHandler{db: db}
}

func (h *ProjectHandler) AdminProjectsHandler(w http.ResponseWriter, r *http.Request) {
	projects, err := h.getAllProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := loadTemplates("", "admin/layout.html", "admin/projects.html")
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:     "Manage Projects",
		Projects:  projects,
		CSRFToken: csrf.Token(r),
		IsAdmin:   true,
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *ProjectHandler) AdminNewProjectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, err := loadTemplates("", "admin/layout.html", "admin/project_form.html")
		if err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}

		data := PageData{
			Title:          "New Project",
			CSRFToken:      csrf.Token(r),
			IsAdmin:        true,
			CoreCategories: models.CoreCategories,
			Project:        &models.Project{}, // Initialize empty project
		}

		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	project := &models.Project{
		Title:       r.FormValue("title"),
		Description: template.HTMLEscapeString(r.FormValue("description")),
		VideoEmbed:  utils.SanitizeVideoEmbed(r.FormValue("video_embed")),
		Date:        time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Insert project
	result, err := tx.Exec(`
        INSERT INTO projects (title, description, video_embed, date, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?)`,
		project.Title, project.Description, project.VideoEmbed,
		project.Date, project.CreatedAt, project.UpdatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	projectID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	project.ID = projectID

	// Handle tags
	tags := strings.Split(r.FormValue("tags"), ",")
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if err := h.addProjectTag(tx, projectID, tag); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Handle images
	if err := saveProjectWithImages(tx, r, project); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/projects", http.StatusSeeOther)
}

func (h *ProjectHandler) addProjectTag(tx *sql.Tx, projectID int64, tagName string) error {
	var tagID int64
	err := tx.QueryRow("SELECT id FROM tags WHERE name = ?", tagName).Scan(&tagID)
	if err == sql.ErrNoRows {
		result, err := tx.Exec("INSERT INTO tags (name) VALUES (?)", tagName)
		if err != nil {
			return err
		}
		tagID, err = result.LastInsertId()
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO project_tags (project_id, tag_id) VALUES (?, ?)", projectID, tagID)
	return err
}

func sanitizeEmbed(embed string) string {
	if !strings.Contains(embed, "youtube.com/embed") &&
		!strings.Contains(embed, "player.vimeo.com/video") {
		return ""
	}
	return embed
}

func parseDate(dateStr string) time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Now()
	}
	return t
}

func (h *ProjectHandler) getAllProjects() ([]models.Project, error) {
	rows, err := h.db.Query(`
        SELECT id, title, description, video_embed, date, created_at, updated_at
        FROM projects
        ORDER BY date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.VideoEmbed,
			&p.Date, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (h *ProjectHandler) GetProjectByID(id int64) (*models.Project, error) {
	project, err := h.getProjectWithTags(id)
	if err != nil {
		return nil, err
	}

	// Get images
	rows, err := h.db.Query(`
        SELECT hash, path FROM images 
        WHERE project_id = ? 
        ORDER BY created_at DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []models.Image
	for rows.Next() {
		var img models.Image
		if err := rows.Scan(&img.Hash, &img.Path); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	project.Images = images

	return project, nil
}

func (h *ProjectHandler) getProjectWithTags(id int64) (*models.Project, error) {
	project := &models.Project{}
	err := h.db.QueryRow(`
        SELECT id, title, description, video_embed, date, created_at, updated_at 
        FROM projects WHERE id = ?`, id).Scan(
		&project.ID, &project.Title, &project.Description, &project.VideoEmbed,
		&project.Date, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Get tags
	rows, err := h.db.Query(`
        SELECT t.name FROM tags t
        JOIN project_tags pt ON t.id = pt.tag_id
        WHERE pt.project_id = ?
        ORDER BY CASE 
            WHEN t.name = 'Commercial' THEN 1
            WHEN t.name = 'Narrative' THEN 2
            WHEN t.name = 'Music Video' THEN 3
            WHEN t.name = 'Documentary' THEN 4
            ELSE 5 END, t.name`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		project.Tags = append(project.Tags, tag)
	}
	return project, nil
}

func (h *ProjectHandler) AdminEditProjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		project, err := h.GetProjectByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl, err := loadTemplates("", "admin/layout.html", "admin/project_form.html")
		if err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}

		data := PageData{
			Title:          "Edit Project",
			Project:        project,
			CSRFToken:      csrf.Token(r),
			CoreCategories: models.CoreCategories,
			IsAdmin:        true,
		}

		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Printf("Template execution error: %v", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}
		return
	}

	// Handle POST - Update project
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Update project
	_, err = tx.Exec(`
        UPDATE projects 
        SET title = ?, description = ?, video_embed = ?, date = ?, updated_at = ?
        WHERE id = ?`,
		r.FormValue("title"),
		r.FormValue("description"),
		sanitizeEmbed(r.FormValue("video_embed")),
		parseDate(r.FormValue("date")),
		time.Now(),
		id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.handleProjectTags(tx, id, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete existing tags
	_, err = tx.Exec("DELETE FROM project_tags WHERE project_id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add new tags
	tags := strings.Split(r.FormValue("tags"), ",")
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if err := h.addProjectTag(tx, id, tag); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Handle new images if any
	if err := saveProjectWithImages(tx, r, &models.Project{ID: id}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/projects", http.StatusSeeOther)
}

func (h *ProjectHandler) AdminDeleteProjectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Get image paths before deletion
	rows, err := tx.Query("SELECT path FROM images WHERE project_id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var imagePaths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		imagePaths = append(imagePaths, path)
	}

	// Delete in order: project_tags, images, project
	_, err = tx.Exec("DELETE FROM project_tags WHERE project_id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM images WHERE project_id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM projects WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete image files after successful DB transaction
	for _, path := range imagePaths {
		if err := os.Remove(path); err != nil {
			log.Printf("Failed to delete image file %s: %v", path, err)
		}
	}

	http.Redirect(w, r, "/admin/projects", http.StatusSeeOther)
}

func (h *ProjectHandler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	projects, err := h.getAllProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nav, err := NewNavigationHandler(h.db).GetNavigation()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config, err := h.getConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Use themed template loading
	themePath := filepath.Join("templates", "themes", config.ThemeName)
	tmpl, err := template.ParseFiles(
		filepath.Join(themePath, "base.html"),
		filepath.Join(themePath, "home.html"),
	)

	if err != nil {
		if config.ThemeName != "default" {
			// Fallback to default theme
			themePath = filepath.Join("templates", "themes", "default")
			tmpl, err = template.ParseFiles(
				filepath.Join(themePath, "base.html"),
				filepath.Join(themePath, "home.html"),
			)
			if err != nil {
				log.Printf("Template error: %v", err)
				http.Error(w, "Template error", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("Template error: %v", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}
	}

	data := PageData{
		Title:      "Portfolio",
		Projects:   projects,
		Navigation: nav,
		Theme:      config.ThemeName,
		About:      config.AboutText,
		IsAdmin:    h.isAdmin(r),
	}

	// Change "base" to "layout" to match the base template definition
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}

func (h *ProjectHandler) handleProjectTags(tx *sql.Tx, projectID int64, r *http.Request) error {
	// Delete existing tags
	if _, err := tx.Exec("DELETE FROM project_tags WHERE project_id = ?", projectID); err != nil {
		return err
	}

	// Handle core categories first
	for _, cat := range r.PostForm["categories[]"] {
		if err := h.addProjectTag(tx, projectID, cat); err != nil {
			return err
		}
	}

	// Handle custom tags
	customTags := strings.Split(r.FormValue("custom_tags"), ",")
	for _, tag := range customTags {
		tag = strings.TrimSpace(tag)
		if tag != "" && !models.IsCoreCategory(tag) {
			if err := h.addProjectTag(tx, projectID, tag); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *ProjectHandler) getConfig() (*models.SiteConfig, error) {
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

func (h *ProjectHandler) isAdmin(r *http.Request) bool {
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

func saveProjectWithImages(tx *sql.Tx, r *http.Request, project *models.Project) error {
	if r.MultipartForm == nil || r.MultipartForm.File["images[]"] == nil {
		return nil
	}

	files := r.MultipartForm.File["images[]"]
	uploadDir := filepath.Join("data", "uploads", "images")
	thumbsDir := filepath.Join("data", "uploads", "thumbnails")

	// Create directories if they don't exist
	for _, dir := range []string{uploadDir, thumbsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Process all images first
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		// Hash the file contents
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			return fmt.Errorf("failed to hash file: %w", err)
		}
		contentHash := hex.EncodeToString(hash.Sum(nil))

		// Reset file pointer
		if _, err := file.Seek(0, 0); err != nil {
			return fmt.Errorf("failed to reset file pointer: %w", err)
		}

		ext := ".jpg" // Force jpg extension for consistency
		newPath := filepath.Join(uploadDir, contentHash+ext)
		thumbPath := filepath.Join(thumbsDir, contentHash+ext)

		// Save original
		if err := saveOriginalFile(file, newPath); err != nil {
			return fmt.Errorf("failed to save original file: %w", err)
		}

		// Reset file pointer for thumbnail
		if _, err := file.Seek(0, 0); err != nil {
			return fmt.Errorf("failed to reset file pointer: %w", err)
		}

		// Create thumbnail
		if err := createThumbnail(file, thumbPath); err != nil {
			return fmt.Errorf("failed to create thumbnail: %w", err)
		}

		// Save image record in database
		_, err = tx.Exec(`
            INSERT INTO images (project_id, hash, path, created_at)
            VALUES (?, ?, ?, ?)`,
			project.ID, contentHash, newPath, time.Now())
		if err != nil {
			return fmt.Errorf("failed to save image record: %w", err)
		}
	}

	return nil
}

func saveOriginalFile(src multipart.File, destPath string) error {
	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func createThumbnail(src multipart.File, destPath string) error {
	img, _, err := image.Decode(src)
	if err != nil {
		return err
	}

	thumbnail := resize.Resize(300, 0, img, resize.Lanczos3)

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	return jpeg.Encode(dst, thumbnail, &jpeg.Options{Quality: 85})
}
