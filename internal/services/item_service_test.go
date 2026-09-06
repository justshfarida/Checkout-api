package services

import (
	"context"
	"errors"
	"testing"

	"checkout-api/internal/models"
)

// ======================================================
// GET ITEM
// ======================================================

func TestItemService_GetItem(
	t *testing.T,
) {
	tests := []struct {
		name string

		id       int
		repoItem *models.Item
		repoErr  error

		wantErr error
	}{
		{
			name: "returns item successfully",

			id: 1,

			repoItem: &models.Item{
				ID:    1,
				Name:  "Sticker Pack",
				Price: 500,
				Stock: 10,
			},
		},
		{
			name: "returns ErrItemNotFound when item does not exist",

			id: 999,

			wantErr: ErrItemNotFound,
		},
		{
			name: "returns repository error",

			id:      1,
			repoErr: errFakeDependency,

			wantErr: errFakeDependency,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				// ARRANGE

				repo :=
					&fakeItemRepository{
						items: map[int]*models.Item{},
						err:   tt.repoErr,
					}

				if tt.repoItem != nil {
					repo.items[tt.repoItem.ID] =
						tt.repoItem
				}

				service :=
					NewItemService(repo)

				// ACT

				got, err :=
					service.GetItem(
						context.Background(),
						tt.id,
					)

				// ASSERT

				if !errors.Is(
					err,
					tt.wantErr,
				) {
					t.Fatalf(
						"GetItem() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				if tt.wantErr != nil {
					if got != nil {
						t.Errorf(
							"GetItem() = %#v, want nil",
							got,
						)
					}

					return
				}

				if got == nil {
					t.Fatal(
						"GetItem() returned nil",
					)
				}

				if got.ID != tt.repoItem.ID {
					t.Errorf(
						"item ID = %d, want %d",
						got.ID,
						tt.repoItem.ID,
					)
				}

				if got.Name != tt.repoItem.Name {
					t.Errorf(
						"item name = %q, want %q",
						got.Name,
						tt.repoItem.Name,
					)
				}
			},
		)
	}
}

// ======================================================
// OFFSET PAGINATION
// ======================================================

func TestItemService_GetItemsOffset(
	t *testing.T,
) {
	tests := []struct {
		name string

		limit  int
		offset int

		repoItems []*models.Item
		repoErr   error

		count    int
		countErr error

		wantErr error

		wantLimit  int
		wantOffset int
		wantTotal  int
		wantLen    int
	}{
		{
			name: "returns items with offset pagination",

			limit:  10,
			offset: 20,

			repoItems: []*models.Item{
				{ID: 21},
				{ID: 22},
			},

			count: 50,

			wantLimit:  10,
			wantOffset: 20,
			wantTotal:  50,
			wantLen:    2,
		},
		{
			name: "negative offset becomes zero",

			limit:  10,
			offset: -5,

			repoItems: []*models.Item{
				{ID: 1},
			},

			count: 1,

			wantLimit:  10,
			wantOffset: 0,
			wantTotal:  1,
			wantLen:    1,
		},
		{
			name: "returns item repository error",

			limit:  10,
			offset: 0,

			repoErr: errFakeDependency,

			wantErr: errFakeDependency,
		},
		{
			name: "returns count repository error",

			limit:  10,
			offset: 0,

			repoItems: []*models.Item{
				{ID: 1},
			},

			countErr: errFakeDependency,

			wantErr: errFakeDependency,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				// ARRANGE

				repo :=
					&fakeItemRepository{
						offsetItems: tt.repoItems,
						offsetErr:   tt.repoErr,
						count:       tt.count,
						countErr:    tt.countErr,
					}

				service :=
					NewItemService(repo)

				// ACT

				got, err :=
					service.GetItemsOffset(
						context.Background(),
						tt.limit,
						tt.offset,
					)

				// ASSERT

				if !errors.Is(
					err,
					tt.wantErr,
				) {
					t.Fatalf(
						"GetItemsOffset() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				if tt.wantErr != nil {
					return
				}

				if got.Limit !=
					tt.wantLimit {

					t.Errorf(
						"limit = %d, want %d",
						got.Limit,
						tt.wantLimit,
					)
				}

				if got.Offset !=
					tt.wantOffset {

					t.Errorf(
						"offset = %d, want %d",
						got.Offset,
						tt.wantOffset,
					)
				}

				if got.Total !=
					tt.wantTotal {

					t.Errorf(
						"total = %d, want %d",
						got.Total,
						tt.wantTotal,
					)
				}

				if len(got.Items) !=
					tt.wantLen {

					t.Errorf(
						"items length = %d, want %d",
						len(got.Items),
						tt.wantLen,
					)
				}

				// Verify normalized values were
				// actually passed to the repository.

				if repo.offsetLimit !=
					tt.wantLimit {

					t.Errorf(
						"repository limit = %d, want %d",
						repo.offsetLimit,
						tt.wantLimit,
					)
				}

				if repo.offsetValue !=
					tt.wantOffset {

					t.Errorf(
						"repository offset = %d, want %d",
						repo.offsetValue,
						tt.wantOffset,
					)
				}
			},
		)
	}
}

// ======================================================
// CURSOR PAGINATION
// ======================================================

func TestItemService_GetItemsCursor(
	t *testing.T,
) {
	tests := []struct {
		name string

		rawCursor string
		limit     int

		repoItems []*models.Item
		repoErr   error

		wantErr error

		wantLen        int
		wantHasMore    bool
		wantNextCursor bool
	}{
		{
			name: "returns page without next cursor",

			rawCursor: "",
			limit:     2,

			repoItems: []*models.Item{
				{ID: 1},
				{ID: 2},
			},

			wantLen:        2,
			wantHasMore:    false,
			wantNextCursor: false,
		},
		{
			name: "returns next cursor when more items exist",

			rawCursor: "",
			limit:     2,

			/*
				Service asks repository for limit + 1.

				Three items returned for a limit of two
				means another page exists.
			*/
			repoItems: []*models.Item{
				{ID: 1},
				{ID: 2},
				{ID: 3},
			},

			wantLen:        2,
			wantHasMore:    true,
			wantNextCursor: true,
		},
		{
			name: "returns invalid cursor error",

			rawCursor: "not-a-valid-cursor",
			limit:     2,

			wantErr: ErrInvalidCursor,
		},
		{
			name: "returns repository error",

			rawCursor: "",
			limit:     2,

			repoErr: errFakeDependency,

			wantErr: errFakeDependency,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				// ARRANGE

				repo :=
					&fakeItemRepository{
						cursorItems: tt.repoItems,
						cursorErr:   tt.repoErr,
					}

				service :=
					NewItemService(repo)

				// ACT

				got, err :=
					service.GetItemsCursor(
						context.Background(),
						tt.rawCursor,
						tt.limit,
					)

				// ASSERT

				if !errors.Is(
					err,
					tt.wantErr,
				) {
					t.Fatalf(
						"GetItemsCursor() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				if tt.wantErr != nil {
					return
				}

				if len(got.Items) !=
					tt.wantLen {

					t.Errorf(
						"items length = %d, want %d",
						len(got.Items),
						tt.wantLen,
					)
				}

				if got.HasMore !=
					tt.wantHasMore {

					t.Errorf(
						"HasMore = %v, want %v",
						got.HasMore,
						tt.wantHasMore,
					)
				}

				hasNextCursor :=
					got.NextCursor != ""

				if hasNextCursor !=
					tt.wantNextCursor {

					t.Errorf(
						"NextCursor present = %v, want %v",
						hasNextCursor,
						tt.wantNextCursor,
					)
				}

				/*
					The repository should receive limit+1.

					That extra row is how the service
					determines HasMore.
				*/
				if repo.cursorLimit !=
					tt.limit+1 {

					t.Errorf(
						"repository limit = %d, want %d",
						repo.cursorLimit,
						tt.limit+1,
					)
				}
			},
		)
	}
}
