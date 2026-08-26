package controllers

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const itemID = 1

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL must be set")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("unable to connect: %v", err)
	}
	defer pool.Close()

	fmt.Println()
	fmt.Println("SELECT ... FOR UPDATE — expect NO oversell")
	resetStock(ctx, pool, itemID, 1)
	runTwoConcurrent(ctx, pool, safeDecrement)
	printStock(ctx, pool, itemID)
}

func runTwoConcurrent(ctx context.Context, pool *pgxpool.Pool, decrement func(context.Context, *pgxpool.Pool, string) (bool, error)) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, 2)

	for _, name := range []string{"goroutine-A", "goroutine-B"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			<-start
			ok, err := decrement(ctx, pool, name)
			switch {
			case err != nil:
				results <- fmt.Sprintf("%s: ERROR: %v", name, err)
			case ok:
				results <- fmt.Sprintf("%s: SUCCESS (bought the item)", name)
			default:
				results <- fmt.Sprintf("%s: FAILED (out of stock)", name)
			}
		}(name)
	}

	close(start)
	wg.Wait()
	close(results)

	for r := range results {
		fmt.Println(" ", r)
	}
}

func safeDecrement(ctx context.Context, pool *pgxpool.Pool, caller string) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var stock int
	err = tx.QueryRow(ctx, "select stock from items where id=$1 for update", itemID).Scan(&stock)
	if err != nil {
		return false, fmt.Errorf("select for update: %w", err)
	}

	if stock <= 0 {
		return false, tx.Commit(ctx)
	}

	time.Sleep(50 * time.Millisecond)

	_, err = tx.Exec(ctx, "update items set stock = stock - 1 where id=$1", itemID)
	if err != nil {
		return false, fmt.Errorf("update: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}
	return true, nil
}

func resetStock(ctx context.Context, pool *pgxpool.Pool, id, stock int) {
	_, err := pool.Exec(ctx, "update items set stock=$1 where id=$2", stock, id)
	if err != nil {
		log.Fatalf("unable to reset stock: %v", err)
	}
}

func printStock(ctx context.Context, pool *pgxpool.Pool, id int) {
	var stock int
	err := pool.QueryRow(ctx, "select stock from items where id=$1", id).Scan(&stock)
	if err != nil {
		log.Fatalf("unable to read stock: %v", err)
	}
	if stock < 0 {
		fmt.Printf("Final stock: %d   -- OVERSOLD! Sold more than existed.\n", stock)
	} else {
		fmt.Printf("Final stock: %d\n", stock)
	}
}
