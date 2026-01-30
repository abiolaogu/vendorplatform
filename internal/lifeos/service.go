// Package lifeos provides core business logic for the LifeOS platform
package lifeos

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/api/lifeos"
)

// Service provides business logic for LifeOS operations
type Service struct {
	db     *pgxpool.Pool
	cache  *redis.Client
	logger *zap.Logger
}

// NewService creates a new LifeOS service
func NewService(db *pgxpool.Pool, cache *redis.Client, logger *zap.Logger) *Service {
	return &Service{
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

// CreateLifeEvent creates a new life event
func (s *Service) CreateLifeEvent(ctx context.Context, req *CreateEventRequest) (*lifeos.LifeEvent, error) {
	event := &lifeos.LifeEvent{
		ID:                  uuid.New(),
		UserID:              req.UserID,
		EventType:           req.EventType,
		EventSubtype:        req.EventSubtype,
		ClusterType:         getClusterForEventType(req.EventType),
		DetectedAt:          time.Now(),
		EventDate:           req.EventDate,
		EventDateFlex:       req.DateFlexibility,
		DetectionMethod:     lifeos.DetectionExplicit,
		DetectionConfidence: 1.0,
		Status:              lifeos.StatusConfirmed,
		Phase:               lifeos.PhaseDiscovery,
		CompletionPct:       0.0,
		Scale:               req.Scale,
		GuestCount:          req.GuestCount,
		Location:            req.Location,
		Budget:              req.Budget,
		Preferences:         req.Preferences,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Calculate planning horizon
	if event.EventDate != nil {
		event.PlanningHorizon = int(time.Until(*event.EventDate).Hours() / 24)
	}

	// Generate required services based on event type
	event.RequiredServices = s.generateRequiredServices(event.EventType, event.Scale)

	// Store in database
	query := `
		INSERT INTO lifeos_events (
			id, user_id, event_type, event_subtype, cluster_type,
			detected_at, event_date, date_flexibility, planning_horizon_days,
			detection_method, detection_confidence, status, phase,
			completion_percentage, scale, guest_count, location,
			budget, preferences, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)`

	_, err := s.db.Exec(ctx, query,
		event.ID, event.UserID, event.EventType, event.EventSubtype, event.ClusterType,
		event.DetectedAt, event.EventDate, event.EventDateFlex, event.PlanningHorizon,
		event.DetectionMethod, event.DetectionConfidence, event.Status, event.Phase,
		event.CompletionPct, event.Scale, event.GuestCount, event.Location,
		event.Budget, event.Preferences, event.CreatedAt, event.UpdatedAt,
	)
	if err != nil {
		s.logger.Error("Failed to create life event", zap.Error(err))
		return nil, fmt.Errorf("failed to create life event: %w", err)
	}

	s.logger.Info("Created life event",
		zap.String("event_id", event.ID.String()),
		zap.String("event_type", string(event.EventType)),
		zap.String("user_id", event.UserID.String()),
	)

	return event, nil
}

// GetLifeEvent retrieves a life event by ID
func (s *Service) GetLifeEvent(ctx context.Context, eventID uuid.UUID) (*lifeos.LifeEvent, error) {
	event := &lifeos.LifeEvent{}

	query := `
		SELECT id, user_id, event_type, event_subtype, cluster_type,
			detected_at, event_date, date_flexibility, planning_horizon_days,
			detection_method, detection_confidence, status, phase,
			completion_percentage, scale, guest_count, location,
			budget, preferences, created_at, updated_at,
			confirmed_at, completed_at
		FROM lifeos_events
		WHERE id = $1`

	err := s.db.QueryRow(ctx, query, eventID).Scan(
		&event.ID, &event.UserID, &event.EventType, &event.EventSubtype, &event.ClusterType,
		&event.DetectedAt, &event.EventDate, &event.EventDateFlex, &event.PlanningHorizon,
		&event.DetectionMethod, &event.DetectionConfidence, &event.Status, &event.Phase,
		&event.CompletionPct, &event.Scale, &event.GuestCount, &event.Location,
		&event.Budget, &event.Preferences, &event.CreatedAt, &event.UpdatedAt,
		&event.ConfirmedAt, &event.CompletedAt,
	)
	if err != nil {
		s.logger.Error("Failed to get life event", zap.Error(err), zap.String("event_id", eventID.String()))
		return nil, fmt.Errorf("failed to get life event: %w", err)
	}

	return event, nil
}

// GetEventPlan generates a comprehensive orchestration plan for the event
func (s *Service) GetEventPlan(ctx context.Context, eventID uuid.UUID) (*EventPlan, error) {
	event, err := s.GetLifeEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}

	plan := &EventPlan{
		EventID:          eventID,
		EventType:        event.EventType,
		CurrentPhase:     event.Phase,
		CompletionPct:    event.CompletionPct,
		RequiredServices: event.RequiredServices,
		Timeline:         s.generateTimeline(event),
		Milestones:       s.generateMilestones(event),
		NextActions:      s.getNextActions(event),
		GeneratedAt:      time.Now(),
	}

	return plan, nil
}

// ConfirmDetectedEvent confirms a system-detected event
func (s *Service) ConfirmDetectedEvent(ctx context.Context, eventID uuid.UUID, confirmed bool) error {
	now := time.Now()
	newStatus := lifeos.StatusConfirmed
	if !confirmed {
		newStatus = lifeos.StatusCancelled
	}

	query := `
		UPDATE lifeos_events
		SET status = $1, confirmed_at = $2, updated_at = $3
		WHERE id = $4 AND status = $5`

	result, err := s.db.Exec(ctx, query, newStatus, now, now, eventID, lifeos.StatusDetected)
	if err != nil {
		s.logger.Error("Failed to confirm detected event", zap.Error(err))
		return fmt.Errorf("failed to confirm event: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("event not found or not in detected state")
	}

	s.logger.Info("Confirmed detected event",
		zap.String("event_id", eventID.String()),
		zap.Bool("confirmed", confirmed),
	)

	return nil
}

// GetDetectedEvents retrieves detected but unconfirmed events for a user
func (s *Service) GetDetectedEvents(ctx context.Context, userID uuid.UUID) ([]*lifeos.LifeEvent, error) {
	query := `
		SELECT id, user_id, event_type, event_subtype, cluster_type,
			detected_at, event_date, date_flexibility, planning_horizon_days,
			detection_method, detection_confidence, status, phase,
			completion_percentage, scale, guest_count, location,
			budget, preferences, created_at, updated_at
		FROM lifeos_events
		WHERE user_id = $1 AND status = $2
		ORDER BY detection_confidence DESC, detected_at DESC
		LIMIT 10`

	rows, err := s.db.Query(ctx, query, userID, lifeos.StatusDetected)
	if err != nil {
		s.logger.Error("Failed to get detected events", zap.Error(err))
		return nil, fmt.Errorf("failed to get detected events: %w", err)
	}
	defer rows.Close()

	var events []*lifeos.LifeEvent
	for rows.Next() {
		event := &lifeos.LifeEvent{}
		err := rows.Scan(
			&event.ID, &event.UserID, &event.EventType, &event.EventSubtype, &event.ClusterType,
			&event.DetectedAt, &event.EventDate, &event.EventDateFlex, &event.PlanningHorizon,
			&event.DetectionMethod, &event.DetectionConfidence, &event.Status, &event.Phase,
			&event.CompletionPct, &event.Scale, &event.GuestCount, &event.Location,
			&event.Budget, &event.Preferences, &event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan event", zap.Error(err))
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// Helper functions

func getClusterForEventType(eventType lifeos.EventType) lifeos.ClusterType {
	clusterMap := map[lifeos.EventType]lifeos.ClusterType{
		lifeos.EventTypeWedding:        lifeos.ClusterCelebrations,
		lifeos.EventTypeBirthday:       lifeos.ClusterCelebrations,
		lifeos.EventTypeGraduation:     lifeos.ClusterCelebrations,
		lifeos.EventTypeRelocation:     lifeos.ClusterHome,
		lifeos.EventTypeRenovation:     lifeos.ClusterHome,
		lifeos.EventTypeTravel:         lifeos.ClusterTravel,
		lifeos.EventTypeBusinessLaunch: lifeos.ClusterBusiness,
		lifeos.EventTypeRetirement:     lifeos.ClusterCelebrations,
		lifeos.EventTypeFuneral:        lifeos.ClusterCelebrations,
		lifeos.EventTypeChildbirth:     lifeos.ClusterHealth,
	}

	if cluster, ok := clusterMap[eventType]; ok {
		return cluster
	}
	return lifeos.ClusterCelebrations
}

func (s *Service) generateRequiredServices(eventType lifeos.EventType, scale lifeos.EventScale) []lifeos.RequiredService {
	// Simplified service generation - in production, this would be more sophisticated
	services := []lifeos.RequiredService{}

	switch eventType {
	case lifeos.EventTypeWedding:
		services = append(services,
			lifeos.RequiredService{
				ID:               uuid.New(),
				CategoryName:     "Venue",
				Priority:         lifeos.PriorityCritical,
				IsRequired:       true,
				Phase:            lifeos.PhaseVendorSelect,
				DeadlineDays:     90,
				BudgetAllocation: 30.0,
				Status:           lifeos.RequirementPending,
			},
			lifeos.RequiredService{
				ID:               uuid.New(),
				CategoryName:     "Catering",
				Priority:         lifeos.PriorityCritical,
				IsRequired:       true,
				Phase:            lifeos.PhaseVendorSelect,
				DeadlineDays:     60,
				BudgetAllocation: 25.0,
				Status:           lifeos.RequirementPending,
			},
			lifeos.RequiredService{
				ID:               uuid.New(),
				CategoryName:     "Photography",
				Priority:         lifeos.PriorityHigh,
				IsRequired:       true,
				Phase:            lifeos.PhaseVendorSelect,
				DeadlineDays:     45,
				BudgetAllocation: 15.0,
				Status:           lifeos.RequirementPending,
			},
		)
	}

	return services
}

func (s *Service) generateTimeline(event *lifeos.LifeEvent) []TimelineItem {
	timeline := []TimelineItem{}

	if event.EventDate == nil {
		return timeline
	}

	// Generate key milestones based on event type
	eventDate := *event.EventDate
	timeline = append(timeline, TimelineItem{
		Date:        eventDate.AddDate(0, 0, -90),
		Phase:       lifeos.PhaseVendorSelect,
		Title:       "Start vendor selection",
		Description: "Begin researching and contacting vendors",
		IsComplete:  false,
	})

	timeline = append(timeline, TimelineItem{
		Date:        eventDate.AddDate(0, 0, -30),
		Phase:       lifeos.PhaseBooking,
		Title:       "Finalize all bookings",
		Description: "Confirm all vendors and make final payments",
		IsComplete:  false,
	})

	timeline = append(timeline, TimelineItem{
		Date:        eventDate.AddDate(0, 0, -7),
		Phase:       lifeos.PhasePreEvent,
		Title:       "Final preparations",
		Description: "Confirm details with all vendors",
		IsComplete:  false,
	})

	return timeline
}

func (s *Service) generateMilestones(event *lifeos.LifeEvent) []Milestone {
	// Simplified milestone generation
	return []Milestone{
		{
			Title:       "Event Created",
			Description: "Your event has been created and planning has begun",
			IsComplete:  true,
			CompletedAt: &event.CreatedAt,
		},
		{
			Title:       "Services Identified",
			Description: "Required services have been identified",
			IsComplete:  len(event.RequiredServices) > 0,
		},
		{
			Title:       "Vendors Selected",
			Description: "All critical vendors have been selected",
			IsComplete:  false,
		},
	}
}

func (s *Service) getNextActions(event *lifeos.LifeEvent) []string {
	actions := []string{}

	switch event.Phase {
	case lifeos.PhaseDiscovery:
		actions = append(actions, "Review your event requirements")
		actions = append(actions, "Set your budget and preferences")
	case lifeos.PhasePlanning:
		actions = append(actions, "Review recommended services")
		actions = append(actions, "Browse vendor options")
	case lifeos.PhaseVendorSelect:
		actions = append(actions, "Contact shortlisted vendors")
		actions = append(actions, "Compare quotes and reviews")
	}

	return actions
}

// Request/Response types

type CreateEventRequest struct {
	UserID           uuid.UUID
	EventType        lifeos.EventType
	EventSubtype     string
	EventDate        *time.Time
	DateFlexibility  lifeos.DateFlexibility
	Scale            lifeos.EventScale
	GuestCount       *int
	Location         *lifeos.Location
	Budget           *lifeos.Budget
	Preferences      lifeos.EventPreferences
}

type EventPlan struct {
	EventID          uuid.UUID                  `json:"event_id"`
	EventType        lifeos.EventType           `json:"event_type"`
	CurrentPhase     lifeos.EventPhase          `json:"current_phase"`
	CompletionPct    float64                    `json:"completion_percentage"`
	RequiredServices []lifeos.RequiredService   `json:"required_services"`
	Timeline         []TimelineItem             `json:"timeline"`
	Milestones       []Milestone                `json:"milestones"`
	NextActions      []string                   `json:"next_actions"`
	GeneratedAt      time.Time                  `json:"generated_at"`
}

type TimelineItem struct {
	Date        time.Time          `json:"date"`
	Phase       lifeos.EventPhase  `json:"phase"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	IsComplete  bool               `json:"is_complete"`
}

type Milestone struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	IsComplete  bool       `json:"is_complete"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
