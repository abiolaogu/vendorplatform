// =============================================================================
// INTEGRATION TESTS
// Tests API endpoints with real database connections
// =============================================================================

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// =============================================================================
// TEST SUITE
// =============================================================================

type IntegrationTestSuite struct {
	suite.Suite
	db     *pgxpool.Pool
	cache  *redis.Client
	router *gin.Engine
}

func (s *IntegrationTestSuite) SetupSuite() {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		s.T().Skip("Skipping integration tests. Set INTEGRATION_TEST=true to run.")
	}

	ctx := context.Background()

	// Connect to test database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://vendorplatform:vendorplatform@localhost:5432/vendorplatform_test?sslmode=disable"
	}

	var err error
	s.db, err = pgxpool.New(ctx, dbURL)
	s.Require().NoError(err)

	// Connect to Redis
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/1"
	}

	opt, err := redis.ParseURL(redisURL)
	s.Require().NoError(err)
	s.cache = redis.NewClient(opt)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	s.router = gin.New()
	
	// Register routes
	s.setupRoutes()
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
	if s.cache != nil {
		s.cache.Close()
	}
}

func (s *IntegrationTestSuite) SetupTest() {
	// Clean up test data before each test
	ctx := context.Background()
	s.db.Exec(ctx, "TRUNCATE users, sessions, transactions, bookings CASCADE")
	s.cache.FlushDB(ctx)
}

func (s *IntegrationTestSuite) setupRoutes() {
	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	api := s.router.Group("/api/v1")
	
	// Auth routes
	auth := api.Group("/auth")
	{
		auth.POST("/register", s.handleRegister)
		auth.POST("/login", s.handleLogin)
	}
	
	// Vendor routes
	vendors := api.Group("/vendors")
	{
		vendors.GET("", s.handleListVendors)
		vendors.GET("/:id", s.handleGetVendor)
	}
}

// =============================================================================
// HEALTH CHECK TESTS
// =============================================================================

func (s *IntegrationTestSuite) TestHealthCheck() {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusOK, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("healthy", response["status"])
}

// =============================================================================
// AUTHENTICATION TESTS
// =============================================================================

func (s *IntegrationTestSuite) TestUserRegistration() {
	payload := map[string]interface{}{
		"email":      "test@example.com",
		"password":   "SecurePassword123!",
		"first_name": "Test",
		"last_name":  "User",
	}
	
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("test@example.com", response["email"])
}

func (s *IntegrationTestSuite) TestUserRegistrationDuplicateEmail() {
	// First registration
	payload := map[string]interface{}{
		"email":      "duplicate@example.com",
		"password":   "SecurePassword123!",
		"first_name": "Test",
		"last_name":  "User",
	}
	
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusCreated, w.Code)
	
	// Second registration with same email
	req = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusConflict, w.Code)
}

func (s *IntegrationTestSuite) TestUserLogin() {
	// Register user first
	s.registerTestUser("login@example.com", "SecurePassword123!")
	
	// Login
	payload := map[string]interface{}{
		"email":    "login@example.com",
		"password": "SecurePassword123!",
	}
	
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.NotEmpty(response["access_token"])
	s.NotEmpty(response["refresh_token"])
}

func (s *IntegrationTestSuite) TestUserLoginInvalidCredentials() {
	payload := map[string]interface{}{
		"email":    "nonexistent@example.com",
		"password": "WrongPassword123!",
	}
	
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusUnauthorized, w.Code)
}

// =============================================================================
// VENDOR TESTS
// =============================================================================

func (s *IntegrationTestSuite) TestListVendors() {
	// Create test vendors
	s.createTestVendor("Vendor 1", "Photography")
	s.createTestVendor("Vendor 2", "Catering")
	
	req := httptest.NewRequest("GET", "/api/v1/vendors?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	
	vendors := response["vendors"].([]interface{})
	s.GreaterOrEqual(len(vendors), 2)
}

func (s *IntegrationTestSuite) TestGetVendorByID() {
	vendorID := s.createTestVendor("Test Vendor", "Photography")
	
	req := httptest.NewRequest("GET", "/api/v1/vendors/"+vendorID.String(), nil)
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("Test Vendor", response["business_name"])
}

func (s *IntegrationTestSuite) TestGetVendorNotFound() {
	nonExistentID := uuid.New()
	
	req := httptest.NewRequest("GET", "/api/v1/vendors/"+nonExistentID.String(), nil)
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusNotFound, w.Code)
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (s *IntegrationTestSuite) registerTestUser(email, password string) uuid.UUID {
	ctx := context.Background()
	userID := uuid.New()
	
	// Hash password (in real code, use bcrypt)
	passwordHash := "$2a$12$hash"
	
	_, err := s.db.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, first_name, last_name, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'Test', 'User', 'customer', 'active', NOW(), NOW())
	`, userID, email, passwordHash)
	s.NoError(err)
	
	return userID
}

func (s *IntegrationTestSuite) createTestVendor(name, category string) uuid.UUID {
	ctx := context.Background()
	vendorID := uuid.New()
	userID := uuid.New()
	
	// Create user first
	_, err := s.db.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, first_name, last_name, role, status, created_at, updated_at)
		VALUES ($1, $2, '$2a$12$hash', 'Vendor', 'Owner', 'vendor', 'active', NOW(), NOW())
	`, userID, name+"@vendor.com")
	s.NoError(err)
	
	// Create vendor
	_, err = s.db.Exec(ctx, `
		INSERT INTO vendors (id, user_id, business_name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'Test vendor description', 'active', NOW(), NOW())
	`, vendorID, userID, name)
	s.NoError(err)
	
	return vendorID
}

// =============================================================================
// HANDLER STUBS (would be replaced with actual handlers)
// =============================================================================

func (s *IntegrationTestSuite) handleRegister(c *gin.Context) {
	var req struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Check if email exists
	var exists bool
	s.db.QueryRow(c.Request.Context(), 
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	
	userID := uuid.New()
	_, err := s.db.Exec(c.Request.Context(), `
		INSERT INTO users (id, email, password_hash, first_name, last_name, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'customer', 'active', NOW(), NOW())
	`, userID, req.Email, "$2a$12$hash", req.FirstName, req.LastName)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"id":         userID,
		"email":      req.Email,
		"first_name": req.FirstName,
		"last_name":  req.LastName,
	})
}

func (s *IntegrationTestSuite) handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	var userID uuid.UUID
	err := s.db.QueryRow(c.Request.Context(),
		"SELECT id FROM users WHERE email = $1", req.Email).Scan(&userID)
	
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"access_token":  "test-access-token",
		"refresh_token": "test-refresh-token",
		"expires_at":    time.Now().Add(15 * time.Minute),
	})
}

func (s *IntegrationTestSuite) handleListVendors(c *gin.Context) {
	rows, err := s.db.Query(c.Request.Context(), `
		SELECT id, business_name, description, status FROM vendors LIMIT 10
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var vendors []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var name, description, status string
		rows.Scan(&id, &name, &description, &status)
		vendors = append(vendors, map[string]interface{}{
			"id":            id,
			"business_name": name,
			"description":   description,
			"status":        status,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{"vendors": vendors})
}

func (s *IntegrationTestSuite) handleGetVendor(c *gin.Context) {
	id := c.Param("id")
	
	var vendor struct {
		ID           uuid.UUID
		BusinessName string
		Description  string
		Status       string
	}
	
	err := s.db.QueryRow(c.Request.Context(), `
		SELECT id, business_name, description, status FROM vendors WHERE id = $1
	`, id).Scan(&vendor.ID, &vendor.BusinessName, &vendor.Description, &vendor.Status)
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vendor not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"id":            vendor.ID,
		"business_name": vendor.BusinessName,
		"description":   vendor.Description,
		"status":        vendor.Status,
	})
}

// =============================================================================
// RUN TESTS
// =============================================================================

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
