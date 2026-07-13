package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "default-src 'none'; frame-ancestors 'none'; sandbox", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "interest-cohort=()", w.Header().Get("Permissions-Policy"))
	assert.Equal(t, "same-origin", w.Header().Get("Cross-Origin-Resource-Policy"))
}

func TestSecurityHeaders_AppliedOnAllMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders())
	router.POST("/test", func(c *gin.Context) { c.Status(http.StatusCreated) })
	router.DELETE("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(method, "/test", nil)
		router.ServeHTTP(w, req)

		assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"), "missing header for method %s", method)
		assert.NotEmpty(t, w.Header().Get("X-Frame-Options"), "missing header for method %s", method)
		assert.NotEmpty(t, w.Header().Get("Content-Security-Policy"), "missing header for method %s", method)
	}
}
