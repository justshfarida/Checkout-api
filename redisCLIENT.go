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

	// MANY ITEMS
	items := map[string]map[string]interface{}{
		"item:1": {
			"name":  "T-Shirt",
			"price": 2500,
			"stock": 120,
		},
		"item:2": {
			"name":  "Developer Hoodie",
			"price": 6000,
			"stock": 45,
		},
		"item:3": {
			"name":  "Sticker Pack",
			"price": 500,
			"stock": 300,
		},
	}

	// CREATE
	for key, item := range items {
		err := client.HSet(ctx, key, item).Err()
		if err != nil {
			log.Printf("failed to insert %s: %v", key, err)
			continue
		}

		fmt.Println("Inserted:", key)
	}

	// READ
	for key := range items {
		item, err := client.HGetAll(ctx, key).Result()
		if err != nil {
			log.Printf("failed to retrieve %s: %v", key, err)
			continue
		}

		fmt.Println("Retrieved:", key, item)
	}

	//UPDATE
	err := client.HSet(ctx, "item:2", "stock", 100).Err()
	if err != nil {
		log.Printf("failed to update item:2: %v", err)
		return
	}

	updatedItem, err := client.HGetAll(ctx, "item:2").Result()
	if err != nil {
		log.Printf("failed to retrieve updated item: %v", err)
		return
	}

	fmt.Println("Updated item:2:", updatedItem)

	//DELETE
	for key := range items {
		deleted, err := client.Del(ctx, key).Result()
		if err != nil {
			log.Printf("failed to delete %s: %v", key, err)
			continue
		}

		fmt.Printf("Deleted %s: %d\n", key, deleted)
	}
}
