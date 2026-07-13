package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestETagCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ComputeETag", func(t *testing.T) {
		data1 := gin.H{"id": 1, "name": "resource"}
		data2 := gin.H{"id": 1, "name": "resource"}
		data3 := gin.H{"id": 2, "name": "resource"}

		etag1 := ComputeETag(data1)
		etag2 := ComputeETag(data2)
		etag3 := ComputeETag(data3)

		assert.NotEmpty(t, etag1)
		assert.Equal(t, etag1, etag2, "ETag generation must be deterministic")
		assert.NotEqual(t, etag1, etag3, "ETag must differ for different payloads")
		assert.Contains(t, etag1, "W/\"", "ETag should represent a weak etag")
	})

	t.Run("CheckIfNoneMatch - Match", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		etag := "W/\"hash123\""
		c.Request.Header.Set("If-None-Match", etag)

		matched := CheckIfNoneMatch(c, etag)
		assert.True(t, matched)
	})

	t.Run("CheckIfNoneMatch - Mismatch", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("If-None-Match", "W/\"different\"")

		matched := CheckIfNoneMatch(c, "W/\"hash123\"")
		assert.False(t, matched)
	})

	t.Run("CheckIfNoneMatch - Unsafe Method", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/test", nil)
		etag := "W/\"hash123\""
		c.Request.Header.Set("If-None-Match", etag)

		matched := CheckIfNoneMatch(c, etag)
		assert.False(t, matched, "CheckIfNoneMatch must reject unsafe HTTP methods")
	})
}
