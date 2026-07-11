// Package service implements core domain business logic.
package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"linkpulse/internal/cache"
	"linkpulse/internal/constants"
	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/models"
	"linkpulse/internal/repository"
	"linkpulse/internal/utils"

	"github.com/google/uuid"
)

// LinkService defines the operations for shortening, resolving, and managing URLs.
type LinkService interface {
	Create(ctx context.Context, req models.CreateLinkRequest, userID *uuid.UUID) (*models.LinkResponse, error)
	Resolve(ctx context.Context, code string) (string, error)
	RecordClick(ctx context.Context, code string, details models.ClickDetails) error
	GetStats(ctx context.Context, code string, userID uuid.UUID) (*models.LinkStatsResponse, error)
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.LinkResponse, error)
	List(ctx context.Context, userID uuid.UUID, query models.ListLinksQuery) (*models.PaginationResponse[models.LinkResponse], error)
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req models.UpdateLinkRequest) (*models.LinkResponse, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

type linkService struct {
	linkRepo             repository.LinkRepository
	analyticsRepo        repository.AnalyticsRepository
	linkCache            cache.LinkCache
	shortCodeLength      int
	maxGenerationRetries int
	baseURL              string
}

// NewLinkService instantiates a new LinkService implementation.
func NewLinkService(
	linkRepo repository.LinkRepository,
	analyticsRepo repository.AnalyticsRepository,
	linkCache cache.LinkCache,
	shortCodeLength int,
	maxGenerationRetries int,
	baseURL string,
) LinkService {
	if shortCodeLength <= 0 {
		shortCodeLength = 7
	}
	if maxGenerationRetries <= 0 {
		maxGenerationRetries = 5
	}
	return &linkService{
		linkRepo:             linkRepo,
		analyticsRepo:        analyticsRepo,
		linkCache:            linkCache,
		shortCodeLength:      shortCodeLength,
		maxGenerationRetries: maxGenerationRetries,
		baseURL:              baseURL,
	}
}

// Create validates request parameters, alias checks, and persists shortened links.
func (s *linkService) Create(ctx context.Context, req models.CreateLinkRequest, userID *uuid.UUID) (*models.LinkResponse, error) {
	if !utils.IsValidURL(req.OriginalURL) {
		return nil, fmt.Errorf("%w: invalid URL format", domainErrors.ErrInvalidInput)
	}

	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("%w: expiration date must be in the future", domainErrors.ErrInvalidInput)
	}

	var shortCode string

	if req.CustomAlias != "" {
		// Enforce alias length limits
		if len(req.CustomAlias) < 3 || len(req.CustomAlias) > 50 {
			return nil, fmt.Errorf("%w: alias length must be between 3 and 50 characters", domainErrors.ErrInvalidInput)
		}

		// Enforce characters format: alphanumeric, hyphens, and underscores allowed
		matched, _ := regexp.MatchString("^[a-zA-Z0-9-_]+$", req.CustomAlias)
		if !matched {
			return nil, fmt.Errorf("%w: alias must contain only alphanumeric characters, hyphens, or underscores", domainErrors.ErrInvalidInput)
		}

		// Enforce reserved words check
		for _, reserved := range constants.ReservedAliases {
			if req.CustomAlias == reserved {
				return nil, fmt.Errorf("%w: custom alias is a reserved word", domainErrors.ErrInvalidInput)
			}
		}

		// Enforce uniqueness validation
		exists, err := s.linkRepo.ExistsAlias(ctx, req.CustomAlias)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("%w: custom alias already in use", domainErrors.ErrAlreadyExists)
		}

		shortCode = req.CustomAlias
	} else {
		// Generate random Base62 slug code and check for collisions
		for i := 0; i < s.maxGenerationRetries; i++ {
			code, err := utils.GenerateBase62Code(s.shortCodeLength)
			if err != nil {
				return nil, err
			}
			existing, err := s.linkRepo.FindByShortCode(ctx, code)
			if errors.Is(err, domainErrors.ErrNotFound) {
				shortCode = code
				break
			}
			if err != nil {
				return nil, err
			}
			if existing != nil {
				continue // Collision, retry
			}
		}
		if shortCode == "" {
			return nil, fmt.Errorf("%w: failed to generate unique short code", domainErrors.ErrInternal)
		}
	}

	link := &models.Link{
		ID:          uuid.New(),
		OriginalURL: req.OriginalURL,
		ShortCode:   shortCode,
		Title:       req.Title,
		UserID:      userID,
		IsActive:    true,
		ExpiresAt:   req.ExpiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.linkRepo.Create(ctx, link); err != nil {
		return nil, err
	}

	return &models.LinkResponse{
		ID:          link.ID,
		OriginalURL: link.OriginalURL,
		ShortCode:   link.ShortCode,
		ShortURL:    utils.BuildShortURL(s.baseURL, link.ShortCode),
		Title:       link.Title,
		ExpiresAt:   link.ExpiresAt,
		IsActive:    link.IsActive,
		ClickCount:  0,
		CreatedAt:   link.CreatedAt,
		UpdatedAt:   link.UpdatedAt,
	}, nil
}

// Resolve fetches destination and handles validation checks (bypasses Redis cache during Day 3).
func (s *linkService) Resolve(ctx context.Context, code string) (string, error) {
	link, err := s.linkRepo.FindByShortCode(ctx, code)
	if err != nil {
		return "", err
	}

	// Active check
	if !link.IsActive {
		return "", fmt.Errorf("%w: link is deactivated", domainErrors.ErrNotFound)
	}

	// Expiration check
	if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
		return "", fmt.Errorf("%w: link expired", domainErrors.ErrGone)
	}

	return link.OriginalURL, nil
}

// RecordClick persists click analytics.
func (s *linkService) RecordClick(ctx context.Context, code string, details models.ClickDetails) error {
	link, err := s.linkRepo.FindByShortCode(ctx, code)
	if err != nil {
		return err
	}

	click := &models.Analytics{
		ID:        uuid.New(),
		LinkID:    link.ID,
		ClickedAt: time.Now(),
		IPHash:    details.IPHash,
		Country:   details.Country,
		City:      details.City,
		Browser:   details.Browser,
		OS:        details.OS,
		Device:    details.Device,
		Referrer:  details.Referrer,
		UserAgent: details.UserAgent,
	}

	return s.analyticsRepo.Create(ctx, click)
}

// GetStats retrieves analytics clicks count.
func (s *linkService) GetStats(ctx context.Context, code string, userID uuid.UUID) (*models.LinkStatsResponse, error) {
	link, err := s.linkRepo.FindByShortCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if link.UserID == nil || *link.UserID != userID {
		return nil, domainErrors.ErrNotFound // returns 404 to prevent enumeration
	}

	count, err := s.analyticsRepo.GetClicksCount(ctx, link.ID)
	if err != nil {
		return nil, err
	}

	return &models.LinkStatsResponse{
		ID:          link.ID,
		ShortCode:   link.ShortCode,
		OriginalURL: link.OriginalURL,
		TotalClicks: count,
	}, nil
}

// GetByID returns details for a link owned by the user.
func (s *linkService) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.LinkResponse, error) {
	link, err := s.linkRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if link.UserID == nil || *link.UserID != userID {
		return nil, domainErrors.ErrNotFound // 404 block for security
	}

	return &models.LinkResponse{
		ID:          link.ID,
		OriginalURL: link.OriginalURL,
		ShortCode:   link.ShortCode,
		ShortURL:    utils.BuildShortURL(s.baseURL, link.ShortCode),
		Title:       link.Title,
		ExpiresAt:   link.ExpiresAt,
		IsActive:    link.IsActive,
		ClickCount:  0,
		CreatedAt:   link.CreatedAt,
		UpdatedAt:   link.UpdatedAt,
	}, nil
}

// List returns a paginated array of user-owned links.
func (s *linkService) List(ctx context.Context, userID uuid.UUID, q models.ListLinksQuery) (*models.PaginationResponse[models.LinkResponse], error) {
	// Sanitize and bound limits
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Limit <= 0 {
		q.Limit = 20
	} else if q.Limit > 100 {
		q.Limit = 100
	}

	// Validate sorting security
	switch q.Sort {
	case "created_at", "updated_at", "expires_at":
		// Whitelisted, allowed
	default:
		return nil, fmt.Errorf("%w: unsupported sort column", domainErrors.ErrInvalidInput)
	}

	links, total, err := s.linkRepo.FindByUser(ctx, userID, q)
	if err != nil {
		return nil, err
	}

	items := make([]models.LinkResponse, len(links))
	for i, l := range links {
		items[i] = models.LinkResponse{
			ID:          l.ID,
			OriginalURL: l.OriginalURL,
			ShortCode:   l.ShortCode,
			ShortURL:    utils.BuildShortURL(s.baseURL, l.ShortCode),
			Title:       l.Title,
			ExpiresAt:   l.ExpiresAt,
			IsActive:    l.IsActive,
			ClickCount:  0,
			CreatedAt:   l.CreatedAt,
			UpdatedAt:   l.UpdatedAt,
		}
	}

	totalPages := int((total + int64(q.Limit) - 1) / int64(q.Limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.PaginationResponse[models.LinkResponse]{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: totalPages,
		Items:      items,
	}, nil
}

// Update partial-modifies a link's configuration.
func (s *linkService) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req models.UpdateLinkRequest) (*models.LinkResponse, error) {
	link, err := s.linkRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if link.UserID == nil || *link.UserID != userID {
		return nil, domainErrors.ErrNotFound
	}

	if req.Title != nil {
		link.Title = *req.Title
	}
	if req.ExpiresAt != nil {
		if req.ExpiresAt.Before(time.Now()) {
			return nil, fmt.Errorf("%w: expiration date must be in the future", domainErrors.ErrInvalidInput)
		}
		link.ExpiresAt = req.ExpiresAt
	}
	if req.IsActive != nil {
		link.IsActive = *req.IsActive
	}

	link.UpdatedAt = time.Now()
	if err := s.linkRepo.Update(ctx, link); err != nil {
		return nil, err
	}

	return &models.LinkResponse{
		ID:          link.ID,
		OriginalURL: link.OriginalURL,
		ShortCode:   link.ShortCode,
		ShortURL:    utils.BuildShortURL(s.baseURL, link.ShortCode),
		Title:       link.Title,
		ExpiresAt:   link.ExpiresAt,
		IsActive:    link.IsActive,
		ClickCount:  0,
		CreatedAt:   link.CreatedAt,
		UpdatedAt:   link.UpdatedAt,
	}, nil
}

// Delete soft-deletes a link if owned by the user.
func (s *linkService) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	link, err := s.linkRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if link.UserID == nil || *link.UserID != userID {
		return domainErrors.ErrNotFound
	}

	return s.linkRepo.SoftDelete(ctx, id)
}
