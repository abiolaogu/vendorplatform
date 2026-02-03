// Package notifications provides types for notification API
package notifications

import (
	"github.com/google/uuid"

	"github.com/BillyRonksGlobal/vendorplatform/internal/notification"
)

// =============================================================================
// REQUEST TYPES
// =============================================================================

// SendNotificationRequest represents a request to send a notification
type SendNotificationRequest struct {
	UserID   uuid.UUID                       `json:"user_id" binding:"required"`
	Type     notification.NotificationType   `json:"type" binding:"required"`
	Title    string                          `json:"title" binding:"required"`
	Body     string                          `json:"body" binding:"required"`
	Data     map[string]interface{}          `json:"data,omitempty"`
	Priority notification.NotificationPriority `json:"priority,omitempty"`
	Channels []notification.NotificationChannel `json:"channels,omitempty"`
}

// RegisterDeviceRequest represents a request to register a device
type RegisterDeviceRequest struct {
	Token    string `json:"token" binding:"required"`
	Platform string `json:"platform" binding:"required"` // 'ios', 'android', 'web'
}

// UpdatePreferencesRequest represents a request to update notification preferences
type UpdatePreferencesRequest struct {
	PushEnabled     bool                             `json:"push_enabled"`
	EmailEnabled    bool                             `json:"email_enabled"`
	SMSEnabled      bool                             `json:"sms_enabled"`
	QuietHoursStart string                           `json:"quiet_hours_start,omitempty"` // "22:00"
	QuietHoursEnd   string                           `json:"quiet_hours_end,omitempty"`   // "08:00"
	DisabledTypes   []notification.NotificationType  `json:"disabled_types,omitempty"`
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// SendNotificationResponse represents a response after sending a notification
type SendNotificationResponse struct {
	Notifications []*notification.Notification `json:"notifications"`
	Message       string                       `json:"message"`
}

// GetNotificationsResponse represents a response with notifications
type GetNotificationsResponse struct {
	Notifications []*notification.Notification `json:"notifications"`
	Total         int                          `json:"total"`
}

// UnreadCountResponse represents a response with unread count
type UnreadCountResponse struct {
	Count int `json:"count"`
}

// GetPreferencesResponse represents a response with notification preferences
type GetPreferencesResponse struct {
	Preferences *notification.UserPreferences `json:"preferences"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
