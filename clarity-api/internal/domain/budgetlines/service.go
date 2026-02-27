package budgetlines

import (
	"context"
	"fmt"
)

// Service defines the business logic contract for the budgetlines domain.
type Service interface {
	List(ctx context.Context, tenantID, userID string, page, perPage int) ([]BudgetLine, int, error)
	Get(ctx context.Context, tenantID, userID, id string) (*BudgetLine, error)
	Create(ctx context.Context, tenantID, userID string, req CreateRequest) (*BudgetLine, error)
	Update(ctx context.Context, tenantID, userID, id string, req UpdateRequest) (*BudgetLine, error)
	Delete(ctx context.Context, tenantID, userID, id string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository, _ ...any) Service {
	return &service{repo: repo}
}

func (s *service) List(ctx context.Context, tenantID, userID string, page, perPage int) ([]BudgetLine, int, error) {
	// TODO: apply role-based visibility before calling repo
	return s.repo.List(ctx, tenantID, page, perPage)
}

func (s *service) Get(ctx context.Context, tenantID, userID, id string) (*BudgetLine, error) {
	// TODO: enforce access control
	m, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("budgetlines.Get: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("budgetlines: not found")
	}
	return m, nil
}

func (s *service) Create(ctx context.Context, tenantID, userID string, req CreateRequest) (*BudgetLine, error) {
	// TODO: validate req, enforce role, call repo.Create, write audit log
	return s.repo.Create(ctx, tenantID, req)
}

func (s *service) Update(ctx context.Context, tenantID, userID, id string, req UpdateRequest) (*BudgetLine, error) {
	// TODO: validate req, enforce role, write audit log
	return s.repo.Update(ctx, tenantID, id, req)
}

func (s *service) Delete(ctx context.Context, tenantID, userID, id string) error {
	// TODO: enforce role, write audit log
	return s.repo.Delete(ctx, tenantID, id)
}
