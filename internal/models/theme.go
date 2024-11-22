// internal/models/theme.go
package models

// Theme represents a website theme configuration
type Theme struct {
	Name        string
	TemplateDir string
	StyleSheet  string
}

// AvailableThemes defines the supported themes and their configurations
var AvailableThemes = map[string]Theme{
	"default": {
		Name:        "Default",
		TemplateDir: "templates/themes/default",
		StyleSheet:  "/static/css/default/main.css",
	},
	"minimal": {
		Name:        "Minimal",
		TemplateDir: "templates/themes/minimal",
		StyleSheet:  "/static/css/minimal/main.css",
	},
}

// GetTheme returns a theme configuration by name with fallback to default
func GetTheme(name string) Theme {
	if theme, ok := AvailableThemes[name]; ok {
		return theme
	}
	return AvailableThemes["default"]
}
