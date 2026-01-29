// Package booking provides booking management functionality
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
	ErrBookingNotFound       = errors.New("booking not found")
	ErrInvalidBookingData    = errors.New("invalid booking data")
	ErrBookingAlreadyExists  = errors.New("booking already exists")
	ErrInsufficientPermission = errors.New("insufficient permission")
	ErrBookingNotCancellable = errors.New("booking cannot be cancelled")
)

// Booking represents a service booking
type Booking struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	VendorID        uuid.UUID  `json:"vendor_id"`
	ServiceID       uuid.UUID  `json:"service_id"`
	ProjectID       *uuid.UUID `json:"project_id,omitempty"`
	BookingNumber   string     `json:"booking_number"`
	ScheduledDate   time.Time  `json:"scheduled_date"`
	ScheduledStart  *time.Time `json:"scheduled_start_time,omitempty"`
	ScheduledEnd    *time.Time `json:"scheduled_end_time,omitempty"`
	DurationMinutes *int       `json:"duration_minutes,omitempty"`
	Timezone        string     `json:"timezone"`
	LocationType    string     `json:"service_location_type"`
	AddressID       *uuid.UUID `json:"service_address_id,omitempty"`
	Quantity        int        `json:"quantity"`
	GuestCount      *int       `json:"guest_count,omitempty"`
	UnitPrice       float64    `json:"unit_price"`
	Subtotal        float64    `json:"subtotal"`
	DiscountAmount  float64    `json:"discount_amount"`
	DiscountReason  string     `json:"discount_reason,omitempty"`
	TaxAmount       float64    `json:"tax_amount"`
	ServiceFee      float64    `json:"service_fee"`
	TotalAmount     float64    `json:"total_amount"`
	Currency        string     `json:"currency"`
	PaymentStatus   string     `json:"payment_status"`
	AmountPaid      float64    `json:"amount_paid"`
	PaymentDueDate  *time.Time `json:"payment_due_date,omitempty"`
	Status          string     `json:"status"`
	CustomerNotes   string     `json:"customer_notes,omitempty"`
	SpecialRequests string     `json:"special_requests,omitempty"`
	VendorNotes     string     `json:"vendor_notes,omitempty"`
	SourceType      string     `json:"source_type"`
	CustomerRating  *float64   `json:"customer_rating,omitempty"`
	CustomerReview  string     `json:"customer_review,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ConfirmedAt     *time.Time `json:"confirmed_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
}

// CreateBookingRequest represents data for creating a booking
type CreateBookingRequest struct {
	UserID          uuid.UUID  `json:"user_id"`
	ServiceID       uuid.UUID  `json:"service_id"`
	ProjectID       *uuid.UUID `json:"project_id,omitempty"`
	ScheduledDate   time.Time  `json:"scheduled_date"`
	ScheduledStart  *time.Time `json:"scheduled_start_time,omitempty"`
	ScheduledEnd    *time.Time `json:"scheduled_end_time,omitempty"`
	DurationMinutes *int       `json:"duration_minutes,omitempty"`
	LocationType    string     `json:"service_location_type"`
	AddressID       *uuid.UUID `json:"service_address_id,omitempty"`
	Quantity        int        `json:"quantity"`
	GuestCount      *int       `json:"guest_count,omitempty"`
	CustomerNotes   string     `json:"customer_notes,omitempty"`
	SpecialRequests string     `json:"special_requests,omitempty"`
	SourceType      string     `json:"source_type,omitempty"`
}

// UpdateBookingRequest represents data for updating a booking
type UpdateBookingRequest struct {
	ScheduledDate   *time.Time `json:"scheduled_date,omitempty"`
	ScheduledStart  *time.Time `json:"scheduled_start_time,omitempty"`
	ScheduledEnd    *time.Time `json:"scheduled_end_time,omitempty"`
	LocationType    *string    `json:"service_location_type,omitempty"`
	AddressID       *uuid.UUID `json:"service_address_id,omitempty"`
	Quantity        *int       `json:"quantity,omitempty"`
	GuestCount      *int       `json:"guest_count,omitempty"`
	CustomerNotes   *string    `json:"customer_notes,omitempty"`
	SpecialRequests *string    `json:"special_requests,omitempty"`
}

// ListBookingsFilter represents filters for listing bookings
type ListBookingsFilter struct {
	UserID    *uuid.UUID
	VendorID  *uuid.UUID
	ServiceID *uuid.UUID
	ProjectID *uuid.UUID
	Status    []string
	FromDate  *time.Time
	ToDate    *time.Time
	Limit     int
	Offset    int
}

// Service handles booking business logic
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

// CreateBooking creates a new booking
func (s *Service) CreateBooking(ctx context.Context, req *CreateBookingRequest) (*Booking, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Get service details and vendor
	var vendorID uuid.UUID
	var unitPrice float64
	var serviceName string
	err := s.db.QueryRow(ctx, `
		SELECT vendor_id, base_price, name
		FROM services
		WHERE id = $1 AND is_active = TRUE
	`, req.ServiceID).Scan(&vendorID, &unitPrice, &serviceName)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("service not found or inactive")
		}
		return nil, fmt.Errorf("failed to fetch service: %w", err)
	}

	// Calculate amounts
	quantity := req.Quantity
	if quantity < 1 {
		quantity = 1
	}

	subtotal := unitPrice * float64(quantity)
	taxAmount := subtotal * 0.075 // 7.5% VAT for Nigeria
	serviceFee := subtotal * 0.10   // 10% platform fee
	totalAmount := subtotal + taxAmount + serviceFee

	// Generate booking number
	bookingNumber := s.generateBookingNumber()

	// Set source type
	sourceType := req.SourceType
	if sourceType == "" {
		sourceType = "direct"
	}

	// Set timezone
	timezone := "Africa/Lagos"

	// Insert booking
	booking := &Booking{
		ID:              uuid.New(),
		UserID:          req.UserID,
		VendorID:        vendorID,
		ServiceID:       req.ServiceID,
		ProjectID:       req.ProjectID,
		BookingNumber:   bookingNumber,
		ScheduledDate:   req.ScheduledDate,
		ScheduledStart:  req.ScheduledStart,
		ScheduledEnd:    req.ScheduledEnd,
		DurationMinutes: req.DurationMinutes,
		Timezone:        timezone,
		LocationType:    req.LocationType,
		AddressID:       req.AddressID,
		Quantity:        quantity,
		GuestCount:      req.GuestCount,
		UnitPrice:       unitPrice,
		Subtotal:        subtotal,
		DiscountAmount:  0,
		TaxAmount:       taxAmount,
		ServiceFee:      serviceFee,
		TotalAmount:     totalAmount,
		Currency:        "NGN",
		PaymentStatus:   "pending",
		AmountPaid:      0,
		Status:          "pending",
		CustomerNotes:   req.CustomerNotes,
		SpecialRequests: req.SpecialRequests,
		SourceType:      sourceType,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO bookings (
			id, user_id, vendor_id, service_id, project_id, booking_number,
			scheduled_date, scheduled_start_time, scheduled_end_time, duration_minutes,
			timezone, service_location_type, service_address_id, quantity, guest_count,
			unit_price, subtotal, discount_amount, tax_amount, service_fee, total_amount,
			currency, payment_status, amount_paid, status, customer_notes, special_requests,
			source_type, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30
		)
	`,
		booking.ID, booking.UserID, booking.VendorID, booking.ServiceID, booking.ProjectID,
		booking.BookingNumber, booking.ScheduledDate, booking.ScheduledStart, booking.ScheduledEnd,
		booking.DurationMinutes, booking.Timezone, booking.LocationType, booking.AddressID,
		booking.Quantity, booking.GuestCount, booking.UnitPrice, booking.Subtotal,
		booking.DiscountAmount, booking.TaxAmount, booking.ServiceFee, booking.TotalAmount,
		booking.Currency, booking.PaymentStatus, booking.AmountPaid, booking.Status,
		booking.CustomerNotes, booking.SpecialRequests, booking.SourceType,
		booking.CreatedAt, booking.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}

	return booking, nil
}

// GetBooking retrieves a booking by ID
func (s *Service) GetBooking(ctx context.Context, id uuid.UUID) (*Booking, error) {
	booking := &Booking{}

	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, vendor_id, service_id, project_id, booking_number,
		       scheduled_date, scheduled_start_time, scheduled_end_time, duration_minutes,
		       timezone, service_location_type, service_address_id, quantity, guest_count,
		       unit_price, subtotal, discount_amount, discount_reason, tax_amount, service_fee,
		       total_amount, currency, payment_status, amount_paid, payment_due_date, status,
		       customer_notes, special_requests, vendor_notes, source_type,
		       customer_rating, customer_review, created_at, updated_at,
		       confirmed_at, completed_at, cancelled_at
		FROM bookings
		WHERE id = $1
	`, id).Scan(
		&booking.ID, &booking.UserID, &booking.VendorID, &booking.ServiceID, &booking.ProjectID,
		&booking.BookingNumber, &booking.ScheduledDate, &booking.ScheduledStart, &booking.ScheduledEnd,
		&booking.DurationMinutes, &booking.Timezone, &booking.LocationType, &booking.AddressID,
		&booking.Quantity, &booking.GuestCount, &booking.UnitPrice, &booking.Subtotal,
		&booking.DiscountAmount, &booking.DiscountReason, &booking.TaxAmount, &booking.ServiceFee,
		&booking.TotalAmount, &booking.Currency, &booking.PaymentStatus, &booking.AmountPaid,
		&booking.PaymentDueDate, &booking.Status, &booking.CustomerNotes, &booking.SpecialRequests,
		&booking.VendorNotes, &booking.SourceType, &booking.CustomerRating, &booking.CustomerReview,
		&booking.CreatedAt, &booking.UpdatedAt, &booking.ConfirmedAt, &booking.CompletedAt,
		&booking.CancelledAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}

	return booking, nil
}

// ListBookings lists bookings with filters
func (s *Service) ListBookings(ctx context.Context, filter *ListBookingsFilter) ([]*Booking, error) {
	query := `
		SELECT id, user_id, vendor_id, service_id, project_id, booking_number,
		       scheduled_date, scheduled_start_time, scheduled_end_time, duration_minutes,
		       timezone, service_location_type, service_address_id, quantity, guest_count,
		       unit_price, subtotal, discount_amount, discount_reason, tax_amount, service_fee,
		       total_amount, currency, payment_status, amount_paid, payment_due_date, status,
		       customer_notes, special_requests, vendor_notes, source_type,
		       customer_rating, customer_review, created_at, updated_at,
		       confirmed_at, completed_at, cancelled_at
		FROM bookings
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argPos)
		args = append(args, *filter.UserID)
		argPos++
	}
	if filter.VendorID != nil {
		query += fmt.Sprintf(" AND vendor_id = $%d", argPos)
		args = append(args, *filter.VendorID)
		argPos++
	}
	if filter.ServiceID != nil {
		query += fmt.Sprintf(" AND service_id = $%d", argPos)
		args = append(args, *filter.ServiceID)
		argPos++
	}
	if filter.ProjectID != nil {
		query += fmt.Sprintf(" AND project_id = $%d", argPos)
		args = append(args, *filter.ProjectID)
		argPos++
	}
	if len(filter.Status) > 0 {
		query += fmt.Sprintf(" AND status = ANY($%d)", argPos)
		args = append(args, filter.Status)
		argPos++
	}
	if filter.FromDate != nil {
		query += fmt.Sprintf(" AND scheduled_date >= $%d", argPos)
		args = append(args, *filter.FromDate)
		argPos++
	}
	if filter.ToDate != nil {
		query += fmt.Sprintf(" AND scheduled_date <= $%d", argPos)
		args = append(args, *filter.ToDate)
		argPos++
	}

	query += " ORDER BY scheduled_date DESC, created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list bookings: %w", err)
	}
	defer rows.Close()

	bookings := []*Booking{}
	for rows.Next() {
		booking := &Booking{}
		err := rows.Scan(
			&booking.ID, &booking.UserID, &booking.VendorID, &booking.ServiceID, &booking.ProjectID,
			&booking.BookingNumber, &booking.ScheduledDate, &booking.ScheduledStart, &booking.ScheduledEnd,
			&booking.DurationMinutes, &booking.Timezone, &booking.LocationType, &booking.AddressID,
			&booking.Quantity, &booking.GuestCount, &booking.UnitPrice, &booking.Subtotal,
			&booking.DiscountAmount, &booking.DiscountReason, &booking.TaxAmount, &booking.ServiceFee,
			&booking.TotalAmount, &booking.Currency, &booking.PaymentStatus, &booking.AmountPaid,
			&booking.PaymentDueDate, &booking.Status, &booking.CustomerNotes, &booking.SpecialRequests,
			&booking.VendorNotes, &booking.SourceType, &booking.CustomerRating, &booking.CustomerReview,
			&booking.CreatedAt, &booking.UpdatedAt, &booking.ConfirmedAt, &booking.CompletedAt,
			&booking.CancelledAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan booking: %w", err)
		}
		bookings = append(bookings, booking)
	}

	return bookings, nil
}

// UpdateBooking updates a booking
func (s *Service) UpdateBooking(ctx context.Context, id uuid.UUID, req *UpdateBookingRequest) (*Booking, error) {
	// Get existing booking
	existing, err := s.GetBooking(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if booking can be updated
	if existing.Status == "completed" || existing.Status == "cancelled" {
		return nil, fmt.Errorf("cannot update %s booking", existing.Status)
	}

	// Build update query
	query := "UPDATE bookings SET updated_at = NOW()"
	args := []interface{}{}
	argPos := 1

	if req.ScheduledDate != nil {
		query += fmt.Sprintf(", scheduled_date = $%d", argPos)
		args = append(args, *req.ScheduledDate)
		argPos++
	}
	if req.ScheduledStart != nil {
		query += fmt.Sprintf(", scheduled_start_time = $%d", argPos)
		args = append(args, *req.ScheduledStart)
		argPos++
	}
	if req.ScheduledEnd != nil {
		query += fmt.Sprintf(", scheduled_end_time = $%d", argPos)
		args = append(args, *req.ScheduledEnd)
		argPos++
	}
	if req.LocationType != nil {
		query += fmt.Sprintf(", service_location_type = $%d", argPos)
		args = append(args, *req.LocationType)
		argPos++
	}
	if req.AddressID != nil {
		query += fmt.Sprintf(", service_address_id = $%d", argPos)
		args = append(args, *req.AddressID)
		argPos++
	}
	if req.Quantity != nil {
		query += fmt.Sprintf(", quantity = $%d", argPos)
		args = append(args, *req.Quantity)
		argPos++

		// Recalculate amounts
		newSubtotal := existing.UnitPrice * float64(*req.Quantity)
		newTax := newSubtotal * 0.075
		newServiceFee := newSubtotal * 0.10
		newTotal := newSubtotal + newTax + newServiceFee

		query += fmt.Sprintf(", subtotal = $%d, tax_amount = $%d, service_fee = $%d, total_amount = $%d",
			argPos, argPos+1, argPos+2, argPos+3)
		args = append(args, newSubtotal, newTax, newServiceFee, newTotal)
		argPos += 4
	}
	if req.GuestCount != nil {
		query += fmt.Sprintf(", guest_count = $%d", argPos)
		args = append(args, *req.GuestCount)
		argPos++
	}
	if req.CustomerNotes != nil {
		query += fmt.Sprintf(", customer_notes = $%d", argPos)
		args = append(args, *req.CustomerNotes)
		argPos++
	}
	if req.SpecialRequests != nil {
		query += fmt.Sprintf(", special_requests = $%d", argPos)
		args = append(args, *req.SpecialRequests)
		argPos++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argPos)
	args = append(args, id)

	_, err = s.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update booking: %w", err)
	}

	return s.GetBooking(ctx, id)
}

// ConfirmBooking confirms a booking (vendor acceptance)
func (s *Service) ConfirmBooking(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE bookings
		SET status = 'confirmed', confirmed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`, id)

	if err != nil {
		return fmt.Errorf("failed to confirm booking: %w", err)
	}

	return nil
}

// StartBooking marks booking as in progress
func (s *Service) StartBooking(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE bookings
		SET status = 'in_progress', updated_at = NOW()
		WHERE id = $1 AND status = 'confirmed'
	`, id)

	if err != nil {
		return fmt.Errorf("failed to start booking: %w", err)
	}

	return nil
}

// CompleteBooking marks booking as completed
func (s *Service) CompleteBooking(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE bookings
		SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'in_progress'
	`, id)

	if err != nil {
		return fmt.Errorf("failed to complete booking: %w", err)
	}

	return nil
}

// CancelBooking cancels a booking
func (s *Service) CancelBooking(ctx context.Context, id uuid.UUID, reason string) error {
	// Get existing booking
	existing, err := s.GetBooking(ctx, id)
	if err != nil {
		return err
	}

	// Check if booking can be cancelled
	if existing.Status == "completed" || existing.Status == "cancelled" {
		return ErrBookingNotCancellable
	}

	_, err = s.db.Exec(ctx, `
		UPDATE bookings
		SET status = 'cancelled', cancelled_at = NOW(), cancellation_reason = $2, updated_at = NOW()
		WHERE id = $1
	`, id, reason)

	if err != nil {
		return fmt.Errorf("failed to cancel booking: %w", err)
	}

	return nil
}

// AddReview adds a customer review for a booking
func (s *Service) AddReview(ctx context.Context, id uuid.UUID, rating float64, review string) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	_, err := s.db.Exec(ctx, `
		UPDATE bookings
		SET customer_rating = $2, customer_review = $3, updated_at = NOW()
		WHERE id = $1 AND status = 'completed'
	`, id, rating, review)

	if err != nil {
		return fmt.Errorf("failed to add review: %w", err)
	}

	// Update vendor's average rating
	go s.updateVendorRating(context.Background(), id)

	return nil
}

// Helper methods

func (s *Service) validateCreateRequest(req *CreateBookingRequest) error {
	if req.UserID == uuid.Nil {
		return fmt.Errorf("%w: user_id is required", ErrInvalidBookingData)
	}
	if req.ServiceID == uuid.Nil {
		return fmt.Errorf("%w: service_id is required", ErrInvalidBookingData)
	}
	if req.ScheduledDate.IsZero() {
		return fmt.Errorf("%w: scheduled_date is required", ErrInvalidBookingData)
	}
	if req.ScheduledDate.Before(time.Now()) {
		return fmt.Errorf("%w: scheduled_date cannot be in the past", ErrInvalidBookingData)
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}
	return nil
}

func (s *Service) generateBookingNumber() string {
	// Format: BK-YYYYMMDD-XXXX (e.g., BK-20260129-A3F7)
	now := time.Now()
	random := uuid.New().String()[:4]
	return fmt.Sprintf("BK-%s-%s", now.Format("20060102"), random)
}

func (s *Service) updateVendorRating(ctx context.Context, bookingID uuid.UUID) {
	// Get vendor ID from booking
	var vendorID uuid.UUID
	err := s.db.QueryRow(ctx, "SELECT vendor_id FROM bookings WHERE id = $1", bookingID).Scan(&vendorID)
	if err != nil {
		return
	}

	// Calculate new average rating
	var avgRating float64
	var ratingCount int
	err = s.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(customer_rating), 0), COUNT(*)
		FROM bookings
		WHERE vendor_id = $1 AND customer_rating IS NOT NULL
	`, vendorID).Scan(&avgRating, &ratingCount)

	if err != nil {
		return
	}

	// Update vendor
	_, _ = s.db.Exec(ctx, `
		UPDATE vendors
		SET rating_average = $2, rating_count = $3, updated_at = NOW()
		WHERE id = $1
	`, vendorID, avgRating, ratingCount)
}
