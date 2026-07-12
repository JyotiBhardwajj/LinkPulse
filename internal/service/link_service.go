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
	"linkpulse/internal/logger"
	"linkpulse/internal/metrics"
	"linkpulse/internal/models"
	"linkpulse/internal/repository"
	"linkpulse/internal/utils"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

// LinkService defines the operations for shortening, resolving, and managing URLs.
type LinkService interface {
	Create(ctx context.Context, req models.CreateLinkRequest, userID *uuid.UUID) (*models.LinkResponse, error)
	Resolve(ctx context.Context, code string) (*models.CachedLink, error)
	RecordClick(ctx context.Context, code string, details models.ClickDetails) error
	GetStats(ctx context.Context, code string, userID uuid.UUID) (*models.LinkStatsResponse, error)
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.LinkResponse, error)
	List(ctx context.Context, userID uuid.UUID, query models.ListLinksQuery) (*models.PaginationResponse[models.LinkResponse], error)
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req models.UpdateLinkRequest) (*models.LinkResponse, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	DeactivateExpiredLinks(ctx context.Context) (int, error)
}

type linkService struct {
	linkRepo             repository.LinkRepository
	analyticsRepo        repository.AnalyticsRepository
	linkCache            cache.LinkCache
	shortCodeLength      int
	maxGenerationRetries int
	baseURL              string
	cacheTTL             time.Duration
	singleflightGroup    singleflight.Group
	metrics              metrics.Metrics
}

// NewLinkService instantiates a new LinkService implementation.
func NewLinkService(
	linkRepo repository.LinkRepository,
	analyticsRepo repository.AnalyticsRepository,
	linkCache cache.LinkCache,
	shortCodeLength int,
	maxGenerationRetries int,
	baseURL string,
	cacheTTL time.Duration,
	metricsTracker metrics.Metrics,
) LinkService {
	if shortCodeLength <= 0 {
		shortCodeLength = 7
	}
	if maxGenerationRetries <= 0 {
		maxGenerationRetries = 5
	}
	if cacheTTL <= 0 {
		cacheTTL = 24 * time.Hour
	}
	return &linkService{
		linkRepo:             linkRepo,
		analyticsRepo:        analyticsRepo,
		linkCache:            linkCache,
		shortCodeLength:      shortCodeLength,
		maxGenerationRetries: maxGenerationRetries,
		baseURL:              baseURL,
		cacheTTL:             cacheTTL,
		metrics:              metricsTracker,
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

	s.metrics.RecordLinkCreated()

	s.submitAudit(ctx, logger.EventLinkCreate, "links", link.ID.String(), userID)

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

// Resolve fetches destination executing the Cache-Aside pattern.
// Protects against cache stamps using singleflight.
func (s *linkService) Resolve(ctx context.Context, code string) (*models.CachedLink, error) {
	// 1. Check Redis Cache
	cached, err := s.linkCache.GetLink(ctx, code)
	if err != nil {
		// Cache error is logged in the cache layer; fallback to repository to prevent service disruption
		cached = nil
	}

	if cached != nil {
		// Cache Hit Lifecycle Checks
		if !cached.IsActive {
			return nil, fmt.Errorf("%w: link is deactivated", domainErrors.ErrNotFound)
		}
		if cached.ExpiresAt != nil && time.Now().After(*cached.ExpiresAt) {
			return nil, fmt.Errorf("%w: link expired", domainErrors.ErrGone)
		}
		return cached, nil
	}

	// 2. Cache Miss - singleflight stampede protection
	res, err, _ := s.singleflightGroup.Do(code, func() (interface{}, error) {
		// Double-check cache inside singleflight callback to handle immediate concurrent releases
		doubleCheck, err := s.linkCache.GetLink(ctx, code)
		if err == nil && doubleCheck != nil {
			return doubleCheck, nil
		}

		link, err := s.linkRepo.FindByShortCode(ctx, code)
		if err != nil {
			return nil, err
		}

		// Lifecycle Validations
		if !link.IsActive {
			return nil, fmt.Errorf("%w: link is deactivated", domainErrors.ErrNotFound)
		}
		if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
			return nil, fmt.Errorf("%w: link expired", domainErrors.ErrGone)
		}

		cachedLink := &models.CachedLink{
			ID:          link.ID,
			OriginalURL: link.OriginalURL,
			ShortCode:   link.ShortCode,
			ExpiresAt:   link.ExpiresAt,
			IsActive:    link.IsActive,
		}

		// Smarter Cache TTL: min(CACHE_TTL, remaining lifetime)
		ttl := s.cacheTTL
		if link.ExpiresAt != nil {
			remaining := time.Until(*link.ExpiresAt)
			if remaining <= 0 {
				return nil, fmt.Errorf("%w: link expired", domainErrors.ErrGone)
			}
			if remaining < ttl {
				ttl = remaining
			}
		}

		// Save to cache asynchronously to prevent blocking response loop
		go func() {
			_ = s.linkCache.SetLink(context.Background(), code, cachedLink, ttl)
		}()

		return cachedLink, nil
	})

	if err != nil {
		return nil, err
	}

	cachedLink, ok := res.(*models.CachedLink)
	if !ok {
		return nil, fmt.Errorf("%w: failed to cast cache result", domainErrors.ErrInternal)
	}

	s.metrics.RecordLinkResolved()
	return cachedLink, nil
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

// Update partial-modifies a link's configuration. Invalidates cache immediately.
func (s *linkService) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req models.UpdateLinkRequest) (*models.LinkResponse, error) {
	link, err := s.linkRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if link.UserID == nil || *link.UserID != userID {
		return nil, domainErrors.ErrNotFound
	}

	// Invalidate cache for the old short code first
	_ = s.linkCache.DeleteLink(ctx, link.ShortCode)

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

	s.metrics.RecordLinkUpdated()

	s.submitAudit(ctx, logger.EventLinkUpdate, "links", link.ID.String(), &userID)

	// Invalidate cache for the updated short code
	_ = s.linkCache.DeleteLink(ctx, link.ShortCode)

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

// Delete soft-deletes a link if owned by the user. Invalidates cache immediately.
func (s *linkService) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	link, err := s.linkRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if link.UserID == nil || *link.UserID != userID {
		return domainErrors.ErrNotFound
	}

	// Invalidate cache immediately
	_ = s.linkCache.DeleteLink(ctx, link.ShortCode)

	if err := s.linkRepo.SoftDelete(ctx, id); err != nil {
		return err
	}

	s.metrics.RecordLinkDeleted()

	s.submitAudit(ctx, logger.EventLinkDelete, "links", id.String(), &userID)
	return nil
}

// DeactivateExpiredLinks executes GORM queries to batch deactivate links and invalidates cache entries.
func (s *linkService) DeactivateExpiredLinks(ctx context.Context) (int, error) {
	links, err := s.linkRepo.DeactivateExpiredLinks(ctx)
	if err != nil {
		return 0, err
	}

	count := len(links)
	if count == 0 {
		return 0, nil
	}

	// Batch delete from cache
	for _, l := range links {
		_ = s.linkCache.DeleteLink(ctx, l.ShortCode)
	}

	return count, nil
}

func (s *linkService) submitAudit(ctx context.Context, event logger.AuditEvent, resource string, resourceID string, userID *uuid.UUID) {
	var uid uuid.UUID
	if userID != nil {
		uid = *userID
	}
	var reqID string
	if val := ctx.Value(constants.RequestIDKey); val != nil {
		if id, ok := val.(string); ok {
			reqID = id
		}
	}
	var ip string
	if val := ctx.Value(constants.ClientIPKey); val != nil {
		if clIP, ok := val.(string); ok {
			ip = clIP
		}
	}
	var ipHash string
	if ip != "" {
		ipHash = utils.HashIP(ip)
	}

	logger.GetAuditLogger().Submit(logger.AuditRecord{
		RequestID:  reqID,
		UserID:     uid,
		Event:      event,
		Resource:   resource,
		ResourceID: resourceID,
		IPHash:     ipHash,
	})
}
