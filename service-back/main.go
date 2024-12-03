package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

var (
	cache   = make(map[string]Order)
	cacheMu sync.RWMutex
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction(zap.WithCaller(true), zap.AddStacktrace(zap.ErrorLevel))
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			log.Fatalf("failed to sync logger: %v", err)
		}
	}(logger) // flushes buffer, if any

	ctx := context.Background()

	// Load the configuration from the JSON file
	config, err := LoadConfig("../config/db.json")
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// Set the DB_CONFIG environment variable
	if err := os.Setenv("DB_CONFIG", config.DBConfig); err != nil {
		fmt.Println("Error setting environment variable:", err)
		return
	}

	// Now you can use os.Getenv("DB_CONFIG") to retrieve the value
	dbConfig := os.Getenv("DB_CONFIG")
	//fmt.Println("DB_CONFIG:", dbConfig)

	// Initialize PostgreSQL connection
	pg, err := NewPG(ctx, dbConfig, logger)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer pg.Close()

	// Create tables
	if err := pg.CreateTables(ctx); err != nil {
		logger.Fatal("Failed to create tables", zap.Error(err))
	}

	if err := LoadCacheFromDB(dbConfig, ctx); err != nil {
		logger.Fatal("Failed to load cache from database", zap.Error(err))

	}
	go Start(logger)

	http.HandleFunc("/order/", func(w http.ResponseWriter, r *http.Request) {
		orderHandle(w, r, logger)
	})
	srv := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	// Channel for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("ListenAndServe(): ", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	<-stop
	logger.Info("Shutting down server...")

	// Create a context with a timeout for the shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown:", zap.Error(err))
	}

	logger.Info("Server exiting")
}

// orderHandle handles HTTP requests for orders.
func orderHandle(w http.ResponseWriter, r *http.Request, logger *zap.Logger) {
	logger.Info("Received request", zap.String("method", r.Method), zap.String("url", r.URL.String()))
	switch r.Method {
	case http.MethodGet:
		handleGetOrder(w, r, logger)
	case http.MethodPost:
		createOrderHandler(w, r, logger)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetOrder retrieves an order from the cache.
func handleGetOrder(w http.ResponseWriter, r *http.Request, logger *zap.Logger) {
	orderUID := r.URL.Path[len("/order/"):]

	logger.Info("Received request to get order", zap.String("orderUID", orderUID))

	cacheMu.RLock()
	order, exists := cache[orderUID]
	cacheMu.RUnlock()

	if !exists {
		logger.Warn("Order not found", zap.String("orderUID", orderUID))
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		logger.Error("Failed to encode order", zap.String("orderUID", orderUID), zap.Error(err))
		http.Error(w, "Failed to encode order", http.StatusInternalServerError)
	} else {
		logger.Info("Successfully retrieved order", zap.String("orderUID", orderUID))
	}
}

// handleCreateOrder creates a new order and stores it in the cache.
func createOrderHandler(w http.ResponseWriter, r *http.Request, logger *zap.Logger) {
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid order data", http.StatusBadRequest)
		return
	}

	// Produce the order to Kafka
	if err := ProduceOrder(order, logger); err != nil {
		http.Error(w, "Failed to produce order", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}
