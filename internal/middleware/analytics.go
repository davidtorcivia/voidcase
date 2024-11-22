// internal/middleware/analytics.go
package middleware

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"voidcase/internal/models" // Add models import
)

type AnalyticsMiddleware struct {
	db *sql.DB
}

func NewAnalyticsMiddleware(db *sql.DB) *AnalyticsMiddleware {
	return &AnalyticsMiddleware{db: db}
}

func (am *AnalyticsMiddleware) TrackPageView(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't track admin pages
		if !strings.HasPrefix(r.URL.Path, "/admin") {
			go am.savePageView(r.URL.Path, r.Header.Get("Referer"))
		}
		next.ServeHTTP(w, r)
	})
}

func (am *AnalyticsMiddleware) savePageView(path, referrer string) {
	_, err := am.db.Exec(`
        INSERT INTO analytics (page_path, referrer, viewed_at)
        VALUES (?, ?, ?)`,
		path, referrer, time.Now())
	if err != nil {
		log.Printf("Failed to save analytics: %v", err)
	}
}

func (am *AnalyticsMiddleware) GetAnalyticsSummary(days int) (*models.AnalyticsSummary, error) {
	summary := &models.AnalyticsSummary{
		TopPages:     make(map[string]int),
		TopReferrers: make(map[string]int),
		ProjectViews: make(map[int64]int),
		DailyViews:   make([]models.DailyViews, 0),
	}

	// Get total views
	err := am.db.QueryRow(`
        SELECT COUNT(*) FROM analytics 
        WHERE viewed_at >= datetime('now', ?)
    `, fmt.Sprintf("-%d days", days)).Scan(&summary.TotalViews)
	if err != nil {
		return nil, err
	}

	// Get top pages
	rows, err := am.db.Query(`
        SELECT page_path, COUNT(*) as count
        FROM analytics
        WHERE viewed_at >= datetime('now', ?)
        GROUP BY page_path
        ORDER BY count DESC
        LIMIT 10
    `, fmt.Sprintf("-%d days", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		var count int
		if err := rows.Scan(&path, &count); err != nil {
			return nil, err
		}
		summary.TopPages[path] = count
	}

	// Get top referrers
	rows, err = am.db.Query(`
        SELECT referrer, COUNT(*) as count
        FROM analytics
        WHERE viewed_at >= datetime('now', ?)
        AND referrer != ''
        GROUP BY referrer
        ORDER BY count DESC
        LIMIT 10`,
		fmt.Sprintf("-%d days", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var referrer string
		var count int
		if err := rows.Scan(&referrer, &count); err != nil {
			return nil, err
		}
		summary.TopReferrers[referrer] = count
	}

	// Get daily views
	rows, err = am.db.Query(`
        SELECT date(viewed_at) as view_date, COUNT(*) as views
        FROM analytics
        WHERE viewed_at > datetime('now', '-? days')
        GROUP BY date(viewed_at)
        ORDER BY view_date`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var viewDate time.Time
		var count int
		if err := rows.Scan(&viewDate, &count); err != nil {
			return nil, err
		}
		summary.DailyViews = append(summary.DailyViews, models.DailyViews{
			Date:  viewDate,
			Count: count,
		})
	}

	return summary, nil
}
