package repository

import (
	"context"

	"github.com/nexusriot/gin-vagrant-demo/internal/domain"
)

// ItemRepository is the persistence interface for items.
// Both the in-memory and Postgres implementations satisfy this interface.
type ItemRepository interface {
	Create(ctx context.Context, item *domain.Item) (*domain.Item, error)
	GetByID(ctx context.Context, id int64) (*domain.Item, error)
	List(ctx context.Context, ownerID string, limit, offset int) ([]*domain.Item, error)
	Update(ctx context.Context, item *domain.Item) (*domain.Item, error)
	Delete(ctx context.Context, id int64, ownerID string) error
	// Ping verifies the backing store is reachable; used by /readyz.
	Ping(ctx context.Context) error
}
