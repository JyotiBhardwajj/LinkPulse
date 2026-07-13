package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"linkpulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestResponseHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("SendSuccess", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Set("RequestID", "test-uuid-123")

		SendSuccess(c, http.StatusOK, "Success", gin.H{"key": "value"})

		assert.Equal(t, http.StatusOK, w.Code)
		var resp models.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "Success", resp.Message)
		assert.Equal(t, "test-uuid-123", resp.RequestID)

		dataMap := resp.Data.(map[string]interface{})
		assert.Equal(t, "value", dataMap["key"])
	})

	t.Run("SendPaginatedSuccess", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Set("RequestID", "test-uuid-123")

		items := []string{"item1", "item2"}
		SendPaginatedSuccess(c, http.StatusOK, "Retrieved", items, 1, 20, 100, 5)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp models.PaginatedResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "Retrieved", resp.Message)
		assert.Equal(t, "test-uuid-123", resp.RequestID)
		assert.Equal(t, 1, resp.Metadata.Page)
		assert.Equal(t, 20, resp.Metadata.PageSize)
		assert.Equal(t, int64(100), resp.Metadata.Total)
		assert.Equal(t, 5, resp.Metadata.TotalPages)
		assert.True(t, resp.Metadata.HasNext)
		assert.False(t, resp.Metadata.HasPrevious)
	})

	t.Run("SendError", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Set("RequestID", "test-uuid-123")

		SendError(c, http.StatusUnauthorized, "Invalid token", "UNAUTHORIZED")

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

		var resp models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
		assert.Equal(t, "Invalid token", resp.Error.Message)
		assert.Equal(t, "https://linkpulse.com/errors/unauthorized", resp.Type)
		assert.Equal(t, "Unauthorized", resp.Title)
		assert.Equal(t, http.StatusUnauthorized, resp.Status)
		assert.Equal(t, "test-uuid-123", resp.RequestID)
	})

	t.Run("SendValidationError", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Set("RequestID", "test-uuid-123")

		details := []models.ValidationError{
			{Field: "email", Rule: "email", Message: "Invalid email"},
		}
		SendValidationError(c, details)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

		var resp models.ValidationErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
		assert.Equal(t, "https://linkpulse.com/errors/validation-error", resp.Type)
		assert.Equal(t, "Validation Error", resp.Title)
		assert.Equal(t, 1, len(resp.Details))
		assert.Equal(t, "email", resp.Details[0].Field)
	})
}
