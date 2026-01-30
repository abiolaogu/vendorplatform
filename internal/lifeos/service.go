// Package lifeos provides life event orchestration business logic
package lifeos

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
	ErrEventNotFound    = errors.New("life event not found")
	ErrInvalidEventData = errors.New("invalid event data")
	ErrEventExists      = errors.New("event already exists")
	ErrUnauthorized     = errors.New("unauthorized")
)

// Service handles life event orchestration operations
type Service struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewService creates a new lifeos service
func NewService(db *pgxpool.Pool, cache *redis.Client) *Service {
	return &Service{
		db:    db,
		cache: cache,
	}
}

// EventType represents the type of life event
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

// EventStatus represents the status of a life event
type EventStatus string

const (
	StatusDetected   EventStatus = "detected"
	StatusConfirmed  EventStatus = "confirmed"
	StatusPlanning   EventStatus = "planning"
	StatusBooked     EventStatus = "booked"
	StatusInProgress EventStatus = "in_progress"
	StatusCompleted  EventStatus = "completed"
	StatusCancelled  EventStatus = "cancelled"
)

// EventPhase represents the current phase of event planning
type EventPhase string

const (
	PhaseDiscovery    EventPhase = "discovery"
	PhasePlanning     EventPhase = "planning"
	PhaseVendorSelect EventPhase = "vendor_select"
	PhaseBooking      EventPhase = "booking"
	PhasePreEvent     EventPhase = "pre_event"
	PhaseEventDay     EventPhase = "event_day"
	PhasePostEvent    EventPhase = "post_event"
)

// LifeEvent represents a detected or declared life event
type LifeEvent struct {
	ID                  uuid.UUID              `json:"id"`
	UserID              uuid.UUID              `json:"user_id"`
	EventType           EventType              `json:"event_type"`
	EventSubtype        string                 `json:"event_subtype,omitempty"`
	ClusterType         string                 `json:"cluster_type"`
	Status              EventStatus            `json:"status"`
	Phase               EventPhase             `json:"phase"`
	DetectionMethod     string                 `json:"detection_method"`
	DetectionConfidence float64                `json:"detection_confidence"`
	EventDate           *time.Time             `json:"event_date,omitempty"`
	GuestCount          *int                   `json:"guest_count,omitempty"`
	Budget              *float64               `json:"budget,omitempty"`
	Currency            string                 `json:"currency,omitempty"`
	Latitude            *float64               `json:"latitude,omitempty"`
	Longitude           *float64               `json:"longitude,omitempty"`
	CompletionPct       float64                `json:"completion_percentage"`
	CustomAttributes    map[string]interface{} `json:"custom_attributes,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	ConfirmedAt         *time.Time             `json:"confirmed_at,omitempty"`
	CompletedAt         *time.Time             `json:"completed_at,omitempty"`
}

// EventOrchestrationPlan represents the orchestration plan for a life event
type EventOrchestrationPlan struct {
	EventID          uuid.UUID           `json:"event_id"`
	ServicePlan      []PlannedService    `json:"service_plan"`
	Timeline         []TimelinePhase     `json:"timeline"`
	BudgetBreakdown  []BudgetItem        `json:"budget_breakdown"`
	CriticalPath     []string            `json:"critical_path"`
	SuggestedBundles []SuggestedBundle   `json:"suggested_bundles,omitempty"`
	Risks            []IdentifiedRisk    `json:"risks,omitempty"`
	NextActions      []RecommendedAction `json:"next_actions"`
	GeneratedAt      time.Time           `json:"generated_at"`
}

// PlannedService represents a service in the event plan
type PlannedService struct {
	CategoryID       uuid.UUID  `json:"category_id"`
	CategoryName     string     `json:"category_name"`
	Priority         string     `json:"priority"`
	Phase            EventPhase `json:"phase"`
	Status           string     `json:"status"`
	BudgetAllocation float64    `json:"budget_allocation"`
	BookByDate       *time.Time `json:"book_by_date,omitempty"`
}

// TimelinePhase represents a phase in the event timeline
type TimelinePhase struct {
	Phase       EventPhase `json:"phase"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     time.Time  `json:"end_date"`
	Status      string     `json:"status"`
	Tasks       []string   `json:"tasks"`
}

// BudgetItem represents a budget allocation item
type BudgetItem struct {
	CategoryID   uuid.UUID `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Allocated    float64   `json:"allocated"`
	Spent        float64   `json:"spent"`
	Percentage   float64   `json:"percentage"`
	Status       string    `json:"status"`
}

// SuggestedBundle represents a bundle recommendation
type SuggestedBundle struct {
	BundleID        uuid.UUID   `json:"bundle_id"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	IncludedItems   []uuid.UUID `json:"included_items"`
	TotalPrice      float64     `json:"total_price"`
	DiscountPercent float64     `json:"discount_percent"`
	Savings         float64     `json:"savings"`
}

// IdentifiedRisk represents a potential risk
type IdentifiedRisk struct {
	Type             string   `json:"type"`
	Description      string   `json:"description"`
	Severity         string   `json:"severity"`
	MitigationSteps  []string `json:"mitigation_steps"`
}

// RecommendedAction represents a recommended next action
type RecommendedAction struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	ActionType  string     `json:"action_type"`
}

// DetectedEvent represents a detected but unconfirmed event
type DetectedEvent struct {
	ID                  uuid.UUID              `json:"id"`
	UserID              uuid.UUID              `json:"user_id"`
	EventType           EventType              `json:"event_type"`
	DetectionMethod     string                 `json:"detection_method"`
	DetectionConfidence float64                `json:"detection_confidence"`
	DetectionSignals    []DetectionSignal      `json:"detection_signals"`
	DetectedAt          time.Time              `json:"detected_at"`
	CustomAttributes    map[string]interface{} `json:"custom_attributes,omitempty"`
}

// DetectionSignal represents evidence for event detection
type DetectionSignal struct {
	SignalType string    `json:"signal_type"`
	Source     string    `json:"source"`
	Value      string    `json:"value"`
	Confidence float64   `json:"confidence"`
	Timestamp  time.Time `json:"timestamp"`
}

// CreateEventRequest represents a request to create a life event
type CreateEventRequest struct {
	UserID         uuid.UUID              `json:"user_id"`
	EventType      EventType              `json:"event_type"`
	EventSubtype   string                 `json:"event_subtype,omitempty"`
	EventDate      *time.Time             `json:"event_date,omitempty"`
	GuestCount     *int                   `json:"guest_count,omitempty"`
	Budget         *float64               `json:"budget,omitempty"`
	Currency       string                 `json:"currency,omitempty"`
	Latitude       *float64               `json:"latitude,omitempty"`
	Longitude      *float64               `json:"longitude,omitempty"`
	CustomAttributes map[string]interface{} `json:"custom_attributes,omitempty"`
}

// CreateEvent creates a new life event
func (s *Service) CreateEvent(ctx context.Context, req *CreateEventRequest) (*LifeEvent, error) {
	if req.UserID == uuid.Nil || req.EventType == "" {
		return nil, ErrInvalidEventData
	}

	event := &LifeEvent{
		ID:                  uuid.New(),
		UserID:              req.UserID,
		EventType:           req.EventType,
		EventSubtype:        req.EventSubtype,
		ClusterType:         getClusterType(req.EventType),
		Status:              StatusConfirmed,
		Phase:               PhaseDiscovery,
		DetectionMethod:     "explicit",
		DetectionConfidence: 1.0,
		EventDate:           req.EventDate,
		GuestCount:          req.GuestCount,
		Budget:              req.Budget,
		Currency:            req.Currency,
		Latitude:            req.Latitude,
		Longitude:           req.Longitude,
		CompletionPct:       0.0,
		CustomAttributes:    req.CustomAttributes,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Get event trigger ID
	var eventTriggerID uuid.UUID
	err := s.db.QueryRow(ctx, `
		SELECT id FROM life_event_triggers
		WHERE slug = $1 AND is_active = TRUE
	`, string(req.EventType)).Scan(&eventTriggerID)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("event type not supported: %s", req.EventType)
		}
		return nil, fmt.Errorf("failed to get event trigger: %w", err)
	}

	// Insert into projects table (life events are treated as projects)
	err = s.db.QueryRow(ctx, `
		INSERT INTO projects (
			id, user_id, event_trigger_id, name, event_type, event_date,
			total_budget, expected_guests, status, completion_percentage,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`,
		event.ID,
		event.UserID,
		eventTriggerID,
		fmt.Sprintf("My %s", req.EventType),
		string(req.EventType),
		req.EventDate,
		req.Budget,
		req.GuestCount,
		string(event.Status),
		event.CompletionPct,
		event.CreatedAt,
		event.UpdatedAt,
	).Scan(&event.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return event, nil
}

// GetEvent retrieves a life event by ID
func (s *Service) GetEvent(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) (*LifeEvent, error) {
	var event LifeEvent
	var eventTriggerID uuid.UUID

	err := s.db.QueryRow(ctx, `
		SELECT
			p.id, p.user_id, p.event_type, p.event_date, p.status,
			p.total_budget, p.expected_guests, p.completion_percentage,
			p.created_at, p.updated_at, p.event_trigger_id
		FROM projects p
		WHERE p.id = $1 AND p.user_id = $2
	`, eventID, userID).Scan(
		&event.ID,
		&event.UserID,
		&event.EventType,
		&event.EventDate,
		&event.Status,
		&event.Budget,
		&event.GuestCount,
		&event.CompletionPct,
		&event.CreatedAt,
		&event.UpdatedAt,
		&eventTriggerID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrEventNotFound
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	event.ClusterType = getClusterType(event.EventType)
	event.Phase = PhaseDiscovery
	event.DetectionMethod = "explicit"
	event.DetectionConfidence = 1.0
	event.Currency = "NGN"

	return &event, nil
}

// GetEventPlan generates an orchestration plan for a life event
func (s *Service) GetEventPlan(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) (*EventOrchestrationPlan, error) {
	// Verify event exists and belongs to user
	event, err := s.GetEvent(ctx, eventID, userID)
	if err != nil {
		return nil, err
	}

	plan := &EventOrchestrationPlan{
		EventID:     event.ID,
		GeneratedAt: time.Now(),
	}

	// Get required categories for this event type
	rows, err := s.db.Query(ctx, `
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
		WHERE let.slug = $1 AND ecm.is_active = TRUE
		ORDER BY ecm.necessity_score DESC, ecm.typical_booking_offset_days DESC
	`, string(event.EventType))

	if err != nil {
		return nil, fmt.Errorf("failed to get service plan: %w", err)
	}
	defer rows.Close()

	var services []PlannedService
	var budgetItems []BudgetItem
	criticalPath := []string{}

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
			s.Priority = "critical"
			criticalPath = append(criticalPath, s.CategoryName)
		case "secondary":
			s.Priority = "high"
		case "optional":
			s.Priority = "medium"
		default:
			s.Priority = "low"
		}

		s.Phase = EventPhase(phase)
		s.Status = "pending"
		s.BudgetAllocation = budgetPct

		// Calculate book-by date if event date is set
		if event.EventDate != nil {
			bookByDate := event.EventDate.AddDate(0, 0, -bookingOffset)
			s.BookByDate = &bookByDate
		}

		services = append(services, s)

		// Add budget item
		if event.Budget != nil {
			allocated := *event.Budget * budgetPct / 100.0
			budgetItems = append(budgetItems, BudgetItem{
				CategoryID:   s.CategoryID,
				CategoryName: s.CategoryName,
				Allocated:    allocated,
				Spent:        0,
				Percentage:   budgetPct,
				Status:       "on_track",
			})
		}
	}

	plan.ServicePlan = services
	plan.BudgetBreakdown = budgetItems
	plan.CriticalPath = criticalPath

	// Generate timeline phases
	if event.EventDate != nil {
		plan.Timeline = generateTimeline(*event.EventDate)
	}

	// Generate next actions
	plan.NextActions = generateNextActions(services, event)

	return plan, nil
}

// ConfirmDetectedEvent confirms a detected event
func (s *Service) ConfirmDetectedEvent(ctx context.Context, detectedEventID uuid.UUID, userID uuid.UUID) (*LifeEvent, error) {
	// For now, this is a placeholder
	// In a full implementation, this would retrieve the detected event and convert it to a confirmed event
	return nil, fmt.Errorf("not implemented")
}

// GetDetectedEvents retrieves detected but unconfirmed events for a user
func (s *Service) GetDetectedEvents(ctx context.Context, userID uuid.UUID) ([]DetectedEvent, error) {
	// For now, return empty list
	// In a full implementation, this would query detected events from behavioral analysis
	return []DetectedEvent{}, nil
}

// Helper functions

func getClusterType(eventType EventType) string {
	switch eventType {
	case EventTypeWedding, EventTypeFuneral, EventTypeBirthday, EventTypeGraduation, EventTypeRetirement:
		return "celebrations"
	case EventTypeRelocation, EventTypeRenovation:
		return "home"
	case EventTypeTravel:
		return "travel"
	case EventTypeChildbirth:
		return "health"
	case EventTypeBusinessLaunch:
		return "business"
	default:
		return "other"
	}
}

func generateTimeline(eventDate time.Time) []TimelinePhase {
	now := time.Now()
	phases := []TimelinePhase{
		{
			Phase:       PhaseDiscovery,
			Name:        "Discovery",
			Description: "Understanding requirements and setting goals",
			StartDate:   now,
			EndDate:     now.AddDate(0, 0, 7),
			Status:      "active",
			Tasks:       []string{"Define budget", "Set guest count", "Choose location"},
		},
		{
			Phase:       PhasePlanning,
			Name:        "Planning",
			Description: "Creating comprehensive service list",
			StartDate:   now.AddDate(0, 0, 7),
			EndDate:     now.AddDate(0, 0, 21),
			Status:      "pending",
			Tasks:       []string{"Review service categories", "Get vendor quotes", "Compare options"},
		},
		{
			Phase:       PhaseVendorSelect,
			Name:        "Vendor Selection",
			Description: "Choosing and booking vendors",
			StartDate:   now.AddDate(0, 0, 21),
			EndDate:     eventDate.AddDate(0, 0, -30),
			Status:      "pending",
			Tasks:       []string{"Book critical vendors", "Finalize contracts", "Pay deposits"},
		},
		{
			Phase:       PhasePreEvent,
			Name:        "Pre-Event",
			Description: "Final preparations and confirmations",
			StartDate:   eventDate.AddDate(0, 0, -30),
			EndDate:     eventDate.AddDate(0, 0, -1),
			Status:      "pending",
			Tasks:       []string{"Confirm all bookings", "Finalize details", "Final payments"},
		},
		{
			Phase:       PhaseEventDay,
			Name:        "Event Day",
			Description: "The big day",
			StartDate:   eventDate,
			EndDate:     eventDate,
			Status:      "pending",
			Tasks:       []string{"Coordinate vendors", "Manage timeline", "Enjoy the moment"},
		},
	}
	return phases
}

func generateNextActions(services []PlannedService, event *LifeEvent) []RecommendedAction {
	actions := []RecommendedAction{}

	// Add first 3 critical services as next actions
	for i, s := range services {
		if i >= 3 {
			break
		}
		if s.Priority == "critical" {
			action := RecommendedAction{
				Title:       fmt.Sprintf("Book %s", s.CategoryName),
				Description: fmt.Sprintf("Find and book a %s vendor for your %s", s.CategoryName, event.EventType),
				Priority:    s.Priority,
				DueDate:     s.BookByDate,
				ActionType:  "book",
			}
			actions = append(actions, action)
		}
	}

	// Add budget setup if budget not set
	if event.Budget == nil {
		actions = append([]RecommendedAction{{
			Title:       "Set Your Budget",
			Description: "Define your total budget to get personalized recommendations",
			Priority:    "high",
			ActionType:  "configure",
		}}, actions...)
	}

	return actions
}
