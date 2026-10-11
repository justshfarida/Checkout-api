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
			name: "merch quantity equal to stock succeeds",

			item: &models.Item{
				ID:       1,
				Stock:    5,
				ItemType: "merch",
			},

			quantity: 5,

			wantUpsertCalled: true,
		},
		{
			// B + I
			name: "merch quantity greater than stock returns insufficient stock",

			item: &models.Item{
				ID:       1,
				Stock:    5,
				ItemType: "merch",
			},

			quantity: 6,

			wantErr: validation.ErrInsufficientStock,

			wantUpsertCalled: false,
		},
		{
			// B: only merch is stock-limited; currency packs are unlimited
			name: "non-merch item quantity above stock is allowed",

			item: &models.Item{
				ID:       1,
				Stock:    0,
				ItemType: "currency",
			},

			quantity: 100,

			wantUpsertCalled: true,
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

// TestCartService_CartWriterDependency demonstrates dependency inversion for
// the cart database write dependency: success, error propagation, and the
// observable write side effect (the writer is called with the expected data).
func TestCartService_CartWriterDependency(t *testing.T) {
	const (
		userID   = 7
		itemID   = 11
		quantity = 2
	)

	tests := []struct {
		name     string
		writeErr error
		wantErr  error
	}{
		{
			name: "success: cart write succeeds",
		},
		{
			name:     "error: database write failure is returned",
			writeErr: errFakeDependency,
			wantErr:  errFakeDependency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &fakeCartRepository{}
			writer := &fakeCartWriter{upsertErr: tt.writeErr}
			itemRepo := &fakeItemRepository{
				items: map[int]*models.Item{
					itemID: {
						ID:    itemID,
						Stock: 10,
					},
				},
			}

			service := NewCartService(reader, writer, itemRepo)

			err := service.UpsertCartItem(
				context.Background(),
				userID,
				itemID,
				quantity,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpsertCartItem() error = %v, want %v", err, tt.wantErr)
			}

			// Side effect / interaction: the service attempted the DB write.
			if !writer.upsertCalled {
				t.Fatal("expected CartWriter.UpsertCartItem to be called")
			}
			if writer.userID != userID || writer.itemID != itemID || writer.quantity != quantity {
				t.Fatalf(
					"writer args = (userID=%d, itemID=%d, quantity=%d), want (%d, %d, %d)",
					writer.userID,
					writer.itemID,
					writer.quantity,
					userID,
					itemID,
					quantity,
				)
			}
		})
	}
}

func TestCartService_Upsert_DoesNotWrite_WhenStockIsInsufficient(t *testing.T) {
	ctx := context.Background()

	itemRepo := &fakeItemRepository{
		items: map[int]*models.Item{
			1: {
				ID:       1,
				Name:     "Keyboard",
				Price:    5000,
				Stock:    2,
				ItemType: "merch",
			},
		},
	}
	writer := &fakeCartWriter{}
	cartRepo := &fakeCartRepository{}

	service := NewCartService(
		cartRepo,
		writer,
		itemRepo,
	)

	err := service.UpsertCartItem(
		ctx,
		7, // userID
		1, // itemID
		5, // quantity — but only 2 are available
	)

	// We expect stock validation to fail.
	if !errors.Is(err, validation.ErrInsufficientStock) {
		t.Fatalf(
			"expected ErrInsufficientStock, got %v",
			err,
		)
	}

	// Most important assertion:
	// invalid operation must NOT reach the DB writer.
	if writer.upsertCalled {
		t.Fatal(
			"expected UpsertCartItem NOT to be called when stock is insufficient",
		)
	}
}

func TestCartService_GetUserCartCursor(t *testing.T) {
	cartItemsN := func(n int) []models.CartItem {
		out := make([]models.CartItem, n)
		for i := range out {
			out[i] = models.CartItem{ID: i + 1}
		}
		return out
	}

	t.Run("extra row means another page: trims and returns cursor of last item", func(t *testing.T) {
		repo := &fakeCartRepository{cursorItems: cartItemsN(3), total: 9}
		service := NewCartService(repo, &fakeCartWriter{}, &fakeItemRepository{})

		got, err := service.GetUserCartCursor(context.Background(), 1, "", 2)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if repo.cursorLimit != 3 {
			t.Errorf("repository limit = %d, want 3 (limit+1)", repo.cursorLimit)
		}
		if len(got.Items) != 2 || !got.HasMore || got.Total != 9 {
			t.Fatalf("got %d items hasMore=%v total=%d, want 2/true/9", len(got.Items), got.HasMore, got.Total)
		}
		if lastID, err := DecodeCursor(got.NextCursor); err != nil || lastID != 2 {
			t.Errorf("next cursor decodes to %d (err %v), want 2", lastID, err)
		}
	})

	t.Run("last page has no cursor and passes decoded cursor to repository", func(t *testing.T) {
		repo := &fakeCartRepository{cursorItems: cartItemsN(1)}
		service := NewCartService(repo, &fakeCartWriter{}, &fakeItemRepository{})

		got, err := service.GetUserCartCursor(context.Background(), 1, EncodeCursor(8), 2)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if repo.cursorLastID != 8 {
			t.Errorf("repository lastID = %d, want 8", repo.cursorLastID)
		}
		if got.HasMore || got.NextCursor != "" {
			t.Errorf("hasMore=%v cursor=%q, want false and empty", got.HasMore, got.NextCursor)
		}
	})

	t.Run("invalid cursor returns ErrInvalidCursor", func(t *testing.T) {
		service := NewCartService(&fakeCartRepository{}, &fakeCartWriter{}, &fakeItemRepository{})

		if _, err := service.GetUserCartCursor(context.Background(), 1, "!!", 2); !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("error = %v, want ErrInvalidCursor", err)
		}
	})

	t.Run("total error is returned", func(t *testing.T) {
		repo := &fakeCartRepository{cursorItems: cartItemsN(1), totalErr: errFakeDependency}
		service := NewCartService(repo, &fakeCartWriter{}, &fakeItemRepository{})

		if _, err := service.GetUserCartCursor(context.Background(), 1, "", 2); !errors.Is(err, errFakeDependency) {
			t.Fatalf("error = %v, want %v", err, errFakeDependency)
		}
	})
}
