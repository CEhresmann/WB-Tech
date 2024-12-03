package main

import (
	"context"
	//"encoding/json"
	"log"
)

func SaveOrder(order Order) (err error) {
	ctx := context.Background()
	pg := PgInstance

	// Start a transaction
	tx, err := pg.db.Begin(ctx)
	if err != nil {
		log.Printf("error starting transaction: %v", err)
		return err
	}

	// Check if the order already exists
	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM orders WHERE order_uid=$1)`, order.OrderUID).Scan(&exists)
	if err != nil {
		err := tx.Rollback(ctx)
		if err != nil {
			log.Fatalf("error rolling back transaction: %v", err)
		}
		log.Printf("error checking if order exists: %v", err)
		return err
	}

	if exists {
		log.Printf("Order with order_uid %s already exists, skipping insert.", order.OrderUID)
		err := tx.Commit(ctx)
		if err != nil {
			log.Fatalf("error commiting transaction: %v", err)
		}
		return err
	}

	// Insert order
	_, err = tx.Exec(ctx, `INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard) 
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature, order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		err := tx.Rollback(ctx)
		if err != nil {
			log.Fatalf("error rolling back transaction: %v", err)
		}
		log.Printf("error inserting order: %v", err)
		return err
	}

	// Insert delivery and payment as before...

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		log.Printf("error committing transaction: %v", err)
		return err
	}
	return nil
}
