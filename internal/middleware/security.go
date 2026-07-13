// Package middleware defines Gin HTTP middlewares.
package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders injects production security headers into every HTTP response.
// These headers prevent common web vulnerabilities such as MIME sniffing,
// clickjacking, and unwanted cross-origin resource sharing.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevents MIME type sniffing in browsers.
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevents this page from being embedded in iframes — stops clickjacking.
		c.Header("X-Frame-Options", "DENY")

		// Instructs browsers not to send referrer information.
		c.Header("Referrer-Policy", "no-referrer")

		// Strict CSP for an API backend: no resources should be loaded from this origin.
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; sandbox")

		// Disables browser features and APIs that are not needed for a JSON API.
		c.Header("Permissions-Policy", "interest-cohort=()")

		// Prevents other origins from reading responses to cross-origin requests.
		c.Header("Cross-Origin-Resource-Policy", "same-origin")

		c.Next()
	}
}
