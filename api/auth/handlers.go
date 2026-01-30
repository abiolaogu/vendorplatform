// Package auth provides HTTP handlers for authentication
package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/auth"
)

// Handler handles authentication HTTP requests
type Handler struct {
	authService *auth.Service
	logger      *zap.Logger
}

// NewHandler creates a new auth handler
func NewHandler(authService *auth.Service, logger *zap.Logger) *Handler {
	return &Handler{
		authService: authService,
		logger:      logger,
	}
}

// RegisterRoutes registers authentication routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	authRoutes := router.Group("/auth")
	{
		// Public routes
		authRoutes.POST("/register", h.Register)
		authRoutes.POST("/login", h.Login)
		authRoutes.POST("/refresh", h.RefreshToken)
		authRoutes.POST("/verify-email", h.VerifyEmail)
		authRoutes.POST("/forgot-password", h.ForgotPassword)
		authRoutes.POST("/reset-password", h.ResetPassword)

		// Protected routes
		protected := authRoutes.Group("")
		protected.Use(h.authService.AuthMiddleware())
		{
			protected.POST("/logout", h.Logout)
			protected.POST("/logout-all", h.LogoutAll)
			protected.POST("/change-password", h.ChangePassword)
			protected.GET("/me", h.GetCurrentUser)
		}
	}
}

// Register handles user registration
func (h *Handler) Register(c *gin.Context) {
	var req auth.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Registration failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful. Please check your email for verification.",
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"role":       user.Role,
			"status":     user.Status,
		},
	})
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract device info
	deviceInfo := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	tokens, user, err := h.authService.Login(c.Request.Context(), req, deviceInfo, ipAddress, userAgent)
	if err != nil {
		h.logger.Info("Login failed", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("User logged in", zap.String("user_id", user.ID.String()), zap.String("email", user.Email))

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user": gin.H{
			"id":              user.ID,
			"email":           user.Email,
			"first_name":      user.FirstName,
			"last_name":       user.LastName,
			"role":            user.Role,
			"status":          user.Status,
			"email_verified":  user.EmailVerified,
			"phone_verified":  user.PhoneVerified,
			"avatar_url":      user.AvatarURL,
		},
		"tokens": tokens,
	})
}

// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	sessionID, exists := c.Get("session_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.authService.Logout(c.Request.Context(), sessionID.(uuid.UUID))
	if err != nil {
		h.logger.Error("Logout failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// LogoutAll handles logout from all devices
func (h *Handler) LogoutAll(c *gin.Context) {
	userID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err = h.authService.LogoutAll(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Logout all failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout from all devices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out from all devices successfully"})
}

// RefreshToken handles token refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokens, err := h.authService.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"tokens":  tokens,
	})
}

// VerifyEmail handles email verification
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.authService.VerifyEmail(c.Request.Context(), req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

// ForgotPassword handles password reset request
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.authService.RequestPasswordReset(c.Request.Context(), req.Email)
	if err != nil {
		h.logger.Error("Password reset request failed", zap.Error(err))
	}

	// Always return success to prevent email enumeration
	response := gin.H{"message": "If the email exists, a password reset link has been sent"}

	// In development, include the token for testing
	if token != "" {
		h.logger.Info("Password reset token generated", zap.String("email", req.Email), zap.String("token", token))
		// Only expose token in development
		// response["dev_token"] = token
	}

	c.JSON(http.StatusOK, response)
}

// ResetPassword handles password reset with token
func (h *Handler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// ChangePassword handles password change for authenticated users
func (h *Handler) ChangePassword(c *gin.Context) {
	userID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.authService.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// GetCurrentUser returns the current authenticated user's information
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userEmail, _ := c.Get("user_email")
	userRole, _ := c.Get("user_role")

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    userID,
			"email": userEmail,
			"role":  userRole,
		},
	})
}
