package purchaseorders

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/albievan/clarity/clarity-api/internal/db"
)

// Repository defines the data-access contract for the purchaseorders domain.
type Repository interface {
	List(ctx context.Context, tenantID string, page, perPage int) ([]PurchaseOrder, int, error)
	GetByID(ctx context.Context, tenantID, id string) (*PurchaseOrder, error)
	Create(ctx context.Context, tenantID string, req CreateRequest) (*PurchaseOrder, error)
	Update(ctx context.Context, tenantID, id string, req UpdateRequest) (*PurchaseOrder, error)
	Delete(ctx context.Context, tenantID, id string) error
}

type repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) Repository {
	return &repository{db: database}
}

func (r *repository) List(ctx context.Context, tenantID string, page, perPage int) ([]PurchaseOrder, int, error) {
	// TODO: SELECT ... FROM purchase_orders WHERE tenant_id=$1 LIMIT $2 OFFSET $3
	_ = tenantID
	return nil, 0, fmt.Errorf("purchaseorders.List: not implemented")
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*PurchaseOrder, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, created_at, updated_at FROM purchase_orders WHERE tenant_id=$1 AND id=$2 LIMIT 1`,
		tenantID, id,
	)
	var m PurchaseOrder
	err := row.Scan(&m.ID, &m.TenantID, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("purchaseorders.GetByID: %w", err)
	}
	return &m, nil
}

func (r *repository) Create(ctx context.Context, tenantID string, req CreateRequest) (*PurchaseOrder, error) {
	// TODO: INSERT INTO purchase_orders (...)
	_ = req
	return nil, fmt.Errorf("purchaseorders.Create: not implemented")
}

func (r *repository) Update(ctx context.Context, tenantID, id string, req UpdateRequest) (*PurchaseOrder, error) {
	// TODO: UPDATE purchase_orders SET ... WHERE tenant_id=$1 AND id=$2
	_ = req
	return nil, fmt.Errorf("purchaseorders.Update: not implemented")
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	// TODO: soft or hard delete depending on domain rules
	return fmt.Errorf("purchaseorders.Delete: not implemented")
}
