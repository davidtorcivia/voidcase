-- schema.sql
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    description TEXT,
    video_embed TEXT,
    date DATE NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL COLLATE NOCASE
);

CREATE TABLE IF NOT EXISTS project_tags (
    project_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (project_id, tag_id)
);

CREATE TABLE IF NOT EXISTS images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    hash TEXT NOT NULL UNIQUE,
    path TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS site_config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    about_text TEXT DEFAULT '',
    contact_info TEXT DEFAULT '',
    tracking_code TEXT DEFAULT '',
    theme_name TEXT NOT NULL DEFAULT 'default',
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS page_views (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    page_path TEXT NOT NULL,
    referrer TEXT,
    project_id INTEGER,
    ip_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS analytics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    page_path TEXT NOT NULL,
    referrer TEXT,
    viewed_at DATETIME NOT NULL,
    project_id INTEGER REFERENCES projects(id) ON DELETE SET NULL,
    path_hash TEXT,
    browser TEXT,
    platform TEXT
);

CREATE INDEX IF NOT EXISTS idx_page_views_created_at ON page_views(created_at);
CREATE INDEX IF NOT EXISTS idx_page_views_ip_hash ON page_views(ip_hash);
CREATE INDEX IF NOT EXISTS idx_project_date ON projects(date);
CREATE INDEX IF NOT EXISTS idx_image_hash ON images(hash);
CREATE INDEX IF NOT EXISTS idx_tag_name ON tags(name);
CREATE INDEX IF NOT EXISTS idx_session_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_pageview_path ON page_views(page_path);
CREATE INDEX IF NOT EXISTS idx_pageview_referrer ON page_views(referrer);
CREATE INDEX IF NOT EXISTS idx_analytics_viewed_at ON analytics(viewed_at);
CREATE INDEX IF NOT EXISTS idx_analytics_project_id ON analytics(project_id);
CREATE INDEX IF NOT EXISTS idx_analytics_path_hash ON analytics(path_hash);
