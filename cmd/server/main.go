package main

import (
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

	// Health check
	api.HandleFunc("/health", handlers.HealthCheck).Methods("GET")

	// Scan endpoints (CRITICAL - core functionality)
	api.HandleFunc("/scans", handlers.HandleScan).Methods("POST")
	api.HandleFunc("/scans/history", handlers.GetScanHistory).Methods("GET")

	// Device endpoints
	api.HandleFunc("/devices", handlers.GetDevices).Methods("GET")
	api.HandleFunc("/devices/{id}", handlers.GetDevice).Methods("GET")
	api.HandleFunc("/devices/{id}/status", handlers.UpdateDeviceStatus).Methods("PUT")
	api.HandleFunc("/devices/{id}/movements", handlers.GetDeviceMovements).Methods("GET")

	// Zone endpoints
	api.HandleFunc("/zones", handlers.GetZones).Methods("GET")
	api.HandleFunc("/zones", handlers.CreateZone).Methods("POST")
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

	// Dashboard/stats
	api.HandleFunc("/dashboard/stats", handlers.GetDashboardStats).Methods("GET")
	api.HandleFunc("/movements", handlers.GetMovements).Methods("GET")

	// Apply middleware
	api.Use(middleware.Logger)
	api.Use(middleware.RecoveryMiddleware)

	// Serve static frontend files (after frontend is built)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/dist")))

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
