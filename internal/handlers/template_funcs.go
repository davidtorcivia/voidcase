// internal/handlers/template_funcs.go
package handlers

import (
	"html/template"
	"strings"
	"time"
	"voidcase/internal/models"
)

var templateFuncs = template.FuncMap{
	"join": strings.Join,
	"contains": func(slice []string, item string) bool {
		for _, s := range slice {
			if s == item {
				return true
			}
		}
		return false
	},
	"isCoreCategory": func(tag string) bool {
		for _, cat := range models.CoreCategories {
			if cat == tag {
				return true
			}
		}
		return false
	},
	"now": time.Now,
	"formatDate": func(t time.Time) string {
		return t.Format("2006-01-02")
	},
}
