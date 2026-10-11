package services

import (
	"context"
	"time"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"

	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

// defaultListQueryTimeout bounds how long listing items may wait on the
// database. A slow or stuck query fails fast instead of holding the request
// (and a DB connection) until the server's WriteTimeout.
const defaultListQueryTimeout = 2 * time.Second

type ItemService struct {
	repo         repositories.ItemRepository
	listRequests metric.Int64Counter
	queryTimeout time.Duration
}
type ItemOffsetResult struct {
	Items  []*models.Item `json:"items"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
	Total  int            `json:"total"`
}

type ItemCursorResult struct {
	Items      []*models.Item `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
	HasMore    bool           `json:"has_more"`
}

func NewItemService(
	repo repositories.ItemRepository,
) *ItemService {
	counter, err := otel.Meter("checkout-api/services").Int64Counter(
		"checkout.items.list.requests",
		metric.WithDescription("Number of item-list service calls"),
		metric.WithUnit("{request}"),
	)

	if err != nil {
		otel.Handle(err)
		counter = nil
	}

	return &ItemService{
		repo:         repo,
		listRequests: counter,
		queryTimeout: defaultListQueryTimeout,
	}
}
func (s *ItemService) GetItem(
	ctx context.Context,
	id int,
) (*models.Item, error) {
	item, err := s.repo.GetItem(
		ctx,
		id,
	)

	// Real database/infrastructure problem.
	if err != nil {
		return nil, err
	}

	// Database worked, but item doesn't exist.
	if item == nil {
		return nil, ErrItemNotFound
	}

	// Success.
	return item, nil
}
func (s *ItemService) GetItemsOffset(
	ctx context.Context,
	limit int,
	offset int,
) (ItemOffsetResult, error) {
	ctx, span := otel.Tracer("checkout-api/services").Start(
		ctx,
		"items.list",
	)
	defer span.End()

	if s.listRequests != nil {
		s.listRequests.Add(ctx, 1)
	}

	limit = NormalizeLimit(limit)
	if offset < 0 {
		offset = 0
	}

	span.SetAttributes(
		attribute.Int("pagination.limit", limit),
		attribute.Int("pagination.offset", offset),
	)

	// Both queries below (items + count) share one deadline. pgx cancels a
	// query when its context expires, returning context.DeadlineExceeded.
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	items, err := s.repo.GetItemsOffset(ctx, limit, offset)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch items")

		slog.ErrorContext(ctx, "Failed to fetch items",
			"error", err.Error(),
			"limit", limit,
			"offset", offset,
		)

		return ItemOffsetResult{}, err
	}

	total, err := s.repo.CountItems(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to count items")

		slog.ErrorContext(ctx, "Failed to count items",
			"error", err.Error(),
		)

		return ItemOffsetResult{}, err
	}

	span.SetAttributes(
		attribute.Int("items.returned", len(items)),
		attribute.Int("items.total", total),
	)

	slog.InfoContext(ctx, "Items listed",
		"limit", limit,
		"offset", offset,
		"items_returned", len(items),
		"items_total", total,
	)

	return ItemOffsetResult{
		Items:  items,
		Limit:  limit,
		Offset: offset,
		Total:  total,
	}, nil
}
func (s *ItemService) GetItemsCursor(
	ctx context.Context,
	rawCursor string,
	limit int,
) (ItemCursorResult, error) {
	limit = NormalizeLimit(limit)

	lastID, err :=
		DecodeCursor(rawCursor)

	if err != nil {
		return ItemCursorResult{}, err
	}

	items, err :=
		s.repo.GetItemsCursor(
			ctx,
			lastID,
			limit+1,
		)

	if err != nil {
		return ItemCursorResult{}, err
	}

	hasMore :=
		len(items) > limit

	if hasMore {
		items = items[:limit]
	}

	var nextCursor string

	if hasMore &&
		len(items) > 0 {

		lastItem :=
			items[len(items)-1]

		nextCursor =
			EncodeCursor(
				lastItem.ID,
			)
	}

	return ItemCursorResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
