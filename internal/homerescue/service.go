// Package homerescue provides emergency home services business logic
package homerescue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Error definitions
var (
	ErrEmergencyNotFound      = errors.New("emergency not found")
	ErrInvalidRequest         = errors.New("invalid request")
	ErrNoTechniciansAvailable = errors.New("no technicians available")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrInvalidUrgency         = errors.New("invalid urgency level")
	ErrSLABreach              = errors.New("SLA deadline breached")
)

// Service handles HomeRescue business logic
type Service struct {
	db     *pgxpool.Pool
	cache  *redis.Client
	logger *zap.Logger
}

// NewService creates a new HomeRescue service
func NewService(db *pgxpool.Pool, cache *redis.Client, logger *zap.Logger) *Service {
	return &Service{
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

// Emergency represents an emergency service request
type Emergency struct {
	ID                 uuid.UUID  `json:"id"`
	UserID             uuid.UUID  `json:"user_id"`
	Category           string     `json:"category"`
	Subcategory        string     `json:"subcategory"`
	Urgency            string     `json:"urgency"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	Address            string     `json:"address"`
	Unit               string     `json:"unit,omitempty"`
	City               string     `json:"city"`
	State              string     `json:"state"`
	PostalCode         string     `json:"postal_code"`
	Latitude           float64    `json:"latitude"`
	Longitude          float64    `json:"longitude"`
	AccessInstructions string     `json:"access_instructions,omitempty"`
	Status             string     `json:"status"`
	AssignedVendorID   *uuid.UUID `json:"assigned_vendor_id,omitempty"`
	AssignedTechID     *uuid.UUID `json:"assigned_tech_id,omitempty"`
	TechLatitude       *float64   `json:"tech_latitude,omitempty"`
	TechLongitude      *float64   `json:"tech_longitude,omitempty"`
	EstimatedArrival   *time.Time `json:"estimated_arrival,omitempty"`
	ActualArrival      *time.Time `json:"actual_arrival,omitempty"`
	ResponseDeadline   time.Time  `json:"response_deadline"`
	ArrivalDeadline    time.Time  `json:"arrival_deadline"`
	EstimatedCost      *float64   `json:"estimated_cost,omitempty"`
	FinalCost          *float64   `json:"final_cost,omitempty"`
	WorkPerformed      string     `json:"work_performed,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
}

// CreateEmergencyRequest represents a request to create an emergency
type CreateEmergencyRequest struct {
	UserID             uuid.UUID `json:"user_id"`
	Category           string    `json:"category"`
	Subcategory        string    `json:"subcategory"`
	Urgency            string    `json:"urgency"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Address            string    `json:"address"`
	Unit               string    `json:"unit,omitempty"`
	City               string    `json:"city"`
	State              string    `json:"state"`
	PostalCode         string    `json:"postal_code"`
	Latitude           float64   `json:"latitude"`
	Longitude          float64   `json:"longitude"`
	AccessInstructions string    `json:"access_instructions,omitempty"`
}

// EmergencyStatus represents the status information of an emergency
type EmergencyStatus struct {
	EmergencyID       uuid.UUID  `json:"emergency_id"`
	Status            string     `json:"status"`
	AssignedTechID    *uuid.UUID `json:"assigned_tech_id,omitempty"`
	AssignedTechName  string     `json:"assigned_tech_name,omitempty"`
	AssignedTechPhone string     `json:"assigned_tech_phone,omitempty"`
	ResponseDeadline  time.Time  `json:"response_deadline"`
	ArrivalDeadline   time.Time  `json:"arrival_deadline"`
	EstimatedArrival  *time.Time `json:"estimated_arrival,omitempty"`
	SLAStatus         string     `json:"sla_status"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// TechLocation represents a technician's current location
type TechLocation struct {
	TechID    uuid.UUID `json:"tech_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}

// EmergencyTracking represents real-time tracking information
type EmergencyTracking struct {
	EmergencyID       uuid.UUID     `json:"emergency_id"`
	Status            string        `json:"status"`
	TechLocation      *TechLocation `json:"tech_location,omitempty"`
	CustomerLocation  *GeoPoint     `json:"customer_location"`
	EstimatedArrival  *time.Time    `json:"estimated_arrival,omitempty"`
	DistanceRemaining *float64      `json:"distance_remaining_km,omitempty"`
	TimeRemaining     *int          `json:"time_remaining_minutes,omitempty"`
	SLAStatus         string        `json:"sla_status"`
}

// GeoPoint represents a geographic coordinate
type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// TechnicianAvailability represents technician availability information
type TechnicianAvailability struct {
	TechID           uuid.UUID       `json:"tech_id"`
	Category         string          `json:"category"`
	IsAvailable      bool            `json:"is_available"`
	CurrentJobs      int             `json:"current_jobs"`
	MaxConcurrentJobs int            `json:"max_concurrent_jobs"`
	Latitude         *float64        `json:"latitude,omitempty"`
	Longitude        *float64        `json:"longitude,omitempty"`
	AvailableSlots   json.RawMessage `json:"available_slots,omitempty"`
}

// SLAMetrics represents SLA compliance metrics
type SLAMetrics struct {
	EmergencyID          uuid.UUID  `json:"emergency_id"`
	ResponseTimeSLA      int        `json:"response_time_sla_minutes"`
	ActualResponseTime   *int       `json:"actual_response_time_minutes,omitempty"`
	ArrivalTimeSLA       int        `json:"arrival_time_sla_minutes"`
	ActualArrivalTime    *int       `json:"actual_arrival_time_minutes,omitempty"`
	SLAStatus            string     `json:"sla_status"`
	RefundPercentage     int        `json:"refund_percentage"`
	RefundAmount         *float64   `json:"refund_amount,omitempty"`
	RefundProcessed      bool       `json:"refund_processed"`
}

// Response time SLAs in minutes based on urgency
var responseSLAMinutes = map[string]int{
	"critical":  30,   // < 30 minutes
	"urgent":    120,  // < 2 hours
	"same_day":  360,  // < 6 hours
	"scheduled": 1440, // < 24 hours
}

// Refund percentages for SLA breaches
var slaRefundPercentages = map[string]int{
	"critical":  100, // 100% refund if missed
	"urgent":    50,  // 50% refund if missed
	"same_day":  25,  // 25% discount if missed
	"scheduled": 0,   // No refund
}

// =============================================================================
// EMERGENCY CREATION AND MANAGEMENT
// =============================================================================

// CreateEmergency creates a new emergency request and starts technician matching
func (s *Service) CreateEmergency(ctx context.Context, req *CreateEmergencyRequest) (*Emergency, error) {
	// Validate request
	if req.UserID == uuid.Nil || req.Category == "" || req.Title == "" {
		return nil, ErrInvalidRequest
	}

	// Validate urgency level
	slaMinutes, ok := responseSLAMinutes[req.Urgency]
	if !ok {
		return nil, ErrInvalidUrgency
	}

	emergency := &Emergency{
		ID:                 uuid.New(),
		UserID:             req.UserID,
		Category:           req.Category,
		Subcategory:        req.Subcategory,
		Urgency:            req.Urgency,
		Title:              req.Title,
		Description:        req.Description,
		Address:            req.Address,
		Unit:               req.Unit,
		City:               req.City,
		State:              req.State,
		PostalCode:         req.PostalCode,
		Latitude:           req.Latitude,
		Longitude:          req.Longitude,
		AccessInstructions: req.AccessInstructions,
		Status:             "new",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Calculate SLA deadlines
	emergency.ResponseDeadline = emergency.CreatedAt.Add(time.Duration(slaMinutes) * time.Minute)
	emergency.ArrivalDeadline = emergency.ResponseDeadline.Add(30 * time.Minute)

	query := `
		INSERT INTO emergencies (
			id, user_id, category, subcategory, urgency, title, description,
			address, unit, city, state, postal_code, latitude, longitude,
			access_instructions, status, response_deadline, arrival_deadline,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	_, err := s.db.Exec(ctx, query,
		emergency.ID, emergency.UserID, emergency.Category, emergency.Subcategory,
		emergency.Urgency, emergency.Title, emergency.Description, emergency.Address,
		emergency.Unit, emergency.City, emergency.State, emergency.PostalCode,
		emergency.Latitude, emergency.Longitude, emergency.AccessInstructions,
		emergency.Status, emergency.ResponseDeadline, emergency.ArrivalDeadline,
		emergency.CreatedAt, emergency.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Failed to create emergency", zap.Error(err))
		return nil, fmt.Errorf("failed to create emergency: %w", err)
	}

	// Initialize SLA metrics
	if err := s.initializeSLAMetrics(ctx, emergency); err != nil {
		s.logger.Error("Failed to initialize SLA metrics", zap.Error(err))
	}

	// Cache emergency for real-time updates
	s.cacheEmergency(ctx, emergency.ID, "new")

	s.logger.Info("Emergency created",
		zap.String("emergency_id", emergency.ID.String()),
		zap.String("category", emergency.Category),
		zap.String("urgency", emergency.Urgency),
	)

	// Start async technician matching
	go s.matchTechnician(context.Background(), emergency.ID)

	return emergency, nil
}

// GetEmergency retrieves an emergency by ID
func (s *Service) GetEmergency(ctx context.Context, id uuid.UUID) (*Emergency, error) {
	query := `
		SELECT id, user_id, category, subcategory, urgency, title, description,
		       address, unit, city, state, postal_code, latitude, longitude,
		       access_instructions, status, assigned_vendor_id, assigned_tech_id,
		       tech_latitude, tech_longitude, estimated_arrival, actual_arrival,
		       response_deadline, arrival_deadline, estimated_cost, final_cost,
		       work_performed, created_at, updated_at, completed_at
		FROM emergencies WHERE id = $1
	`

	emergency := &Emergency{}
	err := s.db.QueryRow(ctx, query, id).Scan(
		&emergency.ID, &emergency.UserID, &emergency.Category, &emergency.Subcategory,
		&emergency.Urgency, &emergency.Title, &emergency.Description, &emergency.Address,
		&emergency.Unit, &emergency.City, &emergency.State, &emergency.PostalCode,
		&emergency.Latitude, &emergency.Longitude, &emergency.AccessInstructions,
		&emergency.Status, &emergency.AssignedVendorID, &emergency.AssignedTechID,
		&emergency.TechLatitude, &emergency.TechLongitude, &emergency.EstimatedArrival,
		&emergency.ActualArrival, &emergency.ResponseDeadline, &emergency.ArrivalDeadline,
		&emergency.EstimatedCost, &emergency.FinalCost, &emergency.WorkPerformed,
		&emergency.CreatedAt, &emergency.UpdatedAt, &emergency.CompletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrEmergencyNotFound
	}
	if err != nil {
		s.logger.Error("Failed to get emergency", zap.Error(err))
		return nil, fmt.Errorf("failed to get emergency: %w", err)
	}

	return emergency, nil
}

// GetEmergencyStatus retrieves the current status of an emergency
func (s *Service) GetEmergencyStatus(ctx context.Context, emergencyID uuid.UUID) (*EmergencyStatus, error) {
	query := `
		SELECT e.id, e.status, e.assigned_tech_id, e.response_deadline,
		       e.arrival_deadline, e.estimated_arrival, e.created_at, e.updated_at,
		       u.full_name, u.phone
		FROM emergencies e
		LEFT JOIN users u ON u.id = e.assigned_tech_id
		WHERE e.id = $1
	`

	var status EmergencyStatus
	var techName, techPhone *string
	var estimatedArrival *time.Time

	err := s.db.QueryRow(ctx, query, emergencyID).Scan(
		&status.EmergencyID, &status.Status, &status.AssignedTechID,
		&status.ResponseDeadline, &status.ArrivalDeadline, &estimatedArrival,
		&status.CreatedAt, &status.UpdatedAt, &techName, &techPhone,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrEmergencyNotFound
	}
	if err != nil {
		s.logger.Error("Failed to get emergency status", zap.Error(err))
		return nil, fmt.Errorf("failed to get emergency status: %w", err)
	}

	if techName != nil {
		status.AssignedTechName = *techName
	}
	if techPhone != nil {
		status.AssignedTechPhone = *techPhone
	}
	status.EstimatedArrival = estimatedArrival

	// Calculate SLA status
	status.SLAStatus = s.calculateSLAStatus(status.ResponseDeadline, status.ArrivalDeadline, status.Status)

	return &status, nil
}

// =============================================================================
// TECHNICIAN MATCHING AND DISPATCH
// =============================================================================

// matchTechnician finds and assigns the nearest available technician
func (s *Service) matchTechnician(ctx context.Context, emergencyID uuid.UUID) {
	s.logger.Info("Starting technician matching", zap.String("emergency_id", emergencyID.String()))

	// Update status to searching
	_, err := s.db.Exec(ctx, `UPDATE emergencies SET status = 'searching', updated_at = NOW() WHERE id = $1`, emergencyID)
	if err != nil {
		s.logger.Error("Failed to update emergency status to searching", zap.Error(err))
		return
	}

	// Get emergency details
	emergency, err := s.GetEmergency(ctx, emergencyID)
	if err != nil {
		s.logger.Error("Failed to get emergency for matching", zap.Error(err))
		return
	}

	// Find available technicians
	technicians, err := s.findAvailableTechnicians(ctx, emergency.Category, emergency.Latitude, emergency.Longitude, 50.0)
	if err != nil || len(technicians) == 0 {
		s.logger.Warn("No technicians available",
			zap.String("category", emergency.Category),
			zap.Error(err),
		)
		s.db.Exec(ctx, `UPDATE emergencies SET status = 'no_technicians_available', updated_at = NOW() WHERE id = $1`, emergencyID)
		return
	}

	s.logger.Info("Found available technicians",
		zap.String("emergency_id", emergencyID.String()),
		zap.Int("count", len(technicians)),
	)

	// Notify technicians in order of proximity (cascade notification)
	for i, tech := range technicians {
		if i >= 5 { // Notify max 5 technicians
			break
		}

		s.logger.Info("Notifying technician",
			zap.String("emergency_id", emergencyID.String()),
			zap.String("tech_id", tech.TechID.String()),
			zap.Int("rank", i+1),
		)

		// In production, this would send push notification
		// For now, we'll mark the emergency as notified
		s.cache.SAdd(ctx, fmt.Sprintf("emergency:notified:%s", emergencyID.String()), tech.TechID.String())
	}

	// Auto-assign to closest technician if critical
	if emergency.Urgency == "critical" && len(technicians) > 0 {
		closestTech := technicians[0]
		s.logger.Info("Auto-assigning critical emergency to closest tech",
			zap.String("emergency_id", emergencyID.String()),
			zap.String("tech_id", closestTech.TechID.String()),
		)

		// Calculate ETA
		distance := calculateDistance(
			*closestTech.Latitude, *closestTech.Longitude,
			emergency.Latitude, emergency.Longitude,
		)
		eta := time.Now().Add(time.Duration(distance/40.0*60) * time.Minute)

		s.AcceptEmergency(ctx, emergencyID, closestTech.TechID, eta)
	}
}

// findAvailableTechnicians finds technicians available for a category within radius
func (s *Service) findAvailableTechnicians(ctx context.Context, category string, lat, lon, radiusKm float64) ([]TechnicianAvailability, error) {
	query := `
		SELECT
			ta.technician_id,
			ta.category,
			ta.is_available,
			ta.current_concurrent_jobs,
			ta.max_concurrent_jobs,
			ta.last_known_latitude,
			ta.last_known_longitude
		FROM technician_availability ta
		WHERE ta.category = $1
		  AND ta.is_available = true
		  AND ta.current_concurrent_jobs < ta.max_concurrent_jobs
		  AND ta.last_known_latitude IS NOT NULL
		  AND ta.last_known_longitude IS NOT NULL
		ORDER BY (
			6371 * acos(
				cos(radians($2)) * cos(radians(ta.last_known_latitude)) *
				cos(radians(ta.last_known_longitude) - radians($3)) +
				sin(radians($2)) * sin(radians(ta.last_known_latitude))
			)
		) ASC
		LIMIT 10
	`

	rows, err := s.db.Query(ctx, query, category, lat, lon)
	if err != nil {
		return nil, fmt.Errorf("failed to find technicians: %w", err)
	}
	defer rows.Close()

	var technicians []TechnicianAvailability
	for rows.Next() {
		var tech TechnicianAvailability
		err := rows.Scan(
			&tech.TechID, &tech.Category, &tech.IsAvailable,
			&tech.CurrentJobs, &tech.MaxConcurrentJobs,
			&tech.Latitude, &tech.Longitude,
		)
		if err != nil {
			s.logger.Error("Failed to scan technician", zap.Error(err))
			continue
		}

		// Filter by radius
		if tech.Latitude != nil && tech.Longitude != nil {
			distance := calculateDistance(*tech.Latitude, *tech.Longitude, lat, lon)
			if distance <= radiusKm {
				technicians = append(technicians, tech)
			}
		}
	}

	return technicians, nil
}

// =============================================================================
// EMERGENCY LIFECYCLE MANAGEMENT
// =============================================================================

// AcceptEmergency marks an emergency as accepted by a technician
func (s *Service) AcceptEmergency(ctx context.Context, emergencyID, techID uuid.UUID, estimatedArrival time.Time) error {
	query := `
		UPDATE emergencies
		SET assigned_tech_id = $2, status = 'accepted', estimated_arrival = $3, updated_at = NOW()
		WHERE id = $1 AND status IN ('new', 'searching')
	`

	result, err := s.db.Exec(ctx, query, emergencyID, techID, estimatedArrival)
	if err != nil {
		s.logger.Error("Failed to accept emergency", zap.Error(err))
		return fmt.Errorf("failed to accept emergency: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("emergency not available for acceptance")
	}

	// Update SLA metrics
	s.updateSLAResponseTime(ctx, emergencyID)

	// Update technician availability
	s.incrementTechnicianJobs(ctx, techID)

	// Cache update
	s.cacheEmergency(ctx, emergencyID, "accepted")

	s.logger.Info("Emergency accepted",
		zap.String("emergency_id", emergencyID.String()),
		zap.String("tech_id", techID.String()),
	)

	return nil
}

// UpdateTechnicianLocation updates the technician's GPS location
func (s *Service) UpdateTechnicianLocation(ctx context.Context, emergencyID uuid.UUID, lat, lon float64) error {
	query := `
		UPDATE emergencies
		SET tech_latitude = $2, tech_longitude = $3, updated_at = NOW()
		WHERE id = $1
	`

	result, err := s.db.Exec(ctx, query, emergencyID, lat, lon)
	if err != nil {
		s.logger.Error("Failed to update tech location", zap.Error(err))
		return fmt.Errorf("failed to update location: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrEmergencyNotFound
	}

	// Cache tech location in Redis for real-time tracking
	emergency, err := s.GetEmergency(ctx, emergencyID)
	if err == nil && emergency.AssignedTechID != nil {
		s.cacheTechLocation(ctx, *emergency.AssignedTechID, lat, lon)
	}

	// Recalculate ETA
	go s.recalculateETA(context.Background(), emergencyID, lat, lon)

	return nil
}

// CompleteEmergency marks the emergency as completed
func (s *Service) CompleteEmergency(ctx context.Context, emergencyID, techID uuid.UUID, workNotes string, finalCost float64) error {
	now := time.Now()

	query := `
		UPDATE emergencies
		SET status = 'completed', work_performed = $2, final_cost = $3,
		    completed_at = $4, updated_at = $4
		WHERE id = $1 AND assigned_tech_id = $5 AND status NOT IN ('completed', 'cancelled')
	`

	result, err := s.db.Exec(ctx, query, emergencyID, workNotes, finalCost, now, techID)
	if err != nil {
		s.logger.Error("Failed to complete emergency", zap.Error(err))
		return fmt.Errorf("failed to complete emergency: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("emergency not found or already completed")
	}

	// Update SLA metrics with completion time
	s.updateSLAArrivalTime(ctx, emergencyID)

	// Decrement technician jobs
	s.decrementTechnicianJobs(ctx, techID)

	// Process refund if SLA was breached
	go s.processSLARefund(context.Background(), emergencyID)

	// Cache update
	s.cacheEmergency(ctx, emergencyID, "completed")

	s.logger.Info("Emergency completed",
		zap.String("emergency_id", emergencyID.String()),
		zap.String("tech_id", techID.String()),
		zap.Float64("final_cost", finalCost),
	)

	return nil
}

// =============================================================================
// REAL-TIME TRACKING
// =============================================================================

// GetEmergencyTracking retrieves real-time tracking information
func (s *Service) GetEmergencyTracking(ctx context.Context, emergencyID uuid.UUID) (*EmergencyTracking, error) {
	// Get emergency basic info
	query := `
		SELECT e.status, e.assigned_tech_id, e.latitude, e.longitude,
		       e.estimated_arrival, e.response_deadline, e.arrival_deadline
		FROM emergencies e
		WHERE e.id = $1
	`

	var status string
	var techID *uuid.UUID
	var custLat, custLon float64
	var estimatedArrival *time.Time
	var responseDeadline, arrivalDeadline time.Time

	err := s.db.QueryRow(ctx, query, emergencyID).Scan(
		&status, &techID, &custLat, &custLon, &estimatedArrival,
		&responseDeadline, &arrivalDeadline,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrEmergencyNotFound
	}
	if err != nil {
		s.logger.Error("Failed to get tracking info", zap.Error(err))
		return nil, fmt.Errorf("failed to get tracking: %w", err)
	}

	tracking := &EmergencyTracking{
		EmergencyID:      emergencyID,
		Status:           status,
		CustomerLocation: &GeoPoint{Latitude: custLat, Longitude: custLon},
		EstimatedArrival: estimatedArrival,
		SLAStatus:        s.calculateSLAStatus(responseDeadline, arrivalDeadline, status),
	}

	// If tech is assigned and en route, get their real-time location from cache
	if techID != nil && (status == "en_route" || status == "accepted" || status == "assigned") {
		techLoc, err := s.getTechLocation(ctx, *techID)
		if err == nil && techLoc != nil {
			tracking.TechLocation = techLoc

			// Calculate distance and time remaining
			distance := calculateDistance(
				techLoc.Latitude, techLoc.Longitude,
				custLat, custLon,
			)
			tracking.DistanceRemaining = &distance

			// Estimate time: 40 km/h average in city
			timeMinutes := int((distance / 40.0) * 60)
			tracking.TimeRemaining = &timeMinutes
		}
	}

	return tracking, nil
}

// =============================================================================
// SLA MONITORING AND REFUNDS
// =============================================================================

// initializeSLAMetrics creates initial SLA metrics record
func (s *Service) initializeSLAMetrics(ctx context.Context, emergency *Emergency) error {
	slaMinutes := responseSLAMinutes[emergency.Urgency]
	arrivalSLA := slaMinutes + 30 // 30 minutes after response for arrival

	query := `
		INSERT INTO emergency_sla_metrics (
			emergency_id, response_time_sla_minutes, arrival_time_sla_minutes,
			sla_status, refund_percentage, refund_processed, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (emergency_id) DO NOTHING
	`

	_, err := s.db.Exec(ctx, query,
		emergency.ID, slaMinutes, arrivalSLA, "pending",
		slaRefundPercentages[emergency.Urgency], false, time.Now(),
	)

	return err
}

// updateSLAResponseTime records actual response time
func (s *Service) updateSLAResponseTime(ctx context.Context, emergencyID uuid.UUID) {
	query := `
		UPDATE emergency_sla_metrics esm
		SET actual_response_time_minutes = EXTRACT(EPOCH FROM (NOW() - e.created_at)) / 60,
		    updated_at = NOW()
		FROM emergencies e
		WHERE esm.emergency_id = $1 AND e.id = $1
	`

	_, err := s.db.Exec(ctx, query, emergencyID)
	if err != nil {
		s.logger.Error("Failed to update SLA response time", zap.Error(err))
	}
}

// updateSLAArrivalTime records actual arrival time
func (s *Service) updateSLAArrivalTime(ctx context.Context, emergencyID uuid.UUID) {
	query := `
		UPDATE emergency_sla_metrics esm
		SET actual_arrival_time_minutes = EXTRACT(EPOCH FROM (NOW() - e.created_at)) / 60,
		    sla_status = CASE
		        WHEN EXTRACT(EPOCH FROM (NOW() - e.created_at)) / 60 <= response_time_sla_minutes THEN 'met'
		        ELSE 'breached'
		    END,
		    updated_at = NOW()
		FROM emergencies e
		WHERE esm.emergency_id = $1 AND e.id = $1
	`

	_, err := s.db.Exec(ctx, query, emergencyID)
	if err != nil {
		s.logger.Error("Failed to update SLA arrival time", zap.Error(err))
	}
}

// processSLARefund processes refund if SLA was breached
func (s *Service) processSLARefund(ctx context.Context, emergencyID uuid.UUID) {
	// Get SLA metrics
	query := `
		SELECT esm.sla_status, esm.refund_percentage, e.final_cost, e.urgency
		FROM emergency_sla_metrics esm
		JOIN emergencies e ON e.id = esm.emergency_id
		WHERE esm.emergency_id = $1 AND esm.refund_processed = false
	`

	var slaStatus string
	var refundPercentage int
	var finalCost *float64
	var urgency string

	err := s.db.QueryRow(ctx, query, emergencyID).Scan(&slaStatus, &refundPercentage, &finalCost, &urgency)
	if err != nil {
		s.logger.Error("Failed to get SLA metrics for refund", zap.Error(err))
		return
	}

	// If SLA was breached and we have a final cost
	if slaStatus == "breached" && finalCost != nil && *finalCost > 0 {
		refundAmount := (*finalCost) * float64(refundPercentage) / 100.0

		s.logger.Info("Processing SLA refund",
			zap.String("emergency_id", emergencyID.String()),
			zap.String("urgency", urgency),
			zap.Int("refund_percentage", refundPercentage),
			zap.Float64("refund_amount", refundAmount),
		)

		// Update SLA metrics with refund amount
		updateQuery := `
			UPDATE emergency_sla_metrics
			SET refund_amount = $2, refund_processed = true, updated_at = NOW()
			WHERE emergency_id = $1
		`

		_, err := s.db.Exec(ctx, updateQuery, emergencyID, refundAmount)
		if err != nil {
			s.logger.Error("Failed to record refund", zap.Error(err))
			return
		}

		// In production, this would trigger actual refund via payment service
		s.logger.Info("SLA refund recorded",
			zap.String("emergency_id", emergencyID.String()),
			zap.Float64("amount", refundAmount),
		)
	}
}

// GetSLAMetrics retrieves SLA metrics for an emergency
func (s *Service) GetSLAMetrics(ctx context.Context, emergencyID uuid.UUID) (*SLAMetrics, error) {
	query := `
		SELECT emergency_id, response_time_sla_minutes, actual_response_time_minutes,
		       arrival_time_sla_minutes, actual_arrival_time_minutes, sla_status,
		       refund_percentage, refund_amount, refund_processed
		FROM emergency_sla_metrics
		WHERE emergency_id = $1
	`

	metrics := &SLAMetrics{}
	err := s.db.QueryRow(ctx, query, emergencyID).Scan(
		&metrics.EmergencyID, &metrics.ResponseTimeSLA, &metrics.ActualResponseTime,
		&metrics.ArrivalTimeSLA, &metrics.ActualArrivalTime, &metrics.SLAStatus,
		&metrics.RefundPercentage, &metrics.RefundAmount, &metrics.RefundProcessed,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrEmergencyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get SLA metrics: %w", err)
	}

	return metrics, nil
}

// =============================================================================
// TECHNICIAN AVAILABILITY MANAGEMENT
// =============================================================================

// UpdateTechnicianAvailability updates technician availability status
func (s *Service) UpdateTechnicianAvailability(ctx context.Context, techID uuid.UUID, isAvailable bool) error {
	query := `
		UPDATE technician_availability
		SET is_available = $2, updated_at = NOW()
		WHERE technician_id = $1
	`

	result, err := s.db.Exec(ctx, query, techID, isAvailable)
	if err != nil {
		return fmt.Errorf("failed to update availability: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("technician not found")
	}

	s.logger.Info("Technician availability updated",
		zap.String("tech_id", techID.String()),
		zap.Bool("is_available", isAvailable),
	)

	return nil
}

// incrementTechnicianJobs increments current job count
func (s *Service) incrementTechnicianJobs(ctx context.Context, techID uuid.UUID) {
	query := `
		UPDATE technician_availability
		SET current_concurrent_jobs = current_concurrent_jobs + 1, updated_at = NOW()
		WHERE technician_id = $1
	`

	_, err := s.db.Exec(ctx, query, techID)
	if err != nil {
		s.logger.Error("Failed to increment tech jobs", zap.Error(err))
	}
}

// decrementTechnicianJobs decrements current job count
func (s *Service) decrementTechnicianJobs(ctx context.Context, techID uuid.UUID) {
	query := `
		UPDATE technician_availability
		SET current_concurrent_jobs = GREATEST(current_concurrent_jobs - 1, 0), updated_at = NOW()
		WHERE technician_id = $1
	`

	_, err := s.db.Exec(ctx, query, techID)
	if err != nil {
		s.logger.Error("Failed to decrement tech jobs", zap.Error(err))
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// cacheEmergency stores emergency status in Redis
func (s *Service) cacheEmergency(ctx context.Context, emergencyID uuid.UUID, status string) {
	key := fmt.Sprintf("emergency:status:%s", emergencyID.String())
	s.cache.Set(ctx, key, status, 30*time.Minute)
}

// cacheTechLocation stores technician location in Redis
func (s *Service) cacheTechLocation(ctx context.Context, techID uuid.UUID, lat, lon float64) {
	key := fmt.Sprintf("tech:location:%s", techID.String())
	location := map[string]interface{}{
		"latitude":  lat,
		"longitude": lon,
		"timestamp": time.Now().Unix(),
	}

	s.cache.HSet(ctx, key, location)
	s.cache.Expire(ctx, key, 5*time.Minute)
}

// getTechLocation retrieves technician location from Redis
func (s *Service) getTechLocation(ctx context.Context, techID uuid.UUID) (*TechLocation, error) {
	key := fmt.Sprintf("tech:location:%s", techID.String())

	result, err := s.cache.HGetAll(ctx, key).Result()
	if err != nil || len(result) == 0 {
		return nil, err
	}

	var lat, lon float64
	var timestamp int64

	fmt.Sscanf(result["latitude"], "%f", &lat)
	fmt.Sscanf(result["longitude"], "%f", &lon)
	fmt.Sscanf(result["timestamp"], "%d", &timestamp)

	return &TechLocation{
		TechID:    techID,
		Latitude:  lat,
		Longitude: lon,
		Timestamp: time.Unix(timestamp, 0),
	}, nil
}

// recalculateETA recalculates estimated arrival time based on current location
func (s *Service) recalculateETA(ctx context.Context, emergencyID uuid.UUID, techLat, techLon float64) {
	// Get emergency location
	var destLat, destLon float64
	err := s.db.QueryRow(ctx, `SELECT latitude, longitude FROM emergencies WHERE id = $1`, emergencyID).
		Scan(&destLat, &destLon)
	if err != nil {
		return
	}

	// Calculate distance
	distanceKm := calculateDistance(techLat, techLon, destLat, destLon)

	// Estimate time (40 km/h average in city)
	estimatedMinutes := int(distanceKm / 40.0 * 60)
	eta := time.Now().Add(time.Duration(estimatedMinutes) * time.Minute)

	// Update ETA
	s.db.Exec(ctx, `UPDATE emergencies SET estimated_arrival = $2, updated_at = NOW() WHERE id = $1`, emergencyID, eta)
}

// calculateSLAStatus determines current SLA compliance status
func (s *Service) calculateSLAStatus(responseDeadline, arrivalDeadline time.Time, status string) string {
	now := time.Now()

	if status == "completed" || status == "cancelled" {
		return "final"
	}

	// Check if response deadline passed
	if now.After(responseDeadline) && status == "new" || status == "searching" {
		return "breached"
	}

	// Check if arrival deadline passed
	if now.After(arrivalDeadline) && status != "completed" {
		return "breached"
	}

	// Check if approaching deadline (within 20% of time)
	responseBuffer := responseDeadline.Sub(now)
	if responseBuffer < time.Duration(responseSLAMinutes["urgent"])*time.Minute/5 {
		return "at_risk"
	}

	return "on_track"
}

// calculateDistance calculates distance between two GPS coordinates using Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := toRadians(lat2 - lat1)
	dLon := toRadians(lon2 - lon1)

	lat1Rad := toRadians(lat1)
	lat2Rad := toRadians(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180.0
}
