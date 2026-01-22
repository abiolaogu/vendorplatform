// =============================================================================
// NOTIFICATION SERVICE
// Multi-channel notification system: Push, Email, SMS, In-App
// =============================================================================

package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// =============================================================================
// TYPES
// =============================================================================

// Notification represents a notification record
type Notification struct {
	ID          uuid.UUID              `json:"id"`
	UserID      uuid.UUID              `json:"user_id"`
	Type        NotificationType       `json:"type"`
	Channel     NotificationChannel    `json:"channel"`
	Title       string                 `json:"title"`
	Body        string                 `json:"body"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Status      NotificationStatus     `json:"status"`
	Priority    NotificationPriority   `json:"priority"`
	ReadAt      *time.Time             `json:"read_at,omitempty"`
	SentAt      *time.Time             `json:"sent_at,omitempty"`
	DeliveredAt *time.Time             `json:"delivered_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

type NotificationType string
const (
	TypeBookingCreated    NotificationType = "booking_created"
	TypeBookingConfirmed  NotificationType = "booking_confirmed"
	TypeBookingCancelled  NotificationType = "booking_cancelled"
	TypePaymentReceived   NotificationType = "payment_received"
	TypePaymentFailed     NotificationType = "payment_failed"
	TypeEmergencyAssigned NotificationType = "emergency_assigned"
	TypeEmergencyUpdate   NotificationType = "emergency_update"
	TypeTechEnRoute       NotificationType = "tech_en_route"
	TypeTechArrived       NotificationType = "tech_arrived"
	TypeReferralReceived  NotificationType = "referral_received"
	TypeReferralConverted NotificationType = "referral_converted"
	TypeNewMessage        NotificationType = "new_message"
	TypeReviewReceived    NotificationType = "review_received"
	TypePromotion         NotificationType = "promotion"
	TypeSystemAlert       NotificationType = "system_alert"
)

type NotificationChannel string
const (
	ChannelPush   NotificationChannel = "push"
	ChannelEmail  NotificationChannel = "email"
	ChannelSMS    NotificationChannel = "sms"
	ChannelInApp  NotificationChannel = "in_app"
)

type NotificationStatus string
const (
	StatusQueued    NotificationStatus = "queued"
	StatusSent      NotificationStatus = "sent"
	StatusDelivered NotificationStatus = "delivered"
	StatusFailed    NotificationStatus = "failed"
	StatusRead      NotificationStatus = "read"
)

type NotificationPriority string
const (
	PriorityLow    NotificationPriority = "low"
	PriorityNormal NotificationPriority = "normal"
	PriorityHigh   NotificationPriority = "high"
	PriorityCritical NotificationPriority = "critical"
)

// UserPreferences for notification settings
type UserPreferences struct {
	UserID          uuid.UUID `json:"user_id"`
	PushEnabled     bool      `json:"push_enabled"`
	EmailEnabled    bool      `json:"email_enabled"`
	SMSEnabled      bool      `json:"sms_enabled"`
	QuietHoursStart string    `json:"quiet_hours_start"` // "22:00"
	QuietHoursEnd   string    `json:"quiet_hours_end"`   // "08:00"
	DisabledTypes   []NotificationType `json:"disabled_types"`
}

// DeviceToken for push notifications
type DeviceToken struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Token     string    `json:"token"`
	Platform  string    `json:"platform"` // 'ios', 'android', 'web'
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// =============================================================================
// SERVICE
// =============================================================================

// Config for notification service
type Config struct {
	// Email (SMTP)
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string
	
	// SMS (Termii)
	TermiiAPIKey  string
	TermiiSender  string
	
	// Push (Firebase)
	FirebaseCredentials string
	
	// Push (OneSignal)
	OneSignalAppID  string
	OneSignalAPIKey string
	
	// Templates
	TemplateDir string
}

// Service handles notifications
type Service struct {
	db        *pgxpool.Pool
	cache     *redis.Client
	config    *Config
	templates map[string]*template.Template
	http      *http.Client
}

// NewService creates a new notification service
func NewService(db *pgxpool.Pool, cache *redis.Client, config *Config) *Service {
	s := &Service{
		db:        db,
		cache:     cache,
		config:    config,
		templates: make(map[string]*template.Template),
		http:      &http.Client{Timeout: 30 * time.Second},
	}
	s.loadTemplates()
	return s
}

func (s *Service) loadTemplates() {
	// Email templates
	templates := []string{
		"booking_created",
		"booking_confirmed",
		"payment_received",
		"emergency_assigned",
		"welcome",
		"password_reset",
		"email_verification",
	}
	
	for _, name := range templates {
		tmpl, err := template.ParseFiles(fmt.Sprintf("%s/%s.html", s.config.TemplateDir, name))
		if err == nil {
			s.templates[name] = tmpl
		}
	}
}

// =============================================================================
// SEND NOTIFICATIONS
// =============================================================================

// SendRequest for sending a notification
type SendRequest struct {
	UserID   uuid.UUID              `json:"user_id"`
	Type     NotificationType       `json:"type"`
	Title    string                 `json:"title"`
	Body     string                 `json:"body"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Priority NotificationPriority   `json:"priority"`
	Channels []NotificationChannel  `json:"channels"` // Empty = all enabled channels
}

// Send sends a notification to a user
func (s *Service) Send(ctx context.Context, req SendRequest) ([]*Notification, error) {
	// Get user preferences
	prefs, err := s.GetUserPreferences(ctx, req.UserID)
	if err != nil {
		prefs = &UserPreferences{
			UserID:       req.UserID,
			PushEnabled:  true,
			EmailEnabled: true,
			SMSEnabled:   false,
		}
	}
	
	// Check if notification type is disabled
	for _, disabled := range prefs.DisabledTypes {
		if disabled == req.Type {
			return nil, nil // User opted out
		}
	}
	
	// Check quiet hours (skip for critical priority)
	if req.Priority != PriorityCritical && s.isQuietHours(prefs) {
		// Queue for later
		return s.queueForLater(ctx, req, prefs)
	}
	
	// Determine channels
	channels := req.Channels
	if len(channels) == 0 {
		// Use all enabled channels
		if prefs.PushEnabled {
			channels = append(channels, ChannelPush)
		}
		if prefs.EmailEnabled {
			channels = append(channels, ChannelEmail)
		}
		if prefs.SMSEnabled && (req.Priority == PriorityHigh || req.Priority == PriorityCritical) {
			channels = append(channels, ChannelSMS)
		}
		// Always add in-app
		channels = append(channels, ChannelInApp)
	}
	
	var notifications []*Notification
	
	for _, channel := range channels {
		notification := &Notification{
			ID:        uuid.New(),
			UserID:    req.UserID,
			Type:      req.Type,
			Channel:   channel,
			Title:     req.Title,
			Body:      req.Body,
			Data:      req.Data,
			Status:    StatusQueued,
			Priority:  req.Priority,
			CreatedAt: time.Now(),
		}
		
		// Send via channel
		var sendErr error
		switch channel {
		case ChannelPush:
			sendErr = s.sendPush(ctx, notification)
		case ChannelEmail:
			sendErr = s.sendEmail(ctx, notification)
		case ChannelSMS:
			sendErr = s.sendSMS(ctx, notification)
		case ChannelInApp:
			sendErr = s.sendInApp(ctx, notification)
		}
		
		if sendErr != nil {
			notification.Status = StatusFailed
		} else {
			notification.Status = StatusSent
			now := time.Now()
			notification.SentAt = &now
		}
		
		// Save notification
		s.saveNotification(ctx, notification)
		notifications = append(notifications, notification)
	}
	
	return notifications, nil
}

// =============================================================================
// PUSH NOTIFICATIONS
// =============================================================================

func (s *Service) sendPush(ctx context.Context, notification *Notification) error {
	// Get user's device tokens
	tokens, err := s.getDeviceTokens(ctx, notification.UserID)
	if err != nil || len(tokens) == 0 {
		return fmt.Errorf("no device tokens found")
	}
	
	// Send via OneSignal
	payload := map[string]interface{}{
		"app_id":            s.config.OneSignalAppID,
		"include_player_ids": tokens,
		"headings":          map[string]string{"en": notification.Title},
		"contents":          map[string]string{"en": notification.Body},
		"data":              notification.Data,
	}
	
	if notification.Priority == PriorityCritical {
		payload["priority"] = 10
		payload["android_channel_id"] = "urgent"
	}
	
	body, _ := json.Marshal(payload)
	
	req, _ := http.NewRequestWithContext(ctx, "POST", 
		"https://onesignal.com/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Authorization", "Basic "+s.config.OneSignalAPIKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("push notification failed with status %d", resp.StatusCode)
	}
	
	return nil
}

func (s *Service) getDeviceTokens(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT token FROM device_tokens 
		WHERE user_id = $1 AND is_active = TRUE
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err == nil {
			tokens = append(tokens, token)
		}
	}
	
	return tokens, nil
}

// RegisterDevice registers a device for push notifications
func (s *Service) RegisterDevice(ctx context.Context, userID uuid.UUID, token, platform string) error {
	// Deactivate existing tokens for this device
	s.db.Exec(ctx, "UPDATE device_tokens SET is_active = FALSE WHERE token = $1", token)
	
	device := &DeviceToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		Platform:  platform,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	_, err := s.db.Exec(ctx, `
		INSERT INTO device_tokens (id, user_id, token, platform, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, device.ID, device.UserID, device.Token, device.Platform, device.IsActive, device.CreatedAt, device.UpdatedAt)
	
	return err
}

// =============================================================================
// EMAIL NOTIFICATIONS
// =============================================================================

func (s *Service) sendEmail(ctx context.Context, notification *Notification) error {
	// Get user email
	var email string
	err := s.db.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", notification.UserID).Scan(&email)
	if err != nil {
		return err
	}
	
	// Build email content
	var htmlBody string
	if tmpl, ok := s.templates[string(notification.Type)]; ok {
		var buf bytes.Buffer
		data := map[string]interface{}{
			"Title": notification.Title,
			"Body":  notification.Body,
			"Data":  notification.Data,
		}
		if err := tmpl.Execute(&buf, data); err == nil {
			htmlBody = buf.String()
		}
	}
	
	if htmlBody == "" {
		htmlBody = fmt.Sprintf("<h1>%s</h1><p>%s</p>", notification.Title, notification.Body)
	}
	
	// Send via SMTP
	msg := fmt.Sprintf("From: %s <%s>\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", 
		s.config.FromName, s.config.FromEmail,
		email, notification.Title, htmlBody)
	
	auth := smtp.PlainAuth("", s.config.SMTPUser, s.config.SMTPPassword, s.config.SMTPHost)
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)
	
	return smtp.SendMail(addr, auth, s.config.FromEmail, []string{email}, []byte(msg))
}

// =============================================================================
// SMS NOTIFICATIONS
// =============================================================================

func (s *Service) sendSMS(ctx context.Context, notification *Notification) error {
	// Get user phone
	var phone string
	err := s.db.QueryRow(ctx, "SELECT phone FROM users WHERE id = $1", notification.UserID).Scan(&phone)
	if err != nil || phone == "" {
		return fmt.Errorf("no phone number found")
	}
	
	// Send via Termii
	payload := map[string]interface{}{
		"to":      phone,
		"from":    s.config.TermiiSender,
		"sms":     fmt.Sprintf("%s: %s", notification.Title, notification.Body),
		"type":    "plain",
		"channel": "generic",
		"api_key": s.config.TermiiAPIKey,
	}
	
	body, _ := json.Marshal(payload)
	
	req, _ := http.NewRequestWithContext(ctx, "POST", 
		"https://api.ng.termii.com/api/sms/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS failed with status %d", resp.StatusCode)
	}
	
	return nil
}

// =============================================================================
// IN-APP NOTIFICATIONS
// =============================================================================

func (s *Service) sendInApp(ctx context.Context, notification *Notification) error {
	// Save to database (already done in main flow)
	// Publish to real-time channel
	pubsubKey := fmt.Sprintf("notifications:%s", notification.UserID)
	
	data, _ := json.Marshal(notification)
	return s.cache.Publish(ctx, pubsubKey, data).Err()
}

// GetUnreadNotifications returns unread notifications for a user
func (s *Service) GetUnreadNotifications(ctx context.Context, userID uuid.UUID, limit int) ([]*Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, type, channel, title, body, data, status, priority, 
		       read_at, sent_at, delivered_at, created_at
		FROM notifications
		WHERE user_id = $1 AND channel = 'in_app' AND read_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var notifications []*Notification
	for rows.Next() {
		var n Notification
		var dataJSON []byte
		
		err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.Channel, &n.Title, &n.Body,
			&dataJSON, &n.Status, &n.Priority, &n.ReadAt, &n.SentAt, 
			&n.DeliveredAt, &n.CreatedAt,
		)
		if err != nil {
			continue
		}
		
		json.Unmarshal(dataJSON, &n.Data)
		notifications = append(notifications, &n)
	}
	
	return notifications, nil
}

// MarkAsRead marks a notification as read
func (s *Service) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	_, err := s.db.Exec(ctx, 
		"UPDATE notifications SET read_at = $1, status = $2 WHERE id = $3",
		time.Now(), StatusRead, notificationID,
	)
	return err
}

// MarkAllAsRead marks all notifications as read for a user
func (s *Service) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.Exec(ctx, 
		"UPDATE notifications SET read_at = $1, status = $2 WHERE user_id = $3 AND read_at IS NULL",
		time.Now(), StatusRead, userID,
	)
	return err
}

// GetUnreadCount returns unread notification count for a user
func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications 
		WHERE user_id = $1 AND channel = 'in_app' AND read_at IS NULL
	`, userID).Scan(&count)
	return count, err
}

// =============================================================================
// USER PREFERENCES
// =============================================================================

// GetUserPreferences returns notification preferences for a user
func (s *Service) GetUserPreferences(ctx context.Context, userID uuid.UUID) (*UserPreferences, error) {
	var prefs UserPreferences
	var disabledTypesJSON []byte
	
	err := s.db.QueryRow(ctx, `
		SELECT user_id, push_enabled, email_enabled, sms_enabled,
		       quiet_hours_start, quiet_hours_end, disabled_types
		FROM notification_preferences WHERE user_id = $1
	`, userID).Scan(
		&prefs.UserID, &prefs.PushEnabled, &prefs.EmailEnabled, &prefs.SMSEnabled,
		&prefs.QuietHoursStart, &prefs.QuietHoursEnd, &disabledTypesJSON,
	)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(disabledTypesJSON, &prefs.DisabledTypes)
	
	return &prefs, nil
}

// UpdateUserPreferences updates notification preferences
func (s *Service) UpdateUserPreferences(ctx context.Context, prefs *UserPreferences) error {
	disabledTypesJSON, _ := json.Marshal(prefs.DisabledTypes)
	
	_, err := s.db.Exec(ctx, `
		INSERT INTO notification_preferences (
			user_id, push_enabled, email_enabled, sms_enabled,
			quiet_hours_start, quiet_hours_end, disabled_types
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
			push_enabled = EXCLUDED.push_enabled,
			email_enabled = EXCLUDED.email_enabled,
			sms_enabled = EXCLUDED.sms_enabled,
			quiet_hours_start = EXCLUDED.quiet_hours_start,
			quiet_hours_end = EXCLUDED.quiet_hours_end,
			disabled_types = EXCLUDED.disabled_types
	`, prefs.UserID, prefs.PushEnabled, prefs.EmailEnabled, prefs.SMSEnabled,
		prefs.QuietHoursStart, prefs.QuietHoursEnd, disabledTypesJSON,
	)
	
	return err
}

// =============================================================================
// HELPERS
// =============================================================================

func (s *Service) saveNotification(ctx context.Context, n *Notification) error {
	dataJSON, _ := json.Marshal(n.Data)
	
	query := `
		INSERT INTO notifications (
			id, user_id, type, channel, title, body, data,
			status, priority, read_at, sent_at, delivered_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	
	_, err := s.db.Exec(ctx, query,
		n.ID, n.UserID, n.Type, n.Channel, n.Title, n.Body, dataJSON,
		n.Status, n.Priority, n.ReadAt, n.SentAt, n.DeliveredAt, n.CreatedAt,
	)
	return err
}

func (s *Service) isQuietHours(prefs *UserPreferences) bool {
	if prefs.QuietHoursStart == "" || prefs.QuietHoursEnd == "" {
		return false
	}
	
	now := time.Now()
	currentTime := now.Format("15:04")
	
	// Simple comparison (doesn't handle midnight crossing well)
	return currentTime >= prefs.QuietHoursStart || currentTime <= prefs.QuietHoursEnd
}

func (s *Service) queueForLater(ctx context.Context, req SendRequest, prefs *UserPreferences) ([]*Notification, error) {
	// Queue in Redis for processing after quiet hours
	data, _ := json.Marshal(req)
	key := fmt.Sprintf("notification:queue:%s", req.UserID)
	s.cache.LPush(ctx, key, data)
	return nil, nil
}

// =============================================================================
// BATCH NOTIFICATIONS
// =============================================================================

// SendBulk sends notifications to multiple users
func (s *Service) SendBulk(ctx context.Context, userIDs []uuid.UUID, notificationType NotificationType, title, body string, data map[string]interface{}) error {
	for _, userID := range userIDs {
		req := SendRequest{
			UserID:   userID,
			Type:     notificationType,
			Title:    title,
			Body:     body,
			Data:     data,
			Priority: PriorityNormal,
		}
		
		// Send async
		go s.Send(context.Background(), req)
	}
	
	return nil
}

// SendToSegment sends notifications to a user segment
func (s *Service) SendToSegment(ctx context.Context, segment string, notificationType NotificationType, title, body string, data map[string]interface{}) error {
	// Query users by segment
	var userIDs []uuid.UUID
	
	var query string
	switch segment {
	case "vendors":
		query = "SELECT id FROM users WHERE role = 'vendor' AND status = 'active'"
	case "customers":
		query = "SELECT id FROM users WHERE role = 'customer' AND status = 'active'"
	case "premium":
		query = "SELECT user_id FROM subscriptions WHERE status = 'active' AND tier = 'premium'"
	default:
		return fmt.Errorf("unknown segment: %s", segment)
	}
	
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err == nil {
			userIDs = append(userIDs, id)
		}
	}
	
	return s.SendBulk(ctx, userIDs, notificationType, title, body, data)
}
