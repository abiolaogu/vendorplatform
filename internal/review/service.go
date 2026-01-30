// Package review provides review and rating management business logic
package review

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
	ErrReviewNotFound      = errors.New("review not found")
	ErrInvalidReviewData   = errors.New("invalid review data")
	ErrDuplicateReview     = errors.New("review already exists for this booking")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrBookingNotCompleted = errors.New("booking must be completed before reviewing")
)

// Service handles review-related operations
type Service struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewService creates a new review service
func NewService(db *pgxpool.Pool, cache *redis.Client) *Service {
	return &Service{
		db:    db,
		cache: cache,
	}
}

// Review represents a review in the system
type Review struct {
	ID       uuid.UUID  `json:"id"`
	VendorID uuid.UUID  `json:"vendor_id"`
	UserID   uuid.UUID  `json:"user_id"`
	BookingID *uuid.UUID `json:"booking_id,omitempty"`

	// Rating
	Rating               int `json:"rating"`
	QualityRating        *int `json:"quality_rating,omitempty"`
	CommunicationRating  *int `json:"communication_rating,omitempty"`
	TimelinessRating     *int `json:"timeliness_rating,omitempty"`
	ValueRating          *int `json:"value_rating,omitempty"`

	// Content
	Title   string   `json:"title,omitempty"`
	Comment string   `json:"comment"`
	ImageURLs []string `json:"image_urls,omitempty"`

	// Status
	IsVerified  bool   `json:"is_verified"`
	IsPublished bool   `json:"is_published"`
	IsFlagged   bool   `json:"is_flagged"`
	FlagReason  string `json:"flag_reason,omitempty"`

	// Engagement
	HelpfulCount    int `json:"helpful_count"`
	NotHelpfulCount int `json:"not_helpful_count"`

	// Vendor Response
	VendorResponse    string     `json:"vendor_response,omitempty"`
	VendorRespondedAt *time.Time `json:"vendor_responded_at,omitempty"`

	// User Info (populated from join)
	UserName   string `json:"user_name,omitempty"`
	UserAvatar string `json:"user_avatar,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateReviewRequest represents a request to create a review
type CreateReviewRequest struct {
	VendorID  uuid.UUID  `json:"vendor_id"`
	UserID    uuid.UUID  `json:"user_id"`
	BookingID *uuid.UUID `json:"booking_id,omitempty"`

	Rating               int    `json:"rating"`
	QualityRating        *int   `json:"quality_rating,omitempty"`
	CommunicationRating  *int   `json:"communication_rating,omitempty"`
	TimelinessRating     *int   `json:"timeliness_rating,omitempty"`
	ValueRating          *int   `json:"value_rating,omitempty"`

	Title     string   `json:"title,omitempty"`
	Comment   string   `json:"comment"`
	ImageURLs []string `json:"image_urls,omitempty"`
}

// UpdateReviewRequest represents a request to update a review
type UpdateReviewRequest struct {
	Rating               *int     `json:"rating,omitempty"`
	QualityRating        *int     `json:"quality_rating,omitempty"`
	CommunicationRating  *int     `json:"communication_rating,omitempty"`
	TimelinessRating     *int     `json:"timeliness_rating,omitempty"`
	ValueRating          *int     `json:"value_rating,omitempty"`

	Title     *string  `json:"title,omitempty"`
	Comment   *string  `json:"comment,omitempty"`
	ImageURLs []string `json:"image_urls,omitempty"`
}

// ReviewListOptions represents options for listing reviews
type ReviewListOptions struct {
	VendorID     *uuid.UUID
	UserID       *uuid.UUID
	MinRating    *int
	IsVerified   *bool
	WithResponse *bool
	Limit        int
	Offset       int
	SortBy       string // created_at, rating, helpful
	SortOrder    string // asc, desc
}

// Create creates a new review
func (s *Service) Create(ctx context.Context, req *CreateReviewRequest) (*Review, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidReviewData, err)
	}

	// If booking_id is provided, verify booking exists and is completed
	if req.BookingID != nil {
		var bookingStatus string
		err := s.db.QueryRow(ctx,
			"SELECT status FROM bookings WHERE id = $1 AND user_id = $2",
			req.BookingID, req.UserID,
		).Scan(&bookingStatus)

		if err == pgx.ErrNoRows {
			return nil, errors.New("booking not found or does not belong to user")
		}
		if err != nil {
			return nil, fmt.Errorf("failed to verify booking: %w", err)
		}

		if bookingStatus != "completed" && bookingStatus != "confirmed" {
			return nil, ErrBookingNotCompleted
		}
	}

	// Check for duplicate review on same booking
	if req.BookingID != nil {
		var exists bool
		err := s.db.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM reviews WHERE user_id = $1 AND booking_id = $2)",
			req.UserID, req.BookingID,
		).Scan(&exists)

		if err != nil {
			return nil, fmt.Errorf("failed to check duplicate: %w", err)
		}
		if exists {
			return nil, ErrDuplicateReview
		}
	}

	// Create review
	review := &Review{
		ID:                  uuid.New(),
		VendorID:            req.VendorID,
		UserID:              req.UserID,
		BookingID:           req.BookingID,
		Rating:              req.Rating,
		QualityRating:       req.QualityRating,
		CommunicationRating: req.CommunicationRating,
		TimelinessRating:    req.TimelinessRating,
		ValueRating:         req.ValueRating,
		Title:               req.Title,
		Comment:             req.Comment,
		ImageURLs:           req.ImageURLs,
		IsVerified:          req.BookingID != nil, // Verified if linked to booking
		IsPublished:         true,
		IsFlagged:           false,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	query := `
		INSERT INTO reviews (
			id, vendor_id, user_id, booking_id,
			rating, quality_rating, communication_rating, timeliness_rating, value_rating,
			title, comment, image_urls,
			is_verified, is_published, is_flagged,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`

	_, err := s.db.Exec(ctx, query,
		review.ID, review.VendorID, review.UserID, review.BookingID,
		review.Rating, review.QualityRating, review.CommunicationRating,
		review.TimelinessRating, review.ValueRating,
		review.Title, review.Comment, review.ImageURLs,
		review.IsVerified, review.IsPublished, review.IsFlagged,
		review.CreatedAt, review.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create review: %w", err)
	}

	// Trigger updates vendor ratings automatically via database trigger

	return review, nil
}

// GetByID retrieves a review by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Review, error) {
	review := &Review{}

	query := `
		SELECT
			r.id, r.vendor_id, r.user_id, r.booking_id,
			r.rating, r.quality_rating, r.communication_rating, r.timeliness_rating, r.value_rating,
			r.title, r.comment, r.image_urls,
			r.is_verified, r.is_published, r.is_flagged, r.flag_reason,
			r.helpful_count, r.not_helpful_count,
			r.vendor_response, r.vendor_responded_at,
			r.created_at, r.updated_at,
			COALESCE(u.first_name || ' ' || u.last_name, u.display_name, 'Anonymous') as user_name,
			u.avatar_url
		FROM reviews r
		LEFT JOIN users u ON r.user_id = u.id
		WHERE r.id = $1
	`

	err := s.db.QueryRow(ctx, query, id).Scan(
		&review.ID, &review.VendorID, &review.UserID, &review.BookingID,
		&review.Rating, &review.QualityRating, &review.CommunicationRating,
		&review.TimelinessRating, &review.ValueRating,
		&review.Title, &review.Comment, &review.ImageURLs,
		&review.IsVerified, &review.IsPublished, &review.IsFlagged, &review.FlagReason,
		&review.HelpfulCount, &review.NotHelpfulCount,
		&review.VendorResponse, &review.VendorRespondedAt,
		&review.CreatedAt, &review.UpdatedAt,
		&review.UserName, &review.UserAvatar,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrReviewNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	return review, nil
}

// List retrieves a list of reviews based on options
func (s *Service) List(ctx context.Context, opts *ReviewListOptions) ([]*Review, int, error) {
	if opts.Limit == 0 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	// Build query with filters
	baseQuery := `
		FROM reviews r
		LEFT JOIN users u ON r.user_id = u.id
		WHERE r.is_published = TRUE
	`
	countQuery := `SELECT COUNT(*) ` + baseQuery
	selectQuery := `
		SELECT
			r.id, r.vendor_id, r.user_id, r.booking_id,
			r.rating, r.quality_rating, r.communication_rating, r.timeliness_rating, r.value_rating,
			r.title, r.comment, r.image_urls,
			r.is_verified, r.is_published, r.is_flagged, r.flag_reason,
			r.helpful_count, r.not_helpful_count,
			r.vendor_response, r.vendor_responded_at,
			r.created_at, r.updated_at,
			COALESCE(u.first_name || ' ' || u.last_name, u.display_name, 'Anonymous') as user_name,
			u.avatar_url
	` + baseQuery

	args := []interface{}{}
	argPos := 1

	// Apply filters
	if opts.VendorID != nil {
		baseQuery += fmt.Sprintf(" AND r.vendor_id = $%d", argPos)
		selectQuery = selectQuery + fmt.Sprintf(" AND r.vendor_id = $%d", argPos)
		countQuery = countQuery + fmt.Sprintf(" AND r.vendor_id = $%d", argPos)
		args = append(args, *opts.VendorID)
		argPos++
	}
	if opts.UserID != nil {
		baseQuery += fmt.Sprintf(" AND r.user_id = $%d", argPos)
		selectQuery = selectQuery + fmt.Sprintf(" AND r.user_id = $%d", argPos)
		countQuery = countQuery + fmt.Sprintf(" AND r.user_id = $%d", argPos)
		args = append(args, *opts.UserID)
		argPos++
	}
	if opts.MinRating != nil {
		baseQuery += fmt.Sprintf(" AND r.rating >= $%d", argPos)
		selectQuery = selectQuery + fmt.Sprintf(" AND r.rating >= $%d", argPos)
		countQuery = countQuery + fmt.Sprintf(" AND r.rating >= $%d", argPos)
		args = append(args, *opts.MinRating)
		argPos++
	}
	if opts.IsVerified != nil {
		baseQuery += fmt.Sprintf(" AND r.is_verified = $%d", argPos)
		selectQuery = selectQuery + fmt.Sprintf(" AND r.is_verified = $%d", argPos)
		countQuery = countQuery + fmt.Sprintf(" AND r.is_verified = $%d", argPos)
		args = append(args, *opts.IsVerified)
		argPos++
	}
	if opts.WithResponse != nil && *opts.WithResponse {
		baseQuery += " AND r.vendor_response IS NOT NULL"
		selectQuery = selectQuery + " AND r.vendor_response IS NOT NULL"
		countQuery = countQuery + " AND r.vendor_response IS NOT NULL"
	}

	// Get total count
	var total int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count reviews: %w", err)
	}

	// Apply sorting
	orderBy := "r.created_at DESC"
	if opts.SortBy == "rating" {
		if opts.SortOrder == "asc" {
			orderBy = "r.rating ASC"
		} else {
			orderBy = "r.rating DESC"
		}
	} else if opts.SortBy == "helpful" {
		orderBy = "r.helpful_count DESC"
	}

	selectQuery = selectQuery + fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderBy, argPos, argPos+1)
	args = append(args, opts.Limit, opts.Offset)

	// Execute query
	rows, err := s.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list reviews: %w", err)
	}
	defer rows.Close()

	reviews := []*Review{}
	for rows.Next() {
		review := &Review{}
		err := rows.Scan(
			&review.ID, &review.VendorID, &review.UserID, &review.BookingID,
			&review.Rating, &review.QualityRating, &review.CommunicationRating,
			&review.TimelinessRating, &review.ValueRating,
			&review.Title, &review.Comment, &review.ImageURLs,
			&review.IsVerified, &review.IsPublished, &review.IsFlagged, &review.FlagReason,
			&review.HelpfulCount, &review.NotHelpfulCount,
			&review.VendorResponse, &review.VendorRespondedAt,
			&review.CreatedAt, &review.UpdatedAt,
			&review.UserName, &review.UserAvatar,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan review: %w", err)
		}
		reviews = append(reviews, review)
	}

	return reviews, total, nil
}

// Update updates a review
func (s *Service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateReviewRequest) (*Review, error) {
	// Verify user owns this review
	var reviewUserID uuid.UUID
	err := s.db.QueryRow(ctx, "SELECT user_id FROM reviews WHERE id = $1", id).Scan(&reviewUserID)
	if err == pgx.ErrNoRows {
		return nil, ErrReviewNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to verify review ownership: %w", err)
	}

	if reviewUserID != userID {
		return nil, ErrUnauthorized
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{id}
	argPos := 2

	if req.Rating != nil {
		if *req.Rating < 1 || *req.Rating > 5 {
			return nil, fmt.Errorf("%w: rating must be between 1 and 5", ErrInvalidReviewData)
		}
		updates = append(updates, fmt.Sprintf("rating = $%d", argPos))
		args = append(args, *req.Rating)
		argPos++
	}
	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argPos))
		args = append(args, *req.Title)
		argPos++
	}
	if req.Comment != nil {
		updates = append(updates, fmt.Sprintf("comment = $%d", argPos))
		args = append(args, *req.Comment)
		argPos++
	}
	if req.ImageURLs != nil {
		updates = append(updates, fmt.Sprintf("image_urls = $%d", argPos))
		args = append(args, req.ImageURLs)
		argPos++
	}

	if len(updates) == 0 {
		return s.GetByID(ctx, id)
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, time.Now())

	query := fmt.Sprintf("UPDATE reviews SET %s WHERE id = $1", joinUpdates(updates))

	_, err = s.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update review: %w", err)
	}

	return s.GetByID(ctx, id)
}

// Delete deletes a review (soft delete by unpublishing)
func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// Verify user owns this review
	var reviewUserID uuid.UUID
	err := s.db.QueryRow(ctx, "SELECT user_id FROM reviews WHERE id = $1", id).Scan(&reviewUserID)
	if err == pgx.ErrNoRows {
		return ErrReviewNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to verify review ownership: %w", err)
	}

	if reviewUserID != userID {
		return ErrUnauthorized
	}

	query := `UPDATE reviews SET is_published = FALSE, updated_at = $1 WHERE id = $2`
	_, err = s.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete review: %w", err)
	}

	return nil
}

// AddVendorResponse adds a vendor response to a review
func (s *Service) AddVendorResponse(ctx context.Context, reviewID uuid.UUID, vendorUserID uuid.UUID, response string) error {
	// Verify vendor owns the vendor account being reviewed
	var vendorID uuid.UUID
	err := s.db.QueryRow(ctx,
		"SELECT r.vendor_id FROM reviews r WHERE r.id = $1",
		reviewID,
	).Scan(&vendorID)

	if err == pgx.ErrNoRows {
		return ErrReviewNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to get review: %w", err)
	}

	// Check if vendor user owns this vendor
	var ownerUserID uuid.UUID
	err = s.db.QueryRow(ctx, "SELECT user_id FROM vendors WHERE id = $1", vendorID).Scan(&ownerUserID)
	if err != nil {
		return fmt.Errorf("failed to verify vendor ownership: %w", err)
	}

	if ownerUserID != vendorUserID {
		return ErrUnauthorized
	}

	// Add response
	now := time.Now()
	query := `
		UPDATE reviews
		SET vendor_response = $1, vendor_responded_at = $2, updated_at = $3
		WHERE id = $4
	`

	_, err = s.db.Exec(ctx, query, response, now, now, reviewID)
	if err != nil {
		return fmt.Errorf("failed to add vendor response: %w", err)
	}

	return nil
}

// VoteHelpful records a helpful vote on a review
func (s *Service) VoteHelpful(ctx context.Context, reviewID uuid.UUID, userID uuid.UUID, isHelpful bool) error {
	query := `
		INSERT INTO review_votes (id, review_id, user_id, is_helpful, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (review_id, user_id)
		DO UPDATE SET is_helpful = EXCLUDED.is_helpful
	`

	_, err := s.db.Exec(ctx, query, uuid.New(), reviewID, userID, isHelpful, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record vote: %w", err)
	}

	return nil
}

// Helper methods

func (s *Service) validateCreateRequest(req *CreateReviewRequest) error {
	if req.VendorID == uuid.Nil {
		return errors.New("vendor_id is required")
	}
	if req.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}
	if req.Rating < 1 || req.Rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}
	if req.Comment == "" {
		return errors.New("comment is required")
	}
	return nil
}

func joinUpdates(updates []string) string {
	result := ""
	for i, update := range updates {
		if i > 0 {
			result += ", "
		}
		result += update
	}
	return result
}
