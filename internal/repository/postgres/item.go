package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexusriot/gin-vagrant-demo/internal/domain"
)

// ItemRepository is the Postgres implementation of repository.ItemRepository.
type ItemRepository struct {
	db *pgxpool.Pool
}

func NewItemRepository(db *pgxpool.Pool) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) Create(ctx context.Context, item *domain.Item) (*domain.Item, error) {
	const q = `
		INSERT INTO items (title, body, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	now := time.Now().UTC()
	err := r.db.QueryRow(ctx, q, item.Title, item.Body, item.OwnerID, now, now).
		Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (r *ItemRepository) GetByID(ctx context.Context, id int64) (*domain.Item, error) {
	const q = `SELECT id, title, body, owner_id, created_at, updated_at FROM items WHERE id = $1`
	item := &domain.Item{}
	err := r.db.QueryRow(ctx, q, id).
		Scan(&item.ID, &item.Title, &item.Body, &item.OwnerID, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return item, err
}

func (r *ItemRepository) List(ctx context.Context, ownerID string, limit, offset int) ([]*domain.Item, error) {
	const q = `
		SELECT id, title, body, owner_id, created_at, updated_at
		FROM items WHERE owner_id = $1
		ORDER BY id LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, ownerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*domain.Item
	for rows.Next() {
		item := &domain.Item{}
		if err := rows.Scan(&item.ID, &item.Title, &item.Body, &item.OwnerID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ItemRepository) Update(ctx context.Context, item *domain.Item) (*domain.Item, error) {
	const q = `
		UPDATE items SET title=$1, body=$2, updated_at=$3
		WHERE id=$4 AND owner_id=$5
		RETURNING id, title, body, owner_id, created_at, updated_at`
	now := time.Now().UTC()
	out := &domain.Item{}
	err := r.db.QueryRow(ctx, q, item.Title, item.Body, now, item.ID, item.OwnerID).
		Scan(&out.ID, &out.Title, &out.Body, &out.OwnerID, &out.CreatedAt, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return out, err
}

func (r *ItemRepository) Delete(ctx context.Context, id int64, ownerID string) error {
	const q = `DELETE FROM items WHERE id=$1 AND owner_id=$2`
	tag, err := r.db.Exec(ctx, q, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ItemRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}
