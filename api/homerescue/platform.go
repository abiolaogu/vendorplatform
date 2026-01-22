// =============================================================================
// HOMERESCUE - EMERGENCY HOME SERVICES PLATFORM
// Comprehensive Technical & Business Specification
// Version: 1.0.0
// =============================================================================

package homerescue

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

HOMERESCUE: Emergency Home Services On-Demand

TAGLINE: "Help arrives in minutes, not hours."

VISION:
HomeRescue is the emergency response system for home crises. When a pipe bursts
at 2 AM, when the power goes out before an important meeting, when you're locked
out in the rain—HomeRescue connects you with verified, available professionals
who can respond immediately. It's Uber for home emergencies.

CORE VALUE PROPOSITION:
"One tap to rescue. Real-time tracking. Guaranteed response."

TARGET SEGMENTS:
1. Homeowners: Primary residence emergencies
2. Renters: Quick fixes without landlord delays
3. Property Managers: Managing multiple units
4. Businesses: Commercial property emergencies
5. Insurance Companies: Preferred vendor network

KEY DIFFERENTIATORS:
1. Real-Time Availability: See who's available NOW, not tomorrow
2. Emergency-First Design: Optimized for speed, not browsing
3. Guaranteed Response Time: SLA with refund if missed
4. Live Tracking: Know exactly when help arrives
5. Instant Documentation: Photos, receipts, reports for insurance

EMERGENCY CATEGORIES:
1. PLUMBING: Burst pipes, severe leaks, blocked drains, no water
2. ELECTRICAL: Power outage, sparking outlets, exposed wires
3. LOCKSMITH: Locked out, broken locks, security breach
4. HVAC: AC failure (in heat), heating failure (in cold)
5. GLASS/WINDOWS: Broken windows, security compromise
6. ROOFING: Active leaks, storm damage
7. PEST: Dangerous infestations, snake/animal removal
8. SECURITY: Alarm issues, break-in damage repair

RESPONSE TIME TIERS:
- CRITICAL (< 30 min): Life safety, security breach, major water damage
- URGENT (< 2 hours): Significant damage potential, no utilities
- SAME-DAY (< 6 hours): Important but contained issues
- SCHEDULED: Non-emergency repairs discovered during emergency response

================================================================================
SECTION 2: CORE DOMAIN TYPES
================================================================================
*/

// =============================================================================
// 2.1 EMERGENCY REQUEST TYPES
// =============================================================================

// EmergencyRequest represents an emergency service request
type EmergencyRequest struct {
	ID                  uuid.UUID              `json:"id"`
	
	// Requester
	UserID              uuid.UUID              `json:"user_id"`
	PropertyID          *uuid.UUID             `json:"property_id,omitempty"`
	
	// Emergency Classification
	Category            EmergencyCategory      `json:"category"`
	Subcategory         string                 `json:"subcategory"`
	Urgency             UrgencyLevel           `json:"urgency"`
	
	// Description
	Title               string                 `json:"title"`
	Description         string                 `json:"description"`
	Photos              []MediaAttachment      `json:"photos,omitempty"`
	VoiceNote           *MediaAttachment       `json:"voice_note,omitempty"`
	
	// Location
	Location            EmergencyLocation      `json:"location"`
	AccessInstructions  string                 `json:"access_instructions"`
	
	// Status & Timeline
	Status              RequestStatus          `json:"status"`
	StatusHistory       []StatusUpdate         `json:"status_history"`
	
	// Assignment
	AssignedVendorID    *uuid.UUID             `json:"assigned_vendor_id,omitempty"`
	AssignedTechID      *uuid.UUID             `json:"assigned_tech_id,omitempty"`
	AssignmentHistory   []Assignment           `json:"assignment_history,omitempty"`
	
	// Response Tracking
	ResponseDeadline    time.Time              `json:"response_deadline"`
	ArrivalDeadline     time.Time              `json:"arrival_deadline"`
	ActualResponseTime  *time.Time             `json:"actual_response_time,omitempty"`
	ActualArrivalTime   *time.Time             `json:"actual_arrival_time,omitempty"`
	
	// Work Details
	DiagnosisNotes      string                 `json:"diagnosis_notes,omitempty"`
	WorkPerformed       string                 `json:"work_performed,omitempty"`
	PartsUsed           []PartUsed             `json:"parts_used,omitempty"`
	WorkPhotos          []MediaAttachment      `json:"work_photos,omitempty"`
	
	// Pricing
	EstimatedCost       *PriceEstimate         `json:"estimated_cost,omitempty"`
	FinalCost           *FinalPrice            `json:"final_cost,omitempty"`
	PaymentStatus       PaymentStatus          `json:"payment_status"`
	
	// Follow-up
	RequiresFollowUp    bool                   `json:"requires_follow_up"`
	FollowUpRequestID   *uuid.UUID             `json:"follow_up_request_id,omitempty"`
	FollowUpNotes       string                 `json:"follow_up_notes,omitempty"`
	
	// Customer Satisfaction
	Rating              *int                   `json:"rating,omitempty"`
	Review              string                 `json:"review,omitempty"`
	
	// Timestamps
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	CompletedAt         *time.Time             `json:"completed_at,omitempty"`
}

type EmergencyCategory string
const (
	CategoryPlumbing   EmergencyCategory = "plumbing"
	CategoryElectrical EmergencyCategory = "electrical"
	CategoryLocksmith  EmergencyCategory = "locksmith"
	CategoryHVAC       EmergencyCategory = "hvac"
	CategoryGlass      EmergencyCategory = "glass"
	CategoryRoofing    EmergencyCategory = "roofing"
	CategoryPest       EmergencyCategory = "pest"
	CategorySecurity   EmergencyCategory = "security"
	CategoryGeneral    EmergencyCategory = "general"
)

type UrgencyLevel string
const (
	UrgencyCritical  UrgencyLevel = "critical"   // < 30 min response
	UrgencyUrgent    UrgencyLevel = "urgent"     // < 2 hour response
	UrgencySameDay   UrgencyLevel = "same_day"   // < 6 hour response
	UrgencyScheduled UrgencyLevel = "scheduled"  // Planned repair
)

// Response time SLAs in minutes
var ResponseTimeSLA = map[UrgencyLevel]int{
	UrgencyCritical:  30,
	UrgencyUrgent:    120,
	UrgencySameDay:   360,
	UrgencyScheduled: 1440, // 24 hours
}

type RequestStatus string
const (
	StatusNew           RequestStatus = "new"
	StatusSearching     RequestStatus = "searching"      // Finding available tech
	StatusAssigned      RequestStatus = "assigned"       // Tech assigned
	StatusAccepted      RequestStatus = "accepted"       // Tech accepted
	StatusEnRoute       RequestStatus = "en_route"       // Tech on the way
	StatusArrived       RequestStatus = "arrived"        // Tech at location
	StatusDiagnosing    RequestStatus = "diagnosing"     // Assessing problem
	StatusQuoted        RequestStatus = "quoted"         // Estimate given
	StatusApproved      RequestStatus = "approved"       // Customer approved work
	StatusInProgress    RequestStatus = "in_progress"    // Work underway
	StatusCompleted     RequestStatus = "completed"      // Work finished
	StatusCancelled     RequestStatus = "cancelled"
	StatusNoShow        RequestStatus = "no_show"        // Tech didn't arrive
	StatusDisputed      RequestStatus = "disputed"
)

type PaymentStatus string
const (
	PaymentPending   PaymentStatus = "pending"
	PaymentHeld      PaymentStatus = "held"      // Pre-authorized
	PaymentCharged   PaymentStatus = "charged"
	PaymentRefunded  PaymentStatus = "refunded"
	PaymentDisputed  PaymentStatus = "disputed"
)

type EmergencyLocation struct {
	Address         string  `json:"address"`
	Unit            string  `json:"unit,omitempty"`
	City            string  `json:"city"`
	State           string  `json:"state"`
	PostalCode      string  `json:"postal_code"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	PropertyType    string  `json:"property_type"` // 'house', 'apartment', 'office', 'commercial'
	GateCode        string  `json:"gate_code,omitempty"`
	ParkingInfo     string  `json:"parking_info,omitempty"`
}

type MediaAttachment struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"` // 'photo', 'video', 'audio'
	URL         string    `json:"url"`
	ThumbnailURL string   `json:"thumbnail_url,omitempty"`
	Caption     string    `json:"caption,omitempty"`
	UploadedAt  time.Time `json:"uploaded_at"`
	UploadedBy  string    `json:"uploaded_by"` // 'customer', 'technician'
}

type StatusUpdate struct {
	Status     RequestStatus `json:"status"`
	Timestamp  time.Time     `json:"timestamp"`
	UpdatedBy  string        `json:"updated_by"` // 'system', 'customer', 'technician', 'support'
	Notes      string        `json:"notes,omitempty"`
	Location   *GeoPoint     `json:"location,omitempty"`
}

type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Assignment struct {
	VendorID    uuid.UUID  `json:"vendor_id"`
	TechID      uuid.UUID  `json:"tech_id"`
	AssignedAt  time.Time  `json:"assigned_at"`
	Response    string     `json:"response"` // 'accepted', 'declined', 'timeout'
	ResponseAt  *time.Time `json:"response_at,omitempty"`
	Reason      string     `json:"reason,omitempty"`
}

type PartUsed struct {
	Name        string  `json:"name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
	IsWarranty  bool    `json:"is_warranty"`
}

type PriceEstimate struct {
	LaborMin    float64 `json:"labor_min"`
	LaborMax    float64 `json:"labor_max"`
	PartsMin    float64 `json:"parts_min"`
	PartsMax    float64 `json:"parts_max"`
	TotalMin    float64 `json:"total_min"`
	TotalMax    float64 `json:"total_max"`
	Currency    string  `json:"currency"`
	ValidFor    int     `json:"valid_for_minutes"`
	Notes       string  `json:"notes,omitempty"`
}

type FinalPrice struct {
	CallOutFee      float64 `json:"call_out_fee"`
	LaborCost       float64 `json:"labor_cost"`
	LaborHours      float64 `json:"labor_hours"`
	PartsCost       float64 `json:"parts_cost"`
	EmergencyPremium float64 `json:"emergency_premium"`
	Subtotal        float64 `json:"subtotal"`
	Tax             float64 `json:"tax"`
	Discount        float64 `json:"discount"`
	Total           float64 `json:"total"`
	Currency        string  `json:"currency"`
}

// =============================================================================
// 2.2 EMERGENCY TECHNICIAN TYPES
// =============================================================================

// EmergencyTechnician represents a tech available for emergency calls
type EmergencyTechnician struct {
	ID                  uuid.UUID              `json:"id"`
	VendorID            uuid.UUID              `json:"vendor_id"`
	UserID              uuid.UUID              `json:"user_id"`
	
	// Profile
	Name                string                 `json:"name"`
	Photo               string                 `json:"photo_url"`
	Phone               string                 `json:"phone"`
	
	// Capabilities
	Categories          []EmergencyCategory    `json:"categories"`
	Certifications      []Certification        `json:"certifications"`
	EquipmentList       []string               `json:"equipment_list"`
	
	// Availability
	IsOnline            bool                   `json:"is_online"`
	CurrentStatus       TechStatus             `json:"current_status"`
	CurrentLocation     *GeoPoint              `json:"current_location,omitempty"`
	LastLocationUpdate  time.Time              `json:"last_location_update"`
	
	// Service Area
	ServiceRadius       float64                `json:"service_radius_km"`
	HomeBase            GeoPoint               `json:"home_base"`
	
	// Performance
	Rating              float64                `json:"rating"`
	CompletedJobs       int                    `json:"completed_jobs"`
	AcceptanceRate      float64                `json:"acceptance_rate"`
	AvgResponseTime     int                    `json:"avg_response_time_minutes"`
	AvgArrivalTime      int                    `json:"avg_arrival_time_minutes"`
	OnTimeRate          float64                `json:"on_time_rate"`
	
	// Current Assignment
	ActiveRequestID     *uuid.UUID             `json:"active_request_id,omitempty"`
	
	// Verification
	IsVerified          bool                   `json:"is_verified"`
	BackgroundChecked   bool                   `json:"background_checked"`
	InsuranceVerified   bool                   `json:"insurance_verified"`
	
	// Schedule
	WorkingHours        []WorkingHours         `json:"working_hours"`
	OnCallSchedule      []OnCallPeriod         `json:"on_call_schedule"`
}

type TechStatus string
const (
	TechAvailable    TechStatus = "available"
	TechBusy         TechStatus = "busy"
	TechEnRoute      TechStatus = "en_route"
	TechOnJob        TechStatus = "on_job"
	TechOffline      TechStatus = "offline"
	TechOnBreak      TechStatus = "on_break"
)

type Certification struct {
	Name        string    `json:"name"`
	Issuer      string    `json:"issuer"`
	Number      string    `json:"number"`
	IssuedDate  time.Time `json:"issued_date"`
	ExpiryDate  time.Time `json:"expiry_date"`
	Verified    bool      `json:"verified"`
}

type WorkingHours struct {
	DayOfWeek   int    `json:"day_of_week"` // 0 = Sunday
	StartTime   string `json:"start_time"`  // "08:00"
	EndTime     string `json:"end_time"`    // "18:00"
	IsEmergency bool   `json:"is_emergency"` // Accepts emergency calls
}

type OnCallPeriod struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Premium   float64   `json:"premium_percentage"` // Extra pay for on-call
}

// =============================================================================
// SECTION 3: DISPATCH ENGINE
// =============================================================================

// DispatchEngine handles emergency request assignment
type DispatchEngine struct {
	db               *pgxpool.Pool
	cache            *redis.Client
	geoService       *GeoService
	notificationSvc  *NotificationService
	pricingEngine    *EmergencyPricingEngine
	
	// Configuration
	config           *DispatchConfig
	
	// Active tracking
	activeTechs      map[uuid.UUID]*TechState
	activeRequests   map[uuid.UUID]*RequestState
	mu               sync.RWMutex
}

type DispatchConfig struct {
	MaxSearchRadius     float64 // km
	InitialSearchRadius float64 // km
	SearchExpansionStep float64 // km
	MaxAssignmentAttempts int
	AssignmentTimeout   time.Duration
	AutoEscalateAfter   time.Duration
}

type TechState struct {
	Tech           *EmergencyTechnician
	Location       GeoPoint
	LastUpdate     time.Time
	PendingRequest *uuid.UUID
}

type RequestState struct {
	Request          *EmergencyRequest
	AssignmentAttempts int
	CurrentSearchRadius float64
	LastAttemptAt    time.Time
}

// NewDispatchEngine creates a new dispatch engine
func NewDispatchEngine(db *pgxpool.Pool, cache *redis.Client) *DispatchEngine {
	return &DispatchEngine{
		db:    db,
		cache: cache,
		config: &DispatchConfig{
			MaxSearchRadius:     50.0,
			InitialSearchRadius: 5.0,
			SearchExpansionStep: 5.0,
			MaxAssignmentAttempts: 10,
			AssignmentTimeout:   2 * time.Minute,
			AutoEscalateAfter:   5 * time.Minute,
		},
		activeTechs:    make(map[uuid.UUID]*TechState),
		activeRequests: make(map[uuid.UUID]*RequestState),
	}
}

// DispatchResult represents the outcome of a dispatch attempt
type DispatchResult struct {
	Success         bool              `json:"success"`
	RequestID       uuid.UUID         `json:"request_id"`
	AssignedTechID  *uuid.UUID        `json:"assigned_tech_id,omitempty"`
	EstimatedArrival *time.Time       `json:"estimated_arrival,omitempty"`
	Message         string            `json:"message"`
	Alternatives    []TechCandidate   `json:"alternatives,omitempty"`
}

type TechCandidate struct {
	TechID          uuid.UUID `json:"tech_id"`
	TechName        string    `json:"tech_name"`
	Distance        float64   `json:"distance_km"`
	EstimatedArrival int      `json:"estimated_arrival_minutes"`
	Rating          float64   `json:"rating"`
	Price           float64   `json:"estimated_price"`
}

// Dispatch attempts to assign a technician to an emergency request
func (e *DispatchEngine) Dispatch(ctx context.Context, request *EmergencyRequest) (*DispatchResult, error) {
	result := &DispatchResult{
		RequestID: request.ID,
	}
	
	// Track request state
	e.mu.Lock()
	e.activeRequests[request.ID] = &RequestState{
		Request:             request,
		AssignmentAttempts:  0,
		CurrentSearchRadius: e.config.InitialSearchRadius,
	}
	e.mu.Unlock()
	
	// Update request status
	request.Status = StatusSearching
	e.updateRequestStatus(ctx, request, "system", "Searching for available technicians")
	
	// Find candidates
	candidates, err := e.findCandidates(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to find candidates: %w", err)
	}
	
	if len(candidates) == 0 {
		result.Success = false
		result.Message = "No technicians available in your area. We're expanding the search."
		
		// Expand search radius
		go e.expandedSearch(ctx, request)
		return result, nil
	}
	
	// Attempt assignment to best candidate
	for _, candidate := range candidates {
		assigned, err := e.attemptAssignment(ctx, request, candidate)
		if err != nil {
			continue
		}
		
		if assigned {
			eta := time.Now().Add(time.Duration(candidate.EstimatedArrival) * time.Minute)
			result.Success = true
			result.AssignedTechID = &candidate.TechID
			result.EstimatedArrival = &eta
			result.Message = fmt.Sprintf("%s is on the way! ETA: %d minutes", candidate.TechName, candidate.EstimatedArrival)
			
			// Store alternatives for customer visibility
			if len(candidates) > 1 {
				result.Alternatives = candidates[1:min(4, len(candidates))]
			}
			
			return result, nil
		}
	}
	
	// No one accepted, provide alternatives
	result.Success = false
	result.Message = "Finding available technicians..."
	result.Alternatives = candidates[:min(5, len(candidates))]
	
	// Continue searching in background
	go e.backgroundDispatch(ctx, request)
	
	return result, nil
}

func (e *DispatchEngine) findCandidates(ctx context.Context, request *EmergencyRequest) ([]TechCandidate, error) {
	e.mu.RLock()
	state := e.activeRequests[request.ID]
	searchRadius := state.CurrentSearchRadius
	e.mu.RUnlock()
	
	// Query available technicians within radius
	query := `
		SELECT 
			et.id,
			et.name,
			et.current_location,
			et.rating,
			et.avg_arrival_time_minutes,
			ST_Distance(
				et.current_location::geography,
				ST_MakePoint($2, $3)::geography
			) / 1000 as distance_km
		FROM emergency_technicians et
		WHERE et.is_online = TRUE
		  AND et.current_status = 'available'
		  AND $1 = ANY(et.categories)
		  AND et.is_verified = TRUE
		  AND ST_DWithin(
			  et.current_location::geography,
			  ST_MakePoint($2, $3)::geography,
			  $4 * 1000
		  )
		ORDER BY distance_km ASC, et.rating DESC
		LIMIT 20
	`
	
	rows, err := e.db.Query(ctx, query, 
		request.Category,
		request.Location.Longitude,
		request.Location.Latitude,
		searchRadius,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var candidates []TechCandidate
	for rows.Next() {
		var c TechCandidate
		var locationJSON []byte
		var avgArrival int
		
		if err := rows.Scan(&c.TechID, &c.TechName, &locationJSON, &c.Rating, &avgArrival, &c.Distance); err != nil {
			continue
		}
		
		// Calculate ETA based on distance and historical data
		c.EstimatedArrival = e.calculateETA(c.Distance, avgArrival)
		
		// Estimate price
		c.Price = e.pricingEngine.EstimatePrice(request.Category, request.Urgency, c.Distance)
		
		candidates = append(candidates, c)
	}
	
	// Sort by composite score (distance + rating + ETA)
	sort.Slice(candidates, func(i, j int) bool {
		scoreI := e.calculateCandidateScore(candidates[i], request.Urgency)
		scoreJ := e.calculateCandidateScore(candidates[j], request.Urgency)
		return scoreI > scoreJ
	})
	
	return candidates, nil
}

func (e *DispatchEngine) calculateETA(distance float64, avgArrival int) int {
	// Base: 2 minutes per km in traffic
	distanceMinutes := int(distance * 2)
	
	// Use historical average as a factor
	if avgArrival > 0 {
		return (distanceMinutes + avgArrival) / 2
	}
	
	return distanceMinutes + 5 // 5 min buffer
}

func (e *DispatchEngine) calculateCandidateScore(c TechCandidate, urgency UrgencyLevel) float64 {
	// Weights depend on urgency
	var distanceWeight, ratingWeight, etaWeight float64
	
	switch urgency {
	case UrgencyCritical:
		distanceWeight = 0.5
		ratingWeight = 0.1
		etaWeight = 0.4
	case UrgencyUrgent:
		distanceWeight = 0.4
		ratingWeight = 0.2
		etaWeight = 0.4
	default:
		distanceWeight = 0.3
		ratingWeight = 0.4
		etaWeight = 0.3
	}
	
	// Normalize scores (inverse for distance and ETA - lower is better)
	distanceScore := 1.0 / (1.0 + c.Distance/10.0)
	etaScore := 1.0 / (1.0 + float64(c.EstimatedArrival)/30.0)
	ratingScore := c.Rating / 5.0
	
	return distanceScore*distanceWeight + etaScore*etaWeight + ratingScore*ratingWeight
}

func (e *DispatchEngine) attemptAssignment(ctx context.Context, request *EmergencyRequest, candidate TechCandidate) (bool, error) {
	// Record assignment attempt
	e.mu.Lock()
	state := e.activeRequests[request.ID]
	state.AssignmentAttempts++
	state.LastAttemptAt = time.Now()
	e.mu.Unlock()
	
	// Update request
	request.AssignedTechID = &candidate.TechID
	request.Status = StatusAssigned
	request.AssignmentHistory = append(request.AssignmentHistory, Assignment{
		TechID:     candidate.TechID,
		AssignedAt: time.Now(),
		Response:   "pending",
	})
	e.updateRequestStatus(ctx, request, "system", fmt.Sprintf("Assigned to %s", candidate.TechName))
	
	// Notify technician
	notification := &TechNotification{
		Type:      "new_emergency",
		RequestID: request.ID,
		Category:  request.Category,
		Urgency:   request.Urgency,
		Distance:  candidate.Distance,
		Address:   request.Location.Address,
		Price:     candidate.Price,
		ExpiresAt: time.Now().Add(e.config.AssignmentTimeout),
	}
	
	e.notificationSvc.NotifyTechnician(ctx, candidate.TechID, notification)
	
	// Wait for response with timeout
	accepted := e.waitForTechResponse(ctx, request.ID, candidate.TechID, e.config.AssignmentTimeout)
	
	if accepted {
		// Update assignment as accepted
		for i := range request.AssignmentHistory {
			if request.AssignmentHistory[i].TechID == candidate.TechID {
				now := time.Now()
				request.AssignmentHistory[i].Response = "accepted"
				request.AssignmentHistory[i].ResponseAt = &now
			}
		}
		request.Status = StatusAccepted
		e.updateRequestStatus(ctx, request, "technician", "Technician accepted the request")
		
		return true, nil
	}
	
	// Tech didn't accept, mark and try next
	for i := range request.AssignmentHistory {
		if request.AssignmentHistory[i].TechID == candidate.TechID {
			now := time.Now()
			request.AssignmentHistory[i].Response = "timeout"
			request.AssignmentHistory[i].ResponseAt = &now
		}
	}
	
	return false, nil
}

func (e *DispatchEngine) waitForTechResponse(ctx context.Context, requestID, techID uuid.UUID, timeout time.Duration) bool {
	// In production, this would use a pub/sub mechanism
	// For now, poll the database
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if time.Now().After(deadline) {
				return false
			}
			
			// Check if tech accepted
			var response string
			e.db.QueryRow(ctx, `
				SELECT ah.response 
				FROM emergency_requests er,
				     jsonb_array_elements(er.assignment_history) ah
				WHERE er.id = $1 
				  AND (ah->>'tech_id')::uuid = $2
				ORDER BY (ah->>'assigned_at')::timestamp DESC
				LIMIT 1
			`, requestID, techID).Scan(&response)
			
			if response == "accepted" {
				return true
			} else if response == "declined" {
				return false
			}
		}
	}
}

func (e *DispatchEngine) expandedSearch(ctx context.Context, request *EmergencyRequest) {
	e.mu.Lock()
	state := e.activeRequests[request.ID]
	state.CurrentSearchRadius += e.config.SearchExpansionStep
	e.mu.Unlock()
	
	if state.CurrentSearchRadius <= e.config.MaxSearchRadius {
		// Retry dispatch with expanded radius
		e.Dispatch(ctx, request)
	} else {
		// Max radius reached, escalate
		e.escalateRequest(ctx, request)
	}
}

func (e *DispatchEngine) backgroundDispatch(ctx context.Context, request *EmergencyRequest) {
	for {
		e.mu.RLock()
		state := e.activeRequests[request.ID]
		if state == nil || request.Status == StatusAccepted || request.Status == StatusCancelled {
			e.mu.RUnlock()
			return
		}
		e.mu.RUnlock()
		
		time.Sleep(30 * time.Second)
		
		// Check if still needs assignment
		if state.AssignmentAttempts >= e.config.MaxAssignmentAttempts {
			e.escalateRequest(ctx, request)
			return
		}
		
		// Expand radius and retry
		e.expandedSearch(ctx, request)
	}
}

func (e *DispatchEngine) escalateRequest(ctx context.Context, request *EmergencyRequest) {
	// Notify support team
	e.notificationSvc.NotifySupport(ctx, &SupportAlert{
		Type:      "dispatch_failure",
		RequestID: request.ID,
		Message:   "Unable to find available technician after maximum attempts",
		Priority:  "high",
	})
	
	// Notify customer
	e.notificationSvc.NotifyCustomer(ctx, request.UserID, &CustomerNotification{
		Type:    "dispatch_escalated",
		Title:   "We're working on it",
		Message: "We're having difficulty finding an available technician. Our support team has been notified and will contact you shortly.",
	})
}

func (e *DispatchEngine) updateRequestStatus(ctx context.Context, request *EmergencyRequest, updatedBy, notes string) {
	update := StatusUpdate{
		Status:    request.Status,
		Timestamp: time.Now(),
		UpdatedBy: updatedBy,
		Notes:     notes,
	}
	request.StatusHistory = append(request.StatusHistory, update)
	request.UpdatedAt = time.Now()
	
	// Persist to database
	historyJSON, _ := json.Marshal(request.StatusHistory)
	assignmentJSON, _ := json.Marshal(request.AssignmentHistory)
	
	e.db.Exec(ctx, `
		UPDATE emergency_requests 
		SET status = $2, 
		    status_history = $3, 
		    assignment_history = $4,
		    assigned_tech_id = $5,
		    updated_at = $6
		WHERE id = $1
	`, request.ID, request.Status, historyJSON, assignmentJSON, request.AssignedTechID, request.UpdatedAt)
}

// =============================================================================
// SECTION 4: REAL-TIME TRACKING
// =============================================================================

// TrackingService provides real-time location tracking
type TrackingService struct {
	db       *pgxpool.Pool
	cache    *redis.Client
	pubsub   *PubSubService
}

// TechLocationUpdate from mobile app
type TechLocationUpdate struct {
	TechID    uuid.UUID `json:"tech_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Accuracy  float64   `json:"accuracy_meters"`
	Speed     float64   `json:"speed_kmh"`
	Heading   float64   `json:"heading_degrees"`
	Timestamp time.Time `json:"timestamp"`
}

// TrackingUpdate sent to customer
type TrackingUpdate struct {
	RequestID        uuid.UUID `json:"request_id"`
	TechID           uuid.UUID `json:"tech_id"`
	TechName         string    `json:"tech_name"`
	TechPhoto        string    `json:"tech_photo"`
	CurrentLocation  GeoPoint  `json:"current_location"`
	DistanceRemaining float64  `json:"distance_remaining_km"`
	ETAMinutes       int       `json:"eta_minutes"`
	Status           string    `json:"status"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// UpdateTechLocation processes a location update from a technician
func (s *TrackingService) UpdateTechLocation(ctx context.Context, update TechLocationUpdate) error {
	// Update tech's current location in database
	_, err := s.db.Exec(ctx, `
		UPDATE emergency_technicians
		SET current_location = ST_MakePoint($2, $3),
		    last_location_update = $4
		WHERE id = $1
	`, update.TechID, update.Longitude, update.Latitude, update.Timestamp)
	
	if err != nil {
		return err
	}
	
	// Cache for real-time access
	locationJSON, _ := json.Marshal(update)
	s.cache.Set(ctx, fmt.Sprintf("tech:location:%s", update.TechID), locationJSON, 5*time.Minute)
	
	// Check if tech has an active request
	var requestID uuid.UUID
	var customerUserID uuid.UUID
	var destLat, destLng float64
	
	err = s.db.QueryRow(ctx, `
		SELECT er.id, er.user_id, er.location->>'latitude', er.location->>'longitude'
		FROM emergency_requests er
		JOIN emergency_technicians et ON et.id = er.assigned_tech_id
		WHERE et.id = $1 AND er.status IN ('accepted', 'en_route')
	`, update.TechID).Scan(&requestID, &customerUserID, &destLat, &destLng)
	
	if err != nil {
		// No active request
		return nil
	}
	
	// Calculate distance remaining
	distance := s.calculateDistance(update.Latitude, update.Longitude, destLat, destLng)
	
	// Calculate ETA based on speed and distance
	eta := s.calculateETA(distance, update.Speed)
	
	// Get tech info
	var techName, techPhoto string
	s.db.QueryRow(ctx, `SELECT name, photo FROM emergency_technicians WHERE id = $1`, update.TechID).Scan(&techName, &techPhoto)
	
	// Create tracking update for customer
	trackingUpdate := TrackingUpdate{
		RequestID:         requestID,
		TechID:            update.TechID,
		TechName:          techName,
		TechPhoto:         techPhoto,
		CurrentLocation:   GeoPoint{Latitude: update.Latitude, Longitude: update.Longitude},
		DistanceRemaining: distance,
		ETAMinutes:        eta,
		Status:            "en_route",
		UpdatedAt:         time.Now(),
	}
	
	// Publish to customer's channel
	s.pubsub.Publish(ctx, fmt.Sprintf("tracking:%s", requestID), trackingUpdate)
	
	// Check for arrival
	if distance < 0.1 { // Within 100 meters
		s.handleArrival(ctx, requestID, update.TechID)
	}
	
	return nil
}

func (s *TrackingService) calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Haversine formula
	const R = 6371 // Earth's radius in km
	
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return R * c
}

func (s *TrackingService) calculateETA(distance, speed float64) int {
	if speed < 5 {
		speed = 30 // Default average speed in city
	}
	
	hours := distance / speed
	minutes := int(hours * 60)
	
	// Add buffer for traffic, parking
	minutes += 3
	
	return minutes
}

func (s *TrackingService) handleArrival(ctx context.Context, requestID, techID uuid.UUID) {
	// Update request status
	now := time.Now()
	s.db.Exec(ctx, `
		UPDATE emergency_requests 
		SET status = 'arrived', 
		    actual_arrival_time = $2,
		    updated_at = $2
		WHERE id = $1
	`, requestID, now)
	
	// Notify customer
	var customerUserID uuid.UUID
	s.db.QueryRow(ctx, `SELECT user_id FROM emergency_requests WHERE id = $1`, requestID).Scan(&customerUserID)
	
	// Publish arrival event
	s.pubsub.Publish(ctx, fmt.Sprintf("tracking:%s", requestID), TrackingUpdate{
		RequestID: requestID,
		TechID:    techID,
		Status:    "arrived",
		UpdatedAt: now,
	})
}

// SubscribeToTracking allows customer to get real-time updates
func (s *TrackingService) SubscribeToTracking(ctx context.Context, requestID uuid.UUID) (<-chan TrackingUpdate, error) {
	return s.pubsub.Subscribe(ctx, fmt.Sprintf("tracking:%s", requestID))
}

// =============================================================================
// SECTION 5: EMERGENCY PRICING ENGINE
// =============================================================================

// EmergencyPricingEngine calculates emergency service pricing
type EmergencyPricingEngine struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// PricingRules for different scenarios
type PricingRules struct {
	Category         EmergencyCategory
	
	// Base Fees
	CallOutFee       float64 // Fixed fee just for showing up
	MinimumCharge    float64 // Minimum job charge
	
	// Labor Rates
	StandardRate     float64 // Per hour, business hours
	AfterHoursRate   float64 // Per hour, nights/weekends
	HolidayRate      float64 // Per hour, holidays
	
	// Urgency Premiums (percentage)
	CriticalPremium  float64
	UrgentPremium    float64
	
	// Distance
	FreeDistanceKM   float64
	PerKMCharge      float64
}

// Default pricing rules
var DefaultPricingRules = map[EmergencyCategory]PricingRules{
	CategoryPlumbing: {
		Category:        CategoryPlumbing,
		CallOutFee:      15000,
		MinimumCharge:   25000,
		StandardRate:    10000,
		AfterHoursRate:  15000,
		HolidayRate:     20000,
		CriticalPremium: 50,
		UrgentPremium:   25,
		FreeDistanceKM:  5,
		PerKMCharge:     500,
	},
	CategoryElectrical: {
		Category:        CategoryElectrical,
		CallOutFee:      15000,
		MinimumCharge:   25000,
		StandardRate:    12000,
		AfterHoursRate:  18000,
		HolidayRate:     24000,
		CriticalPremium: 50,
		UrgentPremium:   25,
		FreeDistanceKM:  5,
		PerKMCharge:     500,
	},
	CategoryLocksmith: {
		Category:        CategoryLocksmith,
		CallOutFee:      10000,
		MinimumCharge:   15000,
		StandardRate:    8000,
		AfterHoursRate:  12000,
		HolidayRate:     16000,
		CriticalPremium: 30,
		UrgentPremium:   15,
		FreeDistanceKM:  5,
		PerKMCharge:     400,
	},
	CategoryHVAC: {
		Category:        CategoryHVAC,
		CallOutFee:      20000,
		MinimumCharge:   30000,
		StandardRate:    15000,
		AfterHoursRate:  22000,
		HolidayRate:     30000,
		CriticalPremium: 40,
		UrgentPremium:   20,
		FreeDistanceKM:  5,
		PerKMCharge:     600,
	},
	CategoryGlass: {
		Category:        CategoryGlass,
		CallOutFee:      15000,
		MinimumCharge:   25000,
		StandardRate:    10000,
		AfterHoursRate:  15000,
		HolidayRate:     20000,
		CriticalPremium: 30,
		UrgentPremium:   15,
		FreeDistanceKM:  5,
		PerKMCharge:     500,
	},
	CategoryRoofing: {
		Category:        CategoryRoofing,
		CallOutFee:      25000,
		MinimumCharge:   40000,
		StandardRate:    15000,
		AfterHoursRate:  22000,
		HolidayRate:     30000,
		CriticalPremium: 50,
		UrgentPremium:   25,
		FreeDistanceKM:  10,
		PerKMCharge:     700,
	},
	CategoryPest: {
		Category:        CategoryPest,
		CallOutFee:      12000,
		MinimumCharge:   20000,
		StandardRate:    8000,
		AfterHoursRate:  12000,
		HolidayRate:     16000,
		CriticalPremium: 40, // Higher for dangerous pests
		UrgentPremium:   20,
		FreeDistanceKM:  5,
		PerKMCharge:     400,
	},
	CategorySecurity: {
		Category:        CategorySecurity,
		CallOutFee:      15000,
		MinimumCharge:   25000,
		StandardRate:    12000,
		AfterHoursRate:  18000,
		HolidayRate:     24000,
		CriticalPremium: 50,
		UrgentPremium:   25,
		FreeDistanceKM:  5,
		PerKMCharge:     500,
	},
}

// EstimatePrice estimates the price for an emergency service
func (e *EmergencyPricingEngine) EstimatePrice(category EmergencyCategory, urgency UrgencyLevel, distance float64) float64 {
	rules, ok := DefaultPricingRules[category]
	if !ok {
		rules = DefaultPricingRules[CategoryGeneral]
	}
	
	// Start with call-out fee
	price := rules.CallOutFee
	
	// Add labor estimate (assume 1 hour average)
	laborRate := e.getLaborRate(rules)
	price += laborRate
	
	// Add urgency premium
	switch urgency {
	case UrgencyCritical:
		price *= (1 + rules.CriticalPremium/100)
	case UrgencyUrgent:
		price *= (1 + rules.UrgentPremium/100)
	}
	
	// Add distance charge
	if distance > rules.FreeDistanceKM {
		extraKM := distance - rules.FreeDistanceKM
		price += extraKM * rules.PerKMCharge
	}
	
	// Ensure minimum charge
	if price < rules.MinimumCharge {
		price = rules.MinimumCharge
	}
	
	return price
}

func (e *EmergencyPricingEngine) getLaborRate(rules PricingRules) float64 {
	now := time.Now()
	hour := now.Hour()
	weekday := now.Weekday()
	
	// Check if holiday (would need holiday calendar)
	// isHoliday := e.isHoliday(now)
	
	// After hours: before 8 AM, after 6 PM, or weekends
	if hour < 8 || hour >= 18 || weekday == time.Saturday || weekday == time.Sunday {
		return rules.AfterHoursRate
	}
	
	return rules.StandardRate
}

// CalculateFinalPrice calculates the final price after work is done
func (e *EmergencyPricingEngine) CalculateFinalPrice(
	category EmergencyCategory,
	urgency UrgencyLevel,
	laborHours float64,
	parts []PartUsed,
	distance float64,
	discountCode string,
) *FinalPrice {
	rules, ok := DefaultPricingRules[category]
	if !ok {
		rules = PricingRules{
			CallOutFee:     15000,
			StandardRate:   10000,
			AfterHoursRate: 15000,
		}
	}
	
	final := &FinalPrice{
		Currency: "NGN",
	}
	
	// Call-out fee
	final.CallOutFee = rules.CallOutFee
	
	// Labor
	laborRate := e.getLaborRate(rules)
	final.LaborHours = laborHours
	final.LaborCost = laborRate * laborHours
	
	// Parts
	for _, part := range parts {
		if !part.IsWarranty {
			final.PartsCost += part.TotalPrice
		}
	}
	
	// Emergency premium
	switch urgency {
	case UrgencyCritical:
		final.EmergencyPremium = (final.CallOutFee + final.LaborCost) * (rules.CriticalPremium / 100)
	case UrgencyUrgent:
		final.EmergencyPremium = (final.CallOutFee + final.LaborCost) * (rules.UrgentPremium / 100)
	}
	
	// Subtotal
	final.Subtotal = final.CallOutFee + final.LaborCost + final.PartsCost + final.EmergencyPremium
	
	// Discount
	if discountCode != "" {
		final.Discount = e.applyDiscount(final.Subtotal, discountCode)
	}
	
	// Tax (VAT 7.5% in Nigeria)
	final.Tax = (final.Subtotal - final.Discount) * 0.075
	
	// Total
	final.Total = final.Subtotal - final.Discount + final.Tax
	
	return final
}

func (e *EmergencyPricingEngine) applyDiscount(subtotal float64, code string) float64 {
	// Look up discount code
	// For now, return 0
	return 0
}

// =============================================================================
// SECTION 6: API HANDLERS
// =============================================================================

// HomeRescueAPI provides the REST API
type HomeRescueAPI struct {
	db              *pgxpool.Pool
	cache           *redis.Client
	dispatchEngine  *DispatchEngine
	trackingService *TrackingService
	pricingEngine   *EmergencyPricingEngine
}

// CreateEmergencyRequest for new emergency
type CreateEmergencyRequest struct {
	Category           EmergencyCategory `json:"category"`
	Subcategory        string            `json:"subcategory,omitempty"`
	Description        string            `json:"description"`
	Location           EmergencyLocation `json:"location"`
	AccessInstructions string            `json:"access_instructions,omitempty"`
	Photos             []string          `json:"photo_urls,omitempty"`
	ContactPhone       string            `json:"contact_phone"`
}

// CreateEmergency handles emergency creation
func (api *HomeRescueAPI) CreateEmergency(ctx context.Context, userID uuid.UUID, req CreateEmergencyRequest) (*EmergencyRequest, error) {
	// Determine urgency based on category and description
	urgency := api.determineUrgency(req.Category, req.Description)
	
	emergency := &EmergencyRequest{
		ID:                 uuid.New(),
		UserID:             userID,
		Category:           req.Category,
		Subcategory:        req.Subcategory,
		Urgency:            urgency,
		Description:        req.Description,
		Location:           req.Location,
		AccessInstructions: req.AccessInstructions,
		Status:             StatusNew,
		StatusHistory: []StatusUpdate{
			{Status: StatusNew, Timestamp: time.Now(), UpdatedBy: "customer"},
		},
		ResponseDeadline:   time.Now().Add(time.Duration(ResponseTimeSLA[urgency]) * time.Minute / 2),
		ArrivalDeadline:    time.Now().Add(time.Duration(ResponseTimeSLA[urgency]) * time.Minute),
		PaymentStatus:      PaymentPending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	
	// Add photos
	for _, url := range req.Photos {
		emergency.Photos = append(emergency.Photos, MediaAttachment{
			ID:         uuid.New(),
			Type:       "photo",
			URL:        url,
			UploadedAt: time.Now(),
			UploadedBy: "customer",
		})
	}
	
	// Save to database
	if err := api.saveEmergency(ctx, emergency); err != nil {
		return nil, err
	}
	
	// Immediately dispatch
	go api.dispatchEngine.Dispatch(ctx, emergency)
	
	return emergency, nil
}

func (api *HomeRescueAPI) determineUrgency(category EmergencyCategory, description string) UrgencyLevel {
	// Keywords that indicate critical urgency
	criticalKeywords := []string{
		"flood", "flooding", "burst", "fire", "smoke", "gas leak", "sparking",
		"no power", "break-in", "broken into", "locked out", "child", "baby",
		"elderly", "disabled", "medical", "emergency",
	}
	
	urgentKeywords := []string{
		"leak", "leaking", "not working", "broken", "stuck", "won't open",
		"no water", "no heat", "no cooling", "pest", "rats", "mice",
	}
	
	descLower := strings.ToLower(description)
	
	for _, kw := range criticalKeywords {
		if strings.Contains(descLower, kw) {
			return UrgencyCritical
		}
	}
	
	for _, kw := range urgentKeywords {
		if strings.Contains(descLower, kw) {
			return UrgencyUrgent
		}
	}
	
	// Category defaults
	switch category {
	case CategorySecurity, CategoryGlass:
		return UrgencyUrgent
	default:
		return UrgencySameDay
	}
}

func (api *HomeRescueAPI) saveEmergency(ctx context.Context, e *EmergencyRequest) error {
	photosJSON, _ := json.Marshal(e.Photos)
	historyJSON, _ := json.Marshal(e.StatusHistory)
	locationJSON, _ := json.Marshal(e.Location)
	
	query := `
		INSERT INTO emergency_requests (
			id, user_id, category, subcategory, urgency,
			title, description, photos, location, access_instructions,
			status, status_history,
			response_deadline, arrival_deadline,
			payment_status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	
	_, err := api.db.Exec(ctx, query,
		e.ID, e.UserID, e.Category, e.Subcategory, e.Urgency,
		e.Title, e.Description, photosJSON, locationJSON, e.AccessInstructions,
		e.Status, historyJSON,
		e.ResponseDeadline, e.ArrivalDeadline,
		e.PaymentStatus, e.CreatedAt, e.UpdatedAt,
	)
	
	return err
}

// GetEmergencyStatus returns current status with tracking info
func (api *HomeRescueAPI) GetEmergencyStatus(ctx context.Context, requestID uuid.UUID) (*EmergencyStatusResponse, error) {
	// Load emergency
	emergency, err := api.loadEmergency(ctx, requestID)
	if err != nil {
		return nil, err
	}
	
	response := &EmergencyStatusResponse{
		RequestID: requestID,
		Status:    emergency.Status,
		Urgency:   emergency.Urgency,
		Category:  emergency.Category,
		CreatedAt: emergency.CreatedAt,
	}
	
	// Add tech info if assigned
	if emergency.AssignedTechID != nil {
		tech, _ := api.loadTech(ctx, *emergency.AssignedTechID)
		if tech != nil {
			response.AssignedTech = &TechInfo{
				ID:       tech.ID,
				Name:     tech.Name,
				Photo:    tech.Photo,
				Phone:    tech.Phone,
				Rating:   tech.Rating,
				JobCount: tech.CompletedJobs,
			}
			
			// Add real-time tracking if en route
			if emergency.Status == StatusEnRoute || emergency.Status == StatusAccepted {
				tracking, _ := api.getLatestTracking(ctx, requestID)
				response.Tracking = tracking
			}
		}
	}
	
	// Add pricing if quoted
	if emergency.EstimatedCost != nil {
		response.Estimate = emergency.EstimatedCost
	}
	
	// Add final price if completed
	if emergency.FinalCost != nil {
		response.FinalPrice = emergency.FinalCost
	}
	
	return response, nil
}

type EmergencyStatusResponse struct {
	RequestID     uuid.UUID         `json:"request_id"`
	Status        RequestStatus     `json:"status"`
	Urgency       UrgencyLevel      `json:"urgency"`
	Category      EmergencyCategory `json:"category"`
	AssignedTech  *TechInfo         `json:"assigned_tech,omitempty"`
	Tracking      *TrackingUpdate   `json:"tracking,omitempty"`
	Estimate      *PriceEstimate    `json:"estimate,omitempty"`
	FinalPrice    *FinalPrice       `json:"final_price,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

type TechInfo struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Photo    string    `json:"photo"`
	Phone    string    `json:"phone"`
	Rating   float64   `json:"rating"`
	JobCount int       `json:"job_count"`
}

func (api *HomeRescueAPI) loadEmergency(ctx context.Context, requestID uuid.UUID) (*EmergencyRequest, error) {
	// Implementation would load from database
	return nil, nil
}

func (api *HomeRescueAPI) loadTech(ctx context.Context, techID uuid.UUID) (*EmergencyTechnician, error) {
	// Implementation would load from database
	return nil, nil
}

func (api *HomeRescueAPI) getLatestTracking(ctx context.Context, requestID uuid.UUID) (*TrackingUpdate, error) {
	// Get from cache or database
	return nil, nil
}

/*
================================================================================
SECTION 7: BUSINESS MODEL
================================================================================

REVENUE STREAMS:

1. SERVICE FEES (Primary Revenue - 70%)
   
   Platform Fee: 15-20% of total job value
   - Standard: 15%
   - Critical Emergency: 18%
   - After-Hours: 20%
   
   Example:
   - ₦50,000 plumbing job → ₦7,500-10,000 platform fee

2. SUBSCRIPTION TIERS - CUSTOMERS (15%)

   FREE:
   - Pay per emergency
   - Standard response times
   - Basic tracking

   HOMERESCUE+ (₦5,000/month):
   - Priority dispatch (jump the queue)
   - 10% discount on all services
   - Free annual home inspection
   - 24/7 dedicated support line
   - Extended warranty on repairs

   HOMERESCUE FAMILY (₦10,000/month):
   - All Premium features
   - Cover up to 3 properties
   - ₦100,000/year emergency fund
   - Concierge service

3. SUBSCRIPTION TIERS - TECHNICIANS (10%)

   BASIC (Free):
   - Receive emergency requests
   - 20% platform fee
   - Basic support

   PRO (₦20,000/month):
   - Priority in dispatch algorithm
   - 15% platform fee
   - Advanced scheduling tools
   - Marketing support
   - Training resources

   ENTERPRISE (₦50,000/month):
   - For service companies
   - Multiple technician accounts
   - 12% platform fee
   - Dedicated account manager
   - API access
   - White-label options

4. INSURANCE PARTNERSHIPS (5%)
   - Preferred vendor network for insurers
   - Direct billing to insurance
   - Fraud prevention services
   - Claims documentation

5. PROPERTY MANAGEMENT CONTRACTS
   - Bulk pricing for property managers
   - SLA-based contracts
   - Monthly retainer + per-incident

KEY METRICS:
- Response Time (actual vs SLA)
- Customer Satisfaction (NPS)
- Technician Utilization Rate
- Job Completion Rate
- Repeat Customer Rate
- Average Job Value
- Platform Fee Revenue per Job

================================================================================
SECTION 8: SLA GUARANTEES
================================================================================

RESPONSE TIME GUARANTEE:
- Critical: Tech responds within 30 minutes or 100% refund of call-out fee
- Urgent: Tech responds within 2 hours or 50% refund
- Same-Day: Tech arrives same day or 25% discount

QUALITY GUARANTEE:
- 30-day warranty on all repairs
- Re-do at no extra cost if issue recurs
- Money-back guarantee if not satisfied

PRICE GUARANTEE:
- Final price within 20% of estimate
- If higher, customer can decline additional work
- Price match if customer finds lower quote

================================================================================
*/

// Placeholder services
type GeoService struct{}
type NotificationService struct{}
type PubSubService struct{}

type TechNotification struct {
	Type      string
	RequestID uuid.UUID
	Category  EmergencyCategory
	Urgency   UrgencyLevel
	Distance  float64
	Address   string
	Price     float64
	ExpiresAt time.Time
}

type SupportAlert struct {
	Type      string
	RequestID uuid.UUID
	Message   string
	Priority  string
}

type CustomerNotification struct {
	Type    string
	Title   string
	Message string
}

func (n *NotificationService) NotifyTechnician(ctx context.Context, techID uuid.UUID, notification *TechNotification) {}
func (n *NotificationService) NotifySupport(ctx context.Context, alert *SupportAlert) {}
func (n *NotificationService) NotifyCustomer(ctx context.Context, userID uuid.UUID, notification *CustomerNotification) {}

func (p *PubSubService) Publish(ctx context.Context, channel string, data interface{}) {}
func (p *PubSubService) Subscribe(ctx context.Context, channel string) (<-chan TrackingUpdate, error) {
	return nil, nil
}

// Helper
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// strings package for keyword matching
import "strings"
