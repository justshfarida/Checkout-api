package store

import (
	"context"
	"fmt"
	"sync"
)

func SimulateWithoutLock(store *PostgresStore) {
	ctx := context.Background()

	if err := store.ResetStock(ctx, 9, 1); err != nil {
		fmt.Println("failed to reset stock:", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		err := store.BuyItemWithoutLock(ctx, 9)
		fmt.Println("goroutine 1:", err)
	}()

	go func() {
		defer wg.Done()

		err := store.BuyItemWithoutLock(ctx, 9)
		fmt.Println("goroutine 2:", err)
	}()

	wg.Wait()
}

func SimulateWithLock(store *PostgresStore) {
	ctx := context.Background()

	if err := store.ResetStock(ctx, 9, 1); err != nil {
		fmt.Println("failed to reset stock:", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		err := store.BuyItemWithLock(ctx, 9)
		fmt.Println("goroutine 1:", err)
	}()

	go func() {
		defer wg.Done()

		err := store.BuyItemWithLock(ctx, 9)
		fmt.Println("goroutine 2:", err)
	}()

	wg.Wait()
}
func (s *PostgresStore) ResetStock(ctx context.Context, itemID int, stock int) error {
	_, err := s.conn.Exec(ctx,
		`UPDATE items
		 SET stock = $1
		 WHERE id = $2`,
		stock,
		itemID,
	)
	return err
}
