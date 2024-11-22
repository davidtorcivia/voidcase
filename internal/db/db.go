// internal/db/db.go
package db

import (
	"database/sql"
	"fmt"
	"time"

	"voidcase/internal/models" // Changed from filmcms to voidcase
)

type DB struct {
	*sql.DB
}

func New(sqlDB *sql.DB) *DB {
	return &DB{sqlDB}
}

// Implement missing methods
func (db *DB) GetProjectsByTag(tag string) ([]models.Project, error) {
	rows, err := db.Query(`
        SELECT p.id, p.title, p.description, p.video_embed, p.date, 
               p.created_at, p.updated_at
        FROM projects p
        JOIN project_tags pt ON p.id = pt.project_id
        JOIN tags t ON pt.tag_id = t.id
        WHERE LOWER(t.name) = LOWER(?)
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
		projects = append(projects, p)
	}
	return projects, nil
}

func (db *DB) GetAllProjects() ([]models.Project, error) {
	rows, err := db.Query(`
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
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.VideoEmbed,
			&p.Date, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}

		// Get tags for this project
		tags, err := db.getProjectTags(p.ID)
		if err != nil {
			return nil, err
		}
		p.Tags = tags

		projects = append(projects, p)
	}
	return projects, nil
}

func (db *DB) getProjectTags(projectID int64) ([]string, error) {
	rows, err := db.Query(`
        SELECT t.name 
        FROM tags t
        JOIN project_tags pt ON t.id = pt.tag_id
        WHERE pt.project_id = ?`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (db *DB) DeleteProject(id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete project tags
	if _, err := tx.Exec("DELETE FROM project_tags WHERE project_id = ?", id); err != nil {
		return err
	}

	// Delete project images
	if _, err := tx.Exec("DELETE FROM images WHERE project_id = ?", id); err != nil {
		return err
	}

	// Delete project
	if _, err := tx.Exec("DELETE FROM projects WHERE id = ?", id); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) UpdateSiteConfig(config *models.SiteConfig) error {
	_, err := db.Exec(`
        UPDATE site_config 
        SET about_text = ?, contact_info = ?, tracking_code = ?, 
            theme_name = ?, updated_at = ?
        WHERE id = 1`,
		config.AboutText, config.ContactInfo, config.TrackingCode,
		config.ThemeName, config.UpdatedAt)
	return err
}

func (db *DB) GetSiteConfig() (*models.SiteConfig, error) {
	config := &models.SiteConfig{}
	err := db.QueryRow(`
        SELECT id, about_text, contact_info, tracking_code, theme_name, updated_at 
        FROM site_config WHERE id = 1
    `).Scan(&config.ID, &config.AboutText, &config.ContactInfo,
		&config.TrackingCode, &config.ThemeName, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		return &models.SiteConfig{ThemeName: "default"}, nil
	}
	return config, err
}

func (db *DB) GetAnalyticsSummary(days int) (*models.AnalyticsSummary, error) {
	summary := &models.AnalyticsSummary{
		ProjectViews: make(map[int64]int),
		TopReferrers: make(map[string]int),
	}

	// Get total page views and unique views
	err := db.QueryRow(`
        SELECT COUNT(*), COUNT(DISTINCT ip_hash) 
        FROM page_views 
        WHERE created_at > datetime('now', '-? days')
    `, days).Scan(&summary.TotalViews, &summary.UniqueViews)
	if err != nil {
		return nil, fmt.Errorf("failed to get page views: %w", err)
	}

	// Get project views
	rows, err := db.Query(`
        SELECT project_id, COUNT(*) as views
        FROM page_views 
        WHERE project_id IS NOT NULL
        AND created_at > datetime('now', '-? days')
        GROUP BY project_id
        ORDER BY views DESC
        LIMIT 10
    `, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get project views: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var projectID int64
		var views int
		if err := rows.Scan(&projectID, &views); err != nil {
			return nil, fmt.Errorf("failed to scan project views: %w", err)
		}
		summary.ProjectViews[projectID] = views
	}

	// Get top referrers
	rows, err = db.Query(`
        SELECT referrer, COUNT(*) as count
        FROM page_views
        WHERE created_at > datetime('now', '-? days')
        AND referrer != ''
        GROUP BY referrer
        ORDER BY count DESC
        LIMIT 10
    `, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get referrers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var referrer string
		var count int
		if err := rows.Scan(&referrer, &count); err != nil {
			return nil, fmt.Errorf("failed to scan referrer: %w", err)
		}
		summary.TopReferrers[referrer] = count
	}

	// Get daily views
	rows, err = db.Query(`
        SELECT date(created_at) as view_date, COUNT(*) as views
        FROM page_views
        WHERE created_at > datetime('now', '-? days')
        GROUP BY date(created_at)
        ORDER BY view_date
    `, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily views: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var viewDate time.Time
		var count int
		if err := rows.Scan(&viewDate, &count); err != nil {
			return nil, fmt.Errorf("failed to scan daily views: %w", err)
		}
		summary.DailyViews = append(summary.DailyViews, models.DailyViews{
			Date:  viewDate,
			Count: count,
		})
	}

	return summary, nil
}

func (db *DB) SavePageView(view *models.PageView) error {
	_, err := db.Exec(`
        INSERT INTO page_views (page_path, referrer, project_id, ip_hash, created_at)
        VALUES (?, ?, ?, ?, ?)
    `, view.PagePath, view.Referrer, view.ProjectID, view.IPHash, view.ViewedAt)

	if err != nil {
		return fmt.Errorf("failed to save page view: %w", err)
	}
	return nil
}
