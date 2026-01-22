// =============================================================================
// VENDORNET - B2B VENDOR PARTNERSHIP NETWORK
// Comprehensive Technical & Business Specification
// Version: 1.0.0
// =============================================================================

package vendornet

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

/*
================================================================================
SECTION 1: PRODUCT VISION & POSITIONING
================================================================================

VENDORNET: The Professional Network for Service Vendors

TAGLINE: "Grow together. Earn together."

VISION:
VendorNet transforms isolated service vendors into a connected ecosystem where
complementary businesses discover each other, form partnerships, share referrals,
and collaborate on large projects. It's LinkedIn meets Stripe for the vendor economy.

CORE VALUE PROPOSITION:
"Turn your network into revenue. Every referral, every collaboration, every
partnership—tracked, managed, and monetized."

TARGET USERS:
1. Individual Vendors: Photographers, caterers, decorators seeking growth
2. Established Businesses: Event companies wanting to expand offerings
3. Vendor Aggregators: Event planners managing vendor networks
4. Enterprise Clients: Hotels, venues, corporate clients needing vendor packages

KEY DIFFERENTIATORS:
1. Automatic Referral Tracking: No manual tracking of who sent whom
2. Smart Matching: AI-powered partnership suggestions
3. Revenue Sharing Infrastructure: Built-in payment splitting
4. Collaborative Bidding: Multiple vendors can bid together on large projects
5. Reputation Network: Cross-vendor reviews and endorsements

NETWORK EFFECTS:
- More vendors → Better matches → More partnerships → More value → More vendors
- Each new vendor adds potential partners for existing vendors
- Network becomes more valuable as it grows

================================================================================
SECTION 2: CORE DOMAIN TYPES
================================================================================
*/

// =============================================================================
// 2.1 VENDOR NETWORK ENTITIES
// =============================================================================

// VendorProfile represents a vendor in the network
type VendorProfile struct {
	ID                 uuid.UUID              `json:"id"`
	VendorID           uuid.UUID              `json:"vendor_id"` // Link to main vendor table
	
	// Network Identity
	NetworkHandle      string                 `json:"network_handle"` // @photographer_joe
	DisplayName        string                 `json:"display_name"`
	Tagline            string                 `json:"tagline"`
	Bio                string                 `json:"bio"`
	
	// Business Info
	PrimaryCategory    uuid.UUID              `json:"primary_category_id"`
	SecondaryCategories []uuid.UUID           `json:"secondary_category_ids"`
	ServiceAreas       []ServiceArea          `json:"service_areas"`
	
	// Network Stats
	ConnectionCount    int                    `json:"connection_count"`
	PartnershipCount   int                    `json:"partnership_count"`
	ReferralsSent      int                    `json:"referrals_sent"`
	ReferralsReceived  int                    `json:"referrals_received"`
	CollaborationCount int                    `json:"collaboration_count"`
	
	// Revenue Stats
	TotalReferralRevenue    float64           `json:"total_referral_revenue"`
	TotalCollabRevenue      float64           `json:"total_collab_revenue"`
	AvgReferralValue        float64           `json:"avg_referral_value"`
	
	// Trust Scores
	NetworkTrustScore       float64           `json:"network_trust_score"` // 0-100
	ResponseRate            float64           `json:"response_rate"`
	ReferralSuccessRate     float64           `json:"referral_success_rate"`
	
	// Preferences
	PartnershipPreferences  PartnershipPrefs  `json:"partnership_preferences"`
	ReferralPreferences     ReferralPrefs     `json:"referral_preferences"`
	
	// Verification
	IsVerified             bool               `json:"is_verified"`
	VerificationBadges     []string           `json:"verification_badges"`
	
	// Settings
	AcceptingNewPartners   bool               `json:"accepting_new_partners"`
	AcceptingReferrals     bool               `json:"accepting_referrals"`
	AutoAcceptReferrals    bool               `json:"auto_accept_referrals"`
	
	// Timestamps
	JoinedNetworkAt        time.Time          `json:"joined_network_at"`
	LastActiveAt           time.Time          `json:"last_active_at"`
}

type ServiceArea struct {
	City      string  `json:"city"`
	State     string  `json:"state"`
	Country   string  `json:"country"`
	RadiusKM  float64 `json:"radius_km"`
}

type PartnershipPrefs struct {
	PreferredCategories   []uuid.UUID `json:"preferred_categories"`
	MinPartnerRating      float64     `json:"min_partner_rating"`
	MinPartnerExperience  int         `json:"min_partner_experience_years"`
	RequireVerification   bool        `json:"require_verification"`
	MaxActivePartners     int         `json:"max_active_partners"`
}

type ReferralPrefs struct {
	DefaultFeeType        FeeType     `json:"default_fee_type"`
	DefaultFeeValue       float64     `json:"default_fee_value"`
	MinReferralValue      float64     `json:"min_referral_value"`
	RequireApproval       bool        `json:"require_approval"`
	AutoPayReferrals      bool        `json:"auto_pay_referrals"`
}

type FeeType string
const (
	FeePercentage FeeType = "percentage"
	FeeFixed      FeeType = "fixed"
	FeeNone       FeeType = "none"
)

// =============================================================================
// 2.2 CONNECTIONS & PARTNERSHIPS
// =============================================================================

// Connection represents a vendor-to-vendor connection (like LinkedIn)
type Connection struct {
	ID               uuid.UUID         `json:"id"`
	
	// Parties
	VendorAID        uuid.UUID         `json:"vendor_a_id"`
	VendorBID        uuid.UUID         `json:"vendor_b_id"`
	
	// Connection Type
	ConnectionType   ConnectionType    `json:"connection_type"`
	RelationshipNote string            `json:"relationship_note"`
	
	// Status
	Status           ConnectionStatus  `json:"status"`
	InitiatedBy      uuid.UUID         `json:"initiated_by"`
	
	// Mutual Categories (what they can refer to each other)
	MutualCategories []uuid.UUID       `json:"mutual_categories"`
	
	// Activity
	LastInteractionAt time.Time        `json:"last_interaction_at"`
	InteractionCount  int              `json:"interaction_count"`
	
	// Timestamps
	RequestedAt      time.Time         `json:"requested_at"`
	AcceptedAt       *time.Time        `json:"accepted_at,omitempty"`
}

type ConnectionType string
const (
	ConnectionPeer        ConnectionType = "peer"        // Same category
	ConnectionComplementary ConnectionType = "complementary" // Different but related
	ConnectionMentor      ConnectionType = "mentor"      // Mentorship
	ConnectionSubcontractor ConnectionType = "subcontractor" // Work relationship
)

type ConnectionStatus string
const (
	ConnectionPending   ConnectionStatus = "pending"
	ConnectionAccepted  ConnectionStatus = "accepted"
	ConnectionDeclined  ConnectionStatus = "declined"
	ConnectionBlocked   ConnectionStatus = "blocked"
)

// Partnership represents a formal business arrangement
type Partnership struct {
	ID                uuid.UUID            `json:"id"`
	
	// Partners
	VendorAID         uuid.UUID            `json:"vendor_a_id"`
	VendorBID         uuid.UUID            `json:"vendor_b_id"`
	
	// Partnership Details
	PartnershipType   PartnershipType      `json:"partnership_type"`
	Name              string               `json:"name"`
	Description       string               `json:"description"`
	
	// Terms
	Terms             PartnershipTerms     `json:"terms"`
	
	// Status
	Status            PartnershipStatus    `json:"status"`
	
	// Performance
	TotalReferrals    int                  `json:"total_referrals"`
	SuccessfulReferrals int                `json:"successful_referrals"`
	TotalRevenue      float64              `json:"total_revenue"`
	VendorARevenue    float64              `json:"vendor_a_revenue"`
	VendorBRevenue    float64              `json:"vendor_b_revenue"`
	
	// Agreement
	AgreementDocURL   string               `json:"agreement_doc_url,omitempty"`
	SignedByA         bool                 `json:"signed_by_a"`
	SignedByB         bool                 `json:"signed_by_b"`
	
	// Timestamps
	ProposedAt        time.Time            `json:"proposed_at"`
	ProposedBy        uuid.UUID            `json:"proposed_by"`
	ActivatedAt       *time.Time           `json:"activated_at,omitempty"`
	ExpiresAt         *time.Time           `json:"expires_at,omitempty"`
	TerminatedAt      *time.Time           `json:"terminated_at,omitempty"`
	TerminationReason string               `json:"termination_reason,omitempty"`
}

type PartnershipType string
const (
	PartnershipReferral      PartnershipType = "referral"       // Simple referral exchange
	PartnershipPreferred     PartnershipType = "preferred"      // Preferred partner status
	PartnershipExclusive     PartnershipType = "exclusive"      // Exclusive in category
	PartnershipJointVenture  PartnershipType = "joint_venture"  // Joint business offering
	PartnershipWhiteLabel    PartnershipType = "white_label"    // Resell services
)

type PartnershipStatus string
const (
	PartnershipProposed   PartnershipStatus = "proposed"
	PartnershipNegotiating PartnershipStatus = "negotiating"
	PartnershipActive     PartnershipStatus = "active"
	PartnershipPaused     PartnershipStatus = "paused"
	PartnershipExpired    PartnershipStatus = "expired"
	PartnershipTerminated PartnershipStatus = "terminated"
)

type PartnershipTerms struct {
	// Referral Terms
	ReferralFeeType        FeeType  `json:"referral_fee_type"`
	ReferralFeeValue       float64  `json:"referral_fee_value"`
	IsBidirectional        bool     `json:"is_bidirectional"`
	
	// A → B Terms (if different)
	AToBFeeType            *FeeType `json:"a_to_b_fee_type,omitempty"`
	AToBFeeValue           *float64 `json:"a_to_b_fee_value,omitempty"`
	
	// B → A Terms (if different)
	BToAFeeType            *FeeType `json:"b_to_a_fee_type,omitempty"`
	BToAFeeValue           *float64 `json:"b_to_a_fee_value,omitempty"`
	
	// Revenue Sharing (for joint ventures)
	RevenueShareA          float64  `json:"revenue_share_a,omitempty"` // Percentage
	RevenueShareB          float64  `json:"revenue_share_b,omitempty"`
	
	// Exclusivity
	IsExclusive            bool     `json:"is_exclusive"`
	ExclusiveCategories    []uuid.UUID `json:"exclusive_categories,omitempty"`
	
	// Service Level
	ResponseTimeHours      int      `json:"response_time_hours"`
	MinAcceptanceRate      float64  `json:"min_acceptance_rate"`
	
	// Duration
	DurationMonths         int      `json:"duration_months"`
	AutoRenew              bool     `json:"auto_renew"`
	
	// Termination
	NoticePeriodDays       int      `json:"notice_period_days"`
}

// =============================================================================
// 2.3 REFERRALS
// =============================================================================

// Referral represents a client referral between vendors
type Referral struct {
	ID                 uuid.UUID           `json:"id"`
	
	// Source and Destination
	SourceVendorID     uuid.UUID           `json:"source_vendor_id"`
	DestVendorID       uuid.UUID           `json:"dest_vendor_id"`
	PartnershipID      *uuid.UUID          `json:"partnership_id,omitempty"`
	
	// Client Info
	ClientUserID       *uuid.UUID          `json:"client_user_id,omitempty"`
	ClientName         string              `json:"client_name"`
	ClientEmail        string              `json:"client_email"`
	ClientPhone        string              `json:"client_phone"`
	
	// Referral Context
	EventType          string              `json:"event_type"`
	EventDate          *time.Time          `json:"event_date,omitempty"`
	ServiceCategory    uuid.UUID           `json:"service_category_id"`
	EstimatedValue     float64             `json:"estimated_value"`
	Notes              string              `json:"notes"`
	
	// Status Tracking
	Status             ReferralStatus      `json:"status"`
	StatusHistory      []StatusChange      `json:"status_history"`
	
	// Outcome
	ConvertedBookingID *uuid.UUID          `json:"converted_booking_id,omitempty"`
	ActualValue        float64             `json:"actual_value"`
	
	// Fee
	FeeType            FeeType             `json:"fee_type"`
	FeeValue           float64             `json:"fee_value"`
	CalculatedFee      float64             `json:"calculated_fee"`
	FeePaid            bool                `json:"fee_paid"`
	FeePaidAt          *time.Time          `json:"fee_paid_at,omitempty"`
	
	// Tracking
	TrackingCode       string              `json:"tracking_code"` // Unique code for tracking
	SourceURL          string              `json:"source_url,omitempty"` // If from link
	
	// Feedback
	SourceFeedback     *ReferralFeedback   `json:"source_feedback,omitempty"`
	DestFeedback       *ReferralFeedback   `json:"dest_feedback,omitempty"`
	
	// Timestamps
	CreatedAt          time.Time           `json:"created_at"`
	ExpiresAt          time.Time           `json:"expires_at"` // Referral validity
	UpdatedAt          time.Time           `json:"updated_at"`
}

type ReferralStatus string
const (
	ReferralPending    ReferralStatus = "pending"     // Sent, waiting for response
	ReferralAccepted   ReferralStatus = "accepted"    // Dest vendor accepted
	ReferralDeclined   ReferralStatus = "declined"    // Dest vendor declined
	ReferralContacted  ReferralStatus = "contacted"   // Dest contacted client
	ReferralQuoted     ReferralStatus = "quoted"      // Quote sent to client
	ReferralConverted  ReferralStatus = "converted"   // Booking confirmed
	ReferralLost       ReferralStatus = "lost"        // Client chose elsewhere
	ReferralExpired    ReferralStatus = "expired"     // No action taken
)

type StatusChange struct {
	Status     ReferralStatus `json:"status"`
	ChangedAt  time.Time      `json:"changed_at"`
	ChangedBy  uuid.UUID      `json:"changed_by"`
	Notes      string         `json:"notes,omitempty"`
}

type ReferralFeedback struct {
	Rating       int       `json:"rating"` // 1-5
	Comment      string    `json:"comment"`
	WouldRepeat  bool      `json:"would_repeat"`
	SubmittedAt  time.Time `json:"submitted_at"`
}

// =============================================================================
// 2.4 COLLABORATIVE BIDDING
// =============================================================================

// CollaborativeBid represents multiple vendors bidding together
type CollaborativeBid struct {
	ID                 uuid.UUID            `json:"id"`
	
	// Opportunity
	OpportunityID      uuid.UUID            `json:"opportunity_id"`
	
	// Lead Vendor
	LeadVendorID       uuid.UUID            `json:"lead_vendor_id"`
	
	// Team
	TeamMembers        []BidTeamMember      `json:"team_members"`
	
	// Bid Details
	TotalBidAmount     float64              `json:"total_bid_amount"`
	Currency           string               `json:"currency"`
	ProposalDoc        string               `json:"proposal_doc_url"`
	PresentationDoc    string               `json:"presentation_doc_url,omitempty"`
	
	// Revenue Split
	SplitAgreement     []RevenueSplit       `json:"split_agreement"`
	
	// Status
	Status             BidStatus            `json:"status"`
	
	// Outcome
	WonBid             bool                 `json:"won_bid"`
	WonAt              *time.Time           `json:"won_at,omitempty"`
	ContractID         *uuid.UUID           `json:"contract_id,omitempty"`
	
	// Timestamps
	CreatedAt          time.Time            `json:"created_at"`
	SubmittedAt        *time.Time           `json:"submitted_at,omitempty"`
	DeadlineAt         time.Time            `json:"deadline_at"`
}

type BidTeamMember struct {
	VendorID       uuid.UUID `json:"vendor_id"`
	Role           string    `json:"role"`
	ServiceScope   string    `json:"service_scope"`
	BidPortion     float64   `json:"bid_portion"`
	Confirmed      bool      `json:"confirmed"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
}

type RevenueSplit struct {
	VendorID     uuid.UUID `json:"vendor_id"`
	Percentage   float64   `json:"percentage"`
	FixedAmount  float64   `json:"fixed_amount,omitempty"`
	Conditions   string    `json:"conditions,omitempty"`
}

type BidStatus string
const (
	BidDraft       BidStatus = "draft"
	BidPending     BidStatus = "pending"     // Waiting for team confirmation
	BidSubmitted   BidStatus = "submitted"
	BidUnderReview BidStatus = "under_review"
	BidWon         BidStatus = "won"
	BidLost        BidStatus = "lost"
	BidWithdrawn   BidStatus = "withdrawn"
)

// Opportunity represents a project that vendors can bid on
type Opportunity struct {
	ID                 uuid.UUID            `json:"id"`
	
	// Client
	ClientUserID       *uuid.UUID           `json:"client_user_id,omitempty"`
	ClientName         string               `json:"client_name"`
	ClientType         string               `json:"client_type"` // 'individual', 'corporate', 'agency'
	
	// Event Details
	Title              string               `json:"title"`
	Description        string               `json:"description"`
	EventType          string               `json:"event_type"`
	EventDate          *time.Time           `json:"event_date,omitempty"`
	EventLocation      string               `json:"event_location"`
	GuestCount         int                  `json:"guest_count"`
	
	// Requirements
	RequiredCategories []uuid.UUID          `json:"required_category_ids"`
	OptionalCategories []uuid.UUID          `json:"optional_category_ids"`
	Requirements       []string             `json:"requirements"`
	
	// Budget
	BudgetMin          float64              `json:"budget_min"`
	BudgetMax          float64              `json:"budget_max"`
	Currency           string               `json:"currency"`
	
	// Status
	Status             OpportunityStatus    `json:"status"`
	Visibility         OpportunityVisibility `json:"visibility"`
	
	// Bidding
	BidDeadline        time.Time            `json:"bid_deadline"`
	BidCount           int                  `json:"bid_count"`
	SelectedBidID      *uuid.UUID           `json:"selected_bid_id,omitempty"`
	
	// Timestamps
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
}

type OpportunityStatus string
const (
	OpportunityOpen      OpportunityStatus = "open"
	OpportunityClosed    OpportunityStatus = "closed"
	OpportunityAwarded   OpportunityStatus = "awarded"
	OpportunityCancelled OpportunityStatus = "cancelled"
)

type OpportunityVisibility string
const (
	VisibilityPublic    OpportunityVisibility = "public"
	VisibilityNetwork   OpportunityVisibility = "network"   // Visible to connected vendors
	VisibilityInvited   OpportunityVisibility = "invited"   // By invitation only
)

// =============================================================================
// SECTION 3: PARTNERSHIP MATCHING ENGINE
// =============================================================================

// PartnershipMatchingEngine finds optimal vendor partnerships
type PartnershipMatchingEngine struct {
	db               *pgxpool.Pool
	cache            *redis.Client
	adjacencyService *AdjacencyService
}

// PartnerMatch represents a potential partnership match
type PartnerMatch struct {
	VendorID         uuid.UUID           `json:"vendor_id"`
	VendorName       string              `json:"vendor_name"`
	Category         string              `json:"category"`
	MatchScore       float64             `json:"match_score"`
	MatchReasons     []MatchReason       `json:"match_reasons"`
	ComplementaryScore float64           `json:"complementary_score"`
	TrustScore       float64             `json:"trust_score"`
	PotentialValue   float64             `json:"potential_value"`
	MutualConnections int                `json:"mutual_connections"`
	Recommendation   string              `json:"recommendation"`
}

type MatchReason struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

// FindPartnerMatches finds potential partners for a vendor
func (e *PartnershipMatchingEngine) FindPartnerMatches(ctx context.Context, vendorID uuid.UUID, limit int) ([]PartnerMatch, error) {
	// Get vendor profile
	profile, err := e.getVendorProfile(ctx, vendorID)
	if err != nil {
		return nil, err
	}
	
	// Get complementary categories
	complementaryCategories := e.adjacencyService.GetComplementaryCategories(profile.PrimaryCategory)
	
	// Find candidate vendors
	candidates, err := e.findCandidates(ctx, vendorID, profile, complementaryCategories)
	if err != nil {
		return nil, err
	}
	
	// Score and rank candidates
	var matches []PartnerMatch
	for _, candidate := range candidates {
		match := e.scoreCandidate(ctx, profile, candidate, complementaryCategories)
		if match.MatchScore > 0.3 { // Minimum threshold
			matches = append(matches, match)
		}
	}
	
	// Sort by match score
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].MatchScore > matches[j].MatchScore
	})
	
	if len(matches) > limit {
		matches = matches[:limit]
	}
	
	return matches, nil
}

func (e *PartnershipMatchingEngine) getVendorProfile(ctx context.Context, vendorID uuid.UUID) (*VendorProfile, error) {
	query := `
		SELECT id, vendor_id, network_handle, display_name, 
		       primary_category_id, secondary_category_ids,
		       network_trust_score, response_rate, referral_success_rate,
		       partnership_preferences
		FROM vendor_profiles
		WHERE vendor_id = $1
	`
	
	var profile VendorProfile
	var secondaryCategories []uuid.UUID
	var prefsJSON []byte
	
	err := e.db.QueryRow(ctx, query, vendorID).Scan(
		&profile.ID, &profile.VendorID, &profile.NetworkHandle, &profile.DisplayName,
		&profile.PrimaryCategory, &secondaryCategories,
		&profile.NetworkTrustScore, &profile.ResponseRate, &profile.ReferralSuccessRate,
		&prefsJSON,
	)
	
	if err != nil {
		return nil, err
	}
	
	profile.SecondaryCategories = secondaryCategories
	json.Unmarshal(prefsJSON, &profile.PartnershipPreferences)
	
	return &profile, nil
}

type CandidateVendor struct {
	VendorID         uuid.UUID
	VendorName       string
	CategoryID       uuid.UUID
	CategoryName     string
	Rating           float64
	ReviewCount      int
	TrustScore       float64
	ResponseRate     float64
	ReferralSuccess  float64
	ExistingPartners int
	IsVerified       bool
}

func (e *PartnershipMatchingEngine) findCandidates(ctx context.Context, excludeVendorID uuid.UUID, profile *VendorProfile, complementaryCategories []uuid.UUID) ([]CandidateVendor, error) {
	query := `
		SELECT 
			vp.vendor_id,
			v.business_name,
			vp.primary_category_id,
			sc.name as category_name,
			v.rating_average,
			v.rating_count,
			vp.network_trust_score,
			vp.response_rate,
			vp.referral_success_rate,
			vp.partnership_count,
			v.is_verified
		FROM vendor_profiles vp
		JOIN vendors v ON v.id = vp.vendor_id
		JOIN service_categories sc ON sc.id = vp.primary_category_id
		WHERE vp.vendor_id != $1
		  AND vp.accepting_new_partners = TRUE
		  AND v.is_active = TRUE
		  AND (vp.primary_category_id = ANY($2) OR vp.primary_category_id != $3)
		  AND NOT EXISTS (
			  SELECT 1 FROM partnerships p
			  WHERE (p.vendor_a_id = $1 AND p.vendor_b_id = vp.vendor_id)
			     OR (p.vendor_b_id = $1 AND p.vendor_a_id = vp.vendor_id)
			  AND p.status = 'active'
		  )
		ORDER BY vp.network_trust_score DESC, v.rating_average DESC
		LIMIT 100
	`
	
	rows, err := e.db.Query(ctx, query, excludeVendorID, complementaryCategories, profile.PrimaryCategory)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var candidates []CandidateVendor
	for rows.Next() {
		var c CandidateVendor
		if err := rows.Scan(
			&c.VendorID, &c.VendorName, &c.CategoryID, &c.CategoryName,
			&c.Rating, &c.ReviewCount, &c.TrustScore, &c.ResponseRate,
			&c.ReferralSuccess, &c.ExistingPartners, &c.IsVerified,
		); err != nil {
			continue
		}
		candidates = append(candidates, c)
	}
	
	return candidates, nil
}

func (e *PartnershipMatchingEngine) scoreCandidate(ctx context.Context, profile *VendorProfile, candidate CandidateVendor, complementaryCategories []uuid.UUID) PartnerMatch {
	match := PartnerMatch{
		VendorID:   candidate.VendorID,
		VendorName: candidate.VendorName,
		Category:   candidate.CategoryName,
		TrustScore: candidate.TrustScore,
	}
	
	var reasons []MatchReason
	totalScore := 0.0
	
	// 1. Complementary Category Score (0.30 weight)
	isComplementary := false
	for _, cat := range complementaryCategories {
		if cat == candidate.CategoryID {
			isComplementary = true
			break
		}
	}
	
	if isComplementary {
		complementaryScore := 0.9
		match.ComplementaryScore = complementaryScore
		totalScore += complementaryScore * 0.30
		reasons = append(reasons, MatchReason{
			Type:        "complementary_category",
			Description: fmt.Sprintf("Services in %s complement your offerings", candidate.CategoryName),
			Score:       complementaryScore,
		})
	} else if candidate.CategoryID != profile.PrimaryCategory {
		// Not direct competitor but not strongly complementary
		totalScore += 0.5 * 0.30
		match.ComplementaryScore = 0.5
	}
	
	// 2. Trust Score (0.25 weight)
	trustScore := candidate.TrustScore / 100.0
	totalScore += trustScore * 0.25
	if trustScore > 0.8 {
		reasons = append(reasons, MatchReason{
			Type:        "high_trust",
			Description: "Highly trusted vendor in the network",
			Score:       trustScore,
		})
	}
	
	// 3. Performance Score (0.20 weight)
	performanceScore := (candidate.ResponseRate + candidate.ReferralSuccess) / 2.0
	totalScore += performanceScore * 0.20
	if performanceScore > 0.85 {
		reasons = append(reasons, MatchReason{
			Type:        "high_performance",
			Description: fmt.Sprintf("%.0f%% response rate, %.0f%% referral success", candidate.ResponseRate*100, candidate.ReferralSuccess*100),
			Score:       performanceScore,
		})
	}
	
	// 4. Rating Score (0.15 weight)
	ratingScore := candidate.Rating / 5.0
	totalScore += ratingScore * 0.15
	if candidate.Rating >= 4.5 {
		reasons = append(reasons, MatchReason{
			Type:        "top_rated",
			Description: fmt.Sprintf("%.1f star rating from %d reviews", candidate.Rating, candidate.ReviewCount),
			Score:       ratingScore,
		})
	}
	
	// 5. Verification Bonus (0.10 weight)
	if candidate.IsVerified {
		totalScore += 1.0 * 0.10
		reasons = append(reasons, MatchReason{
			Type:        "verified",
			Description: "Verified business identity",
			Score:       1.0,
		})
	}
	
	// 6. Mutual Connections Bonus
	mutualCount := e.getMutualConnectionCount(ctx, profile.VendorID, candidate.VendorID)
	match.MutualConnections = mutualCount
	if mutualCount > 0 {
		connectionBonus := math.Min(float64(mutualCount)/10.0, 0.1)
		totalScore += connectionBonus
		reasons = append(reasons, MatchReason{
			Type:        "mutual_connections",
			Description: fmt.Sprintf("%d mutual connections in the network", mutualCount),
			Score:       connectionBonus * 10,
		})
	}
	
	match.MatchScore = math.Min(totalScore, 1.0)
	match.MatchReasons = reasons
	
	// Estimate potential value
	match.PotentialValue = e.estimatePotentialValue(ctx, profile, candidate)
	
	// Generate recommendation
	match.Recommendation = e.generateRecommendation(match)
	
	return match
}

func (e *PartnershipMatchingEngine) getMutualConnectionCount(ctx context.Context, vendorA, vendorB uuid.UUID) int {
	query := `
		SELECT COUNT(*) FROM (
			SELECT vendor_b_id as connected_vendor FROM connections WHERE vendor_a_id = $1 AND status = 'accepted'
			UNION
			SELECT vendor_a_id as connected_vendor FROM connections WHERE vendor_b_id = $1 AND status = 'accepted'
		) a
		JOIN (
			SELECT vendor_b_id as connected_vendor FROM connections WHERE vendor_a_id = $2 AND status = 'accepted'
			UNION
			SELECT vendor_a_id as connected_vendor FROM connections WHERE vendor_b_id = $2 AND status = 'accepted'
		) b ON a.connected_vendor = b.connected_vendor
	`
	
	var count int
	e.db.QueryRow(ctx, query, vendorA, vendorB).Scan(&count)
	return count
}

func (e *PartnershipMatchingEngine) estimatePotentialValue(ctx context.Context, profile *VendorProfile, candidate CandidateVendor) float64 {
	// Estimate based on historical referral values for this category pair
	query := `
		SELECT AVG(actual_value) FROM referrals r
		JOIN vendor_profiles vp_src ON vp_src.vendor_id = r.source_vendor_id
		JOIN vendor_profiles vp_dst ON vp_dst.vendor_id = r.dest_vendor_id
		WHERE vp_src.primary_category_id = $1
		  AND vp_dst.primary_category_id = $2
		  AND r.status = 'converted'
		  AND r.created_at > NOW() - INTERVAL '6 months'
	`
	
	var avgValue float64
	e.db.QueryRow(ctx, query, profile.PrimaryCategory, candidate.CategoryID).Scan(&avgValue)
	
	if avgValue == 0 {
		avgValue = 200000 // Default estimate
	}
	
	// Project annual value based on referral frequency
	annualReferrals := 12.0 // Estimate 1 per month
	return avgValue * annualReferrals
}

func (e *PartnershipMatchingEngine) generateRecommendation(match PartnerMatch) string {
	if match.MatchScore > 0.8 {
		return "Highly recommended partner - strong complementary fit with excellent track record"
	} else if match.MatchScore > 0.6 {
		return "Good potential partner - consider reaching out to explore collaboration"
	} else if match.MatchScore > 0.4 {
		return "Moderate fit - may be worth connecting for occasional referrals"
	}
	return "Consider other options first"
}

// AdjacencyService provides category relationship data
type AdjacencyService struct {
	db *pgxpool.Pool
}

func (s *AdjacencyService) GetComplementaryCategories(categoryID uuid.UUID) []uuid.UUID {
	query := `
		SELECT target_category_id FROM service_adjacencies
		WHERE source_category_id = $1
		  AND adjacency_type = 'complementary'
		  AND is_active = TRUE
		ORDER BY computed_score DESC
		LIMIT 10
	`
	
	rows, _ := s.db.Query(context.Background(), query, categoryID)
	defer rows.Close()
	
	var categories []uuid.UUID
	for rows.Next() {
		var catID uuid.UUID
		rows.Scan(&catID)
		categories = append(categories, catID)
	}
	
	return categories
}

// =============================================================================
// SECTION 4: REFERRAL TRACKING ENGINE
// =============================================================================

// ReferralEngine manages the referral lifecycle
type ReferralEngine struct {
	db               *pgxpool.Pool
	cache            *redis.Client
	notificationSvc  *NotificationService
	paymentSvc       *PaymentService
}

// CreateReferralRequest for sending a referral
type CreateReferralRequest struct {
	SourceVendorID   uuid.UUID `json:"source_vendor_id"`
	DestVendorID     uuid.UUID `json:"dest_vendor_id"`
	ClientName       string    `json:"client_name"`
	ClientEmail      string    `json:"client_email"`
	ClientPhone      string    `json:"client_phone"`
	EventType        string    `json:"event_type"`
	EventDate        *time.Time `json:"event_date,omitempty"`
	ServiceCategory  uuid.UUID `json:"service_category_id"`
	EstimatedValue   float64   `json:"estimated_value"`
	Notes            string    `json:"notes"`
}

// CreateReferral creates a new referral
func (e *ReferralEngine) CreateReferral(ctx context.Context, req CreateReferralRequest) (*Referral, error) {
	// Get partnership terms if exists
	partnership, _ := e.getActivePartnership(ctx, req.SourceVendorID, req.DestVendorID)
	
	// Determine fee structure
	feeType, feeValue := e.determineFee(ctx, req.SourceVendorID, req.DestVendorID, partnership)
	
	referral := &Referral{
		ID:               uuid.New(),
		SourceVendorID:   req.SourceVendorID,
		DestVendorID:     req.DestVendorID,
		ClientName:       req.ClientName,
		ClientEmail:      req.ClientEmail,
		ClientPhone:      req.ClientPhone,
		EventType:        req.EventType,
		EventDate:        req.EventDate,
		ServiceCategory:  req.ServiceCategory,
		EstimatedValue:   req.EstimatedValue,
		Notes:            req.Notes,
		Status:           ReferralPending,
		StatusHistory: []StatusChange{
			{Status: ReferralPending, ChangedAt: time.Now(), ChangedBy: req.SourceVendorID},
		},
		FeeType:          feeType,
		FeeValue:         feeValue,
		TrackingCode:     e.generateTrackingCode(),
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().AddDate(0, 0, 30), // 30 day validity
		UpdatedAt:        time.Now(),
	}
	
	if partnership != nil {
		referral.PartnershipID = &partnership.ID
	}
	
	// Calculate fee
	referral.CalculatedFee = e.calculateFee(referral)
	
	// Save referral
	if err := e.saveReferral(ctx, referral); err != nil {
		return nil, err
	}
	
	// Notify destination vendor
	e.notificationSvc.NotifyNewReferral(ctx, referral)
	
	return referral, nil
}

func (e *ReferralEngine) getActivePartnership(ctx context.Context, vendorA, vendorB uuid.UUID) (*Partnership, error) {
	query := `
		SELECT id, terms FROM partnerships
		WHERE ((vendor_a_id = $1 AND vendor_b_id = $2) OR (vendor_a_id = $2 AND vendor_b_id = $1))
		  AND status = 'active'
		LIMIT 1
	`
	
	var p Partnership
	var termsJSON []byte
	
	err := e.db.QueryRow(ctx, query, vendorA, vendorB).Scan(&p.ID, &termsJSON)
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(termsJSON, &p.Terms)
	return &p, nil
}

func (e *ReferralEngine) determineFee(ctx context.Context, sourceVendor, destVendor uuid.UUID, partnership *Partnership) (FeeType, float64) {
	// If partnership exists, use partnership terms
	if partnership != nil {
		return partnership.Terms.ReferralFeeType, partnership.Terms.ReferralFeeValue
	}
	
	// Get destination vendor's default referral preferences
	query := `SELECT referral_preferences FROM vendor_profiles WHERE vendor_id = $1`
	var prefsJSON []byte
	e.db.QueryRow(ctx, query, destVendor).Scan(&prefsJSON)
	
	var prefs ReferralPrefs
	json.Unmarshal(prefsJSON, &prefs)
	
	if prefs.DefaultFeeType != "" {
		return prefs.DefaultFeeType, prefs.DefaultFeeValue
	}
	
	// Platform default
	return FeePercentage, 10.0
}

func (e *ReferralEngine) calculateFee(referral *Referral) float64 {
	switch referral.FeeType {
	case FeePercentage:
		return referral.EstimatedValue * (referral.FeeValue / 100.0)
	case FeeFixed:
		return referral.FeeValue
	default:
		return 0
	}
}

func (e *ReferralEngine) generateTrackingCode() string {
	// Generate unique tracking code
	return fmt.Sprintf("REF-%s", uuid.New().String()[:8])
}

// UpdateReferralStatus updates the referral status
func (e *ReferralEngine) UpdateReferralStatus(ctx context.Context, referralID uuid.UUID, newStatus ReferralStatus, vendorID uuid.UUID, notes string) error {
	// Get current referral
	referral, err := e.getReferral(ctx, referralID)
	if err != nil {
		return err
	}
	
	// Validate status transition
	if !e.isValidStatusTransition(referral.Status, newStatus) {
		return fmt.Errorf("invalid status transition from %s to %s", referral.Status, newStatus)
	}
	
	// Update status
	referral.Status = newStatus
	referral.StatusHistory = append(referral.StatusHistory, StatusChange{
		Status:    newStatus,
		ChangedAt: time.Now(),
		ChangedBy: vendorID,
		Notes:     notes,
	})
	referral.UpdatedAt = time.Now()
	
	// Handle conversion
	if newStatus == ReferralConverted {
		// Recalculate fee based on actual value
		if referral.ActualValue > 0 {
			referral.CalculatedFee = e.calculateFeeForValue(referral, referral.ActualValue)
		}
	}
	
	// Save
	if err := e.updateReferral(ctx, referral); err != nil {
		return err
	}
	
	// Notify source vendor of status change
	e.notificationSvc.NotifyReferralStatusChange(ctx, referral)
	
	return nil
}

func (e *ReferralEngine) isValidStatusTransition(current, next ReferralStatus) bool {
	validTransitions := map[ReferralStatus][]ReferralStatus{
		ReferralPending:   {ReferralAccepted, ReferralDeclined, ReferralExpired},
		ReferralAccepted:  {ReferralContacted, ReferralLost, ReferralExpired},
		ReferralContacted: {ReferralQuoted, ReferralLost},
		ReferralQuoted:    {ReferralConverted, ReferralLost},
	}
	
	valid, ok := validTransitions[current]
	if !ok {
		return false
	}
	
	for _, v := range valid {
		if v == next {
			return true
		}
	}
	return false
}

func (e *ReferralEngine) calculateFeeForValue(referral *Referral, actualValue float64) float64 {
	switch referral.FeeType {
	case FeePercentage:
		return actualValue * (referral.FeeValue / 100.0)
	case FeeFixed:
		return referral.FeeValue
	default:
		return 0
	}
}

// ProcessReferralPayment handles fee payment for converted referrals
func (e *ReferralEngine) ProcessReferralPayment(ctx context.Context, referralID uuid.UUID) error {
	referral, err := e.getReferral(ctx, referralID)
	if err != nil {
		return err
	}
	
	if referral.Status != ReferralConverted {
		return fmt.Errorf("referral not converted")
	}
	
	if referral.FeePaid {
		return fmt.Errorf("fee already paid")
	}
	
	// Process payment through payment service
	paymentID, err := e.paymentSvc.ProcessReferralFee(ctx, referral)
	if err != nil {
		return err
	}
	
	// Update referral
	now := time.Now()
	referral.FeePaid = true
	referral.FeePaidAt = &now
	
	e.updateReferral(ctx, referral)
	
	// Notify both parties
	e.notificationSvc.NotifyReferralPayment(ctx, referral, paymentID)
	
	return nil
}

func (e *ReferralEngine) getReferral(ctx context.Context, referralID uuid.UUID) (*Referral, error) {
	query := `
		SELECT id, source_vendor_id, dest_vendor_id, partnership_id,
		       client_name, client_email, client_phone,
		       event_type, event_date, service_category_id, estimated_value, notes,
		       status, status_history, actual_value,
		       fee_type, fee_value, calculated_fee, fee_paid, fee_paid_at,
		       tracking_code, created_at, expires_at, updated_at
		FROM referrals
		WHERE id = $1
	`
	
	var r Referral
	var statusHistoryJSON []byte
	
	err := e.db.QueryRow(ctx, query, referralID).Scan(
		&r.ID, &r.SourceVendorID, &r.DestVendorID, &r.PartnershipID,
		&r.ClientName, &r.ClientEmail, &r.ClientPhone,
		&r.EventType, &r.EventDate, &r.ServiceCategory, &r.EstimatedValue, &r.Notes,
		&r.Status, &statusHistoryJSON, &r.ActualValue,
		&r.FeeType, &r.FeeValue, &r.CalculatedFee, &r.FeePaid, &r.FeePaidAt,
		&r.TrackingCode, &r.CreatedAt, &r.ExpiresAt, &r.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(statusHistoryJSON, &r.StatusHistory)
	return &r, nil
}

func (e *ReferralEngine) saveReferral(ctx context.Context, r *Referral) error {
	statusHistoryJSON, _ := json.Marshal(r.StatusHistory)
	
	query := `
		INSERT INTO referrals (
			id, source_vendor_id, dest_vendor_id, partnership_id,
			client_name, client_email, client_phone,
			event_type, event_date, service_category_id, estimated_value, notes,
			status, status_history, actual_value,
			fee_type, fee_value, calculated_fee, fee_paid,
			tracking_code, created_at, expires_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
	`
	
	_, err := e.db.Exec(ctx, query,
		r.ID, r.SourceVendorID, r.DestVendorID, r.PartnershipID,
		r.ClientName, r.ClientEmail, r.ClientPhone,
		r.EventType, r.EventDate, r.ServiceCategory, r.EstimatedValue, r.Notes,
		r.Status, statusHistoryJSON, r.ActualValue,
		r.FeeType, r.FeeValue, r.CalculatedFee, r.FeePaid,
		r.TrackingCode, r.CreatedAt, r.ExpiresAt, r.UpdatedAt,
	)
	
	return err
}

func (e *ReferralEngine) updateReferral(ctx context.Context, r *Referral) error {
	statusHistoryJSON, _ := json.Marshal(r.StatusHistory)
	
	query := `
		UPDATE referrals SET
			status = $2,
			status_history = $3,
			actual_value = $4,
			calculated_fee = $5,
			fee_paid = $6,
			fee_paid_at = $7,
			updated_at = $8
		WHERE id = $1
	`
	
	_, err := e.db.Exec(ctx, query,
		r.ID, r.Status, statusHistoryJSON, r.ActualValue,
		r.CalculatedFee, r.FeePaid, r.FeePaidAt, r.UpdatedAt,
	)
	
	return err
}

// =============================================================================
// SECTION 5: ANALYTICS & INSIGHTS
// =============================================================================

// NetworkAnalytics provides insights for vendors
type NetworkAnalytics struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// VendorNetworkStats represents a vendor's network statistics
type VendorNetworkStats struct {
	// Overview
	TotalConnections      int     `json:"total_connections"`
	ActivePartnerships    int     `json:"active_partnerships"`
	NetworkReach          int     `json:"network_reach"` // 2nd degree connections
	
	// Referrals
	ReferralsSentTotal    int     `json:"referrals_sent_total"`
	ReferralsSentMonth    int     `json:"referrals_sent_month"`
	ReferralsReceivedTotal int    `json:"referrals_received_total"`
	ReferralsReceivedMonth int    `json:"referrals_received_month"`
	ReferralConversionRate float64 `json:"referral_conversion_rate"`
	
	// Revenue
	TotalReferralRevenue  float64 `json:"total_referral_revenue"`
	MonthlyReferralRevenue float64 `json:"monthly_referral_revenue"`
	AvgReferralValue      float64 `json:"avg_referral_value"`
	TotalFeesEarned       float64 `json:"total_fees_earned"`
	TotalFeesPaid         float64 `json:"total_fees_paid"`
	
	// Performance
	ResponseRate          float64 `json:"response_rate"`
	AvgResponseTimeHours  float64 `json:"avg_response_time_hours"`
	
	// Growth
	NewConnectionsMonth   int     `json:"new_connections_month"`
	ReferralGrowthPct     float64 `json:"referral_growth_pct"`
	
	// Top Partners
	TopReferrers          []PartnerStat `json:"top_referrers"`
	TopReceivers          []PartnerStat `json:"top_receivers"`
}

type PartnerStat struct {
	VendorID     uuid.UUID `json:"vendor_id"`
	VendorName   string    `json:"vendor_name"`
	ReferralCount int      `json:"referral_count"`
	TotalValue   float64   `json:"total_value"`
	ConversionRate float64 `json:"conversion_rate"`
}

// GetVendorStats gets comprehensive stats for a vendor
func (a *NetworkAnalytics) GetVendorStats(ctx context.Context, vendorID uuid.UUID) (*VendorNetworkStats, error) {
	stats := &VendorNetworkStats{}
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	// Parallel stat collection
	wg.Add(5)
	
	go func() {
		defer wg.Done()
		s := a.getConnectionStats(ctx, vendorID)
		mu.Lock()
		stats.TotalConnections = s.total
		stats.ActivePartnerships = s.partnerships
		stats.NetworkReach = s.reach
		stats.NewConnectionsMonth = s.newMonth
		mu.Unlock()
	}()
	
	go func() {
		defer wg.Done()
		s := a.getReferralStats(ctx, vendorID)
		mu.Lock()
		stats.ReferralsSentTotal = s.sentTotal
		stats.ReferralsSentMonth = s.sentMonth
		stats.ReferralsReceivedTotal = s.receivedTotal
		stats.ReferralsReceivedMonth = s.receivedMonth
		stats.ReferralConversionRate = s.conversionRate
		mu.Unlock()
	}()
	
	go func() {
		defer wg.Done()
		s := a.getRevenueStats(ctx, vendorID)
		mu.Lock()
		stats.TotalReferralRevenue = s.totalRevenue
		stats.MonthlyReferralRevenue = s.monthlyRevenue
		stats.AvgReferralValue = s.avgValue
		stats.TotalFeesEarned = s.feesEarned
		stats.TotalFeesPaid = s.feesPaid
		mu.Unlock()
	}()
	
	go func() {
		defer wg.Done()
		s := a.getPerformanceStats(ctx, vendorID)
		mu.Lock()
		stats.ResponseRate = s.responseRate
		stats.AvgResponseTimeHours = s.avgResponseTime
		mu.Unlock()
	}()
	
	go func() {
		defer wg.Done()
		mu.Lock()
		stats.TopReferrers = a.getTopReferrers(ctx, vendorID, 5)
		stats.TopReceivers = a.getTopReceivers(ctx, vendorID, 5)
		mu.Unlock()
	}()
	
	wg.Wait()
	
	return stats, nil
}

type connectionStats struct {
	total        int
	partnerships int
	reach        int
	newMonth     int
}

func (a *NetworkAnalytics) getConnectionStats(ctx context.Context, vendorID uuid.UUID) connectionStats {
	var s connectionStats
	
	// Total connections
	a.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM connections
		WHERE (vendor_a_id = $1 OR vendor_b_id = $1) AND status = 'accepted'
	`, vendorID).Scan(&s.total)
	
	// Active partnerships
	a.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM partnerships
		WHERE (vendor_a_id = $1 OR vendor_b_id = $1) AND status = 'active'
	`, vendorID).Scan(&s.partnerships)
	
	// Network reach (2nd degree)
	a.db.QueryRow(ctx, `
		WITH direct AS (
			SELECT CASE WHEN vendor_a_id = $1 THEN vendor_b_id ELSE vendor_a_id END as connected
			FROM connections
			WHERE (vendor_a_id = $1 OR vendor_b_id = $1) AND status = 'accepted'
		)
		SELECT COUNT(DISTINCT c2.vendor_b_id) + COUNT(DISTINCT c2.vendor_a_id)
		FROM direct d
		JOIN connections c2 ON (c2.vendor_a_id = d.connected OR c2.vendor_b_id = d.connected)
		WHERE c2.status = 'accepted'
		  AND c2.vendor_a_id != $1 AND c2.vendor_b_id != $1
	`, vendorID).Scan(&s.reach)
	
	// New this month
	a.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM connections
		WHERE (vendor_a_id = $1 OR vendor_b_id = $1) 
		  AND status = 'accepted'
		  AND accepted_at > DATE_TRUNC('month', CURRENT_DATE)
	`, vendorID).Scan(&s.newMonth)
	
	return s
}

type referralStats struct {
	sentTotal      int
	sentMonth      int
	receivedTotal  int
	receivedMonth  int
	conversionRate float64
}

func (a *NetworkAnalytics) getReferralStats(ctx context.Context, vendorID uuid.UUID) referralStats {
	var s referralStats
	
	// Sent totals
	a.db.QueryRow(ctx, `SELECT COUNT(*) FROM referrals WHERE source_vendor_id = $1`, vendorID).Scan(&s.sentTotal)
	a.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM referrals 
		WHERE source_vendor_id = $1 AND created_at > DATE_TRUNC('month', CURRENT_DATE)
	`, vendorID).Scan(&s.sentMonth)
	
	// Received totals
	a.db.QueryRow(ctx, `SELECT COUNT(*) FROM referrals WHERE dest_vendor_id = $1`, vendorID).Scan(&s.receivedTotal)
	a.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM referrals 
		WHERE dest_vendor_id = $1 AND created_at > DATE_TRUNC('month', CURRENT_DATE)
	`, vendorID).Scan(&s.receivedMonth)
	
	// Conversion rate (for received referrals)
	var converted, total int
	a.db.QueryRow(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'converted'),
			COUNT(*) FILTER (WHERE status NOT IN ('pending', 'expired'))
		FROM referrals WHERE dest_vendor_id = $1
	`, vendorID).Scan(&converted, &total)
	
	if total > 0 {
		s.conversionRate = float64(converted) / float64(total)
	}
	
	return s
}

type revenueStats struct {
	totalRevenue   float64
	monthlyRevenue float64
	avgValue       float64
	feesEarned     float64
	feesPaid       float64
}

func (a *NetworkAnalytics) getRevenueStats(ctx context.Context, vendorID uuid.UUID) revenueStats {
	var s revenueStats
	
	// Total revenue from received referrals
	a.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(actual_value), 0)
		FROM referrals
		WHERE dest_vendor_id = $1 AND status = 'converted'
	`, vendorID).Scan(&s.totalRevenue)
	
	// Monthly revenue
	a.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(actual_value), 0)
		FROM referrals
		WHERE dest_vendor_id = $1 
		  AND status = 'converted'
		  AND created_at > DATE_TRUNC('month', CURRENT_DATE)
	`, vendorID).Scan(&s.monthlyRevenue)
	
	// Average value
	a.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(actual_value), 0)
		FROM referrals
		WHERE dest_vendor_id = $1 AND status = 'converted' AND actual_value > 0
	`, vendorID).Scan(&s.avgValue)
	
	// Fees earned (from sent referrals)
	a.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(calculated_fee), 0)
		FROM referrals
		WHERE source_vendor_id = $1 AND status = 'converted' AND fee_paid = TRUE
	`, vendorID).Scan(&s.feesEarned)
	
	// Fees paid (for received referrals)
	a.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(calculated_fee), 0)
		FROM referrals
		WHERE dest_vendor_id = $1 AND status = 'converted' AND fee_paid = TRUE
	`, vendorID).Scan(&s.feesPaid)
	
	return s
}

type performanceStats struct {
	responseRate    float64
	avgResponseTime float64
}

func (a *NetworkAnalytics) getPerformanceStats(ctx context.Context, vendorID uuid.UUID) performanceStats {
	var s performanceStats
	
	// Response rate
	var responded, total int
	a.db.QueryRow(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE status != 'pending' AND status != 'expired'),
			COUNT(*)
		FROM referrals WHERE dest_vendor_id = $1
	`, vendorID).Scan(&responded, &total)
	
	if total > 0 {
		s.responseRate = float64(responded) / float64(total)
	}
	
	// Average response time (hours to first status change)
	a.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(
			EXTRACT(EPOCH FROM (
				(status_history->1->>'changed_at')::timestamp - created_at
			)) / 3600
		), 24)
		FROM referrals
		WHERE dest_vendor_id = $1 
		  AND jsonb_array_length(status_history) > 1
	`, vendorID).Scan(&s.avgResponseTime)
	
	return s
}

func (a *NetworkAnalytics) getTopReferrers(ctx context.Context, vendorID uuid.UUID, limit int) []PartnerStat {
	query := `
		SELECT 
			r.source_vendor_id,
			v.business_name,
			COUNT(*) as referral_count,
			COALESCE(SUM(r.actual_value), 0) as total_value,
			COUNT(*) FILTER (WHERE r.status = 'converted')::float / NULLIF(COUNT(*), 0) as conversion_rate
		FROM referrals r
		JOIN vendors v ON v.id = r.source_vendor_id
		WHERE r.dest_vendor_id = $1
		GROUP BY r.source_vendor_id, v.business_name
		ORDER BY referral_count DESC
		LIMIT $2
	`
	
	rows, _ := a.db.Query(ctx, query, vendorID, limit)
	defer rows.Close()
	
	var stats []PartnerStat
	for rows.Next() {
		var s PartnerStat
		rows.Scan(&s.VendorID, &s.VendorName, &s.ReferralCount, &s.TotalValue, &s.ConversionRate)
		stats = append(stats, s)
	}
	
	return stats
}

func (a *NetworkAnalytics) getTopReceivers(ctx context.Context, vendorID uuid.UUID, limit int) []PartnerStat {
	query := `
		SELECT 
			r.dest_vendor_id,
			v.business_name,
			COUNT(*) as referral_count,
			COALESCE(SUM(r.calculated_fee) FILTER (WHERE r.fee_paid), 0) as total_value,
			COUNT(*) FILTER (WHERE r.status = 'converted')::float / NULLIF(COUNT(*), 0) as conversion_rate
		FROM referrals r
		JOIN vendors v ON v.id = r.dest_vendor_id
		WHERE r.source_vendor_id = $1
		GROUP BY r.dest_vendor_id, v.business_name
		ORDER BY referral_count DESC
		LIMIT $2
	`
	
	rows, _ := a.db.Query(ctx, query, vendorID, limit)
	defer rows.Close()
	
	var stats []PartnerStat
	for rows.Next() {
		var s PartnerStat
		rows.Scan(&s.VendorID, &s.VendorName, &s.ReferralCount, &s.TotalValue, &s.ConversionRate)
		stats = append(stats, s)
	}
	
	return stats
}

/*
================================================================================
SECTION 6: BUSINESS MODEL
================================================================================

REVENUE STREAMS:

1. SUBSCRIPTION TIERS (Monthly)

   FREE TIER:
   - Basic profile
   - 5 connections
   - Manual referral tracking
   - Basic analytics

   PROFESSIONAL (₦15,000/month):
   - Unlimited connections
   - Auto referral tracking
   - Partnership management
   - Full analytics
   - Priority in partner matching
   - 5% lower platform fee on referral payments

   BUSINESS (₦50,000/month):
   - Everything in Professional
   - Collaborative bidding
   - Team accounts (up to 5)
   - API access
   - White-label referral links
   - Dedicated account manager
   - Custom partnership agreements

   ENTERPRISE (Custom):
   - Unlimited team accounts
   - Custom integrations
   - SLA guarantees
   - Volume discounts

2. TRANSACTION FEES

   - Referral payment processing: 2.5% of fee amount
   - Collaborative bid platform fee: 3% of won contracts
   - Instant payout: 1.5% additional fee

3. PREMIUM FEATURES (à la carte)

   - Featured partner placement: ₦20,000/month
   - Verified business badge: ₦50,000 one-time
   - Advanced analytics report: ₦10,000/report
   - Partnership agreement templates: ₦5,000 each

4. OPPORTUNITY MARKETPLACE

   - Posting fee for large opportunities: ₦25,000+
   - Featured opportunity placement: ₦50,000
   - Bid review/optimization: ₦15,000

================================================================================
*/

// Placeholder services
type NotificationService struct{}

func (n *NotificationService) NotifyNewReferral(ctx context.Context, r *Referral) {}
func (n *NotificationService) NotifyReferralStatusChange(ctx context.Context, r *Referral) {}
func (n *NotificationService) NotifyReferralPayment(ctx context.Context, r *Referral, paymentID string) {}

type PaymentService struct{}

func (p *PaymentService) ProcessReferralFee(ctx context.Context, r *Referral) (string, error) {
	return "PAY-" + uuid.New().String()[:8], nil
}
