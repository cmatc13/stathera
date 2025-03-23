// cmd/orderbook/main.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/cmatc13/stathera/internal/orderbook"
	"github.com/cmatc13/stathera/pkg/config"
)

func main() {
	// Define command-line flags
	configFile := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Set up custom load options
	opts := config.DefaultLoadOptions()
	if *configFile != "" {
		opts.ConfigFile = *configFile
	}

	// Initialize configuration
	cfg, err := config.LoadWithOptions(opts)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Print configuration source for debugging
	if *configFile != "" {
		log.Printf("Configuration loaded from file: %s", *configFile)
	} else if len(os.Getenv("STATHERA_ENV")) > 0 {
		log.Println("Configuration loaded from environment variables")
	} else {
		log.Println("Configuration loaded from defaults")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize orderbook
	ob, err := orderbook.NewRedisOrderBook(cfg.Redis.Address)
	if err != nil {
		log.Fatalf("Failed to initialize orderbook: %v", err)
	}
	defer ob.Close()

	// Set up API server
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
		})
	})

	r.Get("/orderbook", func(w http.ResponseWriter, r *http.Request) {
		depthStr := r.URL.Query().Get("depth")
		depth := int64(10) // Default

		if depthStr != "" {
			if d, err := strconv.ParseInt(depthStr, 10, 64); err == nil && d > 0 {
				depth = d
			}
		}

		orderBookData, err := ob.GetOrderBook(depth)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(orderBookData)
	})

	// Start server
	// cmd/orderbook/main.go (continued)
	server := &http.Server{
		Addr:    ":" + cfg.API.Port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting orderbook API server on port %s", cfg.API.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Handle graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Println("Shutting down gracefully...")
	server.Shutdown(ctx)
	log.Println("Shutdown complete")
}
