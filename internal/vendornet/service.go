// Package vendornet provides B2B vendor partnership functionality
package vendornet

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	ErrPartnershipNotFound = errors.New("partnership not found")
	ErrReferralNotFound    = errors.New("referral not found")
	ErrInvalidTransition   = errors.New("invalid status transition")
	ErrUnauthorized        = errors.New("unauthorized action")
)

// Service handles VendorNet business logic
type Service struct {
	db                *pgxpool.Pool
	cache             *redis.Client
	matchingEngine    *MatchingEngine
	referralEngine    *ReferralEngine
	analyticsEngine   *AnalyticsEngine
}

// NewService creates a new VendorNet service
func NewService(db *pgxpool.Pool, cache *redis.Client) *Service {
	return &Service{
		db:              db,
		cache:           cache,
		matchingEngine:  NewMatchingEngine(db, cache),
		referralEngine:  NewReferralEngine(db, cache),
		analyticsEngine: NewAnalyticsEngine(db, cache),
	}
}

// MatchingEngine provides partner matching
type MatchingEngine struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewMatchingEngine creates a new matching engine
func NewMatchingEngine(db *pgxpool.Pool, cache *redis.Client) *MatchingEngine {
	return &MatchingEngine{
		db:    db,
		cache: cache,
	}
}

// PartnerMatch represents a potential partnership
type PartnerMatch struct {
	VendorID           uuid.UUID      `json:"vendor_id"`
	VendorName         string         `json:"vendor_name"`
	Category           string         `json:"category"`
	MatchScore         float64        `json:"match_score"`
	MatchReasons       []MatchReason  `json:"match_reasons"`
	TrustScore         float64        `json:"trust_score"`
	Rating             float64        `json:"rating"`
	ReviewCount        int            `json:"review_count"`
	MutualConnections  int            `json:"mutual_connections"`
	Recommendation     string         `json:"recommendation"`
}

// MatchReason explains why a vendor is a good match
type MatchReason struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

// FindPartnerMatches finds potential partners for a vendor
func (e *MatchingEngine) FindPartnerMatches(ctx context.Context, vendorID uuid.UUID, limit int) ([]PartnerMatch, error) {
	// For MVP, return vendors in complementary categories
	query := `
		SELECT
			v.id,
			v.business_name,
			sc.name as category_name,
			v.rating_average,
			v.rating_count,
			v.is_verified
		FROM vendors v
		JOIN services s ON s.vendor_id = v.id
		JOIN service_categories sc ON sc.id = s.category_id
		WHERE v.id != $1
		  AND v.is_active = TRUE
		  AND NOT EXISTS (
			  SELECT 1 FROM partnerships p
			  WHERE ((p.vendor_a_id = $1 AND p.vendor_b_id = v.id)
			     OR (p.vendor_b_id = $1 AND p.vendor_a_id = v.id))
			  AND p.status = 'active'
		  )
		GROUP BY v.id, v.business_name, sc.name, v.rating_average, v.rating_count, v.is_verified
		ORDER BY v.rating_average DESC, v.rating_count DESC
		LIMIT $2
	`

	rows, err := e.db.Query(ctx, query, vendorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []PartnerMatch
	for rows.Next() {
		var m PartnerMatch
		var isVerified bool

		err := rows.Scan(
			&m.VendorID,
			&m.VendorName,
			&m.Category,
			&m.Rating,
			&m.ReviewCount,
			&isVerified,
		)
		if err != nil {
			continue
		}

		// Calculate match score
		m.MatchScore = (m.Rating / 5.0) * 0.6
		if isVerified {
			m.MatchScore += 0.2
		}
		m.TrustScore = m.MatchScore * 100

		// Add match reasons
		m.MatchReasons = []MatchReason{
			{
				Type:        "complementary_category",
				Description: "Services complement your offerings",
				Score:       0.8,
			},
		}

		if m.Rating >= 4.5 {
			m.MatchReasons = append(m.MatchReasons, MatchReason{
				Type:        "high_rating",
				Description: "Highly rated by customers",
				Score:       m.Rating / 5.0,
			})
		}

		m.Recommendation = generateRecommendation(m.MatchScore)
		matches = append(matches, m)
	}

	return matches, nil
}

func generateRecommendation(score float64) string {
	if score > 0.8 {
		return "Highly recommended partner - strong fit"
	} else if score > 0.6 {
		return "Good potential partner - worth exploring"
	}
	return "Potential partner - consider for occasional collaboration"
}

// ReferralEngine handles referral management
type ReferralEngine struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewReferralEngine creates a new referral engine
func NewReferralEngine(db *pgxpool.Pool, cache *redis.Client) *ReferralEngine {
	return &ReferralEngine{
		db:    db,
		cache: cache,
	}
}

// CreateReferralRequest represents a referral creation request
type CreateReferralRequest struct {
	SourceVendorID  uuid.UUID  `json:"source_vendor_id"`
	DestVendorID    uuid.UUID  `json:"dest_vendor_id"`
	ClientName      string     `json:"client_name"`
	ClientEmail     string     `json:"client_email"`
	ClientPhone     string     `json:"client_phone"`
	EventType       string     `json:"event_type"`
	EventDate       *time.Time `json:"event_date,omitempty"`
	ServiceCategory uuid.UUID  `json:"service_category_id"`
	EstimatedValue  float64    `json:"estimated_value"`
	Notes           string     `json:"notes"`
}

// Referral represents a client referral
type Referral struct {
	ID              uuid.UUID     `json:"id"`
	SourceVendorID  uuid.UUID     `json:"source_vendor_id"`
	DestVendorID    uuid.UUID     `json:"dest_vendor_id"`
	ClientName      string        `json:"client_name"`
	ClientEmail     string        `json:"client_email"`
	ClientPhone     string        `json:"client_phone"`
	EventType       string        `json:"event_type"`
	EventDate       *time.Time    `json:"event_date,omitempty"`
	ServiceCategory uuid.UUID     `json:"service_category_id"`
	EstimatedValue  float64       `json:"estimated_value"`
	Notes           string        `json:"notes"`
	Status          string        `json:"status"`
	TrackingCode    string        `json:"tracking_code"`
	FeeType         string        `json:"fee_type"`
	FeeValue        float64       `json:"fee_value"`
	CalculatedFee   float64       `json:"calculated_fee"`
	CreatedAt       time.Time     `json:"created_at"`
	ExpiresAt       time.Time     `json:"expires_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// CreateReferral creates a new referral
func (e *ReferralEngine) CreateReferral(ctx context.Context, req CreateReferralRequest) (*Referral, error) {
	referral := &Referral{
		ID:              uuid.New(),
		SourceVendorID:  req.SourceVendorID,
		DestVendorID:    req.DestVendorID,
		ClientName:      req.ClientName,
		ClientEmail:     req.ClientEmail,
		ClientPhone:     req.ClientPhone,
		EventType:       req.EventType,
		EventDate:       req.EventDate,
		ServiceCategory: req.ServiceCategory,
		EstimatedValue:  req.EstimatedValue,
		Notes:           req.Notes,
		Status:          "pending",
		TrackingCode:    generateTrackingCode(),
		FeeType:         "percentage",
		FeeValue:        10.0,
		CalculatedFee:   req.EstimatedValue * 0.10,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().AddDate(0, 0, 30),
		UpdatedAt:       time.Now(),
	}

	query := `
		INSERT INTO referrals (
			id, source_vendor_id, dest_vendor_id, client_name, client_email, client_phone,
			event_type, event_date, service_category_id, estimated_value, notes,
			status, tracking_code, fee_type, fee_value, calculated_fee,
			created_at, expires_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	_, err := e.db.Exec(ctx, query,
		referral.ID, referral.SourceVendorID, referral.DestVendorID,
		referral.ClientName, referral.ClientEmail, referral.ClientPhone,
		referral.EventType, referral.EventDate, referral.ServiceCategory,
		referral.EstimatedValue, referral.Notes, referral.Status,
		referral.TrackingCode, referral.FeeType, referral.FeeValue,
		referral.CalculatedFee, referral.CreatedAt, referral.ExpiresAt,
		referral.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return referral, nil
}

// GetReferral retrieves a referral by ID
func (e *ReferralEngine) GetReferral(ctx context.Context, referralID uuid.UUID) (*Referral, error) {
	query := `
		SELECT
			id, source_vendor_id, dest_vendor_id, client_name, client_email, client_phone,
			event_type, event_date, service_category_id, estimated_value, notes,
			status, tracking_code, fee_type, fee_value, calculated_fee,
			created_at, expires_at, updated_at
		FROM referrals
		WHERE id = $1
	`

	var r Referral
	err := e.db.QueryRow(ctx, query, referralID).Scan(
		&r.ID, &r.SourceVendorID, &r.DestVendorID,
		&r.ClientName, &r.ClientEmail, &r.ClientPhone,
		&r.EventType, &r.EventDate, &r.ServiceCategory,
		&r.EstimatedValue, &r.Notes, &r.Status,
		&r.TrackingCode, &r.FeeType, &r.FeeValue,
		&r.CalculatedFee, &r.CreatedAt, &r.ExpiresAt,
		&r.UpdatedAt,
	)

	if err != nil {
		return nil, ErrReferralNotFound
	}

	return &r, nil
}

// UpdateReferralStatus updates referral status
func (e *ReferralEngine) UpdateReferralStatus(ctx context.Context, referralID uuid.UUID, status string) error {
	query := `
		UPDATE referrals
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := e.db.Exec(ctx, query, referralID, status, time.Now())
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrReferralNotFound
	}

	return nil
}

func generateTrackingCode() string {
	return "REF-" + uuid.New().String()[:8]
}

// AnalyticsEngine provides network analytics
type AnalyticsEngine struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewAnalyticsEngine creates a new analytics engine
func NewAnalyticsEngine(db *pgxpool.Pool, cache *redis.Client) *AnalyticsEngine {
	return &AnalyticsEngine{
		db:    db,
		cache: cache,
	}
}

// NetworkStats represents network statistics
type NetworkStats struct {
	TotalReferrals      int     `json:"total_referrals"`
	ReferralsThisMonth  int     `json:"referrals_this_month"`
	ConversionRate      float64 `json:"conversion_rate"`
	TotalRevenue        float64 `json:"total_revenue"`
	AvgReferralValue    float64 `json:"avg_referral_value"`
	ActivePartnerships  int     `json:"active_partnerships"`
}

// GetNetworkStats retrieves network statistics for a vendor
func (e *AnalyticsEngine) GetNetworkStats(ctx context.Context, vendorID uuid.UUID) (*NetworkStats, error) {
	stats := &NetworkStats{}

	// Total referrals sent
	e.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM referrals
		WHERE source_vendor_id = $1
	`, vendorID).Scan(&stats.TotalReferrals)

	// Referrals this month
	e.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM referrals
		WHERE source_vendor_id = $1
		  AND created_at >= DATE_TRUNC('month', CURRENT_DATE)
	`, vendorID).Scan(&stats.ReferralsThisMonth)

	// Conversion rate
	var converted, total int
	e.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'converted'),
			COUNT(*)
		FROM referrals
		WHERE dest_vendor_id = $1
	`, vendorID).Scan(&converted, &total)

	if total > 0 {
		stats.ConversionRate = float64(converted) / float64(total) * 100
	}

	// Total revenue (from received referrals)
	e.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(estimated_value), 0)
		FROM referrals
		WHERE dest_vendor_id = $1 AND status = 'converted'
	`, vendorID).Scan(&stats.TotalRevenue)

	// Average referral value
	e.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(estimated_value), 0)
		FROM referrals
		WHERE dest_vendor_id = $1 AND status = 'converted'
	`, vendorID).Scan(&stats.AvgReferralValue)

	// Active partnerships
	e.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM partnerships
		WHERE (vendor_a_id = $1 OR vendor_b_id = $1)
		  AND status = 'active'
	`, vendorID).Scan(&stats.ActivePartnerships)

	return stats, nil
}

// Partnership represents a vendor partnership
type Partnership struct {
	ID          uuid.UUID `json:"id"`
	VendorAID   uuid.UUID `json:"vendor_a_id"`
	VendorBID   uuid.UUID `json:"vendor_b_id"`
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Terms       string    `json:"terms"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreatePartnershipRequest represents partnership creation
type CreatePartnershipRequest struct {
	VendorAID   uuid.UUID `json:"vendor_a_id"`
	VendorBID   uuid.UUID `json:"vendor_b_id"`
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Terms       string    `json:"terms"`
}

// CreatePartnership creates a new partnership
func (s *Service) CreatePartnership(ctx context.Context, req CreatePartnershipRequest) (*Partnership, error) {
	partnership := &Partnership{
		ID:          uuid.New(),
		VendorAID:   req.VendorAID,
		VendorBID:   req.VendorBID,
		Type:        req.Type,
		Name:        req.Name,
		Description: req.Description,
		Status:      "proposed",
		Terms:       req.Terms,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO partnerships (
			id, vendor_a_id, vendor_b_id, type, name, description, status, terms, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.db.Exec(ctx, query,
		partnership.ID, partnership.VendorAID, partnership.VendorBID,
		partnership.Type, partnership.Name, partnership.Description,
		partnership.Status, partnership.Terms, partnership.CreatedAt,
		partnership.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return partnership, nil
}

// GetPartnership retrieves a partnership by ID
func (s *Service) GetPartnership(ctx context.Context, partnershipID uuid.UUID) (*Partnership, error) {
	query := `
		SELECT id, vendor_a_id, vendor_b_id, type, name, description, status, terms, created_at, updated_at
		FROM partnerships
		WHERE id = $1
	`

	var p Partnership
	err := s.db.QueryRow(ctx, query, partnershipID).Scan(
		&p.ID, &p.VendorAID, &p.VendorBID, &p.Type,
		&p.Name, &p.Description, &p.Status, &p.Terms,
		&p.CreatedAt, &p.UpdatedAt,
	)

	if err != nil {
		return nil, ErrPartnershipNotFound
	}

	return &p, nil
}

// GetPartnerMatches finds potential partners
func (s *Service) GetPartnerMatches(ctx context.Context, vendorID uuid.UUID, limit int) ([]PartnerMatch, error) {
	return s.matchingEngine.FindPartnerMatches(ctx, vendorID, limit)
}

// CreateReferral creates a new referral
func (s *Service) CreateReferral(ctx context.Context, req CreateReferralRequest) (*Referral, error) {
	return s.referralEngine.CreateReferral(ctx, req)
}

// GetReferral retrieves a referral
func (s *Service) GetReferral(ctx context.Context, referralID uuid.UUID) (*Referral, error) {
	return s.referralEngine.GetReferral(ctx, referralID)
}

// UpdateReferralStatus updates referral status
func (s *Service) UpdateReferralStatus(ctx context.Context, referralID uuid.UUID, status string) error {
	return s.referralEngine.UpdateReferralStatus(ctx, referralID, status)
}

// GetNetworkAnalytics retrieves network analytics
func (s *Service) GetNetworkAnalytics(ctx context.Context, vendorID uuid.UUID) (*NetworkStats, error) {
	return s.analyticsEngine.GetNetworkStats(ctx, vendorID)
}
