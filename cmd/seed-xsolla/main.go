package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Item struct {
	ID          int
	Name        string
	Description string
	Price       int
}

type XsollaItem struct {
	SKU         string            `json:"sku"`
	Name        map[string]string `json:"name"`
	Description map[string]string `json:"description"`
	Prices      []XsollaPrice     `json:"prices"`
	IsEnabled   bool              `json:"is_enabled"`
	IsShowStore bool              `json:"is_show_in_store"`
	IsFree      bool              `json:"is_free"`
}

type XsollaPrice struct {
	Currency  string  `json:"currency"`
	Amount    float64 `json:"amount"`
	IsDefault bool    `json:"is_default"`
	IsEnabled bool    `json:"is_enabled"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("warning: .env not found; using environment variables")
	}

	databaseURL := mustGetEnv("DATABASE_URL")
	projectID := mustGetEnv("XSOLLA_PROJECT_ID")
	apiKey := mustGetEnv("XSOLLA_API_KEY")

	ctx := context.Background()

	// --------------------
	// CONNECT TO DATABASE
	// --------------------

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	// --------------------
	// READ ITEMS
	// --------------------

	rows, err := pool.Query(
		ctx,
		`
		SELECT id, name, COALESCE(description, ''), price
		FROM items
		ORDER BY id
		`,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var items []Item

	for rows.Next() {
		var item Item

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
		); err != nil {
			log.Fatal(err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d database items\n\n", len(items))

	// --------------------
	// UPLOAD TO XSOLLA
	// --------------------

	client := &http.Client{}

	for _, item := range items {
		sku := fmt.Sprintf("item_%d", item.ID)

		xsollaItem := XsollaItem{
			SKU: sku,

			Name: map[string]string{
				"en": item.Name,
			},

			Description: map[string]string{
				"en": item.Description,
			},

			Prices: []XsollaPrice{
				{
					Currency:  "USD",

					// DB stores cents:
					// 2500 -> $25.00
					Amount:    float64(item.Price) / 100,

					IsDefault: true,
					IsEnabled: true,
				},
			},

			IsEnabled:   true,
			IsShowStore: true,
			IsFree:      false,
		}

		data, err := json.Marshal(xsollaItem)
		if err != nil {
			log.Printf("❌ %s: %v\n", sku, err)
			continue
		}

		url := fmt.Sprintf(
			"https://store.xsolla.com/api/v2/project/%s/admin/items/virtual_items",
			projectID,
		)

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			url,
			bytes.NewReader(data),
		)
		if err != nil {
			log.Printf("❌ %s: %v\n", sku, err)
			continue
		}

		req.Header.Set(
			"Content-Type",
			"application/json",
		)

		// Same Store API authentication pattern
		// we're already using.
		req.SetBasicAuth(
			projectID,
			apiKey,
		)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("❌ %s: %v\n", sku, err)
			continue
		}

		responseBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if readErr != nil {
			log.Printf(
				"❌ %s: failed reading response: %v\n",
				sku,
				readErr,
			)
			continue
		}

		if resp.StatusCode < 200 ||
			resp.StatusCode >= 300 {

			log.Printf(
				"❌ %s (%s): %s\n",
				sku,
				resp.Status,
				string(responseBody),
			)

			continue
		}

		fmt.Printf(
			"✅ %s → %s ($%.2f)\n",
			sku,
			item.Name,
			float64(item.Price)/100,
		)
	}

	fmt.Println("\nFinished syncing catalog.")
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)

	if value == "" {
		log.Fatalf("%s is not set", key)
	}

	return value
}