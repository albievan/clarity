package agreements

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/albievan/clarity/clarity-api/internal/db"
)

// Repository defines the data-access contract for the agreements domain.
type Repository interface {
	List(ctx context.Context, tenantID string, page, perPage int) ([]Agreement, int, error)
	GetByID(ctx context.Context, tenantID, id string) (*Agreement, error)
	Create(ctx context.Context, tenantID string, req CreateRequest) (*Agreement, error)
	Update(ctx context.Context, tenantID, id string, req UpdateRequest) (*Agreement, error)
	Delete(ctx context.Context, tenantID, id string) error
}

type repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) Repository {
	return &repository{db: database}
}

func (r *repository) List(ctx context.Context, tenantID string, page, perPage int) ([]Agreement, int, error) {
	// TODO: SELECT ... FROM agreements WHERE tenant_id=$1 LIMIT $2 OFFSET $3
	_ = tenantID
	return nil, 0, fmt.Errorf("agreements.List: not implemented")
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*Agreement, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, created_at, updated_at FROM agreements WHERE tenant_id=$1 AND id=$2 LIMIT 1`,
		tenantID, id,
	)
	var m Agreement
	err := row.Scan(&m.ID, &m.TenantID, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agreements.GetByID: %w", err)
	}
	return &m, nil
}

func (r *repository) Create(ctx context.Context, tenantID string, req CreateRequest) (*Agreement, error) {
	// TODO: INSERT INTO agreements (...)
	_ = req
	return nil, fmt.Errorf("agreements.Create: not implemented")
}

func (r *repository) Update(ctx context.Context, tenantID, id string, req UpdateRequest) (*Agreement, error) {
	// TODO: UPDATE agreements SET ... WHERE tenant_id=$1 AND id=$2
	_ = req
	return nil, fmt.Errorf("agreements.Update: not implemented")
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	// TODO: soft or hard delete depending on domain rules
	return fmt.Errorf("agreements.Delete: not implemented")
}
