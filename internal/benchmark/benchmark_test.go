// Package benchmark provides Go benchmark tests for critical LinkPulse performance paths.
//
// Run benchmarks with:
//
//	go test -bench=. -benchmem ./internal/benchmark/...
package benchmark

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"linkpulse/internal/auth"
	"linkpulse/internal/metrics"
	"linkpulse/internal/models"
	"linkpulse/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ─── JWT Benchmarks ─────────────────────────────────────────────────────────

const (
	benchSecret = "super-secret-jwt-key-for-benchmarking-only"
	benchIssuer = "linkpulse-bench"
)

// BenchmarkJWTGeneration measures access-token signing throughput.
func BenchmarkJWTGeneration(b *testing.B) {
	userID := uuid.New()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := auth.GenerateAccessToken(userID, "bench@example.com", models.RoleUser, benchSecret, 15*time.Minute, benchIssuer)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJWTValidation measures access-token parse+validate throughput.
func BenchmarkJWTValidation(b *testing.B) {
	userID := uuid.New()
	token, err := auth.GenerateAccessToken(userID, "bench@example.com", models.RoleUser, benchSecret, 15*time.Minute, benchIssuer)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := auth.ValidateAccessToken(token, benchSecret, benchIssuer)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ─── JSON Marshal Benchmark ──────────────────────────────────────────────────

// BenchmarkJSONMarshalLink measures marshaling a typical link response payload.
func BenchmarkJSONMarshalLink(b *testing.B) {
	linkID := uuid.New()
	resp := models.LinkResponse{
		ID:          linkID,
		OriginalURL: "https://example.com/very/long/url/that/is/being/shortened",
		ShortCode:   "abc123",
		ShortURL:    "https://lnk.example.com/abc123",
		ClickCount:  42,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ─── Worker Pool Benchmarks ──────────────────────────────────────────────────

// BenchmarkWorkerSubmit measures event submission throughput under concurrent workers.
func BenchmarkWorkerSubmit(b *testing.B) {
	m := metrics.NewNoOpMetrics()
	repo := &noOpAnalyticsRepo{}
	pool := worker.NewWorkerPool(repo, 4, 1024, m)
	ctx := context.Background()
	pool.Start(ctx)

	linkID := uuid.New()
	event := worker.ClickEvent{
		LinkID:    linkID,
		Timestamp: time.Now(),
		UserAgent: "BenchmarkClient/1.0",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = pool.Submit(ctx, event)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = pool.Shutdown(shutdownCtx)
}

// ─── HTTP Handler Benchmark ───────────────────────────────────────────────────

// BenchmarkResolveShortCode measures the HTTP handler response overhead for the
// redirect resolution path, using an in-memory stub service.
func BenchmarkResolveShortCode(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Stub redirect handler to isolate routing + JSON overhead only.
	router.GET("/r/:code", func(c *gin.Context) {
		code := c.Param("code")
		c.JSON(http.StatusOK, gin.H{
			"short_code": code,
			"long_url":   "https://example.com/destination",
		})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/r/abc123", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", w.Code)
		}
	}
}

// BenchmarkCreateLink measures the HTTP routing + JSON decode overhead for the
// link creation path, using a stub handler to isolate infrastructure cost.
func BenchmarkCreateLink(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.POST("/api/v1/links", func(c *gin.Context) {
		var req models.CreateLinkRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":           uuid.New().String(),
			"short_code":   "abc123",
			"original_url": req.OriginalURL,
		})
	})

	body := `{"original_url":"https://example.com/benchmark-link"}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/links", jsonReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
	}
}

// BenchmarkCacheLookup measures JSON unmarshal cost for a typical cache-hit response.
func BenchmarkCacheLookup(b *testing.B) {
	payload, _ := json.Marshal(models.CachedLink{
		OriginalURL: "https://example.com/destination",
		ShortCode:   "abc123",
		IsActive:    true,
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var link models.CachedLink
		if err := json.Unmarshal(payload, &link); err != nil {
			b.Fatal(err)
		}
	}
}
