// Package service provides business logic for service management
package service

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
	ErrServiceNotFound      = errors.New("service not found")
	ErrInvalidInput         = errors.New("invalid input")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrServiceAlreadyExists = errors.New("service already exists")
)

// Service represents a vendor offering
type Service struct {
	ID              uuid.UUID  `json:"id"`
	VendorID        uuid.UUID  `json:"vendor_id"`
	CategoryID      uuid.UUID  `json:"category_id"`
	Name            string     `json:"name"`
	Slug            string     `json:"slug"`
	SKU             *string    `json:"sku,omitempty"`
	ShortDescription *string   `json:"short_description,omitempty"`
	FullDescription *string    `json:"full_description,omitempty"`
	Highlights      []string   `json:"highlights,omitempty"`
	Includes        []string   `json:"includes,omitempty"`
	Excludes        []string   `json:"excludes,omitempty"`
	PricingModel    string     `json:"pricing_model"`
	BasePrice       *float64   `json:"base_price,omitempty"`
	PriceUnit       *string    `json:"price_unit,omitempty"`
	MinPrice        *float64   `json:"min_price,omitempty"`
	MaxPrice        *float64   `json:"max_price,omitempty"`
	Currency        string     `json:"currency"`
	MinQuantity     int        `json:"min_quantity"`
	MaxQuantity     *int       `json:"max_quantity,omitempty"`
	MinGuests       *int       `json:"min_guests,omitempty"`
	MaxGuests       *int       `json:"max_guests,omitempty"`
	DurationMinutes *int       `json:"duration_minutes,omitempty"`
	MinDurationMinutes *int    `json:"min_duration_minutes,omitempty"`
	MaxDurationMinutes *int    `json:"max_duration_minutes,omitempty"`
	SetupTimeMinutes int       `json:"setup_time_minutes"`
	CleanupTimeMinutes int     `json:"cleanup_time_minutes"`
	IsAvailable     bool       `json:"is_available"`
	AvailabilityType string    `json:"availability_type"`
	LeadTimeHours   *int       `json:"lead_time_hours,omitempty"`
	Images          []string   `json:"images,omitempty"`
	Videos          []string   `json:"videos,omitempty"`
	Variations      interface{} `json:"variations,omitempty"`
	Addons          interface{} `json:"addons,omitempty"`
	RatingAverage   float64    `json:"rating_average"`
	RatingCount     int        `json:"rating_count"`
	BookingCount    int        `json:"booking_count"`
	DisplayOrder    int        `json:"display_order"`
	IsFeatured      bool       `json:"is_featured"`
	Tags            []string   `json:"tags,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CreateServiceRequest represents a request to create a service
type CreateServiceRequest struct {
	VendorID         uuid.UUID   `json:"vendor_id"`
	CategoryID       uuid.UUID   `json:"category_id"`
	Name             string      `json:"name"`
	ShortDescription *string     `json:"short_description,omitempty"`
	FullDescription  *string     `json:"full_description,omitempty"`
	Highlights       []string    `json:"highlights,omitempty"`
	Includes         []string    `json:"includes,omitempty"`
	Excludes         []string    `json:"excludes,omitempty"`
	PricingModel     string      `json:"pricing_model"`
	BasePrice        *float64    `json:"base_price,omitempty"`
	PriceUnit        *string     `json:"price_unit,omitempty"`
	MinPrice         *float64    `json:"min_price,omitempty"`
	MaxPrice         *float64    `json:"max_price,omitempty"`
	Currency         string      `json:"currency"`
	MinQuantity      int         `json:"min_quantity"`
	MaxQuantity      *int        `json:"max_quantity,omitempty"`
	DurationMinutes  *int        `json:"duration_minutes,omitempty"`
	Images           []string    `json:"images,omitempty"`
	Tags             []string    `json:"tags,omitempty"`
}

// UpdateServiceRequest represents a request to update a service
type UpdateServiceRequest struct {
	Name             *string     `json:"name,omitempty"`
	ShortDescription *string     `json:"short_description,omitempty"`
	FullDescription  *string     `json:"full_description,omitempty"`
	BasePrice        *float64    `json:"base_price,omitempty"`
	IsAvailable      *bool       `json:"is_available,omitempty"`
	Images           []string    `json:"images,omitempty"`
	Tags             []string    `json:"tags,omitempty"`
}

// ServiceFilter represents filters for service search
type ServiceFilter struct {
	VendorID         *uuid.UUID
	CategoryID       *uuid.UUID
	PricingModel     *string
	MinPrice         *float64
	MaxPrice         *float64
	IsAvailable      *bool
	IsFeatured       *bool
	Tags             []string
	SearchQuery      *string
	Limit            int
	Offset           int
}

// ServiceService handles service business logic
type ServiceService struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewService creates a new service service
func NewService(db *pgxpool.Pool, cache *redis.Client) *ServiceService {
	return &ServiceService{
		db:    db,
		cache: cache,
	}
}

// Create creates a new service
func (s *ServiceService) Create(ctx context.Context, req *CreateServiceRequest) (*Service, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if req.PricingModel == "" {
		return nil, fmt.Errorf("%w: pricing model is required", ErrInvalidInput)
	}

	// Generate slug from name
	slug := generateSlug(req.Name)

	// Check if service with same slug exists for this vendor
	var exists bool
	err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM services WHERE vendor_id = $1 AND slug = $2)",
		req.VendorID, slug,
	).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check service existence: %w", err)
	}
	if exists {
		return nil, ErrServiceAlreadyExists
	}

	service := &Service{
		ID:                 uuid.New(),
		VendorID:           req.VendorID,
		CategoryID:         req.CategoryID,
		Name:               req.Name,
		Slug:               slug,
		ShortDescription:   req.ShortDescription,
		FullDescription:    req.FullDescription,
		Highlights:         req.Highlights,
		Includes:           req.Includes,
		Excludes:           req.Excludes,
		PricingModel:       req.PricingModel,
		BasePrice:          req.BasePrice,
		PriceUnit:          req.PriceUnit,
		MinPrice:           req.MinPrice,
		MaxPrice:           req.MaxPrice,
		Currency:           req.Currency,
		MinQuantity:        req.MinQuantity,
		MaxQuantity:        req.MaxQuantity,
		DurationMinutes:    req.DurationMinutes,
		Images:             req.Images,
		Tags:               req.Tags,
		IsAvailable:        true,
		AvailabilityType:   "always",
		SetupTimeMinutes:   0,
		CleanupTimeMinutes: 0,
		RatingAverage:      0,
		RatingCount:        0,
		BookingCount:       0,
		DisplayOrder:       0,
		IsFeatured:         false,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if service.Currency == "" {
		service.Currency = "NGN"
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO services (
			id, vendor_id, category_id, name, slug, short_description, full_description,
			highlights, includes, excludes, pricing_model, base_price, price_unit,
			min_price, max_price, currency, min_quantity, max_quantity, duration_minutes,
			setup_time_minutes, cleanup_time_minutes, is_available, availability_type,
			images, tags, rating_average, rating_count, booking_count, display_order,
			is_featured, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32
		)
	`,
		service.ID, service.VendorID, service.CategoryID, service.Name, service.Slug,
		service.ShortDescription, service.FullDescription, service.Highlights,
		service.Includes, service.Excludes, service.PricingModel, service.BasePrice,
		service.PriceUnit, service.MinPrice, service.MaxPrice, service.Currency,
		service.MinQuantity, service.MaxQuantity, service.DurationMinutes,
		service.SetupTimeMinutes, service.CleanupTimeMinutes, service.IsAvailable,
		service.AvailabilityType, service.Images, service.Tags, service.RatingAverage,
		service.RatingCount, service.BookingCount, service.DisplayOrder, service.IsFeatured,
		service.CreatedAt, service.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return service, nil
}

// GetByID retrieves a service by ID
func (s *ServiceService) GetByID(ctx context.Context, id uuid.UUID) (*Service, error) {
	service := &Service{}
	err := s.db.QueryRow(ctx, `
		SELECT id, vendor_id, category_id, name, slug, sku, short_description,
			full_description, highlights, includes, excludes, pricing_model, base_price,
			price_unit, min_price, max_price, currency, min_quantity, max_quantity,
			min_guests, max_guests, duration_minutes, min_duration_minutes, max_duration_minutes,
			setup_time_minutes, cleanup_time_minutes, is_available, availability_type,
			lead_time_hours, images, videos, variations, addons, rating_average,
			rating_count, booking_count, display_order, is_featured, tags,
			created_at, updated_at
		FROM services
		WHERE id = $1
	`, id).Scan(
		&service.ID, &service.VendorID, &service.CategoryID, &service.Name, &service.Slug,
		&service.SKU, &service.ShortDescription, &service.FullDescription, &service.Highlights,
		&service.Includes, &service.Excludes, &service.PricingModel, &service.BasePrice,
		&service.PriceUnit, &service.MinPrice, &service.MaxPrice, &service.Currency,
		&service.MinQuantity, &service.MaxQuantity, &service.MinGuests, &service.MaxGuests,
		&service.DurationMinutes, &service.MinDurationMinutes, &service.MaxDurationMinutes,
		&service.SetupTimeMinutes, &service.CleanupTimeMinutes, &service.IsAvailable,
		&service.AvailabilityType, &service.LeadTimeHours, &service.Images, &service.Videos,
		&service.Variations, &service.Addons, &service.RatingAverage, &service.RatingCount,
		&service.BookingCount, &service.DisplayOrder, &service.IsFeatured, &service.Tags,
		&service.CreatedAt, &service.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrServiceNotFound
		}
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return service, nil
}

// List retrieves services with filters
func (s *ServiceService) List(ctx context.Context, filter *ServiceFilter) ([]*Service, error) {
	query := `
		SELECT id, vendor_id, category_id, name, slug, sku, short_description,
			full_description, highlights, includes, excludes, pricing_model, base_price,
			price_unit, min_price, max_price, currency, min_quantity, max_quantity,
			min_guests, max_guests, duration_minutes, min_duration_minutes, max_duration_minutes,
			setup_time_minutes, cleanup_time_minutes, is_available, availability_type,
			lead_time_hours, images, videos, variations, addons, rating_average,
			rating_count, booking_count, display_order, is_featured, tags,
			created_at, updated_at
		FROM services
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	if filter.VendorID != nil {
		query += fmt.Sprintf(" AND vendor_id = $%d", argCount)
		args = append(args, *filter.VendorID)
		argCount++
	}

	if filter.CategoryID != nil {
		query += fmt.Sprintf(" AND category_id = $%d", argCount)
		args = append(args, *filter.CategoryID)
		argCount++
	}

	if filter.PricingModel != nil {
		query += fmt.Sprintf(" AND pricing_model = $%d", argCount)
		args = append(args, *filter.PricingModel)
		argCount++
	}

	if filter.MinPrice != nil {
		query += fmt.Sprintf(" AND base_price >= $%d", argCount)
		args = append(args, *filter.MinPrice)
		argCount++
	}

	if filter.MaxPrice != nil {
		query += fmt.Sprintf(" AND base_price <= $%d", argCount)
		args = append(args, *filter.MaxPrice)
		argCount++
	}

	if filter.IsAvailable != nil {
		query += fmt.Sprintf(" AND is_available = $%d", argCount)
		args = append(args, *filter.IsAvailable)
		argCount++
	}

	if filter.IsFeatured != nil {
		query += fmt.Sprintf(" AND is_featured = $%d", argCount)
		args = append(args, *filter.IsFeatured)
		argCount++
	}

	if len(filter.Tags) > 0 {
		query += fmt.Sprintf(" AND tags && $%d", argCount)
		args = append(args, filter.Tags)
		argCount++
	}

	if filter.SearchQuery != nil && *filter.SearchQuery != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR short_description ILIKE $%d OR full_description ILIKE $%d)", argCount, argCount, argCount)
		searchPattern := "%" + *filter.SearchQuery + "%"
		args = append(args, searchPattern)
		argCount++
	}

	query += " ORDER BY is_featured DESC, rating_average DESC, created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	} else {
		query += " LIMIT 50"
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	defer rows.Close()

	services := []*Service{}
	for rows.Next() {
		service := &Service{}
		err := rows.Scan(
			&service.ID, &service.VendorID, &service.CategoryID, &service.Name, &service.Slug,
			&service.SKU, &service.ShortDescription, &service.FullDescription, &service.Highlights,
			&service.Includes, &service.Excludes, &service.PricingModel, &service.BasePrice,
			&service.PriceUnit, &service.MinPrice, &service.MaxPrice, &service.Currency,
			&service.MinQuantity, &service.MaxQuantity, &service.MinGuests, &service.MaxGuests,
			&service.DurationMinutes, &service.MinDurationMinutes, &service.MaxDurationMinutes,
			&service.SetupTimeMinutes, &service.CleanupTimeMinutes, &service.IsAvailable,
			&service.AvailabilityType, &service.LeadTimeHours, &service.Images, &service.Videos,
			&service.Variations, &service.Addons, &service.RatingAverage, &service.RatingCount,
			&service.BookingCount, &service.DisplayOrder, &service.IsFeatured, &service.Tags,
			&service.CreatedAt, &service.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}
		services = append(services, service)
	}

	return services, nil
}

// Update updates a service
func (s *ServiceService) Update(ctx context.Context, id uuid.UUID, vendorID uuid.UUID, req *UpdateServiceRequest) (*Service, error) {
	// First, verify the service exists and belongs to the vendor
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.VendorID != vendorID {
		return nil, ErrUnauthorized
	}

	// Build dynamic update query
	query := "UPDATE services SET updated_at = NOW()"
	args := []interface{}{}
	argCount := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d, slug = $%d", argCount, argCount+1)
		args = append(args, *req.Name, generateSlug(*req.Name))
		argCount += 2
	}

	if req.ShortDescription != nil {
		query += fmt.Sprintf(", short_description = $%d", argCount)
		args = append(args, *req.ShortDescription)
		argCount++
	}

	if req.FullDescription != nil {
		query += fmt.Sprintf(", full_description = $%d", argCount)
		args = append(args, *req.FullDescription)
		argCount++
	}

	if req.BasePrice != nil {
		query += fmt.Sprintf(", base_price = $%d", argCount)
		args = append(args, *req.BasePrice)
		argCount++
	}

	if req.IsAvailable != nil {
		query += fmt.Sprintf(", is_available = $%d", argCount)
		args = append(args, *req.IsAvailable)
		argCount++
	}

	if req.Images != nil {
		query += fmt.Sprintf(", images = $%d", argCount)
		args = append(args, req.Images)
		argCount++
	}

	if req.Tags != nil {
		query += fmt.Sprintf(", tags = $%d", argCount)
		args = append(args, req.Tags)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND vendor_id = $%d", argCount, argCount+1)
	args = append(args, id, vendorID)

	_, err = s.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update service: %w", err)
	}

	return s.GetByID(ctx, id)
}

// Delete deletes a service
func (s *ServiceService) Delete(ctx context.Context, id uuid.UUID, vendorID uuid.UUID) error {
	// Verify the service exists and belongs to the vendor
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.VendorID != vendorID {
		return ErrUnauthorized
	}

	_, err = s.db.Exec(ctx, "DELETE FROM services WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}

// generateSlug creates a URL-friendly slug from a string
func generateSlug(s string) string {
	// Simple slug generation - in production, use a proper library
	slug := s
	slug = removeSpecialChars(slug)
	return slug
}

func removeSpecialChars(s string) string {
	// Basic implementation - replace spaces with hyphens and lowercase
	result := ""
	for _, char := range s {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			result += string(char)
		} else if char == ' ' {
			result += "-"
		}
	}
	return result
}
