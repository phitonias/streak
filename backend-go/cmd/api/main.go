package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phitonias/streak/internal/api"
	"github.com/phitonias/streak/internal/config"
	"github.com/phitonias/streak/internal/database"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to databases
	if err := connectDatabases(cfg); err != nil {
		log.Fatalf("Failed to connect to databases: %v", err)
	}

	// Setup router
	router := api.SetupRouter(cfg)

	// Create server
	server := &http.Server{
		Addr:           fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 Streak API server (Go) starting on port %s", cfg.Server.Port)
		log.Printf("Environment: %s", cfg.Server.Env)
		log.Printf("CORS origin: %s", cfg.CORS.AllowOrigins)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	// Close database connections
	closeDatabases()

	log.Println("Server exited")
}

func connectDatabases(cfg *config.Config) error {
	// Connect to PostgreSQL
	if err := database.ConnectPostgres(cfg); err != nil {
		return fmt.Errorf("postgres connection failed: %w", err)
	}

	// Connect to Redis
	if err := database.ConnectRedis(cfg); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}

	// Connect to MongoDB
	if err := database.ConnectMongoDB(cfg); err != nil {
		return fmt.Errorf("mongodb connection failed: %w", err)
	}

	log.Println("✅ All databases connected successfully")
	return nil
}

func closeDatabases() {
	log.Println("Closing database connections...")

	if err := database.ClosePostgres(); err != nil {
		log.Printf("Error closing PostgreSQL: %v", err)
	}

	if err := database.CloseRedis(); err != nil {
		log.Printf("Error closing Redis: %v", err)
	}

	if err := database.CloseMongoDB(); err != nil {
		log.Printf("Error closing MongoDB: %v", err)
	}

	log.Println("Database connections closed")
}
