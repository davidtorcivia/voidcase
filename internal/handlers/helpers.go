// internal/handlers/helpers.go
package handlers

import (
	"html/template"
	"path/filepath"
	"strings"
)

func loadTemplates(themePath string, baseTemplate string, pageTemplates ...string) (*template.Template, error) {
	var templates []string

	// For admin templates, use admin directory directly
	if strings.Contains(baseTemplate, "admin/") {
		templates = append(templates, filepath.Join("templates", "admin", "layout.html"))
		for _, tmpl := range pageTemplates {
			templates = append(templates, filepath.Join("templates", "admin", strings.TrimPrefix(tmpl, "admin/")))
		}
		return template.New("layout").Funcs(templateFuncs).ParseFiles(templates...)
	}

	// For themed templates, try theme path first
	templates = append(templates, filepath.Join("templates/themes", themePath, "base.html"))
	for _, tmpl := range pageTemplates {
		templates = append(templates, filepath.Join("templates/themes", themePath, tmpl))
	}

	tmpl, err := template.New("layout").Funcs(templateFuncs).ParseFiles(templates...)
	if err != nil && themePath != "default" {
		// Fallback to default theme
		templates = []string{
			filepath.Join("templates/themes/default/base.html"),
		}
		for _, tmpl := range pageTemplates {
			templates = append(templates, filepath.Join("templates/themes/default", tmpl))
		}
		return template.New("layout").Funcs(templateFuncs).ParseFiles(templates...)
	}
	return tmpl, err
}
