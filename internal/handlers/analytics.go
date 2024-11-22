// internal/handlers/analytics.go
package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"voidcase/internal/middleware"
	"voidcase/internal/models"

	"github.com/gorilla/csrf"
	"golang.org/x/exp/maps"
)

// ProjectViews represents project view statistics
type ProjectViews struct {
	ProjectID int64
	Title     string
	Views     int
}

type AnalyticsHandler struct {
	db *sql.DB
}

func NewAnalyticsHandler(db *sql.DB) *AnalyticsHandler {
	return &AnalyticsHandler{db: db}
}

func (h *AnalyticsHandler) AdminAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}

	analytics, err := middleware.NewAnalyticsMiddleware(h.db).GetAnalyticsSummary(days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get project titles for IDs
	projectIDs := maps.Keys(analytics.ProjectViews)
	projectsMap, err := h.getProjectsForIDs(projectIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert map to slice for template
	var projectsList []models.Project
	for _, project := range projectsMap {
		projectsList = append(projectsList, project)
	}

	tmpl := template.Must(template.ParseFiles(
		"internal/templates/base.html",
		"internal/templates/admin/analytics.html",
	))

	data := PageData{
		Title:     "Analytics",
		Analytics: analytics,
		Projects:  projectsList,
		CSRFToken: csrf.Token(r),
		IsAdmin:   true,
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *AnalyticsHandler) getProjectsForIDs(ids []int64) (map[int64]models.Project, error) {
	if len(ids) == 0 {
		return make(map[int64]models.Project), nil
	}

	query := fmt.Sprintf(`
        SELECT id, title 
        FROM projects 
        WHERE id IN (?%s)`,
		strings.Repeat(",?", len(ids)-1))

	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make(map[int64]models.Project)
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Title); err != nil {
			return nil, err
		}
		projects[p.ID] = p
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return projects, nil
}
