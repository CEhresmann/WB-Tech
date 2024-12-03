package main

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func Start(logger *zap.Logger) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "orders",
		GroupID: "order-consumer-group", // Fixed the group ID format
	})

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			logger.Error("error reading message", zap.Error(err))
			continue // Skip to the next iteration on error
		}

		var order Order
		if err := json.Unmarshal(m.Value, &order); err != nil {
			logger.Error("error unmarshaling message", zap.Error(err))
			continue // Skip to the next iteration on error
		}

		cacheMu.Lock()
		cache[order.OrderUID] = order
		cacheMu.Unlock()

		if err := SaveOrder(order); err != nil {
			logger.Error("error saving order", zap.String("order_uid", order.OrderUID), zap.Error(err))
		} else {
			logger.Info("order saved successfully", zap.String("order_uid", order.OrderUID))
		}
	}
}

func ProduceOrder(order Order, logger *zap.Logger) error {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "orders",
	})

	defer writer.Close()

	orderData, err := json.Marshal(order)
	if err != nil {
		logger.Error("error marshaling order", zap.Error(err))
		return err
	}

	msg := kafka.Message{
		Key:   []byte(order.OrderUID), // You can use a unique key for partitioning
		Value: orderData,
	}

	if err := writer.WriteMessages(context.Background(), msg); err != nil {
		logger.Error("error writing message to Kafka", zap.Error(err))
		return err
	}

	logger.Info("order produced successfully", zap.String("order_uid", order.OrderUID))
	return nil
}
