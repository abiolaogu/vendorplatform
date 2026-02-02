// Package service provides service management business logic
package service

// Note: This service requires the services table with the following columns:
// id, vendor_id, category_id, name, slug, sku, short_description, full_description,
// highlights, includes, excludes, pricing_model, base_price, price_unit, min_price,
// max_price, currency, min_quantity, max_quantity, min_guests, max_guests,
// duration_minutes, is_available, availability_type, lead_time_hours, is_featured,
// rating_average, rating_count, booking_count, status, created_at, updated_at

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	ErrServiceNotFound    = errors.New("service not found")
	ErrInvalidServiceData = errors.New("invalid service data")
	ErrServiceExists      = errors.New("service already exists")
	ErrUnauthorized       = errors.New("unauthorized")
)

// Service handles service-related operations
type ServiceManager struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewServiceManager creates a new service manager
func NewServiceManager(db *pgxpool.Pool, cache *redis.Client) *ServiceManager {
	return &ServiceManager{
		db:    db,
		cache: cache,
	}
}

// ServiceOffering represents a service offered by a vendor
type ServiceOffering struct {
	ID               uuid.UUID `json:"id"`
	VendorID         uuid.UUID `json:"vendor_id"`
	CategoryID       uuid.UUID `json:"category_id"`

	// Identity
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	SKU              string    `json:"sku,omitempty"`

	// Description
	ShortDescription string    `json:"short_description"`
	FullDescription  string    `json:"full_description"`
	Highlights       []string  `json:"highlights,omitempty"`
	Includes         []string  `json:"includes,omitempty"`
	Excludes         []string  `json:"excludes,omitempty"`

	// Pricing
	PricingModel     string    `json:"pricing_model"`
	BasePrice        *float64  `json:"base_price,omitempty"`
	PriceUnit        string    `json:"price_unit,omitempty"`
	MinPrice         *float64  `json:"min_price,omitempty"`
	MaxPrice         *float64  `json:"max_price,omitempty"`
	Currency         string    `json:"currency"`

	// Capacity
	MinQuantity      int       `json:"min_quantity"`
	MaxQuantity      *int      `json:"max_quantity,omitempty"`
	MinGuests        *int      `json:"min_guests,omitempty"`
	MaxGuests        *int      `json:"max_guests,omitempty"`

	// Availability
	DurationMinutes  *int      `json:"duration_minutes,omitempty"`
	IsAvailable      bool      `json:"is_available"`
	AvailabilityType string    `json:"availability_type"`
	LeadTimeHours    int       `json:"lead_time_hours"`

	// Stats
	IsFeatured       bool      `json:"is_featured"`
	RatingAverage    float64   `json:"rating_average"`
	RatingCount      int       `json:"rating_count"`
	BookingCount     int       `json:"booking_count"`

	// Status
	Status           string    `json:"status"` // active, inactive, draft

	// Timestamps
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ServiceListOptions represents options for listing services
type ServiceListOptions struct {
	VendorID   *uuid.UUID
	CategoryID *uuid.UUID
	Status     *string
	IsAvailable *bool
	IsFeatured *bool
	MinPrice   *float64
	MaxPrice   *float64
	SearchQuery *string
	SortBy     string // "created_at", "rating", "price", "popularity"
	SortOrder  string // "asc", "desc"
	Limit      int
	Offset     int
}

// GetByID retrieves a service by ID
func (s *ServiceManager) GetByID(ctx context.Context, id uuid.UUID) (*ServiceOffering, error) {
	query := `
		SELECT id, vendor_id, category_id, name, slug, COALESCE(sku, ''),
		       short_description, full_description,
		       COALESCE(highlights, '{}'), COALESCE(includes, '{}'), COALESCE(excludes, '{}'),
		       pricing_model, base_price, COALESCE(price_unit, ''),
		       min_price, max_price, currency,
		       min_quantity, max_quantity, min_guests, max_guests,
		       duration_minutes, is_available, availability_type, lead_time_hours,
		       is_featured, rating_average, rating_count, booking_count,
		       status, created_at, updated_at
		FROM services
		WHERE id = $1
	`

	var svc ServiceOffering
	err := s.db.QueryRow(ctx, query, id).Scan(
		&svc.ID, &svc.VendorID, &svc.CategoryID, &svc.Name, &svc.Slug, &svc.SKU,
		&svc.ShortDescription, &svc.FullDescription,
		&svc.Highlights, &svc.Includes, &svc.Excludes,
		&svc.PricingModel, &svc.BasePrice, &svc.PriceUnit,
		&svc.MinPrice, &svc.MaxPrice, &svc.Currency,
		&svc.MinQuantity, &svc.MaxQuantity, &svc.MinGuests, &svc.MaxGuests,
		&svc.DurationMinutes, &svc.IsAvailable, &svc.AvailabilityType, &svc.LeadTimeHours,
		&svc.IsFeatured, &svc.RatingAverage, &svc.RatingCount, &svc.BookingCount,
		&svc.Status, &svc.CreatedAt, &svc.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrServiceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return &svc, nil
}

// GetByVendorID retrieves all services for a vendor
func (s *ServiceManager) GetByVendorID(ctx context.Context, vendorID uuid.UUID, opts *ServiceListOptions) ([]*ServiceOffering, int, error) {
	if opts == nil {
		opts = &ServiceListOptions{Limit: 20, Offset: 0}
	}
	opts.VendorID = &vendorID

	return s.List(ctx, opts)
}

// List retrieves services with optional filtering and pagination
func (s *ServiceManager) List(ctx context.Context, opts *ServiceListOptions) ([]*ServiceOffering, int, error) {
	if opts == nil {
		opts = &ServiceListOptions{}
	}

	// Default values
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	// Build WHERE clause
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argPos := 1

	if opts.VendorID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("vendor_id = $%d", argPos))
		args = append(args, *opts.VendorID)
		argPos++
	}

	if opts.CategoryID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("category_id = $%d", argPos))
		args = append(args, *opts.CategoryID)
		argPos++
	}

	if opts.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *opts.Status)
		argPos++
	}

	if opts.IsAvailable != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("is_available = $%d", argPos))
		args = append(args, *opts.IsAvailable)
		argPos++
	}

	if opts.IsFeatured != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("is_featured = $%d", argPos))
		args = append(args, *opts.IsFeatured)
		argPos++
	}

	if opts.MinPrice != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("base_price >= $%d", argPos))
		args = append(args, *opts.MinPrice)
		argPos++
	}

	if opts.MaxPrice != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("base_price <= $%d", argPos))
		args = append(args, *opts.MaxPrice)
		argPos++
	}

	if opts.SearchQuery != nil && *opts.SearchQuery != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR short_description ILIKE $%d)", argPos, argPos))
		args = append(args, "%"+*opts.SearchQuery+"%")
		argPos++
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Build ORDER BY clause
	orderBy := "created_at DESC"
	if opts.SortBy != "" {
		order := "DESC"
		if opts.SortOrder == "asc" {
			order = "ASC"
		}

		switch opts.SortBy {
		case "created_at":
			orderBy = "created_at " + order
		case "rating":
			orderBy = "rating_average " + order
		case "price":
			orderBy = "base_price " + order
		case "popularity":
			orderBy = "booking_count " + order
		case "name":
			orderBy = "name " + order
		}
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM services WHERE %s", whereClause)
	var total int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count services: %w", err)
	}

	// Get services
	query := fmt.Sprintf(`
		SELECT id, vendor_id, category_id, name, slug, COALESCE(sku, ''),
		       short_description, full_description,
		       COALESCE(highlights, '{}'), COALESCE(includes, '{}'), COALESCE(excludes, '{}'),
		       pricing_model, base_price, COALESCE(price_unit, ''),
		       min_price, max_price, currency,
		       min_quantity, max_quantity, min_guests, max_guests,
		       duration_minutes, is_available, availability_type, lead_time_hours,
		       is_featured, rating_average, rating_count, booking_count,
		       status, created_at, updated_at
		FROM services
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argPos, argPos+1)

	args = append(args, opts.Limit, opts.Offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list services: %w", err)
	}
	defer rows.Close()

	services := []*ServiceOffering{}
	for rows.Next() {
		var svc ServiceOffering
		err := rows.Scan(
			&svc.ID, &svc.VendorID, &svc.CategoryID, &svc.Name, &svc.Slug, &svc.SKU,
			&svc.ShortDescription, &svc.FullDescription,
			&svc.Highlights, &svc.Includes, &svc.Excludes,
			&svc.PricingModel, &svc.BasePrice, &svc.PriceUnit,
			&svc.MinPrice, &svc.MaxPrice, &svc.Currency,
			&svc.MinQuantity, &svc.MaxQuantity, &svc.MinGuests, &svc.MaxGuests,
			&svc.DurationMinutes, &svc.IsAvailable, &svc.AvailabilityType, &svc.LeadTimeHours,
			&svc.IsFeatured, &svc.RatingAverage, &svc.RatingCount, &svc.BookingCount,
			&svc.Status, &svc.CreatedAt, &svc.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan service: %w", err)
		}
		services = append(services, &svc)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating services: %w", err)
	}

	return services, total, nil
}
