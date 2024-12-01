package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

var (
	cache   = make(map[string]Order)
	cacheMu sync.RWMutex
)

func main() {
	ctx := context.Background()

	connString := "host=localhost port=5432 user=tim password=123987 dbname=WBdata sslmode=disable"
	pg, err := NewPG(ctx, connString)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pg.Close()

	if err := pg.CreateTables(ctx); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	go start()

	http.HandleFunc("/order/", orderhandle)
	log.Println(http.ListenAndServe(":8080", nil))
}

func orderhandle(w http.ResponseWriter, r *http.Request) {
	orderID := r.URL.Path[len("/order/"):]

	cacheMu.RLock()
	order, ok := cache[orderID]
	cacheMu.RUnlock()

	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
