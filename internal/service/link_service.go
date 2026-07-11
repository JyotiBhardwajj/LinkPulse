// Package service implements core domain business logic.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"linkpulse/internal/cache"
	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/models"
	"linkpulse/internal/repository"
	"linkpulse/internal/utils"

	"github.com/google/uuid"
)

// LinkService defines the operations for shortening and resolving URLs.
type LinkService interface {
	Shorten(ctx context.Context, req models.ShortenLinkRequest, userID *uuid.UUID) (*models.LinkResponse, error)
	Resolve(ctx context.Context, code string) (string, error)
	RecordClick(ctx context.Context, code string, details models.ClickDetails) error
	GetStats(ctx context.Context, code string, userID uuid.UUID) (*models.LinkStatsResponse, error)
}

type linkService struct {
	linkRepo             repository.LinkRepository
	analyticsRepo        repository.AnalyticsRepository
	linkCache            cache.LinkCache
	shortCodeLength      int
	maxGenerationRetries int
}

// NewLinkService instantiates a new LinkService implementation.
func NewLinkService(
	linkRepo repository.LinkRepository,
	analyticsRepo repository.AnalyticsRepository,
	linkCache cache.LinkCache,
	shortCodeLength int,
	maxGenerationRetries int,
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
	}
}

// Shorten validates, generates a unique code, and saves the link mapping.
func (s *linkService) Shorten(ctx context.Context, req models.ShortenLinkRequest, userID *uuid.UUID) (*models.LinkResponse, error) {
	if !utils.IsValidURL(req.OriginalURL) {
		return nil, fmt.Errorf("%w: invalid URL format", domainErrors.ErrInvalidInput)
	}

	var shortCode string

	if req.CustomSlug != "" {
		// Verify custom slug availability
		existing, err := s.linkRepo.FindByCode(ctx, req.CustomSlug)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("%w: custom slug already in use", domainErrors.ErrAlreadyExists)
		}
		if err != nil && !errors.Is(err, domainErrors.ErrNotFound) {
			return nil, err
		}
		shortCode = req.CustomSlug
	} else {
		// Generate random Base62 short code and retry in case of collisions
		for i := 0; i < s.maxGenerationRetries; i++ {
			code, err := utils.GenerateBase62Code(s.shortCodeLength)
			if err != nil {
				return nil, err
			}
			existing, err := s.linkRepo.FindByCode(ctx, code)
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
			return nil, fmt.Errorf("%w: failed to generate a unique short code after multiple retries", domainErrors.ErrInternal)
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

	// Cache the newly created link mapping immediately to speed up the first resolution
	if err := s.linkCache.Set(ctx, shortCode, req.OriginalURL); err != nil {
		slog.Warn("Failed to cache shortened URL on creation", "code", shortCode, "error", err)
	}

	return &models.LinkResponse{
		ID:          link.ID,
		OriginalURL: link.OriginalURL,
		ShortCode:   link.ShortCode,
		Title:       link.Title,
		ExpiresAt:   link.ExpiresAt,
		CreatedAt:   link.CreatedAt,
	}, nil
}

// Resolve looks up the original URL from cache first, then falls back to PostgreSQL.
func (s *linkService) Resolve(ctx context.Context, code string) (string, error) {
	// 1. Cache-first lookup
	cachedURL, err := s.linkCache.Get(ctx, code)
	if err == nil && cachedURL != "" {
		return cachedURL, nil
	}

	// 2. Database fallback
	link, err := s.linkRepo.FindByCode(ctx, code)
	if err != nil {
		return "", err
	}

	// Check if active
	if !link.IsActive {
		return "", fmt.Errorf("%w: link has been deactivated", domainErrors.ErrNotFound)
	}

	// Check expiration
	if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
		return "", fmt.Errorf("%w: link has expired", domainErrors.ErrNotFound)
	}

	// 3. Populate cache for subsequent hits
	if err := s.linkCache.Set(ctx, code, link.OriginalURL); err != nil {
		slog.Warn("Failed to write to Redis cache on resolution fallback", "code", code, "error", err)
	}

	return link.OriginalURL, nil
}

// RecordClick processes and stores click analytics (placeholder designed to be run asynchronously).
func (s *linkService) RecordClick(ctx context.Context, code string, details models.ClickDetails) error {
	link, err := s.linkRepo.FindByCode(ctx, code)
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

	if err := s.analyticsRepo.Create(ctx, click); err != nil {
		slog.Error("Failed to record link click analytics", "code", code, "error", err)
		return err
	}

	return nil
}

// GetStats returns metrics for the given link.
func (s *linkService) GetStats(ctx context.Context, code string, userID uuid.UUID) (*models.LinkStatsResponse, error) {
	link, err := s.linkRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// Ensure the user owns the link to query stats (unless it's an anonymous public query, but for now we restrict)
	if link.UserID == nil || *link.UserID != userID {
		return nil, domainErrors.ErrUnauthorized
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
