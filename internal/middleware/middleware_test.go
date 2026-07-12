package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"linkpulse/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		reqID := GetRequestID(c)
		c.String(http.StatusOK, reqID)
	})

	t.Run("Generates a Request ID if none is supplied", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		respHeader := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, respHeader)
		assert.Equal(t, respHeader, w.Body.String())
	})

	t.Run("Reuses the Request ID if supplied in headers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "custom-tracing-id")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		respHeader := w.Header().Get("X-Request-ID")
		assert.Equal(t, "custom-tracing-id", respHeader)
		assert.Equal(t, "custom-tracing-id", w.Body.String())
	})
}

func TestErrorMasking(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/error", func(c *gin.Context) {
		utils.SendError(c, http.StatusInternalServerError, "GORM DB error: connection reset by peer", "INTERNAL_DB_ERROR")
	})

	req, _ := http.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp.Success)
	assert.Equal(t, "INTERNAL_DB_ERROR", resp.Error.Code)
	// Verify database details are masked and generic message is returned
	assert.Equal(t, "An unexpected error occurred. Please try again later.", resp.Error.Message)
}
