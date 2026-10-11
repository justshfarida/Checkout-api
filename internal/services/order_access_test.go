package services

import (
	"context"
	"errors"
	"testing"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
)

func TestOrderService_GetOrderForUser(t *testing.T) {
	t.Run("asks the repository for the caller's own order", func(t *testing.T) {
		repo := &fakeOrderRepository{orderForUser: &models.UserOrder{ID: 42, UserID: 7}}
		service, _ := newTestOrderService(repo, &fakeCartRepository{}, &fakeItemRepository{},
			&fakeXsollaPaymentService{}, &fakeXsollaRepository{})

		got, err := service.GetOrderForUser(context.Background(), 42, 7)
		if err != nil || got.ID != 42 {
			t.Fatalf("got %+v, %v; want order 42", got, err)
		}
		// Ownership is enforced by passing the caller's ID down to the query.
		if repo.orderForUserID != 42 || repo.orderForUserUserID != 7 {
			t.Errorf("repo called with order=%d user=%d, want 42/7",
				repo.orderForUserID, repo.orderForUserUserID)
		}
	})

	t.Run("not found or not yours becomes ErrOrderNotFound", func(t *testing.T) {
		repo := &fakeOrderRepository{orderForUserErr: repositories.ErrOrderNotFound}
		service, _ := newTestOrderService(repo, &fakeCartRepository{}, &fakeItemRepository{},
			&fakeXsollaPaymentService{}, &fakeXsollaRepository{})

		if _, err := service.GetOrderForUser(context.Background(), 42, 8); !errors.Is(err, ErrOrderNotFound) {
			t.Fatalf("err = %v, want ErrOrderNotFound", err)
		}
	})

	t.Run("database errors are passed through", func(t *testing.T) {
		repo := &fakeOrderRepository{orderForUserErr: errFakeDependency}
		service, _ := newTestOrderService(repo, &fakeCartRepository{}, &fakeItemRepository{},
			&fakeXsollaPaymentService{}, &fakeXsollaRepository{})

		if _, err := service.GetOrderForUser(context.Background(), 42, 7); !errors.Is(err, errFakeDependency) {
			t.Fatalf("err = %v, want the repository error", err)
		}
	})
}
