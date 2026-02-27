package smtypes

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/albievan/clarity/clarity-api/internal/db"
)

// Repository defines the data-access contract for the smtypes domain.
type Repository interface {
	List(ctx context.Context, tenantID string, page, perPage int) ([]SMType, int, error)
	GetByID(ctx context.Context, tenantID, id string) (*SMType, error)
	Create(ctx context.Context, tenantID string, req CreateRequest) (*SMType, error)
	Update(ctx context.Context, tenantID, id string, req UpdateRequest) (*SMType, error)
	Delete(ctx context.Context, tenantID, id string) error
}

type repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) Repository {
	return &repository{db: database}
}

func (r *repository) List(ctx context.Context, tenantID string, page, perPage int) ([]SMType, int, error) {
	// TODO: SELECT ... FROM s_m_types WHERE tenant_id=$1 LIMIT $2 OFFSET $3
	_ = tenantID
	return nil, 0, fmt.Errorf("smtypes.List: not implemented")
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*SMType, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, created_at, updated_at FROM s_m_types WHERE tenant_id=$1 AND id=$2 LIMIT 1`,
		tenantID, id,
	)
	var m SMType
	err := row.Scan(&m.ID, &m.TenantID, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("smtypes.GetByID: %w", err)
	}
	return &m, nil
}

func (r *repository) Create(ctx context.Context, tenantID string, req CreateRequest) (*SMType, error) {
	// TODO: INSERT INTO s_m_types (...)
	_ = req
	return nil, fmt.Errorf("smtypes.Create: not implemented")
}

func (r *repository) Update(ctx context.Context, tenantID, id string, req UpdateRequest) (*SMType, error) {
	// TODO: UPDATE s_m_types SET ... WHERE tenant_id=$1 AND id=$2
	_ = req
	return nil, fmt.Errorf("smtypes.Update: not implemented")
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	// TODO: soft or hard delete depending on domain rules
	return fmt.Errorf("smtypes.Delete: not implemented")
}
