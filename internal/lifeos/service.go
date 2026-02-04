// Package lifeos provides the service layer for life event detection and orchestration
package lifeos

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Service handles life event orchestration business logic
type Service struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewService creates a new LifeOS service instance
func NewService(db *pgxpool.Pool, cache *redis.Client) *Service {
	return &Service{
		db:    db,
		cache: cache,
	}
}

// LifeEvent represents a detected or declared life event
type LifeEvent struct {
	ID                  uuid.UUID              `json:"id"`
	UserID              uuid.UUID              `json:"user_id"`
	EventType           string                 `json:"event_type"`
	EventSubtype        string                 `json:"event_subtype,omitempty"`
	ClusterType         string                 `json:"cluster_type"`
	DetectedAt          time.Time              `json:"detected_at"`
	EventDate           *time.Time             `json:"event_date,omitempty"`
	EventDateFlex       string                 `json:"event_date_flexibility"`
	PlanningHorizonDays int                    `json:"planning_horizon_days"`
	DetectionMethod     string                 `json:"detection_method"`
	DetectionConfidence float64                `json:"detection_confidence"`
	Scale               string                 `json:"scale"`
	GuestCount          *int                   `json:"guest_count,omitempty"`
	Status              string                 `json:"status"`
	Phase               string                 `json:"phase"`
	CompletionPct       float64                `json:"completion_percentage"`
	CustomAttributes    map[string]interface{} `json:"custom_attributes,omitempty"`
	Tags                []string               `json:"tags,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	ConfirmedAt         *time.Time             `json:"confirmed_at,omitempty"`
}

// CreateLifeEventRequest represents a request to create a life event
type CreateLifeEventRequest struct {
	UserID           uuid.UUID              `json:"user_id"`
	EventType        string                 `json:"event_type"`
	EventSubtype     string                 `json:"event_subtype,omitempty"`
	EventDate        *time.Time             `json:"event_date,omitempty"`
	EventDateFlex    string                 `json:"event_date_flexibility,omitempty"`
	DetectionMethod  string                 `json:"detection_method,omitempty"`
	Scale            string                 `json:"scale,omitempty"`
	GuestCount       *int                   `json:"guest_count,omitempty"`
	CustomAttributes map[string]interface{} `json:"custom_attributes,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
}

// EventPlan represents an orchestration plan for a life event
type EventPlan struct {
	EventID           uuid.UUID      `json:"event_id"`
	EventType         string         `json:"event_type"`
	TotalCategories   int            `json:"total_categories"`
	BookedCategories  int            `json:"booked_categories"`
	CompletionPercent float64        `json:"completion_percent"`
	CurrentPhase      string         `json:"current_phase"`
	Phases            []PhasePlan    `json:"phases"`
	Timeline          []TimelineItem `json:"timeline"`
	BudgetSummary     *BudgetSummary `json:"budget_summary,omitempty"`
	NextActions       []NextAction   `json:"next_actions"`
}

// PhasePlan represents a planning phase
type PhasePlan struct {
	Phase          string           `json:"phase"`
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	StartOffset    int              `json:"start_offset_days"`
	EndOffset      int              `json:"end_offset_days"`
	Categories     []CategoryPlan   `json:"categories"`
	IsCompleted    bool             `json:"is_completed"`
	CompletionPct  float64          `json:"completion_percentage"`
}

// CategoryPlan represents a service category in the plan
type CategoryPlan struct {
	CategoryID      string  `json:"category_id"`
	CategoryName    string  `json:"category_name"`
	Priority        string  `json:"priority"`
	IsBooked        bool    `json:"is_booked"`
	BookedServiceID *string `json:"booked_service_id,omitempty"`
	BudgetAlloc     float64 `json:"budget_allocation_percentage"`
}

// TimelineItem represents a timeline entry
type TimelineItem struct {
	Date        time.Time `json:"date"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category,omitempty"`
	IsDeadline  bool      `json:"is_deadline"`
	IsCompleted bool      `json:"is_completed"`
}

// BudgetSummary represents budget information
type BudgetSummary struct {
	TotalBudget     float64 `json:"total_budget"`
	AllocatedAmount float64 `json:"allocated_amount"`
	SpentAmount     float64 `json:"spent_amount"`
	RemainingAmount float64 `json:"remaining_amount"`
	Currency        string  `json:"currency"`
}

// NextAction represents a suggested next action
type NextAction struct {
	ActionType  string    `json:"action_type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    int       `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CategoryID  *string   `json:"category_id,omitempty"`
}

// DetectedEvent represents an event detected by the system
type DetectedEvent struct {
	ID                  uuid.UUID   `json:"id"`
	UserID              uuid.UUID   `json:"user_id"`
	EventType           string      `json:"event_type"`
	DetectionMethod     string      `json:"detection_method"`
	DetectionConfidence float64     `json:"detection_confidence"`
	DetectedAt          time.Time   `json:"detected_at"`
	Signals             []Signal    `json:"signals"`
	IsConfirmed         bool        `json:"is_confirmed"`
}

// Signal represents a detection signal
type Signal struct {
	SignalType string    `json:"signal_type"`
	Source     string    `json:"source"`
	Value      string    `json:"value"`
	Confidence float64   `json:"confidence"`
	Timestamp  time.Time `json:"timestamp"`
}

// CreateLifeEvent creates a new life event
func (s *Service) CreateLifeEvent(ctx context.Context, req *CreateLifeEventRequest) (*LifeEvent, error) {
	// Validate event type
	if !isValidEventType(req.EventType) {
		return nil, fmt.Errorf("invalid event type: %s", req.EventType)
	}

	// Set defaults
	if req.DetectionMethod == "" {
		req.DetectionMethod = "explicit"
	}
	if req.EventDateFlex == "" {
		req.EventDateFlex = "flexible"
	}
	if req.Scale == "" {
		req.Scale = "medium"
	}

	// Determine cluster type based on event type
	clusterType := getClusterType(req.EventType)

	// Calculate planning horizon
	planningHorizon := calculatePlanningHorizon(req.EventDate, req.EventType)

	// Create event
	event := &LifeEvent{
		ID:                  uuid.New(),
		UserID:              req.UserID,
		EventType:           req.EventType,
		EventSubtype:        req.EventSubtype,
		ClusterType:         clusterType,
		DetectedAt:          time.Now(),
		EventDate:           req.EventDate,
		EventDateFlex:       req.EventDateFlex,
		PlanningHorizonDays: planningHorizon,
		DetectionMethod:     req.DetectionMethod,
		DetectionConfidence: 1.0, // Explicit creation has 100% confidence
		Scale:               req.Scale,
		GuestCount:          req.GuestCount,
		Status:              "confirmed",
		Phase:               "discovery",
		CompletionPct:       0.0,
		CustomAttributes:    req.CustomAttributes,
		Tags:                req.Tags,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	now := time.Now()
	event.ConfirmedAt = &now

	// Store in database
	query := `
		INSERT INTO life_events (
			id, user_id, event_type, event_subtype, cluster_type,
			detected_at, event_date, event_date_flexibility, planning_horizon_days,
			detection_method, detection_confidence, scale, guest_count,
			status, phase, completion_percentage, custom_attributes, tags,
			created_at, updated_at, confirmed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)
	`

	attrsJSON, _ := json.Marshal(event.CustomAttributes)
	tagsJSON, _ := json.Marshal(event.Tags)

	_, err := s.db.Exec(ctx, query,
		event.ID, event.UserID, event.EventType, event.EventSubtype, event.ClusterType,
		event.DetectedAt, event.EventDate, event.EventDateFlex, event.PlanningHorizonDays,
		event.DetectionMethod, event.DetectionConfidence, event.Scale, event.GuestCount,
		event.Status, event.Phase, event.CompletionPct, attrsJSON, tagsJSON,
		event.CreatedAt, event.UpdatedAt, event.ConfirmedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create life event: %w", err)
	}

	return event, nil
}

// GetLifeEvent retrieves a life event by ID
func (s *Service) GetLifeEvent(ctx context.Context, eventID uuid.UUID) (*LifeEvent, error) {
	query := `
		SELECT id, user_id, event_type, event_subtype, cluster_type,
		       detected_at, event_date, event_date_flexibility, planning_horizon_days,
		       detection_method, detection_confidence, scale, guest_count,
		       status, phase, completion_percentage, custom_attributes, tags,
		       created_at, updated_at, confirmed_at
		FROM life_events
		WHERE id = $1
	`

	event := &LifeEvent{}
	var attrsJSON, tagsJSON []byte

	err := s.db.QueryRow(ctx, query, eventID).Scan(
		&event.ID, &event.UserID, &event.EventType, &event.EventSubtype, &event.ClusterType,
		&event.DetectedAt, &event.EventDate, &event.EventDateFlex, &event.PlanningHorizonDays,
		&event.DetectionMethod, &event.DetectionConfidence, &event.Scale, &event.GuestCount,
		&event.Status, &event.Phase, &event.CompletionPct, &attrsJSON, &tagsJSON,
		&event.CreatedAt, &event.UpdatedAt, &event.ConfirmedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get life event: %w", err)
	}

	// Unmarshal JSON fields
	if len(attrsJSON) > 0 {
		json.Unmarshal(attrsJSON, &event.CustomAttributes)
	}
	if len(tagsJSON) > 0 {
		json.Unmarshal(tagsJSON, &event.Tags)
	}

	return event, nil
}

// GetEventPlan generates an orchestration plan for a life event
func (s *Service) GetEventPlan(ctx context.Context, eventID uuid.UUID) (*EventPlan, error) {
	// Get the event
	event, err := s.GetLifeEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}

	// Get categories for this event type
	categoriesQuery := `
		SELECT ecm.category_id, sc.name, ecm.role_type, ecm.phase,
		       ecm.necessity_score, ecm.typical_budget_percentage,
		       ecm.typical_booking_offset_days
		FROM event_category_mappings ecm
		JOIN service_categories sc ON sc.id = ecm.category_id
		JOIN life_event_triggers let ON let.id = ecm.event_trigger_id
		WHERE let.event_type = $1 AND ecm.is_active = TRUE
		ORDER BY ecm.phase, ecm.necessity_score DESC
	`

	rows, err := s.db.Query(ctx, categoriesQuery, event.EventType)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch event categories: %w", err)
	}
	defer rows.Close()

	// Group categories by phase
	phaseMap := make(map[string]*PhasePlan)
	totalCategories := 0

	for rows.Next() {
		var categoryID, categoryName, roleType, phase string
		var necessityScore, budgetPct float64
		var bookingOffset int

		if err := rows.Scan(&categoryID, &categoryName, &roleType, &phase,
			&necessityScore, &budgetPct, &bookingOffset); err != nil {
			continue
		}

		totalCategories++

		// Create phase if it doesn't exist
		if _, exists := phaseMap[phase]; !exists {
			phaseMap[phase] = &PhasePlan{
				Phase:       phase,
				Name:        getPhaseDisplayName(phase),
				Description: getPhaseDescription(phase),
				Categories:  []CategoryPlan{},
			}
		}

		// Add category to phase
		phaseMap[phase].Categories = append(phaseMap[phase].Categories, CategoryPlan{
			CategoryID:   categoryID,
			CategoryName: categoryName,
			Priority:     roleType,
			IsBooked:     false,
			BudgetAlloc:  budgetPct,
		})
	}

	// Convert phase map to sorted slice
	phases := []PhasePlan{}
	phaseOrder := []string{"discovery", "planning", "vendor_select", "booking", "pre_event", "event_day", "post_event"}
	for _, phaseName := range phaseOrder {
		if phase, exists := phaseMap[phaseName]; exists {
			phases = append(phases, *phase)
		}
	}

	// Generate timeline if event date is set
	timeline := []TimelineItem{}
	if event.EventDate != nil {
		timeline = generateTimeline(event, phases)
	}

	// Generate next actions
	nextActions := generateNextActions(event, phases)

	plan := &EventPlan{
		EventID:           event.ID,
		EventType:         event.EventType,
		TotalCategories:   totalCategories,
		BookedCategories:  0,
		CompletionPercent: 0.0,
		CurrentPhase:      event.Phase,
		Phases:            phases,
		Timeline:          timeline,
		NextActions:       nextActions,
	}

	return plan, nil
}

// ConfirmDetectedEvent confirms a detected event
func (s *Service) ConfirmDetectedEvent(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE life_events
		SET status = 'confirmed', confirmed_at = $1, updated_at = $1
		WHERE id = $2 AND user_id = $3 AND status = 'detected'
	`

	result, err := s.db.Exec(ctx, query, now, eventID, userID)
	if err != nil {
		return fmt.Errorf("failed to confirm event: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("event not found or already confirmed")
	}

	return nil
}

// GetDetectedEvents retrieves detected but unconfirmed events for a user
func (s *Service) GetDetectedEvents(ctx context.Context, userID uuid.UUID) ([]DetectedEvent, error) {
	query := `
		SELECT id, user_id, event_type, detection_method, detection_confidence,
		       detected_at, COALESCE(confirmed_at, NULL) IS NOT NULL as is_confirmed
		FROM life_events
		WHERE user_id = $1 AND status IN ('detected', 'confirmed')
		ORDER BY detected_at DESC
		LIMIT 10
	`

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch detected events: %w", err)
	}
	defer rows.Close()

	events := []DetectedEvent{}
	for rows.Next() {
		var event DetectedEvent
		if err := rows.Scan(&event.ID, &event.UserID, &event.EventType,
			&event.DetectionMethod, &event.DetectionConfidence, &event.DetectedAt,
			&event.IsConfirmed); err != nil {
			continue
		}
		event.Signals = []Signal{} // Would fetch from detection_signals table
		events = append(events, event)
	}

	return events, nil
}

// Helper functions

func isValidEventType(eventType string) bool {
	validTypes := map[string]bool{
		"wedding": true, "funeral": true, "birthday": true, "relocation": true,
		"renovation": true, "childbirth": true, "travel": true, "business_launch": true,
		"graduation": true, "retirement": true,
	}
	return validTypes[eventType]
}

func getClusterType(eventType string) string {
	clusterMap := map[string]string{
		"wedding":         "celebrations",
		"birthday":        "celebrations",
		"graduation":      "celebrations",
		"relocation":      "home",
		"renovation":      "home",
		"childbirth":      "health",
		"travel":          "travel",
		"business_launch": "business",
		"retirement":      "celebrations",
		"funeral":         "celebrations",
	}
	if cluster, exists := clusterMap[eventType]; exists {
		return cluster
	}
	return "celebrations"
}

func calculatePlanningHorizon(eventDate *time.Time, eventType string) int {
	if eventDate == nil {
		return 90 // Default 3 months
	}

	daysUntil := int(time.Until(*eventDate).Hours() / 24)

	// Minimum planning horizons by event type
	minimums := map[string]int{
		"wedding":         180, // 6 months
		"relocation":      60,  // 2 months
		"renovation":      90,  // 3 months
		"business_launch": 120, // 4 months
		"default":         30,  // 1 month
	}

	minimum := minimums["default"]
	if m, exists := minimums[eventType]; exists {
		minimum = m
	}

	if daysUntil < minimum {
		return minimum
	}
	return daysUntil
}

func getPhaseDisplayName(phase string) string {
	names := map[string]string{
		"discovery":     "Discovery",
		"planning":      "Planning",
		"vendor_select": "Vendor Selection",
		"booking":       "Booking",
		"pre_event":     "Pre-Event",
		"event_day":     "Event Day",
		"post_event":    "Post-Event",
	}
	if name, exists := names[phase]; exists {
		return name
	}
	return phase
}

func getPhaseDescription(phase string) string {
	descriptions := map[string]string{
		"discovery":     "Understanding your needs and preferences",
		"planning":      "Building your service list and budget",
		"vendor_select": "Choosing the right vendors",
		"booking":       "Confirming bookings and contracts",
		"pre_event":     "Final preparations and confirmations",
		"event_day":     "Day of the event coordination",
		"post_event":    "Follow-up and reviews",
	}
	if desc, exists := descriptions[phase]; exists {
		return desc
	}
	return ""
}

func generateTimeline(event *LifeEvent, phases []PhasePlan) []TimelineItem {
	if event.EventDate == nil {
		return []TimelineItem{}
	}

	timeline := []TimelineItem{}
	eventDate := *event.EventDate

	// Add major milestones
	timeline = append(timeline, TimelineItem{
		Date:        eventDate.AddDate(0, 0, -90),
		Title:       "Start Planning",
		Description: "Begin organizing your " + event.EventType,
		IsDeadline:  true,
		IsCompleted: false,
	})

	timeline = append(timeline, TimelineItem{
		Date:        eventDate.AddDate(0, 0, -30),
		Title:       "Final Bookings",
		Description: "Complete all vendor bookings",
		IsDeadline:  true,
		IsCompleted: false,
	})

	timeline = append(timeline, TimelineItem{
		Date:        eventDate.AddDate(0, 0, -7),
		Title:       "Final Confirmations",
		Description: "Confirm all details with vendors",
		IsDeadline:  true,
		IsCompleted: false,
	})

	timeline = append(timeline, TimelineItem{
		Date:        eventDate,
		Title:       event.EventType + " Day",
		Description: "Your special day",
		IsDeadline:  false,
		IsCompleted: false,
	})

	return timeline
}

func generateNextActions(event *LifeEvent, phases []PhasePlan) []NextAction {
	actions := []NextAction{}

	if event.Phase == "discovery" {
		actions = append(actions, NextAction{
			ActionType:  "set_budget",
			Title:       "Set Your Budget",
			Description: "Define how much you want to spend",
			Priority:    1,
		})

		actions = append(actions, NextAction{
			ActionType:  "select_vendors",
			Title:       "Browse Vendors",
			Description: "Explore vendors for your event",
			Priority:    2,
		})
	}

	return actions
}

// ActivitySignal represents user activity for event detection
type ActivitySignal struct {
	UserID     uuid.UUID              `json:"user_id"`
	SignalType string                 `json:"signal_type"` // search, browse, bookmark, inquiry
	Category   string                 `json:"category,omitempty"`
	Keywords   []string               `json:"keywords,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// DetectionResult represents the result of event detection
type DetectionResult struct {
	DetectedEvents []DetectedEvent `json:"detected_events"`
	TotalSignals   int             `json:"total_signals"`
	AnalyzedPeriod string          `json:"analyzed_period"`
}

// BundleOpportunity represents a service bundling recommendation
type BundleOpportunity struct {
	BundleID         string         `json:"bundle_id"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	Categories       []string       `json:"categories"`
	TotalCategories  int            `json:"total_categories"`
	EstimatedSavings float64        `json:"estimated_savings"`
	SavingsPercent   float64        `json:"savings_percentage"`
	Priority         int            `json:"priority"`
	VendorPackages   []VendorBundle `json:"vendor_packages,omitempty"`
}

// VendorBundle represents a vendor offering a service bundle
type VendorBundle struct {
	VendorID       uuid.UUID `json:"vendor_id"`
	VendorName     string    `json:"vendor_name"`
	BundlePrice    float64   `json:"bundle_price"`
	IndividualSum  float64   `json:"individual_sum"`
	DiscountAmount float64   `json:"discount_amount"`
	DiscountPercent float64  `json:"discount_percentage"`
}

// RiskAssessment represents identified risks for an event
type RiskAssessment struct {
	EventID      uuid.UUID    `json:"event_id"`
	OverallRisk  string       `json:"overall_risk"` // low, medium, high, critical
	RiskScore    float64      `json:"risk_score"`   // 0-100
	Risks        []Risk       `json:"risks"`
	Mitigations  []Mitigation `json:"mitigations"`
	AssessedAt   time.Time    `json:"assessed_at"`
}

// Risk represents a specific risk
type Risk struct {
	RiskType    string  `json:"risk_type"` // timeline, budget, vendor, weather, logistics
	Severity    string  `json:"severity"`  // low, medium, high, critical
	Probability float64 `json:"probability"` // 0-1
	Impact      float64 `json:"impact"`      // 0-1
	Description string  `json:"description"`
	AffectedCategories []string `json:"affected_categories,omitempty"`
}

// Mitigation represents a risk mitigation strategy
type Mitigation struct {
	RiskType    string `json:"risk_type"`
	Strategy    string `json:"strategy"`
	Priority    int    `json:"priority"`
	Description string `json:"description"`
	ActionItems []string `json:"action_items"`
}

// BudgetOptimization represents optimized budget allocation
type BudgetOptimization struct {
	EventID             uuid.UUID                  `json:"event_id"`
	TotalBudget         float64                    `json:"total_budget"`
	OptimizedAllocation map[string]CategoryBudget  `json:"optimized_allocation"`
	SavingsOpportunities []SavingsOpportunity      `json:"savings_opportunities"`
	TotalPotentialSavings float64                  `json:"total_potential_savings"`
	RecommendedChanges  []BudgetChange             `json:"recommended_changes"`
}

// CategoryBudget represents budget for a service category
type CategoryBudget struct {
	CategoryID          string  `json:"category_id"`
	CategoryName        string  `json:"category_name"`
	CurrentAllocation   float64 `json:"current_allocation"`
	RecommendedAllocation float64 `json:"recommended_allocation"`
	MarketAverage       float64 `json:"market_average"`
	Priority            string  `json:"priority"`
}

// SavingsOpportunity represents a way to save money
type SavingsOpportunity struct {
	OpportunityType string  `json:"opportunity_type"` // bundle, timing, vendor_negotiation, alternative
	Description     string  `json:"description"`
	EstimatedSavings float64 `json:"estimated_savings"`
	Categories      []string `json:"categories,omitempty"`
}

// BudgetChange represents a recommended budget change
type BudgetChange struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	CurrentAmount float64 `json:"current_amount"`
	RecommendedAmount float64 `json:"recommended_amount"`
	Change       float64 `json:"change"`
	ChangePercent float64 `json:"change_percentage"`
	Reason       string  `json:"reason"`
}

// DetectLifeEvents analyzes user activity signals to detect potential life events
func (s *Service) DetectLifeEvents(ctx context.Context, userID uuid.UUID, lookbackDays int) (*DetectionResult, error) {
	if lookbackDays <= 0 {
		lookbackDays = 30 // Default to last 30 days
	}

	// Query user activity signals from the last N days
	// In a real implementation, this would query search history, browsing patterns, etc.
	// For now, we'll simulate pattern detection

	signals := []ActivitySignal{} // Would be fetched from activity tracking tables

	// Event detection patterns
	eventPatterns := map[string][]string{
		"wedding": {"wedding", "venue", "catering", "photography", "bride", "groom", "marriage"},
		"relocation": {"moving", "relocation", "packing", "truck", "new home", "apartment"},
		"renovation": {"renovation", "remodeling", "contractor", "construction", "home improvement"},
		"childbirth": {"baby", "maternity", "pediatrician", "nursery", "pregnancy"},
		"birthday": {"birthday", "party", "celebration", "cake", "decorations"},
	}

	// Analyze signals and detect patterns
	detectedEvents := []DetectedEvent{}
	eventScores := make(map[string]float64)
	eventSignals := make(map[string][]Signal)

	// Pattern matching logic (simplified)
	for eventType, keywords := range eventPatterns {
		score := 0.0
		matchedSignals := []Signal{}

		// In real implementation, would analyze actual user signals
		// This is a placeholder for the detection logic

		if score > 0.5 { // Confidence threshold
			event := DetectedEvent{
				ID:                  uuid.New(),
				UserID:              userID,
				EventType:           eventType,
				DetectionMethod:     "behavioral",
				DetectionConfidence: score,
				DetectedAt:          time.Now(),
				Signals:             matchedSignals,
				IsConfirmed:         false,
			}
			detectedEvents = append(detectedEvents, event)
		}
	}

	// Store detected events in database
	for _, event := range detectedEvents {
		query := `
			INSERT INTO life_events (
				id, user_id, event_type, cluster_type, detected_at,
				detection_method, detection_confidence, status, phase,
				completion_percentage, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`

		clusterType := getClusterType(event.EventType)

		_, err := s.db.Exec(ctx, query,
			event.ID, userID, event.EventType, clusterType, time.Now(),
			"behavioral", event.DetectionConfidence, "detected", "discovery",
			0.0, time.Now(), time.Now(),
		)
		if err != nil {
			continue // Log error but continue processing
		}

		// Store detection signals
		for _, signal := range event.Signals {
			signalQuery := `
				INSERT INTO life_event_detection_signals (
					event_id, signal_type, source, value, confidence, timestamp
				) VALUES ($1, $2, $3, $4, $5, $6)
			`
			s.db.Exec(ctx, signalQuery,
				event.ID, signal.SignalType, signal.Source, signal.Value,
				signal.Confidence, signal.Timestamp,
			)
		}
	}

	result := &DetectionResult{
		DetectedEvents: detectedEvents,
		TotalSignals:   len(signals),
		AnalyzedPeriod: fmt.Sprintf("Last %d days", lookbackDays),
	}

	return result, nil
}

// GenerateBundleRecommendations identifies multi-service bundle opportunities
func (s *Service) GenerateBundleRecommendations(ctx context.Context, eventID uuid.UUID) ([]BundleOpportunity, error) {
	// Get the event
	event, err := s.GetLifeEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}

	// Get the event plan to see required categories
	plan, err := s.GetEventPlan(ctx, eventID)
	if err != nil {
		return nil, err
	}

	bundles := []BundleOpportunity{}

	// Bundle 1: Core Services Bundle (highest priority categories)
	coreCategories := []string{}
	for _, phase := range plan.Phases {
		for _, cat := range phase.Categories {
			if cat.Priority == "primary" && !cat.IsBooked {
				coreCategories = append(coreCategories, cat.CategoryName)
			}
			if len(coreCategories) >= 5 {
				break
			}
		}
	}

	if len(coreCategories) >= 3 {
		// Calculate estimated savings (10-15% for bundling 3+ services)
		savingsPercent := 10.0 + float64(len(coreCategories)-3)*1.5
		if savingsPercent > 20 {
			savingsPercent = 20 // Cap at 20%
		}

		bundles = append(bundles, BundleOpportunity{
			BundleID:         fmt.Sprintf("bundle-%s-core", event.EventType),
			Name:             fmt.Sprintf("%s Essential Bundle", capitalizeFirst(event.EventType)),
			Description:      "Bundle your core services together for significant savings",
			Categories:       coreCategories,
			TotalCategories:  len(coreCategories),
			EstimatedSavings: 0, // Would calculate based on actual vendor pricing
			SavingsPercent:   savingsPercent,
			Priority:         1,
		})
	}

	// Bundle 2: Full Package (all unbooked categories)
	allCategories := []string{}
	for _, phase := range plan.Phases {
		for _, cat := range phase.Categories {
			if !cat.IsBooked {
				allCategories = append(allCategories, cat.CategoryName)
			}
		}
	}

	if len(allCategories) >= 5 {
		savingsPercent := 15.0 + float64(len(allCategories)-5)*1.0
		if savingsPercent > 25 {
			savingsPercent = 25
		}

		bundles = append(bundles, BundleOpportunity{
			BundleID:         fmt.Sprintf("bundle-%s-complete", event.EventType),
			Name:             fmt.Sprintf("Complete %s Package", capitalizeFirst(event.EventType)),
			Description:      "Book all services together for maximum savings",
			Categories:       allCategories,
			TotalCategories:  len(allCategories),
			EstimatedSavings: 0,
			SavingsPercent:   savingsPercent,
			Priority:         2,
		})
	}

	// Bundle 3: Category-specific bundles (e.g., "Entertainment Bundle", "Food & Beverage Bundle")
	categoryGroups := map[string][]string{
		"Entertainment": {"DJ/Music", "Photography", "Videography", "Entertainment"},
		"Food & Beverage": {"Catering", "Bartending", "Cake", "Food Service"},
		"Decor & Setup": {"Decoration", "Flowers", "Lighting", "Venue Setup"},
	}

	for groupName, groupCategories := range categoryGroups {
		matchedCategories := []string{}
		for _, phase := range plan.Phases {
			for _, cat := range phase.Categories {
				for _, groupCat := range groupCategories {
					if contains(cat.CategoryName, groupCat) && !cat.IsBooked {
						matchedCategories = append(matchedCategories, cat.CategoryName)
						break
					}
				}
			}
		}

		if len(matchedCategories) >= 2 {
			bundles = append(bundles, BundleOpportunity{
				BundleID:         fmt.Sprintf("bundle-%s-%s", event.EventType, slugify(groupName)),
				Name:             fmt.Sprintf("%s Bundle", groupName),
				Description:      fmt.Sprintf("Bundle %s services for better coordination and savings", groupName),
				Categories:       matchedCategories,
				TotalCategories:  len(matchedCategories),
				EstimatedSavings: 0,
				SavingsPercent:   8.0 + float64(len(matchedCategories))*2.0,
				Priority:         3,
			})
		}
	}

	return bundles, nil
}

// AssessEventRisks analyzes potential risks for a life event
func (s *Service) AssessEventRisks(ctx context.Context, eventID uuid.UUID) (*RiskAssessment, error) {
	// Get the event
	event, err := s.GetLifeEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}

	// Get the event plan
	plan, err := s.GetEventPlan(ctx, eventID)
	if err != nil {
		return nil, err
	}

	risks := []Risk{}
	mitigations := []Mitigation{}
	totalRiskScore := 0.0
	riskCount := 0

	// Risk 1: Timeline Risk
	if event.EventDate != nil {
		daysUntilEvent := int(time.Until(*event.EventDate).Hours() / 24)

		if daysUntilEvent < 30 {
			severity := "critical"
			probability := 0.9
			if daysUntilEvent < 14 {
				probability = 1.0
			}

			risks = append(risks, Risk{
				RiskType:    "timeline",
				Severity:    severity,
				Probability: probability,
				Impact:      0.9,
				Description: fmt.Sprintf("Only %d days until event - very tight timeline for booking vendors", daysUntilEvent),
			})

			mitigations = append(mitigations, Mitigation{
				RiskType: "timeline",
				Strategy: "Expedited Booking",
				Priority: 1,
				Description: "Prioritize critical vendors immediately",
				ActionItems: []string{
					"Book venue in next 48 hours",
					"Contact top 3 vendors for each category today",
					"Consider premium/rush fees",
				},
			})

			totalRiskScore += probability * 0.9 * 100
			riskCount++
		} else if daysUntilEvent < 90 {
			risks = append(risks, Risk{
				RiskType:    "timeline",
				Severity:    "medium",
				Probability: 0.6,
				Impact:      0.5,
				Description: fmt.Sprintf("%d days until event - should start booking soon", daysUntilEvent),
			})

			totalRiskScore += 0.6 * 0.5 * 100
			riskCount++
		}
	}

	// Risk 2: Budget Risk (unallocated budget)
	if plan.BudgetSummary != nil {
		remainingPct := (plan.BudgetSummary.RemainingAmount / plan.BudgetSummary.TotalBudget) * 100

		if remainingPct > 70 {
			risks = append(risks, Risk{
				RiskType:    "budget",
				Severity:    "medium",
				Probability: 0.7,
				Impact:      0.6,
				Description: fmt.Sprintf("%.0f%% of budget unallocated - may lead to last-minute overspending", remainingPct),
			})

			mitigations = append(mitigations, Mitigation{
				RiskType: "budget",
				Strategy: "Budget Planning",
				Priority: 2,
				Description: "Allocate budget to categories proactively",
				ActionItems: []string{
					"Set budget limits for each category",
					"Get quotes from 3+ vendors per category",
					"Build in 10% contingency buffer",
				},
			})

			totalRiskScore += 0.7 * 0.6 * 100
			riskCount++
		}
	}

	// Risk 3: Vendor Availability Risk
	unbookedCritical := 0
	criticalCategories := []string{}
	for _, phase := range plan.Phases {
		for _, cat := range phase.Categories {
			if cat.Priority == "primary" && !cat.IsBooked {
				unbookedCritical++
				criticalCategories = append(criticalCategories, cat.CategoryName)
			}
		}
	}

	if unbookedCritical > 0 {
		severity := "high"
		if unbookedCritical >= 3 {
			severity = "critical"
		}

		risks = append(risks, Risk{
			RiskType:    "vendor",
			Severity:    severity,
			Probability: 0.8,
			Impact:      0.9,
			Description: fmt.Sprintf("%d critical vendors not yet booked", unbookedCritical),
			AffectedCategories: criticalCategories,
		})

		mitigations = append(mitigations, Mitigation{
			RiskType: "vendor",
			Strategy: "Immediate Vendor Outreach",
			Priority: 1,
			Description: "Contact and book critical vendors urgently",
			ActionItems: []string{
				"Request quotes from top vendors today",
				"Check availability for your date",
				"Have backup vendor list ready",
			},
		})

		totalRiskScore += 0.8 * 0.9 * 100
		riskCount++
	}

	// Risk 4: Completion Risk
	if plan.CompletionPercent < 20 && event.EventDate != nil {
		daysUntilEvent := int(time.Until(*event.EventDate).Hours() / 24)
		if daysUntilEvent < 60 {
			risks = append(risks, Risk{
				RiskType:    "logistics",
				Severity:    "high",
				Probability: 0.75,
				Impact:      0.7,
				Description: fmt.Sprintf("Only %.0f%% complete with %d days remaining", plan.CompletionPercent, daysUntilEvent),
			})

			totalRiskScore += 0.75 * 0.7 * 100
			riskCount++
		}
	}

	// Calculate overall risk
	avgRiskScore := 0.0
	if riskCount > 0 {
		avgRiskScore = totalRiskScore / float64(riskCount)
	}

	overallRisk := "low"
	if avgRiskScore > 70 {
		overallRisk = "critical"
	} else if avgRiskScore > 50 {
		overallRisk = "high"
	} else if avgRiskScore > 30 {
		overallRisk = "medium"
	}

	assessment := &RiskAssessment{
		EventID:     eventID,
		OverallRisk: overallRisk,
		RiskScore:   avgRiskScore,
		Risks:       risks,
		Mitigations: mitigations,
		AssessedAt:  time.Now(),
	}

	return assessment, nil
}

// OptimizeBudgetAllocation provides optimized budget allocation recommendations
func (s *Service) OptimizeBudgetAllocation(ctx context.Context, eventID uuid.UUID, totalBudget float64) (*BudgetOptimization, error) {
	// Get the event
	event, err := s.GetLifeEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}

	// Get the event plan
	plan, err := s.GetEventPlan(ctx, eventID)
	if err != nil {
		return nil, err
	}

	if totalBudget <= 0 {
		return nil, fmt.Errorf("total budget must be positive")
	}

	// Build optimized allocation based on category priorities and typical percentages
	optimizedAllocation := make(map[string]CategoryBudget)
	totalAllocatedPct := 0.0

	for _, phase := range plan.Phases {
		for _, cat := range phase.Categories {
			if cat.IsBooked {
				continue // Skip already booked categories
			}

			// Use typical budget percentage from database, or default
			budgetPct := cat.BudgetAlloc
			if budgetPct == 0 {
				// Default allocations by priority
				switch cat.Priority {
				case "primary":
					budgetPct = 15.0
				case "secondary":
					budgetPct = 10.0
				case "optional":
					budgetPct = 5.0
				default:
					budgetPct = 8.0
				}
			}

			allocatedAmount := totalBudget * (budgetPct / 100.0)

			optimizedAllocation[cat.CategoryID] = CategoryBudget{
				CategoryID:            cat.CategoryID,
				CategoryName:          cat.CategoryName,
				CurrentAllocation:     0, // Would fetch from current budget
				RecommendedAllocation: allocatedAmount,
				MarketAverage:         allocatedAmount, // Would fetch from market data
				Priority:              cat.Priority,
			}

			totalAllocatedPct += budgetPct
		}
	}

	// Identify savings opportunities
	savingsOpportunities := []SavingsOpportunity{}

	// Opportunity 1: Bundle discount
	if len(optimizedAllocation) >= 3 {
		bundleSavings := totalBudget * 0.12 // 12% average bundle savings
		savingsOpportunities = append(savingsOpportunities, SavingsOpportunity{
			OpportunityType:  "bundle",
			Description:      "Book 3+ services together to receive bundle discount",
			EstimatedSavings: bundleSavings,
		})
	}

	// Opportunity 2: Early booking discount
	if event.EventDate != nil {
		daysUntilEvent := int(time.Until(*event.EventDate).Hours() / 24)
		if daysUntilEvent > 90 {
			earlyBookingSavings := totalBudget * 0.08 // 8% early booking discount
			savingsOpportunities = append(savingsOpportunities, SavingsOpportunity{
				OpportunityType:  "timing",
				Description:      "Book now for early-bird discounts (3+ months in advance)",
				EstimatedSavings: earlyBookingSavings,
			})
		}
	}

	// Opportunity 3: Alternative vendors
	alternativeSavings := totalBudget * 0.15 // 15% by choosing budget-friendly alternatives
	savingsOpportunities = append(savingsOpportunities, SavingsOpportunity{
		OpportunityType:  "alternative",
		Description:      "Consider budget-friendly vendor alternatives without compromising quality",
		EstimatedSavings: alternativeSavings,
	})

	totalPotentialSavings := 0.0
	for _, opp := range savingsOpportunities {
		totalPotentialSavings += opp.EstimatedSavings
	}

	// Generate recommended changes
	recommendedChanges := []BudgetChange{}
	for _, catBudget := range optimizedAllocation {
		change := catBudget.RecommendedAllocation - catBudget.CurrentAllocation
		if change != 0 {
			changePercent := 0.0
			if catBudget.CurrentAllocation > 0 {
				changePercent = (change / catBudget.CurrentAllocation) * 100
			}

			recommendedChanges = append(recommendedChanges, BudgetChange{
				CategoryID:        catBudget.CategoryID,
				CategoryName:      catBudget.CategoryName,
				CurrentAmount:     catBudget.CurrentAllocation,
				RecommendedAmount: catBudget.RecommendedAllocation,
				Change:            change,
				ChangePercent:     changePercent,
				Reason:            fmt.Sprintf("Optimized for %s priority", catBudget.Priority),
			})
		}
	}

	optimization := &BudgetOptimization{
		EventID:               eventID,
		TotalBudget:           totalBudget,
		OptimizedAllocation:   optimizedAllocation,
		SavingsOpportunities:  savingsOpportunities,
		TotalPotentialSavings: totalPotentialSavings,
		RecommendedChanges:    recommendedChanges,
	}

	return optimization, nil
}

// Helper functions for new features

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && haystack[:len(needle)] == needle
}

func slugify(s string) string {
	// Simple slugify - convert spaces to hyphens and lowercase
	result := ""
	for _, r := range s {
		if r == ' ' {
			result += "-"
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if r >= 'A' && r <= 'Z' {
				result += string(r + 32) // Convert to lowercase
			} else {
				result += string(r)
			}
		}
	}
	return result
}
