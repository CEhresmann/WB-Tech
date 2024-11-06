package main

import (
	"database/sql"
	"encoding/json"
	//"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/nats-io/nats.go"
	_ "github.com/lib/pq"
)

type Order struct {
	ID        int             `json:"id"`
	OrderData json.RawMessage `json:"order_data"`
}

var (
	db      *sql.DB
	cache   = make(map[int]Order)
	cacheMu sync.RWMutex
)

func main() {
	var err error
	db, err = sql.Open("postgres", "user=user password=password dbname=orders_db sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Восстановление кэша из БД
	restoreCache()

	// Подключение к NATS
	natsConn, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer natsConn.Close()

	natsConn.Subscribe("orders", func(m *nats.Msg) {
		var order Order
		if err := json.Unmarshal(m.Data, &order); err != nil {
			log.Printf("Ошибка при распарсивании сообщения: %v", err)
			return
		}
		saveOrderToDB(order)
		cacheMu.Lock()
		cache[order.ID] = order
		cacheMu.Unlock()
	})

	http.HandleFunc("/order/", orderHandler)
	log.Println("Сервер запущен на порту 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func orderHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/order/"):]

	cacheMu.RLock()
	order, found := cache[id]
	cacheMu.RUnlock()

	if !found {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func saveOrderToDB(order Order) {
	_, err := db.Exec("INSERT INTO orders (order_data) VALUES ($1)", order.OrderData)
	if err != nil {
		log.Printf("Ошибка при сохранении заказа в БД: %v", err)
	}
}

func restoreCache() {
	rows, err := db.Query("SELECT id, order_data FROM orders")
	if err != nil {
		log.Printf("Ошибка при восстановлении кэша: %v", err)
		return