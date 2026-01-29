// Package vendor provides vendor management business logic
package vendor

// Note: This service requires the vendors table with the following columns:
// id, user_id, business_name, slug, short_description, full_description, email, phone, website,
// address, city, state, country, latitude, longitude, primary_category_id, category_ids,
// business_type, years_in_business, team_size, status, is_verified, verified_at,
// rating_average, rating_count, completed_bookings, response_time_hours,
// subscription_tier, subscription_ends, created_at, updated_at

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
	ErrVendorNotFound    = errors.New("vendor not found")
	ErrInvalidVendorData = errors.New("invalid vendor data")
	ErrVendorExists      = errors.New("vendor already exists")
	ErrUnauthorized      = errors.New("unauthorized")
)

// Service handles vendor-related operations
type Service struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewService creates a new vendor service
func NewService(db *pgxpool.Pool, cache *redis.Client) *Service {
	return &Service{
		db:    db,
		cache: cache,
	}
}

// Vendor represents a vendor in the system
type Vendor struct {
	ID                uuid.UUID              `json:"id"`
	UserID            uuid.UUID              `json:"user_id"`
	BusinessName      string                 `json:"business_name"`
	Slug              string                 `json:"slug"`
	ShortDescription  string                 `json:"short_description"`
	FullDescription   string                 `json:"full_description"`
	Email             string                 `json:"email"`
	Phone             string                 `json:"phone"`
	Website           string                 `json:"website,omitempty"`

	// Location
	Address           string                 `json:"address"`
	City              string                 `json:"city"`
	State             string                 `json:"state"`
	Country           string                 `json:"country"`
	Latitude          *float64               `json:"latitude,omitempty"`
	Longitude         *float64               `json:"longitude,omitempty"`

	// Categories
	PrimaryCategoryID uuid.UUID              `json:"primary_category_id"`
	CategoryIDs       []uuid.UUID            `json:"category_ids"`

	// Business details
	BusinessType      string                 `json:"business_type"` // individual, company, enterprise
	YearsInBusiness   int                    `json:"years_in_business"`
	TeamSize          int                    `json:"team_size"`

	// Status
	Status            string                 `json:"status"` // pending, active, suspended, inactive
	IsVerified        bool                   `json:"is_verified"`
	VerifiedAt        *time.Time             `json:"verified_at,omitempty"`

	// Stats
	RatingAverage     float64                `json:"rating_average"`
	RatingCount       int                    `json:"rating_count"`
	CompletedBookings int                    `json:"completed_bookings"`
	ResponseTime      *int                   `json:"response_time_hours,omitempty"`

	// Subscription
	SubscriptionTier  string                 `json:"subscription_tier"` // basic, professional, business
	SubscriptionEnds  *time.Time             `json:"subscription_ends,omitempty"`

	// Metadata
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// CreateVendorRequest represents a request to create a vendor
type CreateVendorRequest struct {
	UserID            uuid.UUID   `json:"user_id"`
	BusinessName      string      `json:"business_name"`
	ShortDescription  string      `json:"short_description"`
	Email             string      `json:"email"`
	Phone             string      `json:"phone"`
	Address           string      `json:"address"`
	City              string      `json:"city"`
	State             string      `json:"state"`
	Country           string      `json:"country"`
	PrimaryCategoryID uuid.UUID   `json:"primary_category_id"`
	CategoryIDs       []uuid.UUID `json:"category_ids,omitempty"`
	BusinessType      string      `json:"business_type"`
}

// UpdateVendorRequest represents a request to update a vendor
type UpdateVendorRequest struct {
	BusinessName     *string     `json:"business_name,omitempty"`
	ShortDescription *string     `json:"short_description,omitempty"`
	FullDescription  *string     `json:"full_description,omitempty"`
	Email            *string     `json:"email,omitempty"`
	Phone            *string     `json:"phone,omitempty"`
	Website          *string     `json:"website,omitempty"`
	Address          *string     `json:"address,omitempty"`
	City             *string     `json:"city,omitempty"`
	State            *string     `json:"state,omitempty"`
	CategoryIDs      []uuid.UUID `json:"category_ids,omitempty"`
	YearsInBusiness  *int        `json:"years_in_business,omitempty"`
	TeamSize         *int        `json:"team_size,omitempty"`
}

// VendorListOptions represents options for listing vendors
type VendorListOptions struct {
	CategoryID    *uuid.UUID
	City          *string
	State         *string
	Status        *string
	IsVerified    *bool
	MinRating     *float64
	SearchQuery   *string
	Latitude      *float64
	Longitude     *float64
	RadiusKm      *float64
	Limit         int
	Offset        int
	SortBy        string // rating, created_at, bookings
	SortOrder     string // asc, desc
}

// Create creates a new vendor
func (s *Service) Create(ctx context.Context, req *CreateVendorRequest) (*Vendor, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidVendorData, err)
	}

	// Generate slug from business name
	slug := s.generateSlug(req.BusinessName)

	// Check if slug already exists
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM vendors WHERE slug = $1)", slug).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check slug: %w", err)
	}
	if exists {
		// Add random suffix to make unique
		slug = fmt.Sprintf("%s-%s", slug, uuid.New().String()[:8])
	}

	// Insert vendor
	vendor := &Vendor{
		ID:                uuid.New(),
		UserID:            req.UserID,
		BusinessName:      req.BusinessName,
		Slug:              slug,
		ShortDescription:  req.ShortDescription,
		Email:             req.Email,
		Phone:             req.Phone,
		Address:           req.Address,
		City:              req.City,
		State:             req.State,
		Country:           req.Country,
		PrimaryCategoryID: req.PrimaryCategoryID,
		CategoryIDs:       req.CategoryIDs,
		BusinessType:      req.BusinessType,
		Status:            "pending",
		IsVerified:        false,
		SubscriptionTier:  "basic",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	query := `
		INSERT INTO vendors (
			id, user_id, business_name, slug, short_description,
			email, phone, address, city, state, country,
			primary_category_id, category_ids, business_type,
			status, is_verified, subscription_tier, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		)
	`

	_, err = s.db.Exec(ctx, query,
		vendor.ID, vendor.UserID, vendor.BusinessName, vendor.Slug, vendor.ShortDescription,
		vendor.Email, vendor.Phone, vendor.Address, vendor.City, vendor.State, vendor.Country,
		vendor.PrimaryCategoryID, vendor.CategoryIDs, vendor.BusinessType,
		vendor.Status, vendor.IsVerified, vendor.SubscriptionTier, vendor.CreatedAt, vendor.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create vendor: %w", err)
	}

	return vendor, nil
}

// GetByID retrieves a vendor by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Vendor, error) {
	vendor := &Vendor{}

	query := `
		SELECT
			id, user_id, business_name, slug, short_description, full_description,
			email, phone, website, address, city, state, country, latitude, longitude,
			primary_category_id, category_ids, business_type, years_in_business, team_size,
			status, is_verified, verified_at, rating_average, rating_count, completed_bookings,
			response_time_hours, subscription_tier, subscription_ends,
			created_at, updated_at
		FROM vendors
		WHERE id = $1
	`

	err := s.db.QueryRow(ctx, query, id).Scan(
		&vendor.ID, &vendor.UserID, &vendor.BusinessName, &vendor.Slug,
		&vendor.ShortDescription, &vendor.FullDescription,
		&vendor.Email, &vendor.Phone, &vendor.Website,
		&vendor.Address, &vendor.City, &vendor.State, &vendor.Country,
		&vendor.Latitude, &vendor.Longitude,
		&vendor.PrimaryCategoryID, &vendor.CategoryIDs,
		&vendor.BusinessType, &vendor.YearsInBusiness, &vendor.TeamSize,
		&vendor.Status, &vendor.IsVerified, &vendor.VerifiedAt,
		&vendor.RatingAverage, &vendor.RatingCount, &vendor.CompletedBookings,
		&vendor.ResponseTime, &vendor.SubscriptionTier, &vendor.SubscriptionEnds,
		&vendor.CreatedAt, &vendor.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrVendorNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vendor: %w", err)
	}

	return vendor, nil
}

// GetBySlug retrieves a vendor by slug
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Vendor, error) {
	vendor := &Vendor{}

	query := `
		SELECT
			id, user_id, business_name, slug, short_description, full_description,
			email, phone, website, address, city, state, country, latitude, longitude,
			primary_category_id, category_ids, business_type, years_in_business, team_size,
			status, is_verified, verified_at, rating_average, rating_count, completed_bookings,
			response_time_hours, subscription_tier, subscription_ends,
			created_at, updated_at
		FROM vendors
		WHERE slug = $1
	`

	err := s.db.QueryRow(ctx, query, slug).Scan(
		&vendor.ID, &vendor.UserID, &vendor.BusinessName, &vendor.Slug,
		&vendor.ShortDescription, &vendor.FullDescription,
		&vendor.Email, &vendor.Phone, &vendor.Website,
		&vendor.Address, &vendor.City, &vendor.State, &vendor.Country,
		&vendor.Latitude, &vendor.Longitude,
		&vendor.PrimaryCategoryID, &vendor.CategoryIDs,
		&vendor.BusinessType, &vendor.YearsInBusiness, &vendor.TeamSize,
		&vendor.Status, &vendor.IsVerified, &vendor.VerifiedAt,
		&vendor.RatingAverage, &vendor.RatingCount, &vendor.CompletedBookings,
		&vendor.ResponseTime, &vendor.SubscriptionTier, &vendor.SubscriptionEnds,
		&vendor.CreatedAt, &vendor.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrVendorNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vendor: %w", err)
	}

	return vendor, nil
}

// Update updates a vendor
func (s *Service) Update(ctx context.Context, id uuid.UUID, req *UpdateVendorRequest) (*Vendor, error) {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{id}
	argPos := 2

	if req.BusinessName != nil {
		updates = append(updates, fmt.Sprintf("business_name = $%d", argPos))
		args = append(args, *req.BusinessName)
		argPos++
	}
	if req.ShortDescription != nil {
		updates = append(updates, fmt.Sprintf("short_description = $%d", argPos))
		args = append(args, *req.ShortDescription)
		argPos++
	}
	if req.FullDescription != nil {
		updates = append(updates, fmt.Sprintf("full_description = $%d", argPos))
		args = append(args, *req.FullDescription)
		argPos++
	}
	if req.Email != nil {
		updates = append(updates, fmt.Sprintf("email = $%d", argPos))
		args = append(args, *req.Email)
		argPos++
	}
	if req.Phone != nil {
		updates = append(updates, fmt.Sprintf("phone = $%d", argPos))
		args = append(args, *req.Phone)
		argPos++
	}
	if req.Website != nil {
		updates = append(updates, fmt.Sprintf("website = $%d", argPos))
		args = append(args, *req.Website)
		argPos++
	}
	if req.Address != nil {
		updates = append(updates, fmt.Sprintf("address = $%d", argPos))
		args = append(args, *req.Address)
		argPos++
	}
	if req.City != nil {
		updates = append(updates, fmt.Sprintf("city = $%d", argPos))
		args = append(args, *req.City)
		argPos++
	}
	if req.State != nil {
		updates = append(updates, fmt.Sprintf("state = $%d", argPos))
		args = append(args, *req.State)
		argPos++
	}
	if req.CategoryIDs != nil {
		updates = append(updates, fmt.Sprintf("category_ids = $%d", argPos))
		args = append(args, req.CategoryIDs)
		argPos++
	}
	if req.YearsInBusiness != nil {
		updates = append(updates, fmt.Sprintf("years_in_business = $%d", argPos))
		args = append(args, *req.YearsInBusiness)
		argPos++
	}
	if req.TeamSize != nil {
		updates = append(updates, fmt.Sprintf("team_size = $%d", argPos))
		args = append(args, *req.TeamSize)
		argPos++
	}

	if len(updates) == 0 {
		return s.GetByID(ctx, id)
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, time.Now())

	query := fmt.Sprintf("UPDATE vendors SET %s WHERE id = $1", strings.Join(updates, ", "))

	_, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update vendor: %w", err)
	}

	return s.GetByID(ctx, id)
}

// List retrieves a list of vendors based on options
func (s *Service) List(ctx context.Context, opts *VendorListOptions) ([]*Vendor, int, error) {
	if opts.Limit == 0 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	// Build query with filters
	baseQuery := `FROM vendors WHERE 1=1`
	countQuery := `SELECT COUNT(*) ` + baseQuery
	selectQuery := `
		SELECT
			id, user_id, business_name, slug, short_description,
			email, phone, address, city, state, country,
			primary_category_id, category_ids, business_type,
			status, is_verified, rating_average, rating_count, completed_bookings,
			subscription_tier, created_at, updated_at
	` + baseQuery

	args := []interface{}{}
	argPos := 1

	// Apply filters
	if opts.CategoryID != nil {
		baseQuery += fmt.Sprintf(" AND $%d = ANY(category_ids)", argPos)
		args = append(args, *opts.CategoryID)
		argPos++
	}
	if opts.City != nil {
		baseQuery += fmt.Sprintf(" AND LOWER(city) = LOWER($%d)", argPos)
		args = append(args, *opts.City)
		argPos++
	}
	if opts.State != nil {
		baseQuery += fmt.Sprintf(" AND LOWER(state) = LOWER($%d)", argPos)
		args = append(args, *opts.State)
		argPos++
	}
	if opts.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *opts.Status)
		argPos++
	}
	if opts.IsVerified != nil {
		baseQuery += fmt.Sprintf(" AND is_verified = $%d", argPos)
		args = append(args, *opts.IsVerified)
		argPos++
	}
	if opts.MinRating != nil {
		baseQuery += fmt.Sprintf(" AND rating_average >= $%d", argPos)
		args = append(args, *opts.MinRating)
		argPos++
	}
	if opts.SearchQuery != nil && *opts.SearchQuery != "" {
		baseQuery += fmt.Sprintf(" AND (business_name ILIKE $%d OR short_description ILIKE $%d)", argPos, argPos)
		searchTerm := "%" + *opts.SearchQuery + "%"
		args = append(args, searchTerm)
		argPos++
	}

	// Get total count
	var total int
	err := s.db.QueryRow(ctx, countQuery+baseQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count vendors: %w", err)
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

	selectQuery = selectQuery + baseQuery + fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderBy, argPos, argPos+1)
	args = append(args, opts.Limit, opts.Offset)

	// Execute query
	rows, err := s.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list vendors: %w", err)
	}
	defer rows.Close()

	vendors := []*Vendor{}
	for rows.Next() {
		vendor := &Vendor{}
		err := rows.Scan(
			&vendor.ID, &vendor.UserID, &vendor.BusinessName, &vendor.Slug,
			&vendor.ShortDescription, &vendor.Email, &vendor.Phone,
			&vendor.Address, &vendor.City, &vendor.State, &vendor.Country,
			&vendor.PrimaryCategoryID, &vendor.CategoryIDs, &vendor.BusinessType,
			&vendor.Status, &vendor.IsVerified, &vendor.RatingAverage,
			&vendor.RatingCount, &vendor.CompletedBookings, &vendor.SubscriptionTier,
			&vendor.CreatedAt, &vendor.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan vendor: %w", err)
		}
		vendors = append(vendors, vendor)
	}

	return vendors, total, nil
}

// Verify verifies a vendor
func (s *Service) Verify(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `UPDATE vendors SET is_verified = true, verified_at = $1, status = 'active', updated_at = $2 WHERE id = $3`

	_, err := s.db.Exec(ctx, query, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to verify vendor: %w", err)
	}

	return nil
}

// Delete soft deletes a vendor
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE vendors SET status = 'inactive', updated_at = $1 WHERE id = $2`

	_, err := s.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete vendor: %w", err)
	}

	return nil
}

// Helper methods

func (s *Service) validateCreateRequest(req *CreateVendorRequest) error {
	if req.BusinessName == "" {
		return errors.New("business name is required")
	}
	if req.ShortDescription == "" {
		return errors.New("short description is required")
	}
	if req.Email == "" {
		return errors.New("email is required")
	}
	if req.Phone == "" {
		return errors.New("phone is required")
	}
	if req.City == "" {
		return errors.New("city is required")
	}
	if req.State == "" {
		return errors.New("state is required")
	}
	if req.PrimaryCategoryID == uuid.Nil {
		return errors.New("primary category is required")
	}
	return nil
}

func (s *Service) generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "&", "and")
	// Remove special characters
	replacer := strings.NewReplacer(
		"'", "", "\"", "", ",", "", ".", "", "!", "", "?", "",
		"(", "", ")", "", "[", "", "]", "", "{", "", "}", "",
	)
	slug = replacer.Replace(slug)
	return slug
}
