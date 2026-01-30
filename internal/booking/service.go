// Package booking provides booking management business logic
package booking

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	ErrBookingNotFound    = errors.New("booking not found")
	ErrInvalidBookingData = errors.New("invalid booking data")
	ErrBookingExists      = errors.New("booking already exists")
	ErrInvalidStatus      = errors.New("invalid status transition")
	ErrUnauthorized       = errors.New("unauthorized")
)

// Service handles booking-related operations
type Service struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewService creates a new booking service
func NewService(db *pgxpool.Pool, cache *redis.Client) *Service {
	return &Service{
		db:    db,
		cache: cache,
	}
}

// BookingStatus represents the status of a booking
type BookingStatus string

const (
	StatusPending    BookingStatus = "pending"
	StatusConfirmed  BookingStatus = "confirmed"
	StatusInProgress BookingStatus = "in_progress"
	StatusCompleted  BookingStatus = "completed"
	StatusCancelled  BookingStatus = "cancelled"
	StatusRefunded   BookingStatus = "refunded"
)

// Booking represents a service booking in the system
type Booking struct {
	ID          uuid.UUID              `json:"id"`
	BookingCode string                 `json:"booking_code"`
	UserID      uuid.UUID              `json:"user_id"`
	VendorID    uuid.UUID              `json:"vendor_id"`
	ServiceID   uuid.UUID              `json:"service_id"`
	ProjectID   *uuid.UUID             `json:"project_id,omitempty"`

	// Booking details
	ServiceName        string                 `json:"service_name"`
	ServiceDescription string                 `json:"service_description,omitempty"`
	Status             BookingStatus          `json:"status"`

	// Schedule
	ScheduledDate      *time.Time             `json:"scheduled_date,omitempty"`
	ScheduledTime      *string                `json:"scheduled_time,omitempty"`
	Duration           *int                   `json:"duration_minutes,omitempty"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty"`

	// Location
	ServiceLocation    string                 `json:"service_location"`
	Latitude           *float64               `json:"latitude,omitempty"`
	Longitude          *float64               `json:"longitude,omitempty"`

	// Pricing
	BasePrice          float64                `json:"base_price"`
	TaxAmount          float64                `json:"tax_amount"`
	ServiceFee         float64                `json:"service_fee"`
	DiscountAmount     float64                `json:"discount_amount"`
	TotalAmount        float64                `json:"total_amount"`
	Currency           string                 `json:"currency"`

	// Payment
	PaymentStatus      string                 `json:"payment_status"` // pending, paid, failed, refunded
	PaymentMethod      *string                `json:"payment_method,omitempty"`
	TransactionRef     *string                `json:"transaction_ref,omitempty"`

	// Customer info
	CustomerName       string                 `json:"customer_name"`
	CustomerPhone      string                 `json:"customer_phone"`
	CustomerEmail      string                 `json:"customer_email"`

	// Additional details
	Notes              string                 `json:"notes,omitempty"`
	Requirements       map[string]interface{} `json:"requirements,omitempty"`
	CancellationReason *string                `json:"cancellation_reason,omitempty"`

	// Ratings
	CustomerRating     *float64               `json:"customer_rating,omitempty"`
	CustomerReview     *string                `json:"customer_review,omitempty"`
	VendorRating       *float64               `json:"vendor_rating,omitempty"`
	VendorReview       *string                `json:"vendor_review,omitempty"`

	// Metadata
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// CreateBookingRequest represents a request to create a booking
type CreateBookingRequest struct {
	UserID             uuid.UUID              `json:"user_id"`
	VendorID           uuid.UUID              `json:"vendor_id"`
	ServiceID          uuid.UUID              `json:"service_id"`
	ProjectID          *uuid.UUID             `json:"project_id,omitempty"`
	ServiceName        string                 `json:"service_name"`
	ScheduledDate      *time.Time             `json:"scheduled_date,omitempty"`
	ScheduledTime      *string                `json:"scheduled_time,omitempty"`
	ServiceLocation    string                 `json:"service_location"`
	BasePrice          float64                `json:"base_price"`
	Currency           string                 `json:"currency"`
	CustomerName       string                 `json:"customer_name"`
	CustomerPhone      string                 `json:"customer_phone"`
	CustomerEmail      string                 `json:"customer_email"`
	Notes              string                 `json:"notes,omitempty"`
	Requirements       map[string]interface{} `json:"requirements,omitempty"`
}

// UpdateBookingRequest represents a request to update a booking
type UpdateBookingRequest struct {
	ScheduledDate *time.Time `json:"scheduled_date,omitempty"`
	ScheduledTime *string    `json:"scheduled_time,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
}

// BookingListOptions represents options for listing bookings
type BookingListOptions struct {
	UserID        *uuid.UUID
	VendorID      *uuid.UUID
	ServiceID     *uuid.UUID
	ProjectID     *uuid.UUID
	Status        *BookingStatus
	PaymentStatus *string
	DateFrom      *time.Time
	DateTo        *time.Time
	Limit         int
	Offset        int
	SortBy        string // created_at, scheduled_date, total_amount
	SortOrder     string // asc, desc
}

// Create creates a new booking
func (s *Service) Create(ctx context.Context, req *CreateBookingRequest) (*Booking, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBookingData, err)
	}

	// Calculate pricing
	taxAmount := req.BasePrice * 0.075 // 7.5% VAT
	serviceFee := req.BasePrice * 0.10  // 10% platform fee
	totalAmount := req.BasePrice + taxAmount + serviceFee

	// Generate unique booking code
	bookingCode := s.generateBookingCode()

	// Create booking
	booking := &Booking{
		ID:                 uuid.New(),
		BookingCode:        bookingCode,
		UserID:             req.UserID,
		VendorID:           req.VendorID,
		ServiceID:          req.ServiceID,
		ProjectID:          req.ProjectID,
		ServiceName:        req.ServiceName,
		Status:             StatusPending,
		ScheduledDate:      req.ScheduledDate,
		ScheduledTime:      req.ScheduledTime,
		ServiceLocation:    req.ServiceLocation,
		BasePrice:          req.BasePrice,
		TaxAmount:          taxAmount,
		ServiceFee:         serviceFee,
		DiscountAmount:     0,
		TotalAmount:        totalAmount,
		Currency:           req.Currency,
		PaymentStatus:      "pending",
		CustomerName:       req.CustomerName,
		CustomerPhone:      req.CustomerPhone,
		CustomerEmail:      req.CustomerEmail,
		Notes:              req.Notes,
		Requirements:       req.Requirements,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	query := `
		INSERT INTO bookings (
			id, booking_code, user_id, vendor_id, service_id, project_id,
			service_name, status, scheduled_date, scheduled_time,
			service_location, base_price, tax_amount, service_fee,
			discount_amount, total_amount, currency, payment_status,
			customer_name, customer_phone, customer_email, notes,
			requirements, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
		)
	`

	_, err := s.db.Exec(ctx, query,
		booking.ID, booking.BookingCode, booking.UserID, booking.VendorID,
		booking.ServiceID, booking.ProjectID, booking.ServiceName, booking.Status,
		booking.ScheduledDate, booking.ScheduledTime, booking.ServiceLocation,
		booking.BasePrice, booking.TaxAmount, booking.ServiceFee, booking.DiscountAmount,
		booking.TotalAmount, booking.Currency, booking.PaymentStatus,
		booking.CustomerName, booking.CustomerPhone, booking.CustomerEmail,
		booking.Notes, booking.Requirements, booking.CreatedAt, booking.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}

	return booking, nil
}

// GetByID retrieves a booking by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Booking, error) {
	booking := &Booking{}

	query := `
		SELECT
			id, booking_code, user_id, vendor_id, service_id, project_id,
			service_name, service_description, status, scheduled_date, scheduled_time,
			duration_minutes, completed_at, service_location, latitude, longitude,
			base_price, tax_amount, service_fee, discount_amount, total_amount, currency,
			payment_status, payment_method, transaction_ref,
			customer_name, customer_phone, customer_email, notes, requirements,
			cancellation_reason, customer_rating, customer_review,
			vendor_rating, vendor_review, metadata, created_at, updated_at
		FROM bookings
		WHERE id = $1
	`

	err := s.db.QueryRow(ctx, query, id).Scan(
		&booking.ID, &booking.BookingCode, &booking.UserID, &booking.VendorID,
		&booking.ServiceID, &booking.ProjectID, &booking.ServiceName, &booking.ServiceDescription,
		&booking.Status, &booking.ScheduledDate, &booking.ScheduledTime,
		&booking.Duration, &booking.CompletedAt, &booking.ServiceLocation,
		&booking.Latitude, &booking.Longitude, &booking.BasePrice, &booking.TaxAmount,
		&booking.ServiceFee, &booking.DiscountAmount, &booking.TotalAmount, &booking.Currency,
		&booking.PaymentStatus, &booking.PaymentMethod, &booking.TransactionRef,
		&booking.CustomerName, &booking.CustomerPhone, &booking.CustomerEmail,
		&booking.Notes, &booking.Requirements, &booking.CancellationReason,
		&booking.CustomerRating, &booking.CustomerReview, &booking.VendorRating,
		&booking.VendorReview, &booking.Metadata, &booking.CreatedAt, &booking.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrBookingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}

	return booking, nil
}

// GetByCode retrieves a booking by booking code
func (s *Service) GetByCode(ctx context.Context, code string) (*Booking, error) {
	booking := &Booking{}

	query := `
		SELECT
			id, booking_code, user_id, vendor_id, service_id, project_id,
			service_name, status, scheduled_date, scheduled_time,
			service_location, base_price, tax_amount, service_fee,
			discount_amount, total_amount, currency, payment_status,
			customer_name, customer_phone, customer_email,
			created_at, updated_at
		FROM bookings
		WHERE booking_code = $1
	`

	err := s.db.QueryRow(ctx, query, code).Scan(
		&booking.ID, &booking.BookingCode, &booking.UserID, &booking.VendorID,
		&booking.ServiceID, &booking.ProjectID, &booking.ServiceName, &booking.Status,
		&booking.ScheduledDate, &booking.ScheduledTime, &booking.ServiceLocation,
		&booking.BasePrice, &booking.TaxAmount, &booking.ServiceFee,
		&booking.DiscountAmount, &booking.TotalAmount, &booking.Currency,
		&booking.PaymentStatus, &booking.CustomerName, &booking.CustomerPhone,
		&booking.CustomerEmail, &booking.CreatedAt, &booking.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrBookingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}

	return booking, nil
}

// Update updates a booking
func (s *Service) Update(ctx context.Context, id uuid.UUID, req *UpdateBookingRequest) (*Booking, error) {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{id}
	argPos := 2

	if req.ScheduledDate != nil {
		updates = append(updates, fmt.Sprintf("scheduled_date = $%d", argPos))
		args = append(args, *req.ScheduledDate)
		argPos++
	}
	if req.ScheduledTime != nil {
		updates = append(updates, fmt.Sprintf("scheduled_time = $%d", argPos))
		args = append(args, *req.ScheduledTime)
		argPos++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argPos))
		args = append(args, *req.Notes)
		argPos++
	}

	if len(updates) == 0 {
		return s.GetByID(ctx, id)
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, time.Now())

	query := fmt.Sprintf("UPDATE bookings SET %s WHERE id = $1", joinStrings(updates, ", "))

	_, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update booking: %w", err)
	}

	return s.GetByID(ctx, id)
}

// UpdateStatus updates the status of a booking
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus BookingStatus) error {
	// Validate status transition
	booking, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !s.isValidStatusTransition(booking.Status, newStatus) {
		return ErrInvalidStatus
	}

	now := time.Now()
	query := `UPDATE bookings SET status = $1, updated_at = $2 WHERE id = $3`

	// If completing, also set completed_at
	if newStatus == StatusCompleted {
		query = `UPDATE bookings SET status = $1, completed_at = $2, updated_at = $3 WHERE id = $4`
		_, err = s.db.Exec(ctx, query, newStatus, now, now, id)
	} else {
		_, err = s.db.Exec(ctx, query, newStatus, now, id)
	}

	if err != nil {
		return fmt.Errorf("failed to update booking status: %w", err)
	}

	return nil
}

// Cancel cancels a booking
func (s *Service) Cancel(ctx context.Context, id uuid.UUID, reason string) error {
	query := `
		UPDATE bookings
		SET status = $1, cancellation_reason = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := s.db.Exec(ctx, query, StatusCancelled, reason, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to cancel booking: %w", err)
	}

	return nil
}

// UpdatePaymentStatus updates the payment status of a booking
func (s *Service) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status string, transactionRef *string) error {
	query := `
		UPDATE bookings
		SET payment_status = $1, transaction_ref = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := s.db.Exec(ctx, query, status, transactionRef, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return nil
}

// AddRating adds a customer rating to a booking
func (s *Service) AddRating(ctx context.Context, id uuid.UUID, rating float64, review string) error {
	if rating < 1 || rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}

	query := `
		UPDATE bookings
		SET customer_rating = $1, customer_review = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := s.db.Exec(ctx, query, rating, review, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to add rating: %w", err)
	}

	return nil
}

// List retrieves a list of bookings based on options
func (s *Service) List(ctx context.Context, opts *BookingListOptions) ([]*Booking, int, error) {
	if opts.Limit == 0 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	// Build query with filters
	baseQuery := `FROM bookings WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	// Apply filters
	if opts.UserID != nil {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argPos)
		args = append(args, *opts.UserID)
		argPos++
	}
	if opts.VendorID != nil {
		baseQuery += fmt.Sprintf(" AND vendor_id = $%d", argPos)
		args = append(args, *opts.VendorID)
		argPos++
	}
	if opts.ServiceID != nil {
		baseQuery += fmt.Sprintf(" AND service_id = $%d", argPos)
		args = append(args, *opts.ServiceID)
		argPos++
	}
	if opts.ProjectID != nil {
		baseQuery += fmt.Sprintf(" AND project_id = $%d", argPos)
		args = append(args, *opts.ProjectID)
		argPos++
	}
	if opts.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *opts.Status)
		argPos++
	}
	if opts.PaymentStatus != nil {
		baseQuery += fmt.Sprintf(" AND payment_status = $%d", argPos)
		args = append(args, *opts.PaymentStatus)
		argPos++
	}
	if opts.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, *opts.DateFrom)
		argPos++
	}
	if opts.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, *opts.DateTo)
		argPos++
	}

	// Get total count
	countQuery := `SELECT COUNT(*) ` + baseQuery
	var total int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count bookings: %w", err)
	}

	// Apply sorting and pagination
	orderBy := "created_at DESC"
	if opts.SortBy != "" {
		if opts.SortOrder == "asc" {
			orderBy = fmt.Sprintf("%s ASC", opts.SortBy)
		} else {
			orderBy = fmt.Sprintf("%s DESC", opts.SortBy)
		}
	}

	selectQuery := `
		SELECT
			id, booking_code, user_id, vendor_id, service_id, project_id,
			service_name, status, scheduled_date, scheduled_time,
			service_location, base_price, tax_amount, service_fee,
			discount_amount, total_amount, currency, payment_status,
			customer_name, customer_phone, customer_email,
			created_at, updated_at
	` + baseQuery + fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderBy, argPos, argPos+1)
	args = append(args, opts.Limit, opts.Offset)

	// Execute query
	rows, err := s.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list bookings: %w", err)
	}
	defer rows.Close()

	bookings := []*Booking{}
	for rows.Next() {
		booking := &Booking{}
		err := rows.Scan(
			&booking.ID, &booking.BookingCode, &booking.UserID, &booking.VendorID,
			&booking.ServiceID, &booking.ProjectID, &booking.ServiceName, &booking.Status,
			&booking.ScheduledDate, &booking.ScheduledTime, &booking.ServiceLocation,
			&booking.BasePrice, &booking.TaxAmount, &booking.ServiceFee,
			&booking.DiscountAmount, &booking.TotalAmount, &booking.Currency,
			&booking.PaymentStatus, &booking.CustomerName, &booking.CustomerPhone,
			&booking.CustomerEmail, &booking.CreatedAt, &booking.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan booking: %w", err)
		}
		bookings = append(bookings, booking)
	}

	return bookings, total, nil
}

// Helper methods

func (s *Service) validateCreateRequest(req *CreateBookingRequest) error {
	if req.UserID == uuid.Nil {
		return errors.New("user ID is required")
	}
	if req.VendorID == uuid.Nil {
		return errors.New("vendor ID is required")
	}
	if req.ServiceID == uuid.Nil {
		return errors.New("service ID is required")
	}
	if req.ServiceName == "" {
		return errors.New("service name is required")
	}
	if req.ServiceLocation == "" {
		return errors.New("service location is required")
	}
	if req.BasePrice <= 0 {
		return errors.New("base price must be greater than 0")
	}
	if req.CustomerName == "" {
		return errors.New("customer name is required")
	}
	if req.CustomerPhone == "" {
		return errors.New("customer phone is required")
	}
	if req.CustomerEmail == "" {
		return errors.New("customer email is required")
	}
	return nil
}

func (s *Service) generateBookingCode() string {
	// Generate format: BK-YYYYMMDD-XXXX
	timestamp := time.Now().Format("20060102")
	random := uuid.New().String()[:4]
	return fmt.Sprintf("BK-%s-%s", timestamp, random)
}

func (s *Service) isValidStatusTransition(current, next BookingStatus) bool {
	validTransitions := map[BookingStatus][]BookingStatus{
		StatusPending:    {StatusConfirmed, StatusCancelled},
		StatusConfirmed:  {StatusInProgress, StatusCancelled},
		StatusInProgress: {StatusCompleted, StatusCancelled},
		StatusCompleted:  {StatusRefunded},
		StatusCancelled:  {StatusRefunded},
		StatusRefunded:   {},
	}

	allowed, exists := validTransitions[current]
	if !exists {
		return false
	}

	for _, status := range allowed {
		if status == next {
			return true
		}
	}
	return false
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
