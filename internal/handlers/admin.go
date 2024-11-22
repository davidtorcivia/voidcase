// internal/handlers/admin.go
package handlers

import (
	"database/sql"
	"net/http"

	"voidcase/internal/middleware"
	"voidcase/internal/models"

	"github.com/gorilla/csrf"
)

// AdminHandler handles admin dashboard functionality
type AdminHandler struct {
	db *sql.DB
}

// NewAdminHandler creates a new admin handler instance
func NewAdminHandler(db *sql.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// getRecentProjects retrieves the most recent projects up to limit
func (h *AdminHandler) getRecentProjects(limit int) ([]models.Project, error) {
	rows, err := h.db.Query(`
        SELECT id, title, date FROM projects 
        ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Title, &p.Date); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// DashboardHandler renders the admin dashboard with recent projects,
// analytics and category statistics
func (h *AdminHandler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Get recent projects
	recentProjects, err := h.getRecentProjects(5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get analytics summary for last 7 days
	analytics, err := middleware.NewAnalyticsMiddleware(h.db).GetAnalyticsSummary(7)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get category counts
	categories, err := h.getCategoryCounts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:          "Admin Dashboard",
		RecentProjects: recentProjects,
		Analytics:      analytics,
		Categories:     categories,
		CSRFToken:      csrf.Token(r),
		IsAdmin:        true,
	}

	tmpl, err := loadTemplates("", "admin/layout.html", "admin/dashboard.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// getCategoryCounts returns the count of projects per category/tag,
// ordered by core categories first then by count
func (h *AdminHandler) getCategoryCounts() ([]CategoryCount, error) {
	rows, err := h.db.Query(`
        SELECT t.name, COUNT(pt.project_id) as count
        FROM tags t
        LEFT JOIN project_tags pt ON t.id = pt.tag_id
        GROUP BY t.name
        ORDER BY 
            CASE 
                WHEN t.name = 'Commercial' THEN 1
                WHEN t.name = 'Narrative' THEN 2
                WHEN t.name = 'Music Video' THEN 3
                WHEN t.name = 'Documentary' THEN 4
                ELSE 5
            END,
            count DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []CategoryCount
	for rows.Next() {
		var c CategoryCount
		if err := rows.Scan(&c.Name, &c.Count); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}
