// =============================================================================
// LIFEOS - INTELLIGENT LIFE EVENT ORCHESTRATION PLATFORM
// Comprehensive Technical & Business Specification
// Version: 1.0.0
// =============================================================================

package lifeos

import (
	"context"
	"encoding/json"
	"fmt"
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

LIFEOS: Your Life's Operating System

VISION:
LifeOS transforms how people navigate life's significant moments by anticipating
needs, orchestrating services, and removing friction from complex multi-vendor
coordination. It's not just a marketplace—it's an intelligent life companion
that understands context, predicts requirements, and manages the entire journey.

CORE VALUE PROPOSITION:
"When life happens, LifeOS handles it."

TARGET SEGMENTS:
1. Primary: Urban professionals (25-45) experiencing major life transitions
2. Secondary: Parents managing family milestones and household needs
3. Tertiary: Small business owners handling company events

KEY DIFFERENTIATORS:
1. Predictive Intelligence: Detects life events before explicit user action
2. Full-Stack Orchestration: Manages entire service cascade, not just discovery
3. Single Point of Accountability: One platform owns the entire experience
4. Contextual Commerce: Right services at the right time with bundled pricing
5. Proactive vs Reactive: Anticipates needs rather than waiting for searches

PRODUCT PRINCIPLES:
1. Invisible Complexity: Hide orchestration complexity from users
2. Anticipatory Design: Predict before being asked
3. Graceful Degradation: Work even with partial information
4. Trust Through Transparency: Show reasoning, not just results
5. Human-in-the-Loop: AI assists, humans decide

================================================================================
SECTION 2: CORE PLATFORM ARCHITECTURE
================================================================================
*/

// =============================================================================
// 2.1 CORE DOMAIN TYPES
// =============================================================================

// LifeEvent represents a detected or declared life event
type LifeEvent struct {
	ID                uuid.UUID              `json:"id"`
	UserID            uuid.UUID              `json:"user_id"`
	
	// Event Classification
	EventType         EventType              `json:"event_type"`
	EventSubtype      string                 `json:"event_subtype,omitempty"`
	ClusterType       ClusterType            `json:"cluster_type"`
	
	// Timing
	DetectedAt        time.Time              `json:"detected_at"`
	EventDate         *time.Time             `json:"event_date,omitempty"`
	EventDateFlex     DateFlexibility        `json:"event_date_flexibility"`
	PlanningHorizon   int                    `json:"planning_horizon_days"`
	
	// Detection
	DetectionMethod   DetectionMethod        `json:"detection_method"`
	DetectionConfidence float64              `json:"detection_confidence"`
	DetectionSignals  []DetectionSignal      `json:"detection_signals"`
	
	// Event Details
	Scale             EventScale             `json:"scale"`
	GuestCount        *int                   `json:"guest_count,omitempty"`
	Location          *Location              `json:"location,omitempty"`
	Budget            *Budget                `json:"budget,omitempty"`
	
	// Orchestration State
	Status            EventStatus            `json:"status"`
	Phase             EventPhase             `json:"phase"`
	CompletionPct     float64                `json:"completion_percentage"`
	
	// Service Graph
	RequiredServices  []RequiredService      `json:"required_services"`
	BookedServices    []BookedService        `json:"booked_services"`
	SuggestedBundles  []SuggestedBundle      `json:"suggested_bundles"`
	
	// User Preferences
	Preferences       EventPreferences       `json:"preferences"`
	Constraints       []Constraint           `json:"constraints"`
	
	// Metadata
	CustomAttributes  map[string]interface{} `json:"custom_attributes,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	
	// Timestamps
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	ConfirmedAt       *time.Time             `json:"confirmed_at,omitempty"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
}

type EventType string
const (
	EventTypeWedding        EventType = "wedding"
	EventTypeFuneral        EventType = "funeral"
	EventTypeBirthday       EventType = "birthday"
	EventTypeRelocation     EventType = "relocation"
	EventTypeRenovation     EventType = "renovation"
	EventTypeChildbirth     EventType = "childbirth"
	EventTypeTravel         EventType = "travel"
	EventTypeBusinessLaunch EventType = "business_launch"
	EventTypeGraduation     EventType = "graduation"
	EventTypeRetirement     EventType = "retirement"
)

type ClusterType string
const (
	ClusterCelebrations ClusterType = "celebrations"
	ClusterHome         ClusterType = "home"
	ClusterTravel       ClusterType = "travel"
	ClusterHealth       ClusterType = "health"
	ClusterBusiness     ClusterType = "business"
	ClusterEducation    ClusterType = "education"
)

type DetectionMethod string
const (
	DetectionExplicit      DetectionMethod = "explicit"       // User declared
	DetectionBehavioral    DetectionMethod = "behavioral"     // Search/browse patterns
	DetectionCalendar      DetectionMethod = "calendar"       // Calendar integration
	DetectionSocial        DetectionMethod = "social"         // Social signals
	DetectionTransactional DetectionMethod = "transactional"  // Purchase patterns
	DetectionPartner       DetectionMethod = "partner"        // Partner data (with consent)
)

type EventStatus string
const (
	StatusDetected   EventStatus = "detected"    // System detected, not confirmed
	StatusConfirmed  EventStatus = "confirmed"   // User confirmed the event
	StatusPlanning   EventStatus = "planning"    // Actively planning
	StatusBooked     EventStatus = "booked"      // All critical services booked
	StatusInProgress EventStatus = "in_progress" // Event is happening
	StatusCompleted  EventStatus = "completed"   // Event finished
	StatusCancelled  EventStatus = "cancelled"   // User cancelled
)

type EventPhase string
const (
	PhaseDiscovery    EventPhase = "discovery"    // Understanding requirements
	PhasePlanning     EventPhase = "planning"     // Building service list
	PhaseVendorSelect EventPhase = "vendor_select"// Choosing vendors
	PhaseBooking      EventPhase = "booking"      // Confirming bookings
	PhasePreEvent     EventPhase = "pre_event"    // Final preparations
	PhaseEventDay     EventPhase = "event_day"    // Day of event
	PhasePostEvent    EventPhase = "post_event"   // Follow-up actions
)

type DateFlexibility string
const (
	DateFixed    DateFlexibility = "fixed"    // Exact date required
	DateFlexible DateFlexibility = "flexible" // Some flexibility
	DateOpen     DateFlexibility = "open"     // No date yet
)

type EventScale string
const (
	ScaleIntimate EventScale = "intimate"  // < 20 people
	ScaleSmall    EventScale = "small"     // 20-50 people
	ScaleMedium   EventScale = "medium"    // 50-150 people
	ScaleLarge    EventScale = "large"     // 150-500 people
	ScaleMassive  EventScale = "massive"   // 500+ people
)

// DetectionSignal represents evidence for event detection
type DetectionSignal struct {
	SignalType   string    `json:"signal_type"`
	Source       string    `json:"source"`
	Value        string    `json:"value"`
	Confidence   float64   `json:"confidence"`
	Timestamp    time.Time `json:"timestamp"`
}

// RequiredService represents a service needed for the event
type RequiredService struct {
	ID            uuid.UUID     `json:"id"`
	CategoryID    uuid.UUID     `json:"category_id"`
	CategoryName  string        `json:"category_name"`
	
	// Importance
	Priority      ServicePriority `json:"priority"`
	IsRequired    bool          `json:"is_required"`
	
	// Timing
	Phase         EventPhase    `json:"phase"`
	DeadlineDays  int           `json:"deadline_days_before_event"`
	
	// Budget
	BudgetAllocation float64    `json:"budget_allocation_percentage"`
	EstimatedCost    *PriceRange `json:"estimated_cost,omitempty"`
	
	// Status
	Status        ServiceRequirementStatus `json:"status"`
	BookingID     *uuid.UUID    `json:"booking_id,omitempty"`
	
	// Recommendations
	TopVendors    []VendorRecommendation `json:"top_vendors,omitempty"`
}

type ServicePriority string
const (
	PriorityCritical  ServicePriority = "critical"  // Event can't happen without
	PriorityHigh      ServicePriority = "high"      // Very important
	PriorityMedium    ServicePriority = "medium"    // Nice to have
	PriorityLow       ServicePriority = "low"       // Optional enhancement
)

type ServiceRequirementStatus string
const (
	RequirementPending    ServiceRequirementStatus = "pending"
	RequirementSearching  ServiceRequirementStatus = "searching"
	RequirementShortlisted ServiceRequirementStatus = "shortlisted"
	RequirementBooked     ServiceRequirementStatus = "booked"
	RequirementSkipped    ServiceRequirementStatus = "skipped"
)

// Location with rich context
type Location struct {
	Address       string   `json:"address"`
	City          string   `json:"city"`
	State         string   `json:"state"`
	Country       string   `json:"country"`
	PostalCode    string   `json:"postal_code,omitempty"`
	Latitude      float64  `json:"latitude"`
	Longitude     float64  `json:"longitude"`
	PlaceID       string   `json:"place_id,omitempty"` // Google Places ID
	VenueID       *uuid.UUID `json:"venue_id,omitempty"` // If specific venue
	VenueName     string   `json:"venue_name,omitempty"`
}

// Budget with flexibility
type Budget struct {
	TotalAmount    float64         `json:"total_amount"`
	Currency       string          `json:"currency"`
	Flexibility    BudgetFlex      `json:"flexibility"`
	FlexPercentage float64         `json:"flex_percentage"` // How much over budget is OK
	Allocated      float64         `json:"allocated"`       // Amount allocated to services
	Spent          float64         `json:"spent"`           // Amount actually spent
	Breakdown      []BudgetItem    `json:"breakdown,omitempty"`
}

type BudgetFlex string
const (
	BudgetStrict   BudgetFlex = "strict"   // Cannot exceed
	BudgetModerate BudgetFlex = "moderate" // Some flexibility
	BudgetFlexible BudgetFlex = "flexible" // Significant flexibility
)

type BudgetItem struct {
	CategoryID   uuid.UUID `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Allocated    float64   `json:"allocated"`
	Spent        float64   `json:"spent"`
	Percentage   float64   `json:"percentage"`
}

type PriceRange struct {
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Currency string  `json:"currency"`
}

// EventPreferences captures user preferences for the event
type EventPreferences struct {
	Style            string   `json:"style,omitempty"`      // 'traditional', 'modern', 'minimalist'
	Theme            string   `json:"theme,omitempty"`
	ColorPalette     []string `json:"color_palette,omitempty"`
	MustHaves        []string `json:"must_haves,omitempty"`
	DealBreakers     []string `json:"deal_breakers,omitempty"`
	VendorPrefs      VendorPreferences `json:"vendor_preferences"`
	CommunicationPref string  `json:"communication_preference"` // 'email', 'whatsapp', 'phone'
}

type VendorPreferences struct {
	MinRating        float64  `json:"min_rating"`
	PreferVerified   bool     `json:"prefer_verified"`
	PreferExperience bool     `json:"prefer_experienced"`
	PricePreference  string   `json:"price_preference"` // 'budget', 'mid-range', 'premium'
	PreferredVendors []uuid.UUID `json:"preferred_vendors,omitempty"`
	BlockedVendors   []uuid.UUID `json:"blocked_vendors,omitempty"`
}

// Constraint represents a hard requirement
type Constraint struct {
	Type       string `json:"type"`       // 'date', 'budget', 'location', 'dietary', 'accessibility'
	Field      string `json:"field"`
	Operator   string `json:"operator"`   // 'eq', 'neq', 'gt', 'lt', 'in', 'nin'
	Value      interface{} `json:"value"`
	IsHard     bool   `json:"is_hard"`    // Hard = must satisfy, Soft = prefer
}

// =============================================================================
// 2.2 EVENT DETECTION ENGINE
// =============================================================================

// EventDetectionEngine detects life events from various signals
type EventDetectionEngine struct {
	db              *pgxpool.Pool
	cache           *redis.Client
	signalProcessors map[DetectionMethod]SignalProcessor
	mlPredictor     *MLEventPredictor
	config          *DetectionConfig
}

type DetectionConfig struct {
	MinConfidenceThreshold float64
	SignalWindowDays       int
	EnableMLPrediction     bool
	EnableCalendarSync     bool
	EnablePartnerData      bool
}

// SignalProcessor processes specific types of detection signals
type SignalProcessor interface {
	ProcessSignals(ctx context.Context, userID uuid.UUID, window time.Duration) ([]DetectionSignal, error)
	GetEventProbabilities(signals []DetectionSignal) map[EventType]float64
}

// BehavioralSignalProcessor analyzes search and browse patterns
type BehavioralSignalProcessor struct {
	db *pgxpool.Pool
}

func (p *BehavioralSignalProcessor) ProcessSignals(ctx context.Context, userID uuid.UUID, window time.Duration) ([]DetectionSignal, error) {
	// Analyze search patterns
	searchSignals, err := p.analyzeSearchPatterns(ctx, userID, window)
	if err != nil {
		return nil, err
	}
	
	// Analyze browse patterns
	browseSignals, err := p.analyzeBrowsePatterns(ctx, userID, window)
	if err != nil {
		return nil, err
	}
	
	// Analyze interaction patterns
	interactionSignals, err := p.analyzeInteractionPatterns(ctx, userID, window)
	if err != nil {
		return nil, err
	}
	
	signals := append(searchSignals, browseSignals...)
	signals = append(signals, interactionSignals...)
	
	return signals, nil
}

func (p *BehavioralSignalProcessor) analyzeSearchPatterns(ctx context.Context, userID uuid.UUID, window time.Duration) ([]DetectionSignal, error) {
	query := `
		WITH search_clusters AS (
			SELECT 
				search_query,
				detected_intent,
				detected_event_type,
				COUNT(*) as search_count,
				MAX(created_at) as last_search
			FROM search_history
			WHERE user_id = $1
			  AND created_at > NOW() - $2::interval
			GROUP BY search_query, detected_intent, detected_event_type
		),
		event_signals AS (
			SELECT 
				detected_event_type,
				SUM(search_count) as total_searches,
				MAX(last_search) as most_recent,
				ARRAY_AGG(DISTINCT search_query) as queries
			FROM search_clusters
			WHERE detected_event_type IS NOT NULL
			GROUP BY detected_event_type
		)
		SELECT detected_event_type, total_searches, most_recent, queries
		FROM event_signals
		WHERE total_searches >= 3
		ORDER BY total_searches DESC
	`
	
	rows, err := p.db.Query(ctx, query, userID, fmt.Sprintf("%d days", int(window.Hours()/24)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var signals []DetectionSignal
	for rows.Next() {
		var eventType string
		var searchCount int
		var lastSearch time.Time
		var queries []string
		
		if err := rows.Scan(&eventType, &searchCount, &lastSearch, &queries); err != nil {
			continue
		}
		
		// Calculate confidence based on search frequency and recency
		recencyFactor := 1.0 - (time.Since(lastSearch).Hours() / (window.Hours()))
		frequencyFactor := float64(searchCount) / 20.0 // Normalize to 20 searches = 1.0
		confidence := (recencyFactor*0.4 + frequencyFactor*0.6) * 0.8 // Max 0.8 for search signals
		
		signals = append(signals, DetectionSignal{
			SignalType: "search_pattern",
			Source:     "search_history",
			Value:      eventType,
			Confidence: confidence,
			Timestamp:  lastSearch,
		})
	}
	
	return signals, nil
}

func (p *BehavioralSignalProcessor) analyzeBrowsePatterns(ctx context.Context, userID uuid.UUID, window time.Duration) ([]DetectionSignal, error) {
	query := `
		WITH category_views AS (
			SELECT 
				sc.cluster_type,
				COUNT(*) as view_count,
				SUM(ui.duration_seconds) as total_duration,
				MAX(ui.created_at) as last_view
			FROM user_interactions ui
			JOIN services s ON s.id = ui.entity_id AND ui.entity_type = 'service'
			JOIN service_categories sc ON sc.id = s.category_id
			WHERE ui.user_id = $1
			  AND ui.interaction_type = 'view'
			  AND ui.created_at > NOW() - $2::interval
			GROUP BY sc.cluster_type
		)
		SELECT cluster_type, view_count, total_duration, last_view
		FROM category_views
		WHERE view_count >= 5
		ORDER BY view_count DESC
	`
	
	rows, err := p.db.Query(ctx, query, userID, fmt.Sprintf("%d days", int(window.Hours()/24)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var signals []DetectionSignal
	for rows.Next() {
		var clusterType string
		var viewCount int
		var totalDuration int
		var lastView time.Time
		
		if err := rows.Scan(&clusterType, &viewCount, &totalDuration, &lastView); err != nil {
			continue
		}
		
		// Calculate confidence based on engagement depth
		engagementScore := float64(totalDuration) / float64(viewCount) / 60.0 // Average minutes per view
		confidence := (float64(viewCount)/50.0*0.4 + engagementScore/5.0*0.6) * 0.7
		
		signals = append(signals, DetectionSignal{
			SignalType: "browse_pattern",
			Source:     "user_interactions",
			Value:      clusterType,
			Confidence: confidence,
			Timestamp:  lastView,
		})
	}
	
	return signals, nil
}

func (p *BehavioralSignalProcessor) analyzeInteractionPatterns(ctx context.Context, userID uuid.UUID, window time.Duration) ([]DetectionSignal, error) {
	// Analyze saves, shares, inquiries
	query := `
		WITH high_intent_actions AS (
			SELECT 
				sc.cluster_type,
				ui.interaction_type,
				COUNT(*) as action_count,
				MAX(ui.created_at) as last_action
			FROM user_interactions ui
			JOIN services s ON s.id = ui.entity_id AND ui.entity_type = 'service'
			JOIN service_categories sc ON sc.id = s.category_id
			WHERE ui.user_id = $1
			  AND ui.interaction_type IN ('save', 'share', 'inquire', 'add_to_cart')
			  AND ui.created_at > NOW() - $2::interval
			GROUP BY sc.cluster_type, ui.interaction_type
		)
		SELECT cluster_type, 
		       SUM(CASE WHEN interaction_type = 'inquire' THEN action_count * 3
		                WHEN interaction_type = 'add_to_cart' THEN action_count * 2
		                ELSE action_count END) as weighted_score,
		       MAX(last_action) as most_recent
		FROM high_intent_actions
		GROUP BY cluster_type
		HAVING SUM(action_count) >= 2
	`
	
	rows, err := p.db.Query(ctx, query, userID, fmt.Sprintf("%d days", int(window.Hours()/24)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var signals []DetectionSignal
	for rows.Next() {
		var clusterType string
		var weightedScore float64
		var mostRecent time.Time
		
		if err := rows.Scan(&clusterType, &weightedScore, &mostRecent); err != nil {
			continue
		}
		
		confidence := (weightedScore / 30.0) * 0.9 // High-intent signals get up to 0.9
		if confidence > 0.9 {
			confidence = 0.9
		}
		
		signals = append(signals, DetectionSignal{
			SignalType: "high_intent_action",
			Source:     "user_interactions",
			Value:      clusterType,
			Confidence: confidence,
			Timestamp:  mostRecent,
		})
	}
	
	return signals, nil
}

func (p *BehavioralSignalProcessor) GetEventProbabilities(signals []DetectionSignal) map[EventType]float64 {
	// Aggregate signals into event probabilities
	eventScores := make(map[EventType]float64)
	eventCounts := make(map[EventType]int)
	
	clusterToEvents := map[string][]EventType{
		"celebrations": {EventTypeWedding, EventTypeBirthday, EventTypeGraduation},
		"home":         {EventTypeRelocation, EventTypeRenovation},
		"travel":       {EventTypeTravel},
		"health":       {EventTypeChildbirth},
		"business":     {EventTypeBusinessLaunch},
	}
	
	for _, signal := range signals {
		if events, ok := clusterToEvents[signal.Value]; ok {
			for _, event := range events {
				eventScores[event] += signal.Confidence
				eventCounts[event]++
			}
		}
	}
	
	// Normalize to probabilities
	probabilities := make(map[EventType]float64)
	for event, score := range eventScores {
		// Average confidence with count boost
		avgConfidence := score / float64(eventCounts[event])
		countBoost := 1.0 + (float64(eventCounts[event])-1)*0.1 // 10% boost per additional signal
		probabilities[event] = avgConfidence * countBoost
		if probabilities[event] > 1.0 {
			probabilities[event] = 1.0
		}
	}
	
	return probabilities
}

// DetectEvents is the main detection entry point
func (e *EventDetectionEngine) DetectEvents(ctx context.Context, userID uuid.UUID) ([]LifeEvent, error) {
	window := time.Duration(e.config.SignalWindowDays) * 24 * time.Hour
	
	// Collect signals from all processors
	var allSignals []DetectionSignal
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	for method, processor := range e.signalProcessors {
		wg.Add(1)
		go func(m DetectionMethod, p SignalProcessor) {
			defer wg.Done()
			signals, err := p.ProcessSignals(ctx, userID, window)
			if err != nil {
				return
			}
			mu.Lock()
			allSignals = append(allSignals, signals...)
			mu.Unlock()
		}(method, processor)
	}
	wg.Wait()
	
	if len(allSignals) == 0 {
		return nil, nil
	}
	
	// Get event probabilities
	probabilities := e.aggregateProbabilities(allSignals)
	
	// Create life events for high-confidence detections
	var events []LifeEvent
	for eventType, probability := range probabilities {
		if probability >= e.config.MinConfidenceThreshold {
			event := e.createDetectedEvent(userID, eventType, probability, allSignals)
			events = append(events, event)
		}
	}
	
	// Sort by confidence
	sort.Slice(events, func(i, j int) bool {
		return events[i].DetectionConfidence > events[j].DetectionConfidence
	})
	
	return events, nil
}

func (e *EventDetectionEngine) aggregateProbabilities(signals []DetectionSignal) map[EventType]float64 {
	// Use ensemble of processor probabilities
	combined := make(map[EventType]float64)
	counts := make(map[EventType]int)
	
	for _, processor := range e.signalProcessors {
		probs := processor.GetEventProbabilities(signals)
		for event, prob := range probs {
			combined[event] += prob
			counts[event]++
		}
	}
	
	// Average across processors
	for event := range combined {
		combined[event] /= float64(counts[event])
	}
	
	return combined
}

func (e *EventDetectionEngine) createDetectedEvent(userID uuid.UUID, eventType EventType, confidence float64, signals []DetectionSignal) LifeEvent {
	// Filter relevant signals
	var relevantSignals []DetectionSignal
	for _, s := range signals {
		if s.Confidence > 0.3 {
			relevantSignals = append(relevantSignals, s)
		}
	}
	
	return LifeEvent{
		ID:                  uuid.New(),
		UserID:              userID,
		EventType:           eventType,
		ClusterType:         e.getClusterForEvent(eventType),
		DetectedAt:          time.Now(),
		DetectionMethod:     DetectionBehavioral,
		DetectionConfidence: confidence,
		DetectionSignals:    relevantSignals,
		Status:              StatusDetected,
		Phase:               PhaseDiscovery,
		CompletionPct:       0,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
}

func (e *EventDetectionEngine) getClusterForEvent(eventType EventType) ClusterType {
	mapping := map[EventType]ClusterType{
		EventTypeWedding:        ClusterCelebrations,
		EventTypeFuneral:        ClusterCelebrations,
		EventTypeBirthday:       ClusterCelebrations,
		EventTypeRelocation:     ClusterHome,
		EventTypeRenovation:     ClusterHome,
		EventTypeChildbirth:     ClusterHealth,
		EventTypeTravel:         ClusterTravel,
		EventTypeBusinessLaunch: ClusterBusiness,
		EventTypeGraduation:     ClusterCelebrations,
	}
	return mapping[eventType]
}

// =============================================================================
// 2.3 ORCHESTRATION ENGINE
// =============================================================================

// OrchestrationEngine manages the lifecycle of life events
type OrchestrationEngine struct {
	db                *pgxpool.Pool
	cache             *redis.Client
	recommendationSvc *RecommendationService
	bookingSvc        *BookingService
	notificationSvc   *NotificationService
	pricingEngine     *PricingEngine
	scheduler         *EventScheduler
}

// EventOrchestrationPlan represents the full plan for an event
type EventOrchestrationPlan struct {
	EventID          uuid.UUID                `json:"event_id"`
	
	// Timeline
	Phases           []PhasePlan              `json:"phases"`
	CriticalPath     []CriticalMilestone      `json:"critical_path"`
	
	// Services
	ServicePlan      []PlannedService         `json:"service_plan"`
	SuggestedBundles []BundleOption           `json:"suggested_bundles"`
	
	// Budget
	BudgetPlan       BudgetPlan               `json:"budget_plan"`
	
	// Risk Assessment
	Risks            []IdentifiedRisk         `json:"risks"`
	
	// Actions
	NextActions      []RecommendedAction      `json:"next_actions"`
	
	GeneratedAt      time.Time                `json:"generated_at"`
}

type PhasePlan struct {
	Phase            EventPhase               `json:"phase"`
	StartDate        time.Time                `json:"start_date"`
	EndDate          time.Time                `json:"end_date"`
	Tasks            []PhaseTask              `json:"tasks"`
	Dependencies     []uuid.UUID              `json:"dependencies"`
	Status           string                   `json:"status"`
}

type PhaseTask struct {
	ID               uuid.UUID                `json:"id"`
	Title            string                   `json:"title"`
	Description      string                   `json:"description"`
	CategoryID       *uuid.UUID               `json:"category_id,omitempty"`
	DueDate          time.Time                `json:"due_date"`
	Priority         string                   `json:"priority"`
	Status           string                   `json:"status"`
	AssignedTo       string                   `json:"assigned_to"` // 'user', 'vendor', 'platform'
}

type CriticalMilestone struct {
	ID               uuid.UUID                `json:"id"`
	Title            string                   `json:"title"`
	Date             time.Time                `json:"date"`
	ServiceID        *uuid.UUID               `json:"service_id,omitempty"`
	IsMet            bool                     `json:"is_met"`
	BlocksEvent      bool                     `json:"blocks_event"`
}

type PlannedService struct {
	CategoryID       uuid.UUID                `json:"category_id"`
	CategoryName     string                   `json:"category_name"`
	Priority         ServicePriority          `json:"priority"`
	Phase            EventPhase               `json:"phase"`
	BookByDate       time.Time                `json:"book_by_date"`
	EstimatedCost    PriceRange               `json:"estimated_cost"`
	BudgetAllocation float64                  `json:"budget_allocation"`
	Status           string                   `json:"status"`
	RecommendedVendors []VendorRecommendation `json:"recommended_vendors"`
}

type BundleOption struct {
	BundleID         uuid.UUID                `json:"bundle_id"`
	Name             string                   `json:"name"`
	Description      string                   `json:"description"`
	IncludedServices []uuid.UUID              `json:"included_services"`
	TotalPrice       float64                  `json:"total_price"`
	Savings          float64                  `json:"savings"`
	SavingsPercent   float64                  `json:"savings_percent"`
}

type BudgetPlan struct {
	TotalBudget      float64                  `json:"total_budget"`
	AllocatedAmount  float64                  `json:"allocated_amount"`
	SpentAmount      float64                  `json:"spent_amount"`
	RemainingAmount  float64                  `json:"remaining_amount"`
	Categories       []CategoryBudget         `json:"categories"`
	Recommendations  []BudgetRecommendation   `json:"recommendations"`
}

type CategoryBudget struct {
	CategoryID       uuid.UUID                `json:"category_id"`
	CategoryName     string                   `json:"category_name"`
	Allocated        float64                  `json:"allocated"`
	Spent            float64                  `json:"spent"`
	Percentage       float64                  `json:"percentage"`
	Status           string                   `json:"status"` // 'on_track', 'over_budget', 'under_budget'
}

type BudgetRecommendation struct {
	Type             string                   `json:"type"`
	Message          string                   `json:"message"`
	PotentialSavings float64                  `json:"potential_savings,omitempty"`
	Action           string                   `json:"action,omitempty"`
}

type IdentifiedRisk struct {
	ID               uuid.UUID                `json:"id"`
	Type             string                   `json:"type"`
	Description      string                   `json:"description"`
	Severity         string                   `json:"severity"` // 'low', 'medium', 'high', 'critical'
	Likelihood       string                   `json:"likelihood"`
	MitigationSteps  []string                 `json:"mitigation_steps"`
	AffectedServices []uuid.UUID              `json:"affected_services,omitempty"`
}

type RecommendedAction struct {
	ID               uuid.UUID                `json:"id"`
	Title            string                   `json:"title"`
	Description      string                   `json:"description"`
	Priority         string                   `json:"priority"`
	DueDate          *time.Time               `json:"due_date,omitempty"`
	ActionType       string                   `json:"action_type"` // 'book', 'confirm', 'review', 'pay', 'contact'
	RelatedServiceID *uuid.UUID               `json:"related_service_id,omitempty"`
	DeepLink         string                   `json:"deep_link,omitempty"`
}

type VendorRecommendation struct {
	VendorID         uuid.UUID                `json:"vendor_id"`
	VendorName       string                   `json:"vendor_name"`
	ServiceID        uuid.UUID                `json:"service_id"`
	ServiceName      string                   `json:"service_name"`
	Rating           float64                  `json:"rating"`
	ReviewCount      int                      `json:"review_count"`
	Price            float64                  `json:"price"`
	MatchScore       float64                  `json:"match_score"`
	MatchReasons     []string                 `json:"match_reasons"`
	Availability     string                   `json:"availability"` // 'available', 'limited', 'unavailable'
	ResponseTime     string                   `json:"response_time"`
}

// GeneratePlan creates a comprehensive orchestration plan for an event
func (o *OrchestrationEngine) GeneratePlan(ctx context.Context, event *LifeEvent) (*EventOrchestrationPlan, error) {
	plan := &EventOrchestrationPlan{
		EventID:     event.ID,
		GeneratedAt: time.Now(),
	}
	
	// 1. Generate service requirements
	services, err := o.generateServiceRequirements(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service requirements: %w", err)
	}
	plan.ServicePlan = services
	
	// 2. Generate timeline and phases
	phases, milestones, err := o.generateTimeline(ctx, event, services)
	if err != nil {
		return nil, fmt.Errorf("failed to generate timeline: %w", err)
	}
	plan.Phases = phases
	plan.CriticalPath = milestones
	
	// 3. Generate budget plan
	budgetPlan, err := o.generateBudgetPlan(ctx, event, services)
	if err != nil {
		return nil, fmt.Errorf("failed to generate budget plan: %w", err)
	}
	plan.BudgetPlan = budgetPlan
	
	// 4. Find bundle opportunities
	bundles, err := o.findBundleOpportunities(ctx, event, services)
	if err != nil {
		return nil, fmt.Errorf("failed to find bundles: %w", err)
	}
	plan.SuggestedBundles = bundles
	
	// 5. Assess risks
	risks := o.assessRisks(event, plan)
	plan.Risks = risks
	
	// 6. Generate next actions
	actions := o.generateNextActions(event, plan)
	plan.NextActions = actions
	
	return plan, nil
}

func (o *OrchestrationEngine) generateServiceRequirements(ctx context.Context, event *LifeEvent) ([]PlannedService, error) {
	// Get required categories for this event type
	query := `
		SELECT 
			ecm.category_id,
			sc.name as category_name,
			ecm.role_type,
			ecm.phase,
			ecm.typical_booking_offset_days,
			ecm.necessity_score,
			ecm.typical_budget_percentage
		FROM event_category_mappings ecm
		JOIN life_event_triggers let ON let.id = ecm.event_trigger_id
		JOIN service_categories sc ON sc.id = ecm.category_id
		WHERE let.slug = $1
		  AND ecm.is_active = TRUE
		ORDER BY ecm.necessity_score DESC, ecm.typical_booking_offset_days DESC
	`
	
	rows, err := o.db.Query(ctx, query, string(event.EventType))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var services []PlannedService
	for rows.Next() {
		var s PlannedService
		var roleType, phase string
		var bookingOffset int
		var necessity, budgetPct float64
		
		if err := rows.Scan(&s.CategoryID, &s.CategoryName, &roleType, &phase, 
			&bookingOffset, &necessity, &budgetPct); err != nil {
			continue
		}
		
		// Map role to priority
		switch roleType {
		case "primary":
			s.Priority = PriorityCritical
		case "secondary":
			s.Priority = PriorityHigh
		case "optional":
			s.Priority = PriorityMedium
		default:
			s.Priority = PriorityLow
		}
		
		s.Phase = EventPhase(phase)
		s.BudgetAllocation = budgetPct
		
		// Calculate book-by date
		if event.EventDate != nil {
			s.BookByDate = event.EventDate.AddDate(0, 0, -bookingOffset)
		}
		
		// Get price estimates
		s.EstimatedCost = o.estimateServiceCost(ctx, s.CategoryID, event)
		
		// Get vendor recommendations
		s.RecommendedVendors = o.getVendorRecommendations(ctx, s.CategoryID, event, 3)
		
		s.Status = "pending"
		services = append(services, s)
	}
	
	return services, nil
}

func (o *OrchestrationEngine) estimateServiceCost(ctx context.Context, categoryID uuid.UUID, event *LifeEvent) PriceRange {
	// Get price range from actual services in the category
	query := `
		SELECT 
			PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY base_price) as p25,
			PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY base_price) as p75
		FROM services s
		JOIN vendors v ON v.id = s.vendor_id
		WHERE s.category_id = $1
		  AND s.is_available = TRUE
		  AND v.is_active = TRUE
		  AND s.base_price IS NOT NULL
	`
	
	var min, max float64
	o.db.QueryRow(ctx, query, categoryID).Scan(&min, &max)
	
	// Adjust for event scale
	scaleFactor := 1.0
	if event.Scale == ScaleLarge {
		scaleFactor = 1.5
	} else if event.Scale == ScaleMassive {
		scaleFactor = 2.0
	}
	
	return PriceRange{
		Min:      min * scaleFactor,
		Max:      max * scaleFactor,
		Currency: "NGN",
	}
}

func (o *OrchestrationEngine) getVendorRecommendations(ctx context.Context, categoryID uuid.UUID, event *LifeEvent, limit int) []VendorRecommendation {
	query := `
		SELECT 
			v.id as vendor_id,
			v.business_name,
			s.id as service_id,
			s.name as service_name,
			v.rating_average,
			v.rating_count,
			s.base_price,
			v.response_time_minutes
		FROM services s
		JOIN vendors v ON v.id = s.vendor_id
		WHERE s.category_id = $1
		  AND s.is_available = TRUE
		  AND v.is_active = TRUE
		  AND v.is_verified = TRUE
		ORDER BY v.rating_average DESC, v.rating_count DESC
		LIMIT $2
	`
	
	rows, err := o.db.Query(ctx, query, categoryID, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	
	var recs []VendorRecommendation
	for rows.Next() {
		var r VendorRecommendation
		var responseMinutes int
		
		if err := rows.Scan(&r.VendorID, &r.VendorName, &r.ServiceID, &r.ServiceName,
			&r.Rating, &r.ReviewCount, &r.Price, &responseMinutes); err != nil {
			continue
		}
		
		// Calculate match score
		r.MatchScore = o.calculateVendorMatchScore(r, event)
		r.MatchReasons = o.getMatchReasons(r, event)
		r.Availability = "available" // Would check actual availability
		r.ResponseTime = fmt.Sprintf("~%d min", responseMinutes)
		
		recs = append(recs, r)
	}
	
	// Sort by match score
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].MatchScore > recs[j].MatchScore
	})
	
	return recs
}

func (o *OrchestrationEngine) calculateVendorMatchScore(vendor VendorRecommendation, event *LifeEvent) float64 {
	score := 0.0
	
	// Rating contribution (40%)
	score += (vendor.Rating / 5.0) * 0.4
	
	// Review count contribution (20%)
	reviewScore := float64(vendor.ReviewCount) / 100.0
	if reviewScore > 1.0 {
		reviewScore = 1.0
	}
	score += reviewScore * 0.2
	
	// Price match contribution (25%)
	if event.Budget != nil {
		categoryBudget := event.Budget.TotalAmount * 0.1 // Assume 10% per category
		if vendor.Price <= categoryBudget {
			score += 0.25
		} else {
			score += 0.25 * (categoryBudget / vendor.Price)
		}
	} else {
		score += 0.15 // Neutral if no budget set
	}
	
	// Preference match (15%)
	if event.Preferences.VendorPrefs.MinRating > 0 && vendor.Rating >= event.Preferences.VendorPrefs.MinRating {
		score += 0.15
	}
	
	return score
}

func (o *OrchestrationEngine) getMatchReasons(vendor VendorRecommendation, event *LifeEvent) []string {
	var reasons []string
	
	if vendor.Rating >= 4.5 {
		reasons = append(reasons, "Top-rated vendor")
	}
	
	if vendor.ReviewCount >= 50 {
		reasons = append(reasons, "Highly experienced")
	}
	
	if event.Budget != nil {
		categoryBudget := event.Budget.TotalAmount * 0.1
		if vendor.Price <= categoryBudget {
			reasons = append(reasons, "Within budget")
		}
	}
	
	return reasons
}

func (o *OrchestrationEngine) generateTimeline(ctx context.Context, event *LifeEvent, services []PlannedService) ([]PhasePlan, []CriticalMilestone, error) {
	if event.EventDate == nil {
		return nil, nil, fmt.Errorf("event date required for timeline generation")
	}
	
	eventDate := *event.EventDate
	
	// Define phase durations based on event type and planning horizon
	phases := []PhasePlan{
		{
			Phase:     PhaseDiscovery,
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 0, 7),
			Status:    "completed",
		},
		{
			Phase:     PhasePlanning,
			StartDate: time.Now().AddDate(0, 0, 7),
			EndDate:   eventDate.AddDate(0, -3, 0),
			Status:    "active",
		},
		{
			Phase:     PhaseVendorSelect,
			StartDate: eventDate.AddDate(0, -3, 0),
			EndDate:   eventDate.AddDate(0, -2, 0),
			Status:    "pending",
		},
		{
			Phase:     PhaseBooking,
			StartDate: eventDate.AddDate(0, -2, 0),
			EndDate:   eventDate.AddDate(0, -1, 0),
			Status:    "pending",
		},
		{
			Phase:     PhasePreEvent,
			StartDate: eventDate.AddDate(0, 0, -7),
			EndDate:   eventDate.AddDate(0, 0, -1),
			Status:    "pending",
		},
		{
			Phase:     PhaseEventDay,
			StartDate: eventDate,
			EndDate:   eventDate,
			Status:    "pending",
		},
		{
			Phase:     PhasePostEvent,
			StartDate: eventDate.AddDate(0, 0, 1),
			EndDate:   eventDate.AddDate(0, 0, 14),
			Status:    "pending",
		},
	}
	
	// Generate tasks for each phase
	for i := range phases {
		phases[i].Tasks = o.generatePhaseTasks(phases[i].Phase, services, eventDate)
	}
	
	// Generate critical milestones
	var milestones []CriticalMilestone
	for _, svc := range services {
		if svc.Priority == PriorityCritical {
			milestones = append(milestones, CriticalMilestone{
				ID:          uuid.New(),
				Title:       fmt.Sprintf("Book %s", svc.CategoryName),
				Date:        svc.BookByDate,
				IsMet:       svc.Status == "booked",
				BlocksEvent: true,
			})
		}
	}
	
	// Sort milestones by date
	sort.Slice(milestones, func(i, j int) bool {
		return milestones[i].Date.Before(milestones[j].Date)
	})
	
	return phases, milestones, nil
}

func (o *OrchestrationEngine) generatePhaseTasks(phase EventPhase, services []PlannedService, eventDate time.Time) []PhaseTask {
	var tasks []PhaseTask
	
	switch phase {
	case PhaseDiscovery:
		tasks = append(tasks, PhaseTask{
			ID:          uuid.New(),
			Title:       "Confirm event date",
			Description: "Set the final date for your event",
			Priority:    "high",
			Status:      "pending",
			AssignedTo:  "user",
		})
		tasks = append(tasks, PhaseTask{
			ID:          uuid.New(),
			Title:       "Set budget",
			Description: "Define your total budget for the event",
			Priority:    "high",
			Status:      "pending",
			AssignedTo:  "user",
		})
		
	case PhasePlanning:
		for _, svc := range services {
			if svc.Priority == PriorityCritical || svc.Priority == PriorityHigh {
				tasks = append(tasks, PhaseTask{
					ID:          uuid.New(),
					Title:       fmt.Sprintf("Research %s options", svc.CategoryName),
					Description: fmt.Sprintf("Review and shortlist %s vendors", svc.CategoryName),
					CategoryID:  &svc.CategoryID,
					DueDate:     svc.BookByDate.AddDate(0, 0, -14),
					Priority:    string(svc.Priority),
					Status:      "pending",
					AssignedTo:  "user",
				})
			}
		}
		
	case PhaseBooking:
		for _, svc := range services {
			tasks = append(tasks, PhaseTask{
				ID:          uuid.New(),
				Title:       fmt.Sprintf("Book %s", svc.CategoryName),
				Description: fmt.Sprintf("Confirm booking with selected %s vendor", svc.CategoryName),
				CategoryID:  &svc.CategoryID,
				DueDate:     svc.BookByDate,
				Priority:    string(svc.Priority),
				Status:      "pending",
				AssignedTo:  "user",
			})
		}
		
	case PhasePreEvent:
		tasks = append(tasks, PhaseTask{
			ID:          uuid.New(),
			Title:       "Confirm all vendors",
			Description: "Call each vendor to confirm details and timing",
			DueDate:     eventDate.AddDate(0, 0, -3),
			Priority:    "high",
			Status:      "pending",
			AssignedTo:  "user",
		})
		tasks = append(tasks, PhaseTask{
			ID:          uuid.New(),
			Title:       "Final payments",
			Description: "Complete all outstanding vendor payments",
			DueDate:     eventDate.AddDate(0, 0, -2),
			Priority:    "high",
			Status:      "pending",
			AssignedTo:  "user",
		})
	}
	
	return tasks
}

func (o *OrchestrationEngine) generateBudgetPlan(ctx context.Context, event *LifeEvent, services []PlannedService) (BudgetPlan, error) {
	plan := BudgetPlan{
		TotalBudget:     0,
		AllocatedAmount: 0,
		SpentAmount:     0,
		Categories:      make([]CategoryBudget, 0),
		Recommendations: make([]BudgetRecommendation, 0),
	}
	
	if event.Budget != nil {
		plan.TotalBudget = event.Budget.TotalAmount
	} else {
		// Estimate budget based on event type and scale
		plan.TotalBudget = o.estimateTotalBudget(event)
	}
	
	// Allocate budget to categories
	for _, svc := range services {
		allocated := plan.TotalBudget * (svc.BudgetAllocation / 100.0)
		plan.AllocatedAmount += allocated
		
		plan.Categories = append(plan.Categories, CategoryBudget{
			CategoryID:   svc.CategoryID,
			CategoryName: svc.CategoryName,
			Allocated:    allocated,
			Spent:        0,
			Percentage:   svc.BudgetAllocation,
			Status:       "on_track",
		})
	}
	
	plan.RemainingAmount = plan.TotalBudget - plan.SpentAmount
	
	// Generate recommendations
	if plan.AllocatedAmount > plan.TotalBudget {
		plan.Recommendations = append(plan.Recommendations, BudgetRecommendation{
			Type:    "warning",
			Message: "Your planned services exceed your budget. Consider reducing scope or increasing budget.",
		})
	}
	
	return plan, nil
}

func (o *OrchestrationEngine) estimateTotalBudget(event *LifeEvent) float64 {
	// Base budgets by event type (in NGN)
	baseBudgets := map[EventType]float64{
		EventTypeWedding:    5000000,
		EventTypeBirthday:   200000,
		EventTypeRelocation: 500000,
		EventTypeRenovation: 3000000,
	}
	
	base := baseBudgets[event.EventType]
	if base == 0 {
		base = 1000000 // Default
	}
	
	// Adjust for scale
	scaleFactor := map[EventScale]float64{
		ScaleIntimate: 0.5,
		ScaleSmall:    0.75,
		ScaleMedium:   1.0,
		ScaleLarge:    1.5,
		ScaleMassive:  2.5,
	}
	
	return base * scaleFactor[event.Scale]
}

func (o *OrchestrationEngine) findBundleOpportunities(ctx context.Context, event *LifeEvent, services []PlannedService) ([]BundleOption, error) {
	// Get category IDs
	var categoryIDs []uuid.UUID
	for _, svc := range services {
		categoryIDs = append(categoryIDs, svc.CategoryID)
	}
	
	// Find bundles that match these categories
	query := `
		SELECT 
			sb.id,
			sb.name,
			sb.short_description,
			sb.category_ids,
			sb.discount_percentage
		FROM service_bundles sb
		JOIN life_event_triggers let ON let.id = sb.event_trigger_id
		WHERE let.slug = $1
		  AND sb.is_active = TRUE
		  AND sb.category_ids && $2
		ORDER BY sb.discount_percentage DESC
		LIMIT 5
	`
	
	rows, err := o.db.Query(ctx, query, string(event.EventType), categoryIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var bundles []BundleOption
	for rows.Next() {
		var b BundleOption
		var discountPct float64
		
		if err := rows.Scan(&b.BundleID, &b.Name, &b.Description, &b.IncludedServices, &discountPct); err != nil {
			continue
		}
		
		// Calculate pricing
		b.TotalPrice = o.calculateBundlePrice(ctx, b.BundleID)
		regularPrice := b.TotalPrice / (1 - discountPct/100)
		b.Savings = regularPrice - b.TotalPrice
		b.SavingsPercent = discountPct
		
		bundles = append(bundles, b)
	}
	
	return bundles, nil
}

func (o *OrchestrationEngine) calculateBundlePrice(ctx context.Context, bundleID uuid.UUID) float64 {
	var totalPrice float64
	o.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(bva.bundle_price), 0)
		FROM bundle_vendor_assignments bva
		WHERE bva.bundle_id = $1
	`, bundleID).Scan(&totalPrice)
	return totalPrice
}

func (o *OrchestrationEngine) assessRisks(event *LifeEvent, plan *EventOrchestrationPlan) []IdentifiedRisk {
	var risks []IdentifiedRisk
	
	// Check timeline risks
	if event.EventDate != nil {
		daysUntilEvent := int(time.Until(*event.EventDate).Hours() / 24)
		
		for _, svc := range plan.ServicePlan {
			if svc.Priority == PriorityCritical {
				daysUntilDeadline := int(time.Until(svc.BookByDate).Hours() / 24)
				if daysUntilDeadline < 7 {
					risks = append(risks, IdentifiedRisk{
						ID:          uuid.New(),
						Type:        "timeline",
						Description: fmt.Sprintf("%s booking deadline approaching", svc.CategoryName),
						Severity:    "high",
						Likelihood:  "certain",
						MitigationSteps: []string{
							fmt.Sprintf("Book %s immediately", svc.CategoryName),
							"Consider multiple vendor options",
						},
					})
				}
			}
		}
		
		if daysUntilEvent < 30 {
			risks = append(risks, IdentifiedRisk{
				ID:          uuid.New(),
				Type:        "timeline",
				Description: "Less than 30 days until event - limited vendor availability expected",
				Severity:    "medium",
				Likelihood:  "likely",
				MitigationSteps: []string{
					"Focus on available vendors only",
					"Be prepared for premium pricing",
				},
			})
		}
	}
	
	// Check budget risks
	if plan.BudgetPlan.AllocatedAmount > plan.BudgetPlan.TotalBudget*1.1 {
		risks = append(risks, IdentifiedRisk{
			ID:          uuid.New(),
			Type:        "budget",
			Description: "Planned services significantly exceed budget",
			Severity:    "high",
			Likelihood:  "certain",
			MitigationSteps: []string{
				"Review and prioritize services",
				"Consider bundle discounts",
				"Adjust budget expectations",
			},
		})
	}
	
	return risks
}

func (o *OrchestrationEngine) generateNextActions(event *LifeEvent, plan *EventOrchestrationPlan) []RecommendedAction {
	var actions []RecommendedAction
	
	// Add action for each pending critical service
	for _, svc := range plan.ServicePlan {
		if svc.Status == "pending" && svc.Priority == PriorityCritical {
			actions = append(actions, RecommendedAction{
				ID:               uuid.New(),
				Title:            fmt.Sprintf("Find %s vendor", svc.CategoryName),
				Description:      fmt.Sprintf("Browse and shortlist %s vendors for your event", svc.CategoryName),
				Priority:         "high",
				DueDate:          &svc.BookByDate,
				ActionType:       "book",
				RelatedServiceID: &svc.CategoryID,
				DeepLink:         fmt.Sprintf("/search?category=%s&event=%s", svc.CategoryID, event.ID),
			})
		}
	}
	
	// Add budget action if not set
	if event.Budget == nil {
		actions = append(actions, RecommendedAction{
			ID:          uuid.New(),
			Title:       "Set your budget",
			Description: "Define your total budget to get better recommendations",
			Priority:    "medium",
			ActionType:  "confirm",
			DeepLink:    fmt.Sprintf("/events/%s/budget", event.ID),
		})
	}
	
	// Sort by priority and due date
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Priority != actions[j].Priority {
			priorityOrder := map[string]int{"high": 0, "medium": 1, "low": 2}
			return priorityOrder[actions[i].Priority] < priorityOrder[actions[j].Priority]
		}
		if actions[i].DueDate != nil && actions[j].DueDate != nil {
			return actions[i].DueDate.Before(*actions[j].DueDate)
		}
		return false
	})
	
	// Limit to top 5 actions
	if len(actions) > 5 {
		actions = actions[:5]
	}
	
	return actions
}

// =============================================================================
// 2.4 API HANDLERS
// =============================================================================

// LifeOSAPI provides the REST API for LifeOS
type LifeOSAPI struct {
	detectionEngine     *EventDetectionEngine
	orchestrationEngine *OrchestrationEngine
	db                  *pgxpool.Pool
}

// CreateEventRequest for manual event creation
type CreateEventRequest struct {
	EventType    EventType       `json:"event_type"`
	EventSubtype string          `json:"event_subtype,omitempty"`
	EventDate    *time.Time      `json:"event_date,omitempty"`
	DateFlex     DateFlexibility `json:"date_flexibility"`
	Location     *Location       `json:"location,omitempty"`
	GuestCount   *int            `json:"guest_count,omitempty"`
	Budget       *Budget         `json:"budget,omitempty"`
	Preferences  *EventPreferences `json:"preferences,omitempty"`
}

// GetDetectedEvents returns events detected for a user
func (api *LifeOSAPI) GetDetectedEvents(ctx context.Context, userID uuid.UUID) ([]LifeEvent, error) {
	return api.detectionEngine.DetectEvents(ctx, userID)
}

// CreateEvent creates a new life event
func (api *LifeOSAPI) CreateEvent(ctx context.Context, userID uuid.UUID, req CreateEventRequest) (*LifeEvent, error) {
	event := &LifeEvent{
		ID:              uuid.New(),
		UserID:          userID,
		EventType:       req.EventType,
		EventSubtype:    req.EventSubtype,
		ClusterType:     api.detectionEngine.getClusterForEvent(req.EventType),
		DetectedAt:      time.Now(),
		EventDate:       req.EventDate,
		EventDateFlex:   req.DateFlex,
		DetectionMethod: DetectionExplicit,
		DetectionConfidence: 1.0,
		GuestCount:      req.GuestCount,
		Location:        req.Location,
		Budget:          req.Budget,
		Status:          StatusConfirmed,
		Phase:           PhasePlanning,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	
	if req.Preferences != nil {
		event.Preferences = *req.Preferences
	}
	
	// Determine scale from guest count
	if req.GuestCount != nil {
		event.Scale = api.determineScale(*req.GuestCount)
	} else {
		event.Scale = ScaleMedium
	}
	
	// Save to database
	if err := api.saveEvent(ctx, event); err != nil {
		return nil, err
	}
	
	return event, nil
}

func (api *LifeOSAPI) determineScale(guestCount int) EventScale {
	switch {
	case guestCount < 20:
		return ScaleIntimate
	case guestCount < 50:
		return ScaleSmall
	case guestCount < 150:
		return ScaleMedium
	case guestCount < 500:
		return ScaleLarge
	default:
		return ScaleMassive
	}
}

// GetEventPlan returns the orchestration plan for an event
func (api *LifeOSAPI) GetEventPlan(ctx context.Context, eventID uuid.UUID) (*EventOrchestrationPlan, error) {
	// Load event
	event, err := api.loadEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	
	// Generate plan
	return api.orchestrationEngine.GeneratePlan(ctx, event)
}

// ConfirmDetectedEvent confirms a detected event
func (api *LifeOSAPI) ConfirmDetectedEvent(ctx context.Context, eventID uuid.UUID, updates CreateEventRequest) (*LifeEvent, error) {
	event, err := api.loadEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	
	// Apply updates
	if updates.EventDate != nil {
		event.EventDate = updates.EventDate
	}
	if updates.Location != nil {
		event.Location = updates.Location
	}
	if updates.GuestCount != nil {
		event.GuestCount = updates.GuestCount
		event.Scale = api.determineScale(*updates.GuestCount)
	}
	if updates.Budget != nil {
		event.Budget = updates.Budget
	}
	if updates.Preferences != nil {
		event.Preferences = *updates.Preferences
	}
	
	event.Status = StatusConfirmed
	event.Phase = PhasePlanning
	now := time.Now()
	event.ConfirmedAt = &now
	event.UpdatedAt = now
	
	// Save updates
	if err := api.updateEvent(ctx, event); err != nil {
		return nil, err
	}
	
	return event, nil
}

func (api *LifeOSAPI) saveEvent(ctx context.Context, event *LifeEvent) error {
	// Insert into database
	query := `
		INSERT INTO life_events (
			id, user_id, event_type, event_subtype, cluster_type,
			detected_at, event_date, event_date_flexibility, planning_horizon_days,
			detection_method, detection_confidence, detection_signals,
			scale, guest_count, location, budget,
			status, phase, completion_percentage,
			preferences, constraints, custom_attributes, tags,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
		)
	`
	
	locationJSON, _ := json.Marshal(event.Location)
	budgetJSON, _ := json.Marshal(event.Budget)
	signalsJSON, _ := json.Marshal(event.DetectionSignals)
	prefsJSON, _ := json.Marshal(event.Preferences)
	constraintsJSON, _ := json.Marshal(event.Constraints)
	customJSON, _ := json.Marshal(event.CustomAttributes)
	
	_, err := api.db.Exec(ctx, query,
		event.ID, event.UserID, event.EventType, event.EventSubtype, event.ClusterType,
		event.DetectedAt, event.EventDate, event.EventDateFlex, event.PlanningHorizon,
		event.DetectionMethod, event.DetectionConfidence, signalsJSON,
		event.Scale, event.GuestCount, locationJSON, budgetJSON,
		event.Status, event.Phase, event.CompletionPct,
		prefsJSON, constraintsJSON, customJSON, event.Tags,
		event.CreatedAt, event.UpdatedAt,
	)
	
	return err
}

func (api *LifeOSAPI) loadEvent(ctx context.Context, eventID uuid.UUID) (*LifeEvent, error) {
	query := `
		SELECT 
			id, user_id, event_type, event_subtype, cluster_type,
			detected_at, event_date, event_date_flexibility, planning_horizon_days,
			detection_method, detection_confidence, detection_signals,
			scale, guest_count, location, budget,
			status, phase, completion_percentage,
			preferences, constraints, custom_attributes, tags,
			created_at, updated_at, confirmed_at, completed_at
		FROM life_events
		WHERE id = $1
	`
	
	var event LifeEvent
	var locationJSON, budgetJSON, signalsJSON, prefsJSON, constraintsJSON, customJSON []byte
	
	err := api.db.QueryRow(ctx, query, eventID).Scan(
		&event.ID, &event.UserID, &event.EventType, &event.EventSubtype, &event.ClusterType,
		&event.DetectedAt, &event.EventDate, &event.EventDateFlex, &event.PlanningHorizon,
		&event.DetectionMethod, &event.DetectionConfidence, &signalsJSON,
		&event.Scale, &event.GuestCount, &locationJSON, &budgetJSON,
		&event.Status, &event.Phase, &event.CompletionPct,
		&prefsJSON, &constraintsJSON, &customJSON, &event.Tags,
		&event.CreatedAt, &event.UpdatedAt, &event.ConfirmedAt, &event.CompletedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(locationJSON, &event.Location)
	json.Unmarshal(budgetJSON, &event.Budget)
	json.Unmarshal(signalsJSON, &event.DetectionSignals)
	json.Unmarshal(prefsJSON, &event.Preferences)
	json.Unmarshal(constraintsJSON, &event.Constraints)
	json.Unmarshal(customJSON, &event.CustomAttributes)
	
	return &event, nil
}

func (api *LifeOSAPI) updateEvent(ctx context.Context, event *LifeEvent) error {
	query := `
		UPDATE life_events SET
			event_date = $2,
			event_date_flexibility = $3,
			scale = $4,
			guest_count = $5,
			location = $6,
			budget = $7,
			status = $8,
			phase = $9,
			completion_percentage = $10,
			preferences = $11,
			updated_at = $12,
			confirmed_at = $13
		WHERE id = $1
	`
	
	locationJSON, _ := json.Marshal(event.Location)
	budgetJSON, _ := json.Marshal(event.Budget)
	prefsJSON, _ := json.Marshal(event.Preferences)
	
	_, err := api.db.Exec(ctx, query,
		event.ID,
		event.EventDate, event.EventDateFlex,
		event.Scale, event.GuestCount, locationJSON, budgetJSON,
		event.Status, event.Phase, event.CompletionPct,
		prefsJSON, event.UpdatedAt, event.ConfirmedAt,
	)
	
	return err
}

/*
================================================================================
SECTION 3: BUSINESS MODEL
================================================================================

REVENUE STREAMS:

1. TRANSACTION FEES (Primary - 60% of revenue)
   - 8-15% commission on all bookings made through platform
   - Tiered by vendor category (higher for premium services)
   - Bundle commission: 12-18% (higher due to orchestration value)

2. SUBSCRIPTION - VENDORS (20% of revenue)
   - Free: Basic listing, 15% commission
   - Basic (₦10,000/mo): Enhanced listing, 12% commission, analytics
   - Pro (₦30,000/mo): Featured placement, 10% commission, leads, CRM tools
   - Enterprise: Custom pricing for chains/franchises

3. SUBSCRIPTION - CONSUMERS (10% of revenue)
   - Free: Basic event tracking, recommendations
   - Premium (₦5,000/mo): AI planner, priority support, exclusive discounts
   - Family (₦12,000/mo): Multiple events, shared planning, concierge

4. FINANCING (5% of revenue)
   - Event loans: Partner with fintech for event financing
   - BNPL: Pay-in-4 for large bookings
   - Interest spread: 2-4% of financed amount

5. DATA & INSIGHTS (5% of revenue)
   - Market intelligence reports for vendors
   - Trend analysis for event planners
   - API access for enterprise integrations

KEY METRICS:
- GMV (Gross Merchandise Value)
- Take Rate (Commission %)
- Events Created per User
- Event Completion Rate
- NPS (Net Promoter Score)
- Vendor Retention Rate
- Average Revenue per Event

================================================================================
SECTION 4: IMPLEMENTATION ROADMAP
================================================================================

PHASE 1: FOUNDATION (Months 1-3)
- Core event creation and tracking
- Manual service discovery and booking
- Basic vendor marketplace
- User authentication and profiles

PHASE 2: INTELLIGENCE (Months 4-6)
- Event detection engine
- Basic recommendation system
- Timeline and task management
- Budget tracking

PHASE 3: ORCHESTRATION (Months 7-9)
- Full orchestration engine
- Bundle creation and pricing
- Multi-vendor coordination
- Payment integration

PHASE 4: SCALE (Months 10-12)
- ML-powered predictions
- Partner integrations (calendar, social)
- Enterprise features
- API platform

================================================================================
*/

// Placeholder types for complete compilation
type RecommendationService struct{}
type BookingService struct{}
type NotificationService struct{}
type PricingEngine struct{}
type EventScheduler struct{}
type MLEventPredictor struct{}
type BookedService struct{}
type SuggestedBundle struct{}
