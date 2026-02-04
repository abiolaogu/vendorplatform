package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/BillyRonksGlobal/vendorplatform/internal/notification"
)

// NotificationAdapter adapts the notification service to work with auth service
type NotificationAdapter struct {
	service *notification.Service
}

// NewNotificationAdapter creates a new notification adapter
func NewNotificationAdapter(service *notification.Service) *NotificationAdapter {
	return &NotificationAdapter{
		service: service,
	}
}

// Send sends a notification using the notification service
func (a *NotificationAdapter) Send(ctx context.Context, req SendNotificationRequest) error {
	// Convert to notification service's SendRequest
	userID, err := uuid.Parse(req.UserID.String())
	if err != nil {
		return err
	}

	// Map priority string to notification.NotificationPriority
	priority := notification.PriorityNormal
	switch req.Priority {
	case "low":
		priority = notification.PriorityLow
	case "high":
		priority = notification.PriorityHigh
	case "critical":
		priority = notification.PriorityCritical
	}

	// Map channels
	var channels []notification.NotificationChannel
	for _, ch := range req.Channels {
		switch ch {
		case "email":
			channels = append(channels, notification.ChannelEmail)
		case "push":
			channels = append(channels, notification.ChannelPush)
		case "sms":
			channels = append(channels, notification.ChannelSMS)
		case "in_app":
			channels = append(channels, notification.ChannelInApp)
		}
	}

	// Map type string to NotificationType (simplified)
	notifType := notification.NotificationType(req.Type)

	notifReq := notification.SendRequest{
		UserID:   userID,
		Type:     notifType,
		Title:    req.Title,
		Body:     req.Body,
		Data:     req.Data,
		Priority: priority,
		Channels: channels,
	}

	_, err = a.service.Send(ctx, notifReq)
	return err
}
