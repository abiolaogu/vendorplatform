// Package booking provides booking management functionality
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

var (
	ErrInvalidStatus = errors.New("invalid status transition")
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

// StartBooking marks booking as in progress
func (s *Service) StartBooking(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE bookings
		SET status = 'in_progress', updated_at = NOW()
		WHERE id = $1 AND status = 'confirmed'
	`, id)

	if err != nil {
		return fmt.Errorf("failed to start booking: %w", err)
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

// CompleteBooking marks booking as completed
func (s *Service) CompleteBooking(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE bookings
		SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'in_progress'
	`, id)

	if err != nil {
		return fmt.Errorf("failed to complete booking: %w", err)
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
