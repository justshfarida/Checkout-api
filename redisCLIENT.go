package main

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

func runRedisCRUD() {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("failed to connect to Redis: %v", err)
		return
	}

	const key = "item:9"

	// CREATE
	err := client.HSet(ctx, key, map[string]interface{}{
		"name":  "Race Condition Test Item",
		"price": 100,
		"stock": 1,
	}).Err()
	if err != nil {
		log.Printf("failed to insert item: %v", err)
		return
	}

	fmt.Println("Inserted Redis item")

	// READ
	item, err := client.HGetAll(ctx, key).Result()
	if err != nil {
		log.Printf("failed to retrieve item: %v", err)
		return
	}

	fmt.Println("Retrieved:", item)

	// UPDATE
	err = client.HSet(ctx, key, "stock", 10).Err()
	if err != nil {
		log.Printf("failed to update item: %v", err)
		return
	}

	updatedItem, err := client.HGetAll(ctx, key).Result()
	if err != nil {
		log.Printf("failed to retrieve updated item: %v", err)
		return
	}

	fmt.Println("Updated:", updatedItem)

	// DELETE
	deleted, err := client.Del(ctx, key).Result()
	if err != nil {
		log.Printf("failed to delete item: %v", err)
		return
	}

	fmt.Println("Deleted keys:", deleted)
}
