package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"storagecore/config"
	"storagecore/internal/handlers"
	"storagecore/internal/middleware"
	"storagecore/internal/repository"
)

// spaHandler serves the SPA and falls back to index.html for client-side routes
func spaHandler(w http.ResponseWriter, r *http.Request) {
	// Build file path
	path := "./web/dist" + r.URL.Path

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist - serve index.html with injected config
		serveIndexWithConfig(w, r)
		return
	}

	// File exists - serve it
	http.FileServer(http.Dir("./web/dist")).ServeHTTP(w, r)
}

// serveIndexWithConfig injects runtime configuration into index.html
func serveIndexWithConfig(w http.ResponseWriter, r *http.Request) {
	// Read the index.html file
	file, err := os.Open("./web/dist/index.html")
	if err != nil {
		http.Error(w, "Could not open index.html", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Could not read index.html", http.StatusInternalServerError)
		return
	}

	// Get domain configuration from environment variables
	// These should be just the domain (e.g., "rent.server-nt.de")
	// without protocol or port - the frontend will add the protocol
	rentalCoreDomain := os.Getenv("RENTALCORE_DOMAIN")
	storageCoreDomain := os.Getenv("STORAGECORE_DOMAIN")

	// Create config injection script
	configScript := fmt.Sprintf(`<script>window.__APP_CONFIG__={rentalCoreDomain:"%s",storageCoreDomain:"%s"};</script>`, rentalCoreDomain, storageCoreDomain)

	// Inject the script before </head>
	modifiedContent := bytes.Replace(content, []byte("</head>"), []byte(configScript+"</head>"), 1)

	// Serve the modified content
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(modifiedContent)
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	if err := repository.InitDatabase(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer repository.CloseDatabase()

	// Setup router
	router := mux.NewRouter()

	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Auth endpoints (public - no auth required)
	api.HandleFunc("/auth/login", handlers.Login).Methods("POST")
	api.HandleFunc("/auth/logout", handlers.Logout).Methods("POST")

	// Health check (public)
	api.HandleFunc("/health", handlers.HealthCheck).Methods("GET")

	// Protected routes - apply auth middleware
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.AuthMiddleware)

	// Auth status endpoint (requires authentication)
	protected.HandleFunc("/auth/me", handlers.GetCurrentUser).Methods("GET")

	// Scan endpoints (CRITICAL - core functionality)
	protected.HandleFunc("/scans", handlers.HandleScan).Methods("POST")
	protected.HandleFunc("/scans/history", handlers.GetScanHistory).Methods("GET")

	// Device endpoints
	api.HandleFunc("/devices", handlers.GetDevices).Methods("GET")
	api.HandleFunc("/devices/{id}", handlers.GetDevice).Methods("GET")
	api.HandleFunc("/devices/{id}/status", handlers.UpdateDeviceStatus).Methods("PUT")
	api.HandleFunc("/devices/{id}/movements", handlers.GetDeviceMovements).Methods("GET")

	// Zone endpoints
	api.HandleFunc("/zones", handlers.GetZones).Methods("GET")
	api.HandleFunc("/zones", handlers.CreateZone).Methods("POST")
	api.HandleFunc("/zones/scan", handlers.GetZoneByBarcode).Methods("GET") // Zone barcode lookup
	api.HandleFunc("/zones/{id}/devices", handlers.GetZoneDevices).Methods("GET") // Must be before /zones/{id}
	api.HandleFunc("/zones/{id}", handlers.GetZone).Methods("GET")
	api.HandleFunc("/zones/{id}", handlers.UpdateZone).Methods("PUT")
	api.HandleFunc("/zones/{id}", handlers.DeleteZone).Methods("DELETE")

	// Job endpoints
	api.HandleFunc("/jobs", handlers.GetJobs).Methods("GET")
	api.HandleFunc("/jobs/{id}", handlers.GetJobSummary).Methods("GET")
	api.HandleFunc("/jobs/{id}/complete", handlers.CompleteJob).Methods("POST")

	// Case endpoints
	api.HandleFunc("/cases", handlers.GetCases).Methods("GET")
	api.HandleFunc("/cases/{id}", handlers.GetCase).Methods("GET")
	api.HandleFunc("/cases/{id}/contents", handlers.GetCaseContents).Methods("GET")

	// Maintenance endpoints
	api.HandleFunc("/defects", handlers.GetDefects).Methods("GET")
	api.HandleFunc("/defects", handlers.CreateDefect).Methods("POST")
	api.HandleFunc("/defects/{id}", handlers.UpdateDefect).Methods("PUT")
	api.HandleFunc("/maintenance/inspections", handlers.GetInspections).Methods("GET")
	api.HandleFunc("/maintenance/stats", handlers.GetMaintenanceStats).Methods("GET")

	// Dashboard/stats
	api.HandleFunc("/dashboard/stats", handlers.GetDashboardStats).Methods("GET")
	api.HandleFunc("/movements", handlers.GetMovements).Methods("GET")

	// Apply middleware
	api.Use(middleware.Logger)
	api.Use(middleware.RecoveryMiddleware)

	// Serve static frontend files with SPA fallback
	router.PathPrefix("/").HandlerFunc(spaHandler)

	// CORS setup
	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	// Server configuration
	server := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("StorageCore server starting on %s:%s", cfg.Server.Host, cfg.Server.Port)
		log.Printf("Environment: %s", cfg.App.Environment)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	repository.CloseDatabase()
	log.Println("Server stopped")
}
