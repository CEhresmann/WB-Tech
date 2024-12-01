package main

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"log"
)

func start() {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "orders",
		GroupID: "order-consumer group",
	})

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("error reading message: %v", err)
		}

		var order Order
		if err := json.Unmarshal(m.Value, &order); err != nil {
			log.Printf("error unmarshaling message: %v", err)
		}

		cacheMu.Lock()
		cache[order.OrderUID] = order
		cacheMu.Unlock()

		saveOrder(order)
	}
}
