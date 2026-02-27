package smtypes

import (
	"context"
	"fmt"
)

// Service defines the business logic contract for the smtypes domain.
type Service interface {
	List(ctx context.Context, tenantID, userID string, page, perPage int) ([]SMType, int, error)
	Get(ctx context.Context, tenantID, userID, id string) (*SMType, error)
	Create(ctx context.Context, tenantID, userID string, req CreateRequest) (*SMType, error)
	Update(ctx context.Context, tenantID, userID, id string, req UpdateRequest) (*SMType, error)
	Delete(ctx context.Context, tenantID, userID, id string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository, _ ...any) Service {
	return &service{repo: repo}
}

func (s *service) List(ctx context.Context, tenantID, userID string, page, perPage int) ([]SMType, int, error) {
	// TODO: apply role-based visibility before calling repo
	return s.repo.List(ctx, tenantID, page, perPage)
}

func (s *service) Get(ctx context.Context, tenantID, userID, id string) (*SMType, error) {
	// TODO: enforce access control
	m, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("smtypes.Get: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("smtypes: not found")
	}
	return m, nil
}

func (s *service) Create(ctx context.Context, tenantID, userID string, req CreateRequest) (*SMType, error) {
	// TODO: validate req, enforce role, call repo.Create, write audit log
	return s.repo.Create(ctx, tenantID, req)
}

func (s *service) Update(ctx context.Context, tenantID, userID, id string, req UpdateRequest) (*SMType, error) {
	// TODO: validate req, enforce role, write audit log
	return s.repo.Update(ctx, tenantID, id, req)
}

func (s *service) Delete(ctx context.Context, tenantID, userID, id string) error {
	// TODO: enforce role, write audit log
	return s.repo.Delete(ctx, tenantID, id)
}
