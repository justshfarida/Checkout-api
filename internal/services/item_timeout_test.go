package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"checkout-api/internal/models"
)

// slowItemRepository simulates a database that never answers: it blocks
// until the caller's context is cancelled, like pgx does on a stuck query.
type slowItemRepository struct {
	fakeItemRepository
	sawDeadline bool
}

func (r *slowItemRepository) GetItemsOffset(
	ctx context.Context,
	limit int,
	offset int,
) ([]*models.Item, error) {
	_, r.sawDeadline = ctx.Deadline()
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestItemService_GetItemsOffset_TimesOutOnSlowDatabase(t *testing.T) {
	repo := &slowItemRepository{}
	service := NewItemService(repo)
	service.queryTimeout = 50 * time.Millisecond // keep the test fast

	start := time.Now()
	_, err := service.GetItemsOffset(context.Background(), 10, 0)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	if !repo.sawDeadline {
		t.Error("repository was called without a deadline on its context")
	}
	if elapsed > time.Second {
		t.Errorf("call took %v; it should give up shortly after the 50ms timeout", elapsed)
	}
}

func TestItemService_DefaultListQueryTimeoutIsSet(t *testing.T) {
	if got := NewItemService(&fakeItemRepository{}).queryTimeout; got != 2*time.Second {
		t.Errorf("default queryTimeout = %v, want 2s", got)
	}
}
