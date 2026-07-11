package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"linkpulse/internal/auth"
	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/middleware"
	"linkpulse/internal/models"
	"linkpulse/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Define local mock implementations to prevent handler test cycle imports
type localUserRepo struct{}

func (m *localUserRepo) Create(ctx context.Context, user *models.User) error { return nil }
func (m *localUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return &models.User{ID: id, Email: "user@example.com"}, nil
}
func (m *localUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, nil
}

type localRefreshRepo struct{}

func (m *localRefreshRepo) Create(ctx context.Context, token *models.RefreshToken) error { return nil }
func (m *localRefreshRepo) FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	return nil, nil
}
func (m *localRefreshRepo) Revoke(ctx context.Context, hash string) error { return nil }
func (m *localRefreshRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	return nil
}

// Redefine mock repositories locally inside handler tests
type localLinkRepo struct {
	links map[uuid.UUID]*models.Link
}

func (m *localLinkRepo) Create(ctx context.Context, link *models.Link) error {
	m.links[link.ID] = link
	return nil
}
func (m *localLinkRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Link, error) {
	l, exists := m.links[id]
	if !exists {
		return nil, domainErrors.ErrNotFound
	}
	return l, nil
}
func (m *localLinkRepo) FindByShortCode(ctx context.Context, code string) (*models.Link, error) {
	for _, l := range m.links {
		if l.ShortCode == code {
			return l, nil
		}
	}
	return nil, domainErrors.ErrNotFound
}
func (m *localLinkRepo) FindByUser(ctx context.Context, userID uuid.UUID, q models.ListLinksQuery) ([]models.Link, int64, error) {
	var result []models.Link
	for _, l := range m.links {
		if l.UserID != nil && *l.UserID == userID {
			result = append(result, *l)
		}
	}
	return result, int64(len(result)), nil
}
func (m *localLinkRepo) Update(ctx context.Context, link *models.Link) error {
	m.links[link.ID] = link
	return nil
}
func (m *localLinkRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	delete(m.links, id)
	return nil
}
func (m *localLinkRepo) ExistsAlias(ctx context.Context, alias string) (bool, error) {
	for _, l := range m.links {
		if l.ShortCode == alias {
			return true, nil
		}
	}
	return false, nil
}

type localAnalyticsRepo struct{}

func (m *localAnalyticsRepo) Create(ctx context.Context, click *models.Analytics) error { return nil }
func (m *localAnalyticsRepo) GetClicksCount(ctx context.Context, linkID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *localAnalyticsRepo) GetBrowserDistribution(ctx context.Context, linkID uuid.UUID) (map[string]int64, error) {
	return nil, nil
}
func (m *localAnalyticsRepo) GetClicksOverTime(ctx context.Context, linkID uuid.UUID, interval string) ([]models.ClickTimeMetric, error) {
	return nil, nil
}

type localLinkCache struct{}

func (m *localLinkCache) Set(ctx context.Context, code string, url string) error { return nil }
func (m *localLinkCache) Get(ctx context.Context, code string) (string, error)   { return "", nil }
func (m *localLinkCache) Delete(ctx context.Context, code string) error          { return nil }

func TestLinkHandler_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &localUserRepo{}
	refreshRepo := &localRefreshRepo{}
	linkRepo := &localLinkRepo{links: make(map[uuid.UUID]*models.Link)}
	analyticsRepo := &localAnalyticsRepo{}
	linkCache := &localLinkCache{}
	secret := "handlertestsecretkeythatisreallylong"
	issuer := "linkpulse-api"
	accessTTL := 5 * time.Minute
	refreshTTL := 24 * time.Hour
	baseURL := "http://localhost:8080"

	authService := service.NewAuthService(userRepo, refreshRepo, secret, accessTTL, refreshTTL, issuer)
	linkService := service.NewLinkService(linkRepo, analyticsRepo, linkCache, 7, 5, baseURL)

	linkHandler := NewLinkHandler(linkService)

	// Router setup
	r := gin.New()
	authMiddleware := middleware.Auth(secret, issuer)

	api := r.Group("/api/v1")
	{
		links := api.Group("/links", authMiddleware)
		{
			links.POST("", linkHandler.Create)
			links.GET("", linkHandler.List)
			links.GET("/:id", linkHandler.Get)
			links.PATCH("/:id", linkHandler.Update)
			links.DELETE("/:id", linkHandler.Delete)
		}
	}
	r.GET("/r/:code", linkHandler.Resolve)

	// Forge token manually for simplicity
	claimsUserID := uuid.New()
	token, _ := auth.GenerateAccessToken(claimsUserID, "user@example.com", secret, accessTTL, issuer)
	_ = authService // Bypasses unused warning

	t.Run("POST /links creates shortened URL", func(t *testing.T) {
		reqBody, _ := json.Marshal(models.CreateLinkRequest{
			OriginalURL: "https://stripe.com",
			Title:       "Stripe Developers",
		})
		req, _ := http.NewRequest("POST", "/api/v1/links", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Success bool                `json:"success"`
			Data    models.LinkResponse `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "https://stripe.com", resp.Data.OriginalURL)
		assert.Equal(t, baseURL+"/r/"+resp.Data.ShortCode, resp.Data.ShortURL)

		// Test resolve (redirect 302 Found)
		reqResolve, _ := http.NewRequest("GET", "/r/"+resp.Data.ShortCode, nil)
		wResolve := httptest.NewRecorder()
		r.ServeHTTP(wResolve, reqResolve)

		assert.Equal(t, http.StatusFound, wResolve.Code)
		assert.Equal(t, "https://stripe.com", wResolve.Header().Get("Location"))

		// Test GET /links listing
		reqList, _ := http.NewRequest("GET", "/api/v1/links", nil)
		reqList.Header.Set("Authorization", "Bearer "+token)
		wList := httptest.NewRecorder()
		r.ServeHTTP(wList, reqList)

		assert.Equal(t, http.StatusOK, wList.Code)
		var listResp struct {
			Success bool                                           `json:"success"`
			Data    models.PaginationResponse[models.LinkResponse] `json:"data"`
		}
		_ = json.Unmarshal(wList.Body.Bytes(), &listResp)
		assert.Equal(t, int64(1), listResp.Data.Total)
		assert.Equal(t, "https://stripe.com", listResp.Data.Items[0].OriginalURL)

		// Test PATCH /links/:id
		newTitle := "Stripe API Reference"
		updateBody, _ := json.Marshal(models.UpdateLinkRequest{
			Title: &newTitle,
		})
		reqPatch, _ := http.NewRequest("PATCH", "/api/v1/links/"+resp.Data.ID.String(), bytes.NewBuffer(updateBody))
		reqPatch.Header.Set("Authorization", "Bearer "+token)
		reqPatch.Header.Set("Content-Type", "application/json")
		wPatch := httptest.NewRecorder()
		r.ServeHTTP(wPatch, reqPatch)

		assert.Equal(t, http.StatusOK, wPatch.Code)
		var patchResp struct {
			Success bool                `json:"success"`
			Data    models.LinkResponse `json:"data"`
		}
		_ = json.Unmarshal(wPatch.Body.Bytes(), &patchResp)
		assert.Equal(t, "Stripe API Reference", patchResp.Data.Title)

		// Test DELETE /links/:id (204 No Content)
		reqDelete, _ := http.NewRequest("DELETE", "/api/v1/links/"+resp.Data.ID.String(), nil)
		reqDelete.Header.Set("Authorization", "Bearer "+token)
		wDelete := httptest.NewRecorder()
		r.ServeHTTP(wDelete, reqDelete)

		assert.Equal(t, http.StatusNoContent, wDelete.Code)
	})
}
