package aijustification

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/albievan/clarity/clarity-api/internal/db"
)

// Repository defines the data-access contract for the aijustification domain.
type Repository interface {
	List(ctx context.Context, tenantID string, page, perPage int) ([]AIJustification, int, error)
	GetByID(ctx context.Context, tenantID, id string) (*AIJustification, error)
	Create(ctx context.Context, tenantID string, req CreateRequest) (*AIJustification, error)
	Update(ctx context.Context, tenantID, id string, req UpdateRequest) (*AIJustification, error)
	Delete(ctx context.Context, tenantID, id string) error
}

type repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) Repository {
	return &repository{db: database}
}

func (r *repository) List(ctx context.Context, tenantID string, page, perPage int) ([]AIJustification, int, error) {
	// TODO: SELECT ... FROM a_i_justifications WHERE tenant_id=$1 LIMIT $2 OFFSET $3
	_ = tenantID
	return nil, 0, fmt.Errorf("aijustification.List: not implemented")
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*AIJustification, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, created_at, updated_at FROM a_i_justifications WHERE tenant_id=$1 AND id=$2 LIMIT 1`,
		tenantID, id,
	)
	var m AIJustification
	err := row.Scan(&m.ID, &m.TenantID, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("aijustification.GetByID: %w", err)
	}
	return &m, nil
}

func (r *repository) Create(ctx context.Context, tenantID string, req CreateRequest) (*AIJustification, error) {
	// TODO: INSERT INTO a_i_justifications (...)
	_ = req
	return nil, fmt.Errorf("aijustification.Create: not implemented")
}

func (r *repository) Update(ctx context.Context, tenantID, id string, req UpdateRequest) (*AIJustification, error) {
	// TODO: UPDATE a_i_justifications SET ... WHERE tenant_id=$1 AND id=$2
	_ = req
	return nil, fmt.Errorf("aijustification.Update: not implemented")
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	// TODO: soft or hard delete depending on domain rules
	return fmt.Errorf("aijustification.Delete: not implemented")
}
