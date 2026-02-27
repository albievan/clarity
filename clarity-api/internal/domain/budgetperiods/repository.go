package budgetperiods

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/albievan/clarity/clarity-api/internal/db"
)

// Repository defines the data-access contract for the budgetperiods domain.
type Repository interface {
	List(ctx context.Context, tenantID string, page, perPage int) ([]BudgetPeriod, int, error)
	GetByID(ctx context.Context, tenantID, id string) (*BudgetPeriod, error)
	Create(ctx context.Context, tenantID string, req CreateRequest) (*BudgetPeriod, error)
	Update(ctx context.Context, tenantID, id string, req UpdateRequest) (*BudgetPeriod, error)
	Delete(ctx context.Context, tenantID, id string) error
}

type repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) Repository {
	return &repository{db: database}
}

func (r *repository) List(ctx context.Context, tenantID string, page, perPage int) ([]BudgetPeriod, int, error) {
	// TODO: SELECT ... FROM budget_periods WHERE tenant_id=$1 LIMIT $2 OFFSET $3
	_ = tenantID
	return nil, 0, fmt.Errorf("budgetperiods.List: not implemented")
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*BudgetPeriod, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, created_at, updated_at FROM budget_periods WHERE tenant_id=$1 AND id=$2 LIMIT 1`,
		tenantID, id,
	)
	var m BudgetPeriod
	err := row.Scan(&m.ID, &m.TenantID, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("budgetperiods.GetByID: %w", err)
	}
	return &m, nil
}

func (r *repository) Create(ctx context.Context, tenantID string, req CreateRequest) (*BudgetPeriod, error) {
	// TODO: INSERT INTO budget_periods (...)
	_ = req
	return nil, fmt.Errorf("budgetperiods.Create: not implemented")
}

func (r *repository) Update(ctx context.Context, tenantID, id string, req UpdateRequest) (*BudgetPeriod, error) {
	// TODO: UPDATE budget_periods SET ... WHERE tenant_id=$1 AND id=$2
	_ = req
	return nil, fmt.Errorf("budgetperiods.Update: not implemented")
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	// TODO: soft or hard delete depending on domain rules
	return fmt.Errorf("budgetperiods.Delete: not implemented")
}
