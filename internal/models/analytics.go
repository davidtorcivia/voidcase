// internal/models/analytics.go
package models

import "time"

type PageView struct {
	ID        int64     `db:"id"`
	PagePath  string    `db:"page_path"`
	Referrer  string    `db:"referrer"`
	ViewedAt  time.Time `db:"viewed_at"`
	ProjectID *int64    `db:"project_id"`
	IPHash    string    `db:"ip_hash"`
}

type AnalyticsSummary struct {
	TotalViews   int            `json:"total_views"`
	UniqueViews  int            `json:"unique_views"`
	ProjectViews map[int64]int  `json:"project_views"`
	TopPages     map[string]int `json:"top_pages"`
	TopReferrers map[string]int `json:"top_referrers"`
	DailyViews   []DailyViews   `json:"daily_views"`
}

type DailyViews struct {
	Date  time.Time `json:"date"`
	Count int       `json:"count"`
}

type ReferrerCount struct {
	Referrer string
	Count    int64
}
