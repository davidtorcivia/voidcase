package handlers

import "database/sql"

type NavigationHandler struct {
	db *sql.DB
}

func NewNavigationHandler(db *sql.DB) *NavigationHandler {
	return &NavigationHandler{db: db}
}

func (h *NavigationHandler) GetNavigation() ([]string, error) {
	// Get all unique tags ordered by core categories first, then custom tags
	rows, err := h.db.Query(`
        SELECT DISTINCT t.name 
        FROM tags t
        JOIN project_tags pt ON t.id = pt.tag_id
        ORDER BY 
            CASE 
                WHEN t.name = 'Commercial' THEN 1
                WHEN t.name = 'Narrative' THEN 2 
                WHEN t.name = 'Music Video' THEN 3
                WHEN t.name = 'Documentary' THEN 4
                ELSE 5
            END,
            t.name ASC
    `)
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
