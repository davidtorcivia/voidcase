// cmd/server/main.go
package main

import (
	"database/sql"
	_ "embed"
	"flag"
	"log"
	"net/http"

	"voidcase/internal/handlers"
	"voidcase/internal/middleware"
	"voidcase/internal/models"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func initializeDatabase(db *sql.DB, adminPassword string) error {
	// Use the embedded schema from models
	if _, err := db.Exec(models.SchemaSQL); err != nil {
		return err
	}

	// Create admin user if none exists
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), 12)
		if err != nil {
			return err
		}

		if _, err := db.Exec(`
            INSERT INTO users (username, password_hash, created_at)
            VALUES (?, ?, datetime('now'))
        `, "admin", string(hash)); err != nil {
			return err
		}

		// Updated site_config insertion with all fields
		if _, err := db.Exec(`
            INSERT INTO site_config (
                about_text, 
                contact_info, 
                tracking_code, 
                theme_name, 
                updated_at
            ) VALUES (
                '', 
                '', 
                '', 
                'default', 
                datetime('now')
            )
        `); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// Command line flags
	dbPath := flag.String("db", "./data/db/filmcms.db", "Path to SQLite database")
	adminPass := flag.String("adminpass", "admin", "Initial admin password")
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	// Initialize filesystem
	if err := initializeFileSystem(); err != nil {
		log.Fatal("Failed to initialize filesystem:", err)
	}

	// Initialize database
	db, err := sql.Open("sqlite3", *dbPath+"?_timeout=5000&_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := initializeDatabase(db, *adminPass); err != nil {
		log.Fatal(err)
	}

	// Router setup
	r := mux.NewRouter() // Move this up before using it
	r.StrictSlash(true)  // enforce trailing slashes

	// Update static file server to use new structure
	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	uploadsFs := http.FileServer(http.Dir("data/uploads"))
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", uploadsFs))

	// Initialize handlers
	projectHandler := handlers.NewProjectHandler(db)
	authHandler := handlers.NewAuthHandler(db)
	adminHandler := handlers.NewAdminHandler(db)
	tagHandler := handlers.NewTagHandler(db)
	configHandler := handlers.NewConfigHandler(db)
	pageHandler := handlers.NewPageHandler(db)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(db)
	analyticsMiddleware := middleware.NewAnalyticsMiddleware(db)

	// Set up middleware
	r.Use(csrf.Protect([]byte("32-byte-long-auth-key")))
	r.Use(analyticsMiddleware.TrackPageView)

	// Public routes
	r.HandleFunc("/", projectHandler.HomeHandler)
	r.HandleFunc("/login", authHandler.LoginHandler)
	r.HandleFunc("/logout", authHandler.LogoutHandler)
	r.HandleFunc("/tag/{tag}", tagHandler.TagHandler)
	r.HandleFunc("/about", pageHandler.AboutHandler)

	// Admin routes
	admin := r.PathPrefix("/admin").Subrouter()
	admin.Use(authMiddleware.RequireAuth)
	admin.HandleFunc("/", adminHandler.DashboardHandler)
	admin.HandleFunc("/projects", projectHandler.AdminProjectsHandler)
	admin.HandleFunc("/project/new", projectHandler.AdminNewProjectHandler)
	admin.HandleFunc("/project/{id}/edit", projectHandler.AdminEditProjectHandler)
	admin.HandleFunc("/project/{id}/delete", projectHandler.AdminDeleteProjectHandler)
	admin.HandleFunc("/settings", configHandler.AdminConfigHandler)

	log.Printf("Server starting on http://localhost:%s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, r))
}
