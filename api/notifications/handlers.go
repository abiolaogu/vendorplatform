// Package notifications provides HTTP handlers for notification management
package notifications

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/notification"
)

// Handler handles notification HTTP requests
type Handler struct {
	notificationService *notification.Service
	logger              *zap.Logger
}

// NewHandler creates a new notification handler
func NewHandler(notificationService *notification.Service, logger *zap.Logger) *Handler {
	return &Handler{
		notificationService: notificationService,
		logger:              logger,
	}
}

// RegisterRoutes registers notification routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	notifications := router.Group("/notifications")
	{
		// Send notification (admin/system only)
		notifications.POST("", h.SendNotification)

		// Get user's notifications
		notifications.GET("", h.GetNotifications)
		notifications.GET("/unread", h.GetUnreadNotifications)
		notifications.GET("/unread/count", h.GetUnreadCount)

		// Mark as read
		notifications.PUT("/:id/read", h.MarkAsRead)
		notifications.PUT("/read-all", h.MarkAllAsRead)

		// Device management
		notifications.POST("/devices", h.RegisterDevice)

		// Preferences
		notifications.GET("/preferences", h.GetPreferences)
		notifications.PUT("/preferences", h.UpdatePreferences)
	}
}

// =============================================================================
// HANDLERS
// =============================================================================

// SendNotification handles POST /api/v1/notifications
// @Summary Send a notification
// @Description Send a notification to a user (admin/system only)
// @Tags notifications
// @Accept json
// @Produce json
// @Param notification body SendNotificationRequest true "Notification details"
// @Success 200 {object} SendNotificationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications [post]
func (h *Handler) SendNotification(c *gin.Context) {
	var req SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Validate required fields
	if req.UserID == uuid.Nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "user_id is required"})
		return
	}
	if req.Type == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "type is required"})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "title is required"})
		return
	}
	if req.Body == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "body is required"})
		return
	}

	// Set default priority if not provided
	if req.Priority == "" {
		req.Priority = notification.PriorityNormal
	}

	// Build send request
	sendReq := notification.SendRequest{
		UserID:   req.UserID,
		Type:     req.Type,
		Title:    req.Title,
		Body:     req.Body,
		Data:     req.Data,
		Priority: req.Priority,
		Channels: req.Channels,
	}

	// Send notification
	notifications, err := h.notificationService.Send(c.Request.Context(), sendReq)
	if err != nil {
		h.logger.Error("Failed to send notification",
			zap.Error(err),
			zap.String("user_id", req.UserID.String()),
			zap.String("type", string(req.Type)),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to send notification"})
		return
	}

	c.JSON(http.StatusOK, SendNotificationResponse{
		Notifications: notifications,
		Message:       "Notification sent successfully",
	})
}

// GetNotifications handles GET /api/v1/notifications
// @Summary Get user notifications
// @Description Get all notifications for the authenticated user
// @Tags notifications
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Success 200 {object} GetNotificationsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications [get]
func (h *Handler) GetNotifications(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Parse limit
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := uuid.Parse(limitStr); err == nil {
			limit = int(l.ID())
		}
	}

	// Get notifications
	notifications, err := h.notificationService.GetUnreadNotifications(c.Request.Context(), userID, limit)
	if err != nil {
		h.logger.Error("Failed to get notifications",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve notifications"})
		return
	}

	c.JSON(http.StatusOK, GetNotificationsResponse{
		Notifications: notifications,
		Total:         len(notifications),
	})
}

// GetUnreadNotifications handles GET /api/v1/notifications/unread
// @Summary Get unread notifications
// @Description Get all unread notifications for the authenticated user
// @Tags notifications
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Success 200 {object} GetNotificationsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/unread [get]
func (h *Handler) GetUnreadNotifications(c *gin.Context) {
	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Parse limit
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := uuid.Parse(limitStr); err == nil {
			limit = int(l.ID())
		}
	}

	// Get unread notifications
	notifications, err := h.notificationService.GetUnreadNotifications(c.Request.Context(), userID, limit)
	if err != nil {
		h.logger.Error("Failed to get unread notifications",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve notifications"})
		return
	}

	c.JSON(http.StatusOK, GetNotificationsResponse{
		Notifications: notifications,
		Total:         len(notifications),
	})
}

// GetUnreadCount handles GET /api/v1/notifications/unread/count
// @Summary Get unread count
// @Description Get count of unread notifications for the authenticated user
// @Tags notifications
// @Produce json
// @Success 200 {object} UnreadCountResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/unread/count [get]
func (h *Handler) GetUnreadCount(c *gin.Context) {
	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Get unread count
	count, err := h.notificationService.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get unread count",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve unread count"})
		return
	}

	c.JSON(http.StatusOK, UnreadCountResponse{
		Count: count,
	})
}

// MarkAsRead handles PUT /api/v1/notifications/:id/read
// @Summary Mark notification as read
// @Description Mark a specific notification as read
// @Tags notifications
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/{id}/read [put]
func (h *Handler) MarkAsRead(c *gin.Context) {
	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Parse notification ID
	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid notification ID"})
		return
	}

	// Mark as read
	err = h.notificationService.MarkAsRead(c.Request.Context(), notificationID)
	if err != nil {
		h.logger.Error("Failed to mark notification as read",
			zap.Error(err),
			zap.String("user_id", userID.String()),
			zap.String("notification_id", notificationID.String()),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Notification marked as read",
	})
}

// MarkAllAsRead handles PUT /api/v1/notifications/read-all
// @Summary Mark all notifications as read
// @Description Mark all notifications as read for the authenticated user
// @Tags notifications
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/read-all [put]
func (h *Handler) MarkAllAsRead(c *gin.Context) {
	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Mark all as read
	err = h.notificationService.MarkAllAsRead(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to mark all notifications as read",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to mark all notifications as read"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "All notifications marked as read",
	})
}

// RegisterDevice handles POST /api/v1/notifications/devices
// @Summary Register device for push notifications
// @Description Register a device token for push notifications
// @Tags notifications
// @Accept json
// @Produce json
// @Param device body RegisterDeviceRequest true "Device details"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/devices [post]
func (h *Handler) RegisterDevice(c *gin.Context) {
	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Validate required fields
	if req.Token == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "token is required"})
		return
	}
	if req.Platform == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "platform is required"})
		return
	}

	// Register device
	err = h.notificationService.RegisterDevice(c.Request.Context(), userID, req.Token, req.Platform)
	if err != nil {
		h.logger.Error("Failed to register device",
			zap.Error(err),
			zap.String("user_id", userID.String()),
			zap.String("platform", req.Platform),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to register device"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Device registered successfully",
	})
}

// GetPreferences handles GET /api/v1/notifications/preferences
// @Summary Get notification preferences
// @Description Get notification preferences for the authenticated user
// @Tags notifications
// @Produce json
// @Success 200 {object} GetPreferencesResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/preferences [get]
func (h *Handler) GetPreferences(c *gin.Context) {
	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Get preferences
	prefs, err := h.notificationService.GetUserPreferences(c.Request.Context(), userID)
	if err != nil {
		// If preferences don't exist, return defaults
		prefs = &notification.UserPreferences{
			UserID:       userID,
			PushEnabled:  true,
			EmailEnabled: true,
			SMSEnabled:   false,
			DisabledTypes: []notification.NotificationType{},
		}
	}

	c.JSON(http.StatusOK, GetPreferencesResponse{
		Preferences: prefs,
	})
}

// UpdatePreferences handles PUT /api/v1/notifications/preferences
// @Summary Update notification preferences
// @Description Update notification preferences for the authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Param preferences body UpdatePreferencesRequest true "Preference details"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/preferences [put]
func (h *Handler) UpdatePreferences(c *gin.Context) {
	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Build preferences
	prefs := &notification.UserPreferences{
		UserID:          userID,
		PushEnabled:     req.PushEnabled,
		EmailEnabled:    req.EmailEnabled,
		SMSEnabled:      req.SMSEnabled,
		QuietHoursStart: req.QuietHoursStart,
		QuietHoursEnd:   req.QuietHoursEnd,
		DisabledTypes:   req.DisabledTypes,
	}

	// Update preferences
	err = h.notificationService.UpdateUserPreferences(c.Request.Context(), prefs)
	if err != nil {
		h.logger.Error("Failed to update preferences",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update preferences"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Preferences updated successfully",
	})
}

// =============================================================================
// HELPERS
// =============================================================================

func getUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	// Try to get user ID from context (set by auth middleware)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, ErrUnauthorized
	}

	userID, ok := userIDInterface.(uuid.UUID)
	if !ok {
		// Try parsing as string
		userIDStr, ok := userIDInterface.(string)
		if !ok {
			return uuid.Nil, ErrUnauthorized
		}

		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			return uuid.Nil, ErrUnauthorized
		}

		return parsedID, nil
	}

	return userID, nil
}

// Error definitions
var (
	ErrUnauthorized = gin.Error{
		Err:  http.ErrNotSupported,
		Type: gin.ErrorTypePrivate,
	}
)
