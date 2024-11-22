// internal/handlers/types.go
package handlers

import (
	"html/template"
	"voidcase/internal/models"
)

// PageData represents the data passed to templates
type PageData struct {
	Title          string
	Projects       []models.Project
	Project        *models.Project
	Navigation     []string
	Categories     []CategoryCount
	Theme          string
	About          string
	Contact        string
	TrackingCode   template.HTML
	Analytics      *models.AnalyticsSummary
	RecentProjects []models.Project
	CSRFToken      string
	Error          string
	IsAdmin        bool
	SiteConfig     *models.SiteConfig
	CoreCategories []string
}

// CategoryCount tracks number of projects per category
type CategoryCount struct {
	Name  string
	Count int
}
