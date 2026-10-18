// Package redisdedup is the Redis version of webhooklab.Deduper (task 5).
// It lives in its own package so the core webhooklab package (and its
// tests) do not need Redis.
package redisdedup

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// keyPrefix keeps our keys apart from anything else in the same Redis.
const keyPrefix = "webhooklab:seen:"

// Deduper remembers event ids in Redis with SET key value NX EX ttl:
//
//	NX  = only set the key if it does Not eXist yet. Redis does the check
//	      and the write as one atomic command, so with two receivers behind
//	      a load balancer only one of them gets "OK" for the same event.
//	EX  = the key EXpires after ttl seconds, so Redis does not grow
//	      forever. The ttl must be longer than the provider keeps retrying
//	      (e.g. 24h), or a very late retry would be processed again.
//
// Unlike sync.Map it survives a receiver restart and is shared by all
// instances.
type Deduper struct {
	client *redis.Client
	ttl    time.Duration
}

// New connects to Redis at addr (e.g. "localhost:6379").
func New(addr string, ttl time.Duration) *Deduper {
	return &Deduper{
		client: redis.NewClient(&redis.Options{Addr: addr}),
		ttl:    ttl,
	}
}

// Ping checks that Redis is reachable (used at startup).
func (d *Deduper) Ping(ctx context.Context) error {
	if err := d.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

// FirstSeen implements webhooklab.Deduper. go-redis sends
// "SET webhooklab:seen:<id> <now> EX <ttl> NX"; the reply is OK (true)
// when the key was created, nil (false) when it already existed.
func (d *Deduper) FirstSeen(ctx context.Context, eventID string) (bool, error) {
	created, err := d.client.SetNX(ctx, keyPrefix+eventID, time.Now().Unix(), d.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis SET NX %s: %w", eventID, err)
	}
	return created, nil
}

// Close closes the connection pool.
func (d *Deduper) Close() error { return d.client.Close() }
