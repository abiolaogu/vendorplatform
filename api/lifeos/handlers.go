// Package lifeos provides HTTP handlers for the LifeOS platform
package lifeos

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// LifeOSAPI provides the REST API interface for LifeOS
type LifeOSAPI struct {
	detectionEngine     *EventDetectionEngine
	orchestrationEngine *OrchestrationEngine
	db                  *pgxpool.Pool
	cache               *redis.Client
	logger              *slog.Logger
}

// NewLifeOSAPI creates a new LifeOS API instance
func NewLifeOSAPI(db *pgxpool.Pool, cache *redis.Client, zapLogger *zap.Logger) *LifeOSAPI {
	// Create slog logger
	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Initialize detection engine
	detectionConfig := &DetectionConfig{
		MinConfidenceThreshold: 0.6,
		SignalWindowDays:       30,
		EnableMLPrediction:     false, // Disabled for now
		EnableCalendarSync:     false,
		EnablePartnerData:      false,
	}

	detectionEngine := &EventDetectionEngine{
		db:     db,
		cache:  cache,
		config: detectionConfig,
		signalProcessors: make(map[DetectionMethod]SignalProcessor),
	}

	// Register signal processors
	detectionEngine.signalProcessors[DetectionBehavioral] = &BehavioralSignalProcessor{db: db}

	// Initialize orchestration engine
	orchestrationEngine := &OrchestrationEngine{
		db:    db,
		cache: cache,
	}

	return &LifeOSAPI{
		detectionEngine:     detectionEngine,
		orchestrationEngine: orchestrationEngine,
		db:                  db,
		cache:               cache,
		logger:              slogLogger,
	}
}

// GetEvent retrieves a life event by ID
func (api *LifeOSAPI) GetEvent(ctx context.Context, eventID uuid.UUID) (*LifeEvent, error) {
	return api.loadEvent(ctx, eventID)
}

// GetDetectedEvents returns events detected for a user
func (api *LifeOSAPI) GetDetectedEvents(ctx context.Context, userID uuid.UUID) ([]LifeEvent, error) {
	// First, try to detect new events
	detectedEvents, err := api.detectionEngine.DetectEvents(ctx, userID)
	if err != nil {
		api.logger.Error("Failed to detect events", "error", err, "user_id", userID)
		// Continue to return existing detected events even if detection fails
	}

	// Save newly detected events that aren't already in the database
	for i := range detectedEvents {
		// Check if this event type is already detected recently
		exists, err := api.eventTypeExistsRecently(ctx, userID, detectedEvents[i].EventType)
		if err != nil || exists {
			continue
		}

		// Save the new detection
		if err := api.saveEvent(ctx, &detectedEvents[i]); err != nil {
			api.logger.Error("Failed to save detected event", "error", err)
		}
	}

	// Return all detected events (not yet confirmed) for this user
	return api.loadUserDetectedEvents(ctx, userID)
}

// eventTypeExistsRecently checks if an event type was detected recently (last 30 days)
func (api *LifeOSAPI) eventTypeExistsRecently(ctx context.Context, userID uuid.UUID, eventType EventType) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM life_events
			WHERE user_id = $1
			  AND event_type = $2
			  AND detected_at > NOW() - INTERVAL '30 days'
		)
	`
	var exists bool
	err := api.db.QueryRow(ctx, query, userID, eventType).Scan(&exists)
	return exists, err
}

// loadUserDetectedEvents loads all detected events for a user
func (api *LifeOSAPI) loadUserDetectedEvents(ctx context.Context, userID uuid.UUID) ([]LifeEvent, error) {
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
		WHERE user_id = $1
		  AND status IN ('detected', 'confirmed', 'planning')
		ORDER BY detection_confidence DESC, detected_at DESC
		LIMIT 10
	`

	rows, err := api.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []LifeEvent
	for rows.Next() {
		event, err := api.scanLifeEvent(rows)
		if err != nil {
			api.logger.Error("Failed to scan life event", "error", err)
			continue
		}
		events = append(events, *event)
	}

	return events, nil
}

// scanLifeEvent is a helper to scan a row into a LifeEvent struct
func (api *LifeOSAPI) scanLifeEvent(scanner interface {
	Scan(dest ...interface{}) error
}) (*LifeEvent, error) {
	var event LifeEvent
	var locationJSON, budgetJSON, signalsJSON, prefsJSON, constraintsJSON, customJSON []byte

	err := scanner.Scan(
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

	// Unmarshal JSON fields
	if len(locationJSON) > 0 {
		var loc Location
		if err := json.Unmarshal(locationJSON, &loc); err == nil {
			event.Location = &loc
		}
	}

	if len(budgetJSON) > 0 {
		var budget Budget
		if err := json.Unmarshal(budgetJSON, &budget); err == nil {
			event.Budget = &budget
		}
	}

	if len(signalsJSON) > 0 {
		json.Unmarshal(signalsJSON, &event.DetectionSignals)
	}

	if len(prefsJSON) > 0 {
		json.Unmarshal(prefsJSON, &event.Preferences)
	}

	if len(constraintsJSON) > 0 {
		json.Unmarshal(constraintsJSON, &event.Constraints)
	}

	if len(customJSON) > 0 {
		json.Unmarshal(customJSON, &event.CustomAttributes)
	}

	return &event, nil
}
