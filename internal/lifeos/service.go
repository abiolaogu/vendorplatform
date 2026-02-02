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
