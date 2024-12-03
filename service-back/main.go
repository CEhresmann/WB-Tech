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
	logger, err := zap.NewProduction(zap.WithCaller(true), zap.AddStacktrace(zap.ErrorLevel))
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			log.Fatalf("failed to sync logger: %v", err)
		}
	}(logger) // flushes buffer

	ctx := context.Background()
	config, err := LoadConfig("db.json")
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}
	if err := os.Setenv("DB_CONFIG", config.DBConfig); err != nil {
		fmt.Println("Error setting environment variable:", err)
		return
	}
	dbConfig := os.Getenv("DB_CONFIG")
	//fmt.Println("DB_CONFIG:", dbConfig)

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
	<-stop
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown:", zap.Error(err))
	}

	logger.Info("Server exiting")
}

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

func createOrderHandler(w http.ResponseWriter, r *http.Request, logger *zap.Logger) {
	var order Order

	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		logger.Error("Failed to decode order data", zap.Error(err))
		http.Error(w, "Invalid order data", http.StatusBadRequest)
		return
	}

	if err := ProduceOrder(order, logger); err != nil {
		logger.Error("Failed to produce order", zap.Error(err))
		http.Error(w, "Failed to produce order", http.StatusInternalServerError)
		return
	}

	logger.Info("Order created successfully", zap.String("orderID", order.OrderUID))

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(order); err != nil {
		logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
