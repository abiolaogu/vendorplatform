// Package vendornet provides B2B partnership network business logic
package vendornet

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	ErrPartnershipNotFound   = errors.New("partnership not found")
	ErrPartnershipExists     = errors.New("partnership already exists")
	ErrInvalidPartnershipData = errors.New("invalid partnership data")
	ErrSelfPartnership       = errors.New("cannot create partnership with self")
	ErrReferralNotFound      = errors.New("referral not found")
	ErrInvalidReferralData   = errors.New("invalid referral data")
	ErrUnauthorized          = errors.New("unauthorized")
)

// Service handles VendorNet partnership and referral operations
type Service struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewService creates a new VendorNet service
func NewService(db *pgxpool.Pool, cache *redis.Client) *Service {
	return &Service{
		db:    db,
		cache: cache,
	}
}

// Partnership represents a vendor-to-vendor partnership
type Partnership struct {
	ID                     uuid.UUID  `json:"id"`
	VendorAID              uuid.UUID  `json:"vendor_a_id"`
	VendorBID              uuid.UUID  `json:"vendor_b_id"`
	PartnershipType        string     `json:"partnership_type"` // referral, preferred, exclusive, joint_venture, white_label
	ReferralFeeType        *string    `json:"referral_fee_type,omitempty"` // percentage, fixed, none
	ReferralFeeValue       *float64   `json:"referral_fee_value,omitempty"`
	IsBidirectional        bool       `json:"is_bidirectional"`
	TotalReferrals         int        `json:"total_referrals"`
	SuccessfulReferrals    int        `json:"successful_referrals"`
	TotalRevenueGenerated  float64    `json:"total_revenue_generated"`
	Status                 string     `json:"status"` // pending, active, paused, terminated
	InitiatedBy            uuid.UUID  `json:"initiated_by"`
	TermsAndConditions     *string    `json:"terms_and_conditions,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	ActivatedAt            *time.Time `json:"activated_at,omitempty"`
	ExpiresAt              *time.Time `json:"expires_at,omitempty"`
}

// Referral represents a client referral from one vendor to another
type Referral struct {
	ID              uuid.UUID  `json:"id"`
	SourceVendorID  uuid.UUID  `json:"source_vendor_id"`
	DestVendorID    uuid.UUID  `json:"dest_vendor_id"`
	ClientName      *string    `json:"client_name,omitempty"`
	ClientEmail     *string    `json:"client_email,omitempty"`
	ClientPhone     *string    `json:"client_phone,omitempty"`
	EventType       *string    `json:"event_type,omitempty"`
	EventDate       *time.Time `json:"event_date,omitempty"`
	EstimatedValue  *int64     `json:"estimated_value,omitempty"`
	Status          string     `json:"status"` // pending, accepted, contacted, quoted, converted, lost
	StatusHistory   []byte     `json:"status_history,omitempty"`
	FeeType         *string    `json:"fee_type,omitempty"` // percentage, fixed, none
	FeeValue        *float64   `json:"fee_value,omitempty"`
	FeePaid         bool       `json:"fee_paid"`
	TrackingCode    string     `json:"tracking_code"`
	Notes           *string    `json:"notes,omitempty"`
	Feedback        *string    `json:"feedback,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ConvertedAt     *time.Time `json:"converted_at,omitempty"`
}

// CreatePartnershipRequest represents a request to create a partnership
type CreatePartnershipRequest struct {
	VendorAID          uuid.UUID  `json:"vendor_a_id"`
	VendorBID          uuid.UUID  `json:"vendor_b_id"`
	PartnershipType    string     `json:"partnership_type"`
	ReferralFeeType    *string    `json:"referral_fee_type,omitempty"`
	ReferralFeeValue   *float64   `json:"referral_fee_value,omitempty"`
	IsBidirectional    bool       `json:"is_bidirectional"`
	InitiatedBy        uuid.UUID  `json:"initiated_by"`
	TermsAndConditions *string    `json:"terms_and_conditions,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
}

// CreateReferralRequest represents a request to create a referral
type CreateReferralRequest struct {
	SourceVendorID uuid.UUID  `json:"source_vendor_id"`
	DestVendorID   uuid.UUID  `json:"dest_vendor_id"`
	ClientName     *string    `json:"client_name,omitempty"`
	ClientEmail    *string    `json:"client_email,omitempty"`
	ClientPhone    *string    `json:"client_phone,omitempty"`
	EventType      *string    `json:"event_type,omitempty"`
	EventDate      *time.Time `json:"event_date,omitempty"`
	EstimatedValue *int64     `json:"estimated_value,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
}

// UpdateReferralStatusRequest represents a request to update referral status
type UpdateReferralStatusRequest struct {
	Status   string  `json:"status"`
	Feedback *string `json:"feedback,omitempty"`
}

// PartnerMatch represents a potential partner recommendation
type PartnerMatch struct {
	VendorID         uuid.UUID `json:"vendor_id"`
	BusinessName     string    `json:"business_name"`
	PrimaryCategory  string    `json:"primary_category"`
	Rating           float64   `json:"rating"`
	CompletedBookings int      `json:"completed_bookings"`
	MatchScore       float64   `json:"match_score"`
	MatchReason      string    `json:"match_reason"`
}

// NetworkAnalytics represents vendor network analytics
type NetworkAnalytics struct {
	VendorID              uuid.UUID `json:"vendor_id"`
	TotalPartnerships     int       `json:"total_partnerships"`
	ActivePartnerships    int       `json:"active_partnerships"`
	TotalReferralsSent    int       `json:"total_referrals_sent"`
	TotalReferralsReceived int      `json:"total_referrals_received"`
	ConversionRate        float64   `json:"conversion_rate"`
	TotalRevenueShared    float64   `json:"total_revenue_shared"`
	TotalRevenueEarned    float64   `json:"total_revenue_earned"`
}

// =============================================================================
// PARTNERSHIP OPERATIONS
// =============================================================================

// CreatePartnership creates a new vendor partnership
func (s *Service) CreatePartnership(ctx context.Context, req *CreatePartnershipRequest) (*Partnership, error) {
	// Validate
	if req.VendorAID == uuid.Nil || req.VendorBID == uuid.Nil {
		return nil, ErrInvalidPartnershipData
	}
	if req.VendorAID == req.VendorBID {
		return nil, ErrSelfPartnership
	}

	// Validate partnership type
	validTypes := map[string]bool{
		"referral":      true,
		"preferred":     true,
		"exclusive":     true,
		"joint_venture": true,
		"white_label":   true,
	}
	if !validTypes[req.PartnershipType] {
		return nil, fmt.Errorf("%w: invalid partnership type", ErrInvalidPartnershipData)
	}

	// Check if partnership already exists (in either direction)
	var existingID uuid.UUID
	err := s.db.QueryRow(ctx, `
		SELECT id FROM vendor_partnerships
		WHERE (vendor_a_id = $1 AND vendor_b_id = $2)
		   OR (vendor_a_id = $2 AND vendor_b_id = $1)
		LIMIT 1
	`, req.VendorAID, req.VendorBID).Scan(&existingID)

	if err == nil {
		return nil, ErrPartnershipExists
	}

	// Create partnership
	now := time.Now()
	partnership := &Partnership{
		ID:                 uuid.New(),
		VendorAID:          req.VendorAID,
		VendorBID:          req.VendorBID,
		PartnershipType:    req.PartnershipType,
		ReferralFeeType:    req.ReferralFeeType,
		ReferralFeeValue:   req.ReferralFeeValue,
		IsBidirectional:    req.IsBidirectional,
		Status:             "pending",
		InitiatedBy:        req.InitiatedBy,
		TermsAndConditions: req.TermsAndConditions,
		ExpiresAt:          req.ExpiresAt,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	query := `
		INSERT INTO vendor_partnerships (
			id, vendor_a_id, vendor_b_id, partnership_type,
			referral_fee_type, referral_fee_value, is_bidirectional,
			status, initiated_by, terms_and_conditions, expires_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = s.db.Exec(ctx, query,
		partnership.ID, partnership.VendorAID, partnership.VendorBID,
		partnership.PartnershipType, partnership.ReferralFeeType,
		partnership.ReferralFeeValue, partnership.IsBidirectional,
		partnership.Status, partnership.InitiatedBy,
		partnership.TermsAndConditions, partnership.ExpiresAt,
		partnership.CreatedAt, partnership.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create partnership: %w", err)
	}

	return partnership, nil
}

// GetPartnership retrieves a partnership by ID
func (s *Service) GetPartnership(ctx context.Context, partnershipID uuid.UUID) (*Partnership, error) {
	query := `
		SELECT id, vendor_a_id, vendor_b_id, partnership_type,
		       referral_fee_type, referral_fee_value, is_bidirectional,
		       total_referrals, successful_referrals, total_revenue_generated,
		       status, initiated_by, terms_and_conditions,
		       created_at, updated_at, activated_at, expires_at
		FROM vendor_partnerships
		WHERE id = $1
	`

	var p Partnership
	err := s.db.QueryRow(ctx, query, partnershipID).Scan(
		&p.ID, &p.VendorAID, &p.VendorBID, &p.PartnershipType,
		&p.ReferralFeeType, &p.ReferralFeeValue, &p.IsBidirectional,
		&p.TotalReferrals, &p.SuccessfulReferrals, &p.TotalRevenueGenerated,
		&p.Status, &p.InitiatedBy, &p.TermsAndConditions,
		&p.CreatedAt, &p.UpdatedAt, &p.ActivatedAt, &p.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPartnershipNotFound
		}
		return nil, fmt.Errorf("failed to get partnership: %w", err)
	}

	return &p, nil
}

// GetPartnerMatches returns potential partner recommendations for a vendor
func (s *Service) GetPartnerMatches(ctx context.Context, vendorID uuid.UUID, limit int) ([]*PartnerMatch, error) {
	// Get vendor's primary category
	var primaryCategoryID uuid.UUID
	err := s.db.QueryRow(ctx, `
		SELECT primary_category_id FROM vendors WHERE id = $1
	`, vendorID).Scan(&primaryCategoryID)

	if err != nil {
		return nil, fmt.Errorf("failed to get vendor category: %w", err)
	}

	// Find complementary vendors using adjacency data
	// This is a simplified version - in production, use the recommendation engine
	query := `
		WITH vendor_categories AS (
			SELECT category_ids FROM vendors WHERE id = $1
		),
		existing_partnerships AS (
			SELECT vendor_b_id as partner_id FROM vendor_partnerships
			WHERE vendor_a_id = $1 AND status = 'active'
			UNION
			SELECT vendor_a_id as partner_id FROM vendor_partnerships
			WHERE vendor_b_id = $1 AND status = 'active'
		)
		SELECT
			v.id,
			v.business_name,
			c.name as primary_category,
			v.rating_average,
			v.completed_bookings,
			0.8 as match_score,
			'Complementary services' as match_reason
		FROM vendors v
		JOIN categories c ON c.id = v.primary_category_id
		WHERE v.id != $1
		  AND v.status = 'active'
		  AND v.is_verified = true
		  AND v.id NOT IN (SELECT partner_id FROM existing_partnerships)
		ORDER BY v.rating_average DESC, v.completed_bookings DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, vendorID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get partner matches: %w", err)
	}
	defer rows.Close()

	var matches []*PartnerMatch
	for rows.Next() {
		var m PartnerMatch
		err := rows.Scan(
			&m.VendorID, &m.BusinessName, &m.PrimaryCategory,
			&m.Rating, &m.CompletedBookings, &m.MatchScore, &m.MatchReason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan partner match: %w", err)
		}
		matches = append(matches, &m)
	}

	return matches, nil
}

// =============================================================================
// REFERRAL OPERATIONS
// =============================================================================

// CreateReferral creates a new referral
func (s *Service) CreateReferral(ctx context.Context, req *CreateReferralRequest) (*Referral, error) {
	// Validate
	if req.SourceVendorID == uuid.Nil || req.DestVendorID == uuid.Nil {
		return nil, ErrInvalidReferralData
	}
	if req.SourceVendorID == req.DestVendorID {
		return nil, fmt.Errorf("%w: cannot refer to self", ErrInvalidReferralData)
	}

	// Generate tracking code
	trackingCode := fmt.Sprintf("REF-%s", uuid.New().String()[:8])

	now := time.Now()
	referral := &Referral{
		ID:             uuid.New(),
		SourceVendorID: req.SourceVendorID,
		DestVendorID:   req.DestVendorID,
		ClientName:     req.ClientName,
		ClientEmail:    req.ClientEmail,
		ClientPhone:    req.ClientPhone,
		EventType:      req.EventType,
		EventDate:      req.EventDate,
		EstimatedValue: req.EstimatedValue,
		Status:         "pending",
		StatusHistory:  []byte("[]"),
		TrackingCode:   trackingCode,
		Notes:          req.Notes,
		FeePaid:        false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Get partnership details if exists
	var feeType *string
	var feeValue *float64
	err := s.db.QueryRow(ctx, `
		SELECT referral_fee_type, referral_fee_value
		FROM vendor_partnerships
		WHERE ((vendor_a_id = $1 AND vendor_b_id = $2)
		    OR (vendor_a_id = $2 AND vendor_b_id = $1 AND is_bidirectional = true))
		  AND status = 'active'
		LIMIT 1
	`, req.SourceVendorID, req.DestVendorID).Scan(&feeType, &feeValue)

	if err == nil {
		referral.FeeType = feeType
		referral.FeeValue = feeValue
	}

	query := `
		INSERT INTO referrals (
			id, source_vendor_id, dest_vendor_id, client_name,
			client_email, client_phone, event_type, event_date,
			estimated_value, status, status_history, fee_type,
			fee_value, tracking_code, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err = s.db.Exec(ctx, query,
		referral.ID, referral.SourceVendorID, referral.DestVendorID,
		referral.ClientName, referral.ClientEmail, referral.ClientPhone,
		referral.EventType, referral.EventDate, referral.EstimatedValue,
		referral.Status, referral.StatusHistory, referral.FeeType,
		referral.FeeValue, referral.TrackingCode, referral.Notes,
		referral.CreatedAt, referral.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create referral: %w", err)
	}

	return referral, nil
}

// UpdateReferralStatus updates the status of a referral
func (s *Service) UpdateReferralStatus(ctx context.Context, referralID uuid.UUID, req *UpdateReferralStatusRequest) (*Referral, error) {
	// Validate status
	validStatuses := map[string]bool{
		"pending":   true,
		"accepted":  true,
		"contacted": true,
		"quoted":    true,
		"converted": true,
		"lost":      true,
	}
	if !validStatuses[req.Status] {
		return nil, fmt.Errorf("%w: invalid status", ErrInvalidReferralData)
	}

	now := time.Now()
	var convertedAt *time.Time
	if req.Status == "converted" {
		convertedAt = &now
	}

	query := `
		UPDATE referrals
		SET status = $2,
		    feedback = $3,
		    updated_at = $4,
		    converted_at = $5
		WHERE id = $1
	`

	result, err := s.db.Exec(ctx, query, referralID, req.Status, req.Feedback, now, convertedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update referral status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil, ErrReferralNotFound
	}

	// If converted, update partnership metrics
	if req.Status == "converted" {
		_, err = s.db.Exec(ctx, `
			UPDATE vendor_partnerships
			SET successful_referrals = successful_referrals + 1,
			    updated_at = $3
			WHERE ((vendor_a_id = (SELECT source_vendor_id FROM referrals WHERE id = $1)
			    AND vendor_b_id = (SELECT dest_vendor_id FROM referrals WHERE id = $1))
			   OR (vendor_a_id = (SELECT dest_vendor_id FROM referrals WHERE id = $1)
			    AND vendor_b_id = (SELECT source_vendor_id FROM referrals WHERE id = $1)
			    AND is_bidirectional = true))
			  AND status = 'active'
		`, referralID, referralID, now)

		if err != nil {
			// Log error but don't fail the request
			// In production, use proper logging
			fmt.Printf("Warning: failed to update partnership metrics: %v\n", err)
		}
	}

	// Retrieve updated referral
	return s.GetReferral(ctx, referralID)
}

// GetReferral retrieves a referral by ID
func (s *Service) GetReferral(ctx context.Context, referralID uuid.UUID) (*Referral, error) {
	query := `
		SELECT id, source_vendor_id, dest_vendor_id, client_name,
		       client_email, client_phone, event_type, event_date,
		       estimated_value, status, status_history, fee_type,
		       fee_value, fee_paid, tracking_code, notes, feedback,
		       created_at, updated_at, converted_at
		FROM referrals
		WHERE id = $1
	`

	var r Referral
	err := s.db.QueryRow(ctx, query, referralID).Scan(
		&r.ID, &r.SourceVendorID, &r.DestVendorID, &r.ClientName,
		&r.ClientEmail, &r.ClientPhone, &r.EventType, &r.EventDate,
		&r.EstimatedValue, &r.Status, &r.StatusHistory, &r.FeeType,
		&r.FeeValue, &r.FeePaid, &r.TrackingCode, &r.Notes, &r.Feedback,
		&r.CreatedAt, &r.UpdatedAt, &r.ConvertedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrReferralNotFound
		}
		return nil, fmt.Errorf("failed to get referral: %w", err)
	}

	return &r, nil
}

// =============================================================================
// ANALYTICS OPERATIONS
// =============================================================================

// GetNetworkAnalytics returns network analytics for a vendor
func (s *Service) GetNetworkAnalytics(ctx context.Context, vendorID uuid.UUID) (*NetworkAnalytics, error) {
	analytics := &NetworkAnalytics{
		VendorID: vendorID,
	}

	// Get partnership stats
	err := s.db.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_partnerships,
			COUNT(*) FILTER (WHERE status = 'active') as active_partnerships
		FROM vendor_partnerships
		WHERE vendor_a_id = $1 OR vendor_b_id = $1
	`, vendorID).Scan(&analytics.TotalPartnerships, &analytics.ActivePartnerships)

	if err != nil {
		return nil, fmt.Errorf("failed to get partnership stats: %w", err)
	}

	// Get referral stats
	err = s.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE source_vendor_id = $1) as sent,
			COUNT(*) FILTER (WHERE dest_vendor_id = $1) as received,
			COUNT(*) FILTER (WHERE source_vendor_id = $1 AND status = 'converted') as converted
		FROM referrals
		WHERE source_vendor_id = $1 OR dest_vendor_id = $1
	`, vendorID).Scan(
		&analytics.TotalReferralsSent,
		&analytics.TotalReferralsReceived,
		new(int), // temp var for converted count
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get referral stats: %w", err)
	}

	// Calculate conversion rate
	if analytics.TotalReferralsSent > 0 {
		var convertedCount int
		err = s.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM referrals
			WHERE source_vendor_id = $1 AND status = 'converted'
		`, vendorID).Scan(&convertedCount)

		if err == nil {
			analytics.ConversionRate = float64(convertedCount) / float64(analytics.TotalReferralsSent) * 100
		}
	}

	// Get revenue stats (simplified - in production, link to payment records)
	err = s.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(vp.total_revenue_generated), 0) as shared
		FROM vendor_partnerships vp
		WHERE vp.vendor_a_id = $1 AND vp.status = 'active'
	`, vendorID).Scan(&analytics.TotalRevenueShared)

	if err != nil {
		return nil, fmt.Errorf("failed to get revenue stats: %w", err)
	}

	return analytics, nil
}
