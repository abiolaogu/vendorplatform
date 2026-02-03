// =============================================================================
// AUTHENTICATION & AUTHORIZATION SERVICE
// JWT-based authentication with role-based access control
// =============================================================================

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// =============================================================================
// TYPES
// =============================================================================

// User represents a platform user
type User struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	Phone         string     `json:"phone,omitempty"`
	PasswordHash  string     `json:"-"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	Role          UserRole   `json:"role"`
	Status        UserStatus `json:"status"`
	EmailVerified bool       `json:"email_verified"`
	PhoneVerified bool       `json:"phone_verified"`
	AvatarURL     string     `json:"avatar_url,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
}

type UserRole string
const (
	RoleCustomer    UserRole = "customer"
	RoleVendor      UserRole = "vendor"
	RoleTechnician  UserRole = "technician"
	RoleAdmin       UserRole = "admin"
	RoleSuperAdmin  UserRole = "superadmin"
)

type UserStatus string
const (
	StatusPending   UserStatus = "pending"
	StatusActive    UserStatus = "active"
	StatusSuspended UserStatus = "suspended"
	StatusDeleted   UserStatus = "deleted"
)

// Session represents an active user session
type Session struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	RefreshToken string    `json:"-"`
	DeviceInfo   string    `json:"device_info"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// TokenPair contains access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// Claims for JWT tokens
type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      UserRole  `json:"role"`
	SessionID uuid.UUID `json:"session_id"`
	jwt.RegisteredClaims
}

// =============================================================================
// SERVICE
// =============================================================================

// Config for auth service
type Config struct {
	JWTSecret           string
	AccessTokenExpiry   time.Duration
	RefreshTokenExpiry  time.Duration
	BCryptCost          int
	MaxSessionsPerUser  int
	VerificationExpiry  time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		JWTSecret:          "change-me-in-production",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BCryptCost:         12,
		MaxSessionsPerUser: 5,
		VerificationExpiry: 24 * time.Hour,
	}
}

// NotificationSender interface for sending notifications
type NotificationSender interface {
	Send(ctx context.Context, req SendNotificationRequest) error
}

// SendNotificationRequest matches the notification service's SendRequest
type SendNotificationRequest struct {
	UserID   uuid.UUID
	Type     string
	Title    string
	Body     string
	Data     map[string]interface{}
	Priority string
	Channels []string
}

// Service handles authentication
type Service struct {
	db           *pgxpool.Pool
	cache        *redis.Client
	config       *Config
	notification NotificationSender
}

// NewService creates a new auth service
func NewService(db *pgxpool.Pool, cache *redis.Client, config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}
	return &Service{
		db:     db,
		cache:  cache,
		config: config,
	}
}

// SetNotificationService sets the notification service for sending emails
func (s *Service) SetNotificationService(notificationService NotificationSender) {
	s.notification = notificationService
}

// =============================================================================
// REGISTRATION
// =============================================================================

// RegisterRequest for user registration
type RegisterRequest struct {
	Email     string   `json:"email" binding:"required,email"`
	Password  string   `json:"password" binding:"required,min=8"`
	FirstName string   `json:"first_name" binding:"required"`
	LastName  string   `json:"last_name" binding:"required"`
	Phone     string   `json:"phone"`
	Role      UserRole `json:"role"`
}

// Register creates a new user account
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*User, error) {
	// Check if email already exists
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.config.BCryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Set default role
	if req.Role == "" {
		req.Role = RoleCustomer
	}

	// Create user
	user := &User{
		ID:           uuid.New(),
		Email:        strings.ToLower(req.Email),
		Phone:        req.Phone,
		PasswordHash: string(hash),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         req.Role,
		Status:       StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	query := `
		INSERT INTO users (id, email, phone, password_hash, first_name, last_name, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = s.db.Exec(ctx, query, 
		user.ID, user.Email, user.Phone, user.PasswordHash,
		user.FirstName, user.LastName, user.Role, user.Status,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate verification token
	verificationToken, err := s.generateVerificationToken(ctx, user.ID, "email")
	if err != nil {
		// Log but don't fail - user is created
		fmt.Printf("failed to generate verification token: %v\n", err)
	} else {
		// Send verification email
		if err := s.sendVerificationEmail(ctx, user, verificationToken); err != nil {
			fmt.Printf("failed to send verification email: %v\n", err)
		}
	}

	return user, nil
}

// =============================================================================
// LOGIN
// =============================================================================

// LoginRequest for user login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login authenticates a user and returns tokens
func (s *Service) Login(ctx context.Context, req LoginRequest, deviceInfo, ipAddress, userAgent string) (*TokenPair, *User, error) {
	// Find user by email
	var user User
	var passwordHash string
	
	query := `
		SELECT id, email, phone, password_hash, first_name, last_name, role, status, 
		       email_verified, phone_verified, avatar_url, created_at, updated_at, last_login_at
		FROM users WHERE email = $1
	`
	err := s.db.QueryRow(ctx, query, strings.ToLower(req.Email)).Scan(
		&user.ID, &user.Email, &user.Phone, &passwordHash,
		&user.FirstName, &user.LastName, &user.Role, &user.Status,
		&user.EmailVerified, &user.PhoneVerified, &user.AvatarURL,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	)
	if err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	// Check status
	if user.Status != StatusActive && user.Status != StatusPending {
		return nil, nil, errors.New("account is not active")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	// Create session
	session, err := s.createSession(ctx, user.ID, deviceInfo, ipAddress, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate tokens
	tokens, err := s.generateTokenPair(user, session.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update last login
	s.db.Exec(ctx, "UPDATE users SET last_login_at = $1 WHERE id = $2", time.Now(), user.ID)

	return tokens, &user, nil
}

// =============================================================================
// SESSION MANAGEMENT
// =============================================================================

func (s *Service) createSession(ctx context.Context, userID uuid.UUID, deviceInfo, ipAddress, userAgent string) (*Session, error) {
	// Check existing sessions and remove oldest if exceeds limit
	var count int
	s.db.QueryRow(ctx, "SELECT COUNT(*) FROM sessions WHERE user_id = $1", userID).Scan(&count)
	
	if count >= s.config.MaxSessionsPerUser {
		// Delete oldest session
		s.db.Exec(ctx, `
			DELETE FROM sessions WHERE id = (
				SELECT id FROM sessions WHERE user_id = $1 ORDER BY created_at ASC LIMIT 1
			)
		`, userID)
	}

	// Generate refresh token
	refreshToken, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:           uuid.New(),
		UserID:       userID,
		RefreshToken: refreshToken,
		DeviceInfo:   deviceInfo,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(s.config.RefreshTokenExpiry),
		CreatedAt:    time.Now(),
	}

	query := `
		INSERT INTO sessions (id, user_id, refresh_token, device_info, ip_address, user_agent, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = s.db.Exec(ctx, query,
		session.ID, session.UserID, session.RefreshToken,
		session.DeviceInfo, session.IPAddress, session.UserAgent,
		session.ExpiresAt, session.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// Logout invalidates a session
func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	_, err := s.db.Exec(ctx, "DELETE FROM sessions WHERE id = $1", sessionID)
	return err
}

// LogoutAll invalidates all sessions for a user
func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1", userID)
	return err
}

// =============================================================================
// TOKEN MANAGEMENT
// =============================================================================

func (s *Service) generateTokenPair(user User, sessionID uuid.UUID) (*TokenPair, error) {
	now := time.Now()
	expiresAt := now.Add(s.config.AccessTokenExpiry)

	// Access token claims
	claims := &Claims{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "vendorplatform",
			Subject:   user.ID.String(),
		},
	}

	// Create access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessTokenString, err := accessToken.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken validates an access token and returns claims
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshTokens refreshes the access token using a refresh token
func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Find session by refresh token
	var session Session
	var user User
	
	query := `
		SELECT s.id, s.user_id, s.expires_at, 
		       u.id, u.email, u.role, u.status
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.refresh_token = $1 AND s.expires_at > NOW()
	`
	err := s.db.QueryRow(ctx, query, refreshToken).Scan(
		&session.ID, &session.UserID, &session.ExpiresAt,
		&user.ID, &user.Email, &user.Role, &user.Status,
	)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	if user.Status != StatusActive {
		return nil, errors.New("account is not active")
	}

	// Generate new tokens
	tokens, err := s.generateTokenPair(user, session.ID)
	if err != nil {
		return nil, err
	}

	// Update session with new refresh token
	s.db.Exec(ctx, "UPDATE sessions SET refresh_token = $1 WHERE id = $2", tokens.RefreshToken, session.ID)

	return tokens, nil
}

// =============================================================================
// VERIFICATION
// =============================================================================

func (s *Service) generateVerificationToken(ctx context.Context, userID uuid.UUID, tokenType string) (string, error) {
	token, err := generateSecureToken(32)
	if err != nil {
		return "", err
	}

	// Store in Redis with expiry
	key := fmt.Sprintf("verify:%s:%s", tokenType, token)
	err = s.cache.Set(ctx, key, userID.String(), s.config.VerificationExpiry).Err()
	if err != nil {
		return "", err
	}

	return token, nil
}

// VerifyEmail verifies a user's email address
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	key := fmt.Sprintf("verify:email:%s", token)
	userIDStr, err := s.cache.Get(ctx, key).Result()
	if err != nil {
		return errors.New("invalid or expired verification token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return errors.New("invalid verification token")
	}

	// Update user
	_, err = s.db.Exec(ctx, "UPDATE users SET email_verified = TRUE, status = 'active', updated_at = $1 WHERE id = $2", time.Now(), userID)
	if err != nil {
		return err
	}

	// Delete token
	s.cache.Del(ctx, key)

	return nil
}

// =============================================================================
// PASSWORD MANAGEMENT
// =============================================================================

// RequestPasswordReset initiates password reset
func (s *Service) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	var userID uuid.UUID
	err := s.db.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", strings.ToLower(email)).Scan(&userID)
	if err != nil {
		// Don't reveal if email exists
		return "", nil
	}

	token, err := s.generateVerificationToken(ctx, userID, "password_reset")
	if err != nil {
		return "", err
	}

	// Get user info for email
	var user User
	err = s.db.QueryRow(ctx, "SELECT id, email, first_name, last_name FROM users WHERE id = $1", userID).Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName,
	)
	if err != nil {
		return "", err
	}

	// Send password reset email
	if err := s.sendPasswordResetEmail(ctx, &user, token); err != nil {
		fmt.Printf("failed to send password reset email: %v\n", err)
	}

	return token, nil
}

// ResetPassword resets password with token
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	key := fmt.Sprintf("verify:password_reset:%s", token)
	userIDStr, err := s.cache.Get(ctx, key).Result()
	if err != nil {
		return errors.New("invalid or expired reset token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return errors.New("invalid reset token")
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.config.BCryptCost)
	if err != nil {
		return err
	}

	// Update password
	_, err = s.db.Exec(ctx, "UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3", string(hash), time.Now(), userID)
	if err != nil {
		return err
	}

	// Invalidate all sessions
	s.LogoutAll(ctx, userID)

	// Delete token
	s.cache.Del(ctx, key)

	return nil
}

// ChangePassword changes password for authenticated user
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	var currentHash string
	err := s.db.QueryRow(ctx, "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&currentHash)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(oldPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.config.BCryptCost)
	if err != nil {
		return err
	}

	// Update password
	_, err = s.db.Exec(ctx, "UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3", string(hash), time.Now(), userID)
	return err
}

// =============================================================================
// MIDDLEWARE
// =============================================================================

// AuthMiddleware validates JWT tokens
func (s *Service) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}

		claims, err := s.ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Verify session still exists
		var exists bool
		s.db.QueryRow(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM sessions WHERE id = $1)", claims.SessionID).Scan(&exists)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("session_id", claims.SessionID)

		c.Next()
	}
}

// RequireRole middleware checks if user has required role
func RequireRole(roles ...UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		role := userRole.(UserRole)
		for _, r := range roles {
			if role == r || role == RoleSuperAdmin {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

// =============================================================================
// HELPERS
// =============================================================================

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GetUserFromContext extracts user ID from gin context
func GetUserFromContext(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user not found in context")
	}
	return userID.(uuid.UUID), nil
}

// GetRoleFromContext extracts user role from gin context
func GetRoleFromContext(c *gin.Context) (UserRole, error) {
	role, exists := c.Get("user_role")
	if !exists {
		return "", errors.New("role not found in context")
	}
	return role.(UserRole), nil
}

// =============================================================================
// EMAIL NOTIFICATIONS
// =============================================================================

// sendVerificationEmail sends an email verification link to the user
func (s *Service) sendVerificationEmail(ctx context.Context, user *User, token string) error {
	if s.notification == nil {
		return errors.New("notification service not configured")
	}

	// Build verification URL (this would be the frontend URL)
	baseURL := getEnv("FRONTEND_URL", "https://vendorplatform.com")
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", baseURL, token)

	// Create notification request
	req := SendNotificationRequest{
		UserID:   user.ID,
		Type:     "email_verification",
		Title:    "Verify Your Email Address",
		Body:     "Welcome to VendorPlatform! Please verify your email address to get started.",
		Data: map[string]interface{}{
			"FirstName":        user.FirstName,
			"VerificationURL":  verificationURL,
			"VerificationCode": token[:8], // Show first 8 chars as code
		},
		Priority: "high",
		Channels: []string{"email"},
	}

	return s.notification.Send(ctx, req)
}

// sendPasswordResetEmail sends a password reset link to the user
func (s *Service) sendPasswordResetEmail(ctx context.Context, user *User, token string) error {
	if s.notification == nil {
		return errors.New("notification service not configured")
	}

	// Build reset URL (this would be the frontend URL)
	baseURL := getEnv("FRONTEND_URL", "https://vendorplatform.com")
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)

	// Create notification request
	req := SendNotificationRequest{
		UserID: user.ID,
		Type:   "password_reset",
		Title:  "Reset Your Password",
		Body:   "We received a request to reset your password. Click the link to create a new password.",
		Data: map[string]interface{}{
			"FirstName": user.FirstName,
			"ResetURL":  resetURL,
			"ResetCode": token[:8], // Show first 8 chars as code
		},
		Priority: "high",
		Channels: []string{"email"},
	}

	return s.notification.Send(ctx, req)
}

// Helper to get environment variable (duplicated for package independence)
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
