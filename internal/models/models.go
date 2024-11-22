// internal/models/models.go
package models

import (
	"time"
)

type User struct {
	ID           int64     `db:"id"`
	Username     string    `db:"username"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}

type Project struct {
	ID          int64     `db:"id"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	VideoEmbed  string    `db:"video_embed"`
	Date        time.Time `db:"date"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	Tags        []string  `db:"-"`
	Images      []Image   `db:"-"`
}

type Tag struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

type ProjectTag struct {
	ProjectID int64 `db:"project_id"`
	TagID     int64 `db:"tag_id"`
}

type Image struct {
	ID        int64     `db:"id"`
	ProjectID int64     `db:"project_id"`
	Hash      string    `db:"hash"`
	Path      string    `db:"path"`
	CreatedAt time.Time `db:"created_at"`
}

type CategoryCount struct {
	Name  string
	Count int
}

var CoreCategories = []string{
	"Commercial",
	"Narrative",
	"Music Video",
	"Documentary",
}

func IsCoreCategory(tag string) bool {
	for _, cat := range CoreCategories {
		if cat == tag {
			return true
		}
	}
	return false
}

type SiteConfig struct {
	ID           int64     `db:"id"`
	AboutText    string    `db:"about_text"`
	ContactInfo  string    `db:"contact_info"`
	TrackingCode string    `db:"tracking_code"`
	ThemeName    string    `db:"theme_name"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type Session struct {
	ID        string    `db:"id"`
	UserID    int64     `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
	ExpiresAt time.Time `db:"expires_at"`
}
