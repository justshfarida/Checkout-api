package services

import (
	"context"
	"errors"
	"testing"

	"checkout-api/internal/models"
	"checkout-api/internal/validation"
)

func TestCartService_GetUserCartOffset(
	t *testing.T,
) {
	tests := []struct {
		name string

		limit  int
		offset int

		repoItems []models.CartItem
		repoErr   error

		total    int
		totalErr error

		wantErr error

		wantLen    int
		wantLimit  int
		wantOffset int
		wantTotal  int
	}{
		{
			// Z: zero items
			name: "empty cart returns empty items",

			limit:  10,
			offset: 0,

			repoItems: []models.CartItem{},
			total:     0,

			wantLen:    0,
			wantLimit:  10,
			wantOffset: 0,
			wantTotal:  0,
		},
		{
			// O: one item
			name: "returns one cart item",

			limit:  10,
			offset: 0,

			repoItems: []models.CartItem{
				{
					ID:       1,
					Quantity: 1,
				},
			},

			total: 2500,

			wantLen:    1,
			wantLimit:  10,
			wantOffset: 0,
			wantTotal:  2500,
		},
		{
			// M: many items
			name: "returns many cart items",

			limit:  10,
			offset: 0,

			repoItems: []models.CartItem{
				{ID: 1},
				{ID: 2},
				{ID: 3},
			},

			total: 5000,

			wantLen:    3,
			wantLimit:  10,
			wantOffset: 0,
			wantTotal:  5000,
		},
		{
			// B: boundary
			name: "negative offset becomes zero",

			limit:  10,
			offset: -5,

			repoItems: []models.CartItem{
				{ID: 1},
			},

			total: 1000,

			wantLen:    1,
			wantLimit:  10,
			wantOffset: 0,
			wantTotal:  1000,
		},
		{
			// E: dependency failure
			name: "returns cart repository error",

			limit:  10,
			offset: 0,

			repoErr: errFakeDependency,
			wantErr: errFakeDependency,
		},
		{
			// E
			name: "returns total repository error",

			limit:  10,
			offset: 0,

			repoItems: []models.CartItem{
				{ID: 1},
			},

			totalErr: errFakeDependency,
			wantErr:  errFakeDependency,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				repo :=
					&fakeCartRepository{
						items:    tt.repoItems,
						getErr:   tt.repoErr,
						total:    tt.total,
						totalErr: tt.totalErr,
					}

				service :=
					NewCartService(
						repo,
						&fakeItemRepository{},
					)

				got, err :=
					service.GetUserCartOffset(
						context.Background(),
						7,
						tt.limit,
						tt.offset,
					)

				if !errors.Is(
					err,
					tt.wantErr,
				) {
					t.Fatalf(
						"GetUserCartOffset() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				if tt.wantErr != nil {
					return
				}

				if len(got.Items) != tt.wantLen {
					t.Errorf(
						"items length = %d, want %d",
						len(got.Items),
						tt.wantLen,
					)
				}

				if got.Limit != tt.wantLimit {
					t.Errorf(
						"limit = %d, want %d",
						got.Limit,
						tt.wantLimit,
					)
				}

				if got.Offset != tt.wantOffset {
					t.Errorf(
						"offset = %d, want %d",
						got.Offset,
						tt.wantOffset,
					)
				}

				if got.Total != tt.wantTotal {
					t.Errorf(
						"total = %d, want %d",
						got.Total,
						tt.wantTotal,
					)
				}

				if repo.offsetValue != tt.wantOffset {
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
func TestCartService_UpsertCartItem(
	t *testing.T,
) {
	tests := []struct {
		name string

		item     *models.Item
		itemErr  error
		quantity int

		upsertErr error

		wantErr          error
		wantUpsertCalled bool
	}{
		{
			// S + O
			name: "valid item is added successfully",

			item: &models.Item{
				ID:    1,
				Stock: 10,
			},

			quantity: 1,

			wantUpsertCalled: true,
		},
		{
			// B
			name: "quantity equal to stock succeeds",

			item: &models.Item{
				ID:    1,
				Stock: 5,
			},

			quantity: 5,

			wantUpsertCalled: true,
		},
		{
			// B + I
			name: "quantity greater than stock returns insufficient stock",

			item: &models.Item{
				ID:    1,
				Stock: 5,
			},

			quantity: 6,

			wantErr: validation.ErrInsufficientStock,

			wantUpsertCalled: false,
		},
		{
			// Z-ish / missing dependency result
			name: "missing item returns ErrItemNotFound",

			item: nil,

			quantity: 1,

			wantErr: ErrItemNotFound,

			wantUpsertCalled: false,
		},
		{
			// E
			name: "item repository error is returned",

			itemErr: errFakeDependency,

			quantity: 1,

			wantErr: errFakeDependency,

			wantUpsertCalled: false,
		},
		{
			// E
			name: "upsert repository error is returned",

			item: &models.Item{
				ID:    1,
				Stock: 10,
			},

			quantity: 1,

			upsertErr: errFakeDependency,

			wantErr: errFakeDependency,

			wantUpsertCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				const (
					userID = 7
					itemID = 1
				)

				itemRepo :=
					&fakeItemRepository{
						items: map[int]*models.Item{},
						err:   tt.itemErr,
					}

				if tt.item != nil {
					itemRepo.items[itemID] =
						tt.item
				}

				cartRepo :=
					&fakeCartRepository{
						upsertErr: tt.upsertErr,
					}

				service :=
					NewCartService(
						cartRepo,
						itemRepo,
					)

				err :=
					service.UpsertCartItem(
						context.Background(),
						userID,
						itemID,
						tt.quantity,
					)

				if !errors.Is(
					err,
					tt.wantErr,
				) {
					t.Fatalf(
						"UpsertCartItem() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				if cartRepo.upsertCalled !=
					tt.wantUpsertCalled {

					t.Errorf(
						"UpsertCartItem repository called = %v, want %v",
						cartRepo.upsertCalled,
						tt.wantUpsertCalled,
					)
				}

				if tt.wantUpsertCalled {
					if cartRepo.upsertUserID != userID {
						t.Errorf(
							"userID = %d, want %d",
							cartRepo.upsertUserID,
							userID,
						)
					}

					if cartRepo.upsertItemID != itemID {
						t.Errorf(
							"itemID = %d, want %d",
							cartRepo.upsertItemID,
							itemID,
						)
					}

					if cartRepo.upsertQuantity !=
						tt.quantity {

						t.Errorf(
							"quantity = %d, want %d",
							cartRepo.upsertQuantity,
							tt.quantity,
						)
					}
				}
			},
		)
	}
}
func TestCartService_DeleteUserCart(
	t *testing.T,
) {
	tests := []struct {
		name string

		repoErr error
		wantErr error
	}{
		{
			name: "deletes cart successfully",
		},
		{
			name:    "returns repository error",
			repoErr: errFakeDependency,
			wantErr: errFakeDependency,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				const userID = 7

				repo :=
					&fakeCartRepository{
						deleteErr: tt.repoErr,
					}

				service :=
					NewCartService(
						repo,
						&fakeItemRepository{},
					)

				err :=
					service.DeleteUserCart(
						context.Background(),
						userID,
					)

				if !errors.Is(
					err,
					tt.wantErr,
				) {
					t.Fatalf(
						"DeleteUserCart() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				if !repo.deleteCalled {
					t.Fatal(
						"DeleteUserCart repository method was not called",
					)
				}

				if repo.deletedUserID != userID {
					t.Errorf(
						"userID = %d, want %d",
						repo.deletedUserID,
						userID,
					)
				}
			},
		)
	}
}
func TestCartService_RemoveCartItem(
	t *testing.T,
) {
	tests := []struct {
		name string

		repoErr error
		wantErr error
	}{
		{
			name: "removes cart item successfully",
		},
		{
			name:    "returns repository error",
			repoErr: errFakeDependency,
			wantErr: errFakeDependency,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				const (
					userID = 7
					itemID = 3
				)

				repo :=
					&fakeCartRepository{
						removeErr: tt.repoErr,
					}

				service :=
					NewCartService(
						repo,
						&fakeItemRepository{},
					)

				err :=
					service.RemoveCartItem(
						context.Background(),
						userID,
						itemID,
					)

				if !errors.Is(
					err,
					tt.wantErr,
				) {
					t.Fatalf(
						"RemoveCartItem() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				if !repo.removeCalled {
					t.Fatal(
						"RemoveCartItem repository method was not called",
					)
				}

				if repo.removeUserID != userID {
					t.Errorf(
						"userID = %d, want %d",
						repo.removeUserID,
						userID,
					)
				}

				if repo.removeItemID != itemID {
					t.Errorf(
						"itemID = %d, want %d",
						repo.removeItemID,
						itemID,
					)
				}
			},
		)
	}
}
