package service

import (
	"context"
	"errors"
	"testing"
	"time"

	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// In-memory mock LinkRepository
type mockLinkRepo struct {
	links map[uuid.UUID]*models.Link
}

func newMockLinkRepo() *mockLinkRepo {
	return &mockLinkRepo{links: make(map[uuid.UUID]*models.Link)}
}

func (m *mockLinkRepo) Create(ctx context.Context, link *models.Link) error {
	for _, l := range m.links {
		if l.ShortCode == link.ShortCode && l.DeletedAt.Time.IsZero() {
			return domainErrors.ErrAlreadyExists
		}
	}
	m.links[link.ID] = link
	return nil
}

func (m *mockLinkRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Link, error) {
	l, exists := m.links[id]
	if !exists || !l.DeletedAt.Time.IsZero() {
		return nil, domainErrors.ErrNotFound
	}
	return l, nil
}

func (m *mockLinkRepo) FindByShortCode(ctx context.Context, code string) (*models.Link, error) {
	for _, l := range m.links {
		if l.ShortCode == code && l.DeletedAt.Time.IsZero() {
			return l, nil
		}
	}
	return nil, domainErrors.ErrNotFound
}

func (m *mockLinkRepo) FindByUser(ctx context.Context, userID uuid.UUID, q models.ListLinksQuery) ([]models.Link, int64, error) {
	var result []models.Link
	for _, l := range m.links {
		// Filter deleted
		if !l.DeletedAt.Time.IsZero() {
			if q.Status != "deleted" {
				continue
			}
		} else {
			if q.Status == "deleted" {
				continue
			}
		}

		if l.UserID == nil || *l.UserID != userID {
			continue
		}

		// Apply status
		now := time.Now()
		if q.Status == "active" {
			if !l.IsActive || (l.ExpiresAt != nil && l.ExpiresAt.Before(now)) {
				continue
			}
		} else if q.Status == "inactive" {
			if l.IsActive {
				continue
			}
		} else if q.Status == "expired" {
			if l.ExpiresAt == nil || l.ExpiresAt.After(now) {
				continue
			}
		}

		// Apply search
		if q.Search != "" {
			match := false
			if l.Title == q.Search || l.ShortCode == q.Search {
				match = true
			}
			if !match {
				continue
			}
		}

		result = append(result, *l)
	}

	total := int64(len(result))
	offset := (q.Page - 1) * q.Limit
	if offset >= len(result) {
		return []models.Link{}, total, nil
	}
	end := offset + q.Limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (m *mockLinkRepo) Update(ctx context.Context, link *models.Link) error {
	m.links[link.ID] = link
	return nil
}

func (m *mockLinkRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	l, exists := m.links[id]
	if !exists || !l.DeletedAt.Time.IsZero() {
		return domainErrors.ErrNotFound
	}
	l.DeletedAt.Time = time.Now()
	l.DeletedAt.Valid = true
	return nil
}

func (m *mockLinkRepo) ExistsAlias(ctx context.Context, alias string) (bool, error) {
	for _, l := range m.links {
		if l.ShortCode == alias && l.DeletedAt.Time.IsZero() {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockLinkRepo) DeactivateExpiredLinks(ctx context.Context) ([]models.Link, error) {
	var deactivated []models.Link
	now := time.Now()
	for _, l := range m.links {
		if l.IsActive && l.ExpiresAt != nil && l.ExpiresAt.Before(now) {
			l.IsActive = false
			deactivated = append(deactivated, *l)
		}
	}
	return deactivated, nil
}

// In-memory mock AnalyticsRepository
type mockAnalyticsRepo struct{}

func (m *mockAnalyticsRepo) Create(ctx context.Context, analytics *models.Analytics) error {
	return nil
}
func (m *mockAnalyticsRepo) GetClicksCount(ctx context.Context, linkID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockAnalyticsRepo) GetOverview(ctx context.Context, userID uuid.UUID) (*models.AnalyticsOverview, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) GetClicksOverTime(ctx context.Context, q models.AnalyticsQuery) ([]models.ClickTimeMetric, error) {
	return []models.ClickTimeMetric{}, nil
}
func (m *mockAnalyticsRepo) GetBrowserDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error) {
	return make(map[string]int64), nil
}
func (m *mockAnalyticsRepo) GetDeviceDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) GetReferrerDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) GetTopLinks(ctx context.Context, userID uuid.UUID, limit int) ([]models.TopLinkMetric, error) {
	return nil, nil
}

// Simple in-memory mock LinkCache
type mockLinkCache struct {
	store map[string]*models.CachedLink
}

func (m *mockLinkCache) GetLink(ctx context.Context, code string) (*models.CachedLink, error) {
	val, exists := m.store[code]
	if !exists {
		return nil, nil
	}
	return val, nil
}

func (m *mockLinkCache) SetLink(ctx context.Context, code string, link *models.CachedLink, ttl time.Duration) error {
	m.store[code] = link
	return nil
}

func (m *mockLinkCache) DeleteLink(ctx context.Context, code string) error {
	delete(m.store, code)
	return nil
}

func (m *mockLinkCache) Exists(ctx context.Context, code string) (bool, error) {
	_, exists := m.store[code]
	return exists, nil
}

func TestLinkService_CreateAndResolve(t *testing.T) {
	linkRepo := newMockLinkRepo()
	analyticsRepo := &mockAnalyticsRepo{}
	linkCache := &mockLinkCache{store: make(map[string]*models.CachedLink)}
	baseURL := "https://linkpulse.com"
	service := NewLinkService(linkRepo, analyticsRepo, linkCache, 7, 5, baseURL, 24*time.Hour)

	ctx := context.Background()
	userID := uuid.New()

	t.Run("Create Link with valid parameters succeeds", func(t *testing.T) {
		req := models.CreateLinkRequest{
			OriginalURL: "https://google.com",
			Title:       "Google Homepage",
		}
		resp, err := service.Create(ctx, req, &userID)
		require.NoError(t, err)
		assert.Equal(t, "https://google.com", resp.OriginalURL)
		assert.NotEmpty(t, resp.ShortCode)
		assert.Equal(t, baseURL+"/r/"+resp.ShortCode, resp.ShortURL)
		assert.True(t, resp.IsActive)
	})

	t.Run("Create Link with invalid URL fails", func(t *testing.T) {
		req := models.CreateLinkRequest{
			OriginalURL: "not-a-url",
		}
		resp, err := service.Create(ctx, req, &userID)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.Is(err, domainErrors.ErrInvalidInput))
	})

	t.Run("Create Link with custom alias succeeds", func(t *testing.T) {
		req := models.CreateLinkRequest{
			OriginalURL: "https://stripe.com",
			CustomAlias: "stripe-home",
		}
		resp, err := service.Create(ctx, req, &userID)
		require.NoError(t, err)
		assert.Equal(t, "stripe-home", resp.ShortCode)
		assert.Equal(t, baseURL+"/r/stripe-home", resp.ShortURL)
	})

	t.Run("Create Link with duplicate custom alias fails", func(t *testing.T) {
		req := models.CreateLinkRequest{
			OriginalURL: "https://stripe.com/docs",
			CustomAlias: "stripe-home", // already exists
		}
		resp, err := service.Create(ctx, req, &userID)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.Is(err, domainErrors.ErrAlreadyExists))
	})

	t.Run("Create Link with reserved alias fails", func(t *testing.T) {
		req := models.CreateLinkRequest{
			OriginalURL: "https://stripe.com",
			CustomAlias: "swagger",
		}
		resp, err := service.Create(ctx, req, &userID)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.Is(err, domainErrors.ErrInvalidInput))
	})

	t.Run("Resolve valid short code succeeds", func(t *testing.T) {
		cached, err := service.Resolve(ctx, "stripe-home")
		require.NoError(t, err)
		assert.Equal(t, "https://stripe.com", cached.OriginalURL)
	})

	t.Run("Resolve expired short code returns ErrGone", func(t *testing.T) {
		pastTime := time.Now().Add(-5 * time.Minute)
		l := &models.Link{
			ID:          uuid.New(),
			OriginalURL: "https://expired.com",
			ShortCode:   "expired-slug",
			UserID:      &userID,
			IsActive:    true,
			ExpiresAt:   &pastTime,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		_ = linkRepo.Create(ctx, l)

		dest, err := service.Resolve(ctx, "expired-slug")
		assert.Error(t, err)
		assert.Nil(t, dest)
		assert.True(t, errors.Is(err, domainErrors.ErrGone))
	})
}

func TestLinkService_Management(t *testing.T) {
	linkRepo := newMockLinkRepo()
	analyticsRepo := &mockAnalyticsRepo{}
	linkCache := &mockLinkCache{store: make(map[string]*models.CachedLink)}
	baseURL := "https://linkpulse.com"
	service := NewLinkService(linkRepo, analyticsRepo, linkCache, 7, 5, baseURL, 24*time.Hour)

	ctx := context.Background()
	userA := uuid.New()
	userB := uuid.New()

	// Seed links
	reqA := models.CreateLinkRequest{
		OriginalURL: "https://stripe.com",
		Title:       "Stripe Home",
		CustomAlias: "user-a-stripe",
	}
	respA, _ := service.Create(ctx, reqA, &userA)

	t.Run("GetByID owned link succeeds", func(t *testing.T) {
		resp, err := service.GetByID(ctx, respA.ID, userA)
		require.NoError(t, err)
		assert.Equal(t, respA.ID, resp.ID)
	})

	t.Run("GetByID unowned link returns NotFound (to prevent enumeration)", func(t *testing.T) {
		resp, err := service.GetByID(ctx, respA.ID, userB)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.Is(err, domainErrors.ErrNotFound))
	})

	t.Run("Update PATCH title succeeds", func(t *testing.T) {
		newTitle := "Stripe API Documentation"
		reqUpdate := models.UpdateLinkRequest{
			Title: &newTitle,
		}
		resp, err := service.Update(ctx, respA.ID, userA, reqUpdate)
		require.NoError(t, err)
		assert.Equal(t, "Stripe API Documentation", resp.Title)
	})

	t.Run("Soft delete owned link succeeds", func(t *testing.T) {
		err := service.Delete(ctx, respA.ID, userA)
		assert.NoError(t, err)

		// Assert it is no longer retrievable
		resp, err := service.GetByID(ctx, respA.ID, userA)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}
