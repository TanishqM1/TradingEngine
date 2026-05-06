package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/TanishqM1/Orderbook/internal/engine"
	"github.com/TanishqM1/Orderbook/internal/handlers"
	"github.com/TanishqM1/Orderbook/internal/loadbalancer"
	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"
)

// in this file, I setup the logger, engine manager, load balancer, and routes.

func main() {
	log.SetReportCaller(true)

	// Determine engine binary path
	// First check ENGINE_PATH env var, then try to find it relative to working directory
	engineBinaryPath := os.Getenv("ENGINE_PATH")
	if engineBinaryPath == "" {
		// Try common locations
		possiblePaths := []string{
			"backend/engine/server",           // from project root
			"../engine/server",                // from backend/cmd/api when using go run with correct cwd
			"../../engine/server",             // from backend/cmd/api
			"engine/server",                   // from backend/
			filepath.Join(os.Getenv("HOME"), "Desktop/prsnl/TradingEngine/backend/engine/server"), // absolute fallback
		}

		for _, path := range possiblePaths {
			absPath, err := filepath.Abs(path)
			if err != nil {
				continue
			}
			if _, err := os.Stat(absPath); err == nil {
				engineBinaryPath = absPath
				log.Infof("Found engine binary at: %s", engineBinaryPath)
				break
			}
		}
	}

	if engineBinaryPath == "" {
		log.Warn("Engine binary not found. Set ENGINE_PATH env var or ensure backend/engine/server exists.")
		log.Warn("Falling back to single-engine mode (engine must be started manually on port 6060)")
	} else {
		// Convert to absolute path
		absPath, err := filepath.Abs(engineBinaryPath)
		if err == nil {
			engineBinaryPath = absPath
		}
	}

	// Initialize engine manager and load balancer
	engineManager := engine.NewManager(engineBinaryPath)
	balancer := loadbalancer.New()

	// Initialize handlers with the manager and balancer
	handlers.InitDistributed(engineManager, balancer)

	var r *chi.Mux = chi.NewRouter()
	// setup routes
	handlers.Handler(r)
	fmt.Println("Starting Distributed Trading Engine API on :8000")
	log.Info("Go API server listening on :8000")

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\nShutting down... stopping all engines")
		engineManager.StopAllEngines()
		os.Exit(0)
	}()

	err := http.ListenAndServe("localhost:8000", r)

	// if for some reason the server does not start.
	if err != nil {
		log.Error(err)
	}
}
