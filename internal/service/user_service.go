// Package service implements core domain business logic.
package service

import (
	"context"
	"time"

	"linkpulse/internal/models"
	"linkpulse/internal/repository"
	"linkpulse/internal/utils"

	"github.com/google/uuid"
)

// UserService defines user management operations.
type UserService interface {
	Register(ctx context.Context, req models.UserRegisterRequest) (*models.UserResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.UserResponse, error)
}

type userService struct {
	userRepo repository.UserRepository
}

// NewUserService creates a new instance of UserService.
func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

// Register hashes the user password and persists the user record.
func (s *userService) Register(ctx context.Context, req models.UserRegisterRequest) (*models.UserResponse, error) {
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return &models.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}

// GetByID fetches user details by their ID.
func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*models.UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &models.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}
