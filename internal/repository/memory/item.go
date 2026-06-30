package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/nexusriot/gin-vagrant-demo/internal/domain"
)

// ItemRepository is a thread-safe in-memory implementation used by default and in tests.
type ItemRepository struct {
	mu      sync.RWMutex
	items   map[int64]*domain.Item
	counter int64
}

func NewItemRepository() *ItemRepository {
	return &ItemRepository{items: make(map[int64]*domain.Item)}
}

func (r *ItemRepository) Create(_ context.Context, item *domain.Item) (*domain.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counter++
	now := time.Now().UTC()
	item.ID = r.counter
	item.CreatedAt = now
	item.UpdatedAt = now
	clone := *item
	r.items[item.ID] = &clone
	return &clone, nil
}

func (r *ItemRepository) GetByID(_ context.Context, id int64) (*domain.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *item
	return &clone, nil
}

func (r *ItemRepository) List(_ context.Context, ownerID string, limit, offset int) ([]*domain.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*domain.Item
	for _, item := range r.items {
		if item.OwnerID == ownerID {
			clone := *item
			result = append(result, &clone)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	if offset >= len(result) {
		return []*domain.Item{}, nil
	}
	result = result[offset:]
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (r *ItemRepository) Update(_ context.Context, item *domain.Item) (*domain.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.items[item.ID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if existing.OwnerID != item.OwnerID {
		return nil, domain.ErrForbidden
	}
	item.CreatedAt = existing.CreatedAt
	item.UpdatedAt = time.Now().UTC()
	clone := *item
	r.items[item.ID] = &clone
	return &clone, nil
}

func (r *ItemRepository) Delete(_ context.Context, id int64, ownerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return domain.ErrNotFound
	}
	if item.OwnerID != ownerID {
		return domain.ErrForbidden
	}
	delete(r.items, id)
	return nil
}

func (r *ItemRepository) Ping(_ context.Context) error { return nil }
