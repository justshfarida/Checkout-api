// fake-webhook sends a signed "order_paid" webhook to the local API, exactly
// like Xsolla would, so the async flow can be tested without a real payment.
// Dev tool only: it reads XSOLLA_WEBHOOK_SECRET from .env.
//
//	go run ./cmd/fake-webhook -order 123
package main

import (
	"crypto/sha1" // #nosec G505 -- Xsolla's signature format is SHA-1(body + secret)
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	orderID := flag.Int("order", 0, "id of a PENDING order to mark as paid")
	url := flag.String("url", "http://localhost:8080/webhook", "webhook URL")
	flag.Parse()

	if *orderID <= 0 {
		log.Fatal("usage: go run ./cmd/fake-webhook -order <pending order id>")
	}

	_ = godotenv.Load()
	secret := os.Getenv("XSOLLA_WEBHOOK_SECRET")
	if secret == "" {
		log.Fatal("XSOLLA_WEBHOOK_SECRET is not set (.env)")
	}

	// A new transaction id each run, like a real new payment.
	transactionID := time.Now().UnixNano()
	body := fmt.Sprintf(
		`{"notification_type":"order_paid","custom_parameters":{"order_id":"%d"},"billing":{"transaction":{"id":%d}}}`,
		*orderID, transactionID,
	)

	sum := sha1.Sum([]byte(body + secret)) // #nosec G401 -- required by Xsolla's protocol
	req, err := http.NewRequest(http.MethodPost, *url, strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Signature "+hex.EncodeToString(sum[:]))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	reply, _ := io.ReadAll(resp.Body)

	fmt.Printf("HTTP %d %s\n", resp.StatusCode, reply)
}
