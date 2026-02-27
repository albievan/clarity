package budgets

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/albievan/clarity/clarity-api/internal/db"
)

// Repository defines the data-access contract for the budgets domain.
type Repository interface {
	List(ctx context.Context, tenantID string, page, perPage int) ([]Budget, int, error)
	GetByID(ctx context.Context, tenantID, id string) (*Budget, error)
	Create(ctx context.Context, tenantID string, req CreateRequest) (*Budget, error)
	Update(ctx context.Context, tenantID, id string, req UpdateRequest) (*Budget, error)
	Delete(ctx context.Context, tenantID, id string) error
}

type repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) Repository {
	return &repository{db: database}
}

func (r *repository) List(ctx context.Context, tenantID string, page, perPage int) ([]Budget, int, error) {
	// TODO: SELECT ... FROM budgets WHERE tenant_id=$1 LIMIT $2 OFFSET $3
	_ = tenantID
	return nil, 0, fmt.Errorf("budgets.List: not implemented")
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*Budget, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, created_at, updated_at FROM budgets WHERE tenant_id=$1 AND id=$2 LIMIT 1`,
		tenantID, id,
	)
	var m Budget
	err := row.Scan(&m.ID, &m.TenantID, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("budgets.GetByID: %w", err)
	}
	return &m, nil
}

func (r *repository) Create(ctx context.Context, tenantID string, req CreateRequest) (*Budget, error) {
	// TODO: INSERT INTO budgets (...)
	_ = req
	return nil, fmt.Errorf("budgets.Create: not implemented")
}

func (r *repository) Update(ctx context.Context, tenantID, id string, req UpdateRequest) (*Budget, error) {
	// TODO: UPDATE budgets SET ... WHERE tenant_id=$1 AND id=$2
	_ = req
	return nil, fmt.Errorf("budgets.Update: not implemented")
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	// TODO: soft or hard delete depending on domain rules
	return fmt.Errorf("budgets.Delete: not implemented")
}
