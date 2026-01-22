// =============================================================================
// MIDDLEWARE PACKAGE
// Common HTTP middleware for the API server
// =============================================================================

package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// =============================================================================
// REQUEST ID MIDDLEWARE
// =============================================================================

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		
		c.Next()
	}
}

// =============================================================================
// LOGGING MIDDLEWARE
// =============================================================================

// Logger logs HTTP requests
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		
		// Process request
		c.Next()
		
		// Skip logging for health checks
		if path == "/health" || path == "/ready" {
			return
		}
		
		latency := time.Since(start)
		status := c.Writer.Status()
		
		// Get request ID from context
		requestID, _ := c.Get("request_id")
		
		// Log fields
		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
			zap.Int("body_size", c.Writer.Size()),
		}
		
		if requestID != nil {
			fields = append(fields, zap.String("request_id", requestID.(string)))
		}
		
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.String("user_id", userID.(uuid.UUID).String()))
		}
		
		// Log errors
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				fields = append(fields, zap.String("error", e.Error()))
			}
		}
		
		// Log based on status code
		switch {
		case status >= 500:
			logger.Error("Server error", fields...)
		case status >= 400:
			logger.Warn("Client error", fields...)
		default:
			logger.Info("Request completed", fields...)
		}
	}
}

// =============================================================================
// RECOVERY MIDDLEWARE
// =============================================================================

// Recovery recovers from panics and logs them
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.Stack("stack"),
				)
				
				// Return 500 error
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal server error",
					"message": "An unexpected error occurred",
				})
			}
		}()
		
		c.Next()
	}
}

// =============================================================================
// CORS MIDDLEWARE
// =============================================================================

// CORSConfig for CORS middleware
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

// CORS enables Cross-Origin Resource Sharing
func CORS(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		allowOrigin := ""
		for _, o := range config.AllowOrigins {
			if o == "*" || o == origin {
				allowOrigin = origin
				if o == "*" && !config.AllowCredentials {
					allowOrigin = "*"
				}
				break
			}
		}
		
		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
			c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
			
			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			
			if config.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", string(rune(config.MaxAge.Seconds())))
			}
		}
		
		// Handle preflight
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// =============================================================================
// RATE LIMITING MIDDLEWARE
// =============================================================================

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow(key string) bool
}

// RateLimit limits requests based on IP or user
func RateLimit(limiter RateLimiter, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFunc(c)
		
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many requests",
				"message": "You have exceeded the rate limit. Please try again later.",
			})
			return
		}
		
		c.Next()
	}
}

// IPKeyFunc returns client IP as rate limit key
func IPKeyFunc(c *gin.Context) string {
	return c.ClientIP()
}

// UserKeyFunc returns user ID as rate limit key
func UserKeyFunc(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(uuid.UUID).String()
	}
	return c.ClientIP()
}

// =============================================================================
// SECURITY HEADERS MIDDLEWARE
// =============================================================================

// SecureHeaders adds security headers
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Content Security Policy (basic)
		c.Header("Content-Security-Policy", "default-src 'self'")
		
		// HTTPS only (if in production)
		// c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		c.Next()
	}
}

// =============================================================================
// TIMEOUT MIDDLEWARE
// =============================================================================

// Timeout aborts requests that exceed the timeout
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create context with timeout
		// Note: This is a simplified version. For production,
		// use gin-contrib/timeout or similar
		
		done := make(chan struct{})
		
		go func() {
			c.Next()
			close(done)
		}()
		
		select {
		case <-done:
			return
		case <-time.After(timeout):
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error":   "Request timeout",
				"message": "The request took too long to process",
			})
		}
	}
}

// =============================================================================
// COMPRESSION MIDDLEWARE
// =============================================================================

// Gzip enables gzip compression
// Note: For production, use gin-contrib/gzip
func Gzip() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if client accepts gzip
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}
		
		// For actual gzip, use gin-contrib/gzip
		// This is a placeholder
		c.Next()
	}
}

// =============================================================================
// REQUEST SIZE LIMIT MIDDLEWARE
// =============================================================================

// RequestSizeLimit limits the request body size
func RequestSizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// =============================================================================
// API VERSION MIDDLEWARE
// =============================================================================

// APIVersion adds API version header
func APIVersion(version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-API-Version", version)
		c.Next()
	}
}

// =============================================================================
// CONTENT TYPE MIDDLEWARE
// =============================================================================

// JSONContentType ensures JSON content type for API routes
func JSONContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Next()
	}
}

// =============================================================================
// HEALTH CHECK BYPASS
// =============================================================================

// SkipHealthCheck skips middleware for health check endpoints
func SkipHealthCheck(middleware gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/ready" {
			c.Next()
			return
		}
		middleware(c)
	}
}
