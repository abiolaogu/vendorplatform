// Package homerescue provides emergency home services functionality
package homerescue

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
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

// EmergencyRequest represents an emergency service request
type EmergencyRequest struct {
	ID               uuid.UUID         `json:"id"`
	UserID           uuid.UUID         `json:"user_id"`
	Category         string            `json:"category"`
	Subcategory      string            `json:"subcategory"`
	Urgency          string            `json:"urgency"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	Location         EmergencyLocation `json:"location"`
	Status           string            `json:"status"`
	AssignedTechID   *uuid.UUID        `json:"assigned_tech_id,omitempty"`
	ResponseDeadline time.Time         `json:"response_deadline"`
	ArrivalDeadline  time.Time         `json:"arrival_deadline"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// EmergencyLocation represents the location of an emergency
type EmergencyLocation struct {
	Address    string  `json:"address"`
	Unit       string  `json:"unit,omitempty"`
	City       string  `json:"city"`
	State      string  `json:"state"`
	PostalCode string  `json:"postal_code"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}

// CreateEmergencyRequest represents the input for creating an emergency
type CreateEmergencyRequest struct {
	UserID          uuid.UUID         `json:"user_id"`
	Category        string            `json:"category"`
	Subcategory     string            `json:"subcategory"`
	Urgency         string            `json:"urgency"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Location        EmergencyLocation `json:"location"`
	AccessInfo      string            `json:"access_instructions,omitempty"`
}

// EmergencyStatus represents the status of an emergency
type EmergencyStatus struct {
	EmergencyID      uuid.UUID  `json:"emergency_id"`
	Status           string     `json:"status"`
	AssignedTechID   *uuid.UUID `json:"assigned_tech_id,omitempty"`
	AssignedTechName string     `json:"assigned_tech_name,omitempty"`
	AssignedTechPhone string    `json:"assigned_tech_phone,omitempty"`
	ResponseDeadline time.Time  `json:"response_deadline"`
	ArrivalDeadline  time.Time  `json:"arrival_deadline"`
	EstimatedArrival *time.Time `json:"estimated_arrival,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
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
}

// GeoPoint represents a geographic coordinate
type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Response time SLAs in minutes based on urgency
var responseSLAMinutes = map[string]int{
	"critical":  30,
	"urgent":    120,
	"same_day":  360,
	"scheduled": 1440,
}

// CreateEmergency creates a new emergency request
func (s *Service) CreateEmergency(ctx context.Context, req *CreateEmergencyRequest) (*EmergencyRequest, error) {
	emergencyID := uuid.New()
	now := time.Now()

	// Calculate SLA deadlines based on urgency
	slaMinutes, ok := responseSLAMinutes[req.Urgency]
	if !ok {
		slaMinutes = 120 // Default to urgent
	}

	responseDeadline := now.Add(time.Duration(slaMinutes) * time.Minute)
	arrivalDeadline := responseDeadline.Add(30 * time.Minute) // 30 min after response

	// Insert emergency into database
	query := `
		INSERT INTO emergencies (
			id, user_id, category, subcategory, urgency, title, description,
			address, unit, city, state, postal_code, latitude, longitude,
			access_instructions, status, response_deadline, arrival_deadline,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	_, err := s.db.Exec(ctx, query,
		emergencyID, req.UserID, req.Category, req.Subcategory, req.Urgency,
		req.Title, req.Description,
		req.Location.Address, req.Location.Unit, req.Location.City,
		req.Location.State, req.Location.PostalCode,
		req.Location.Latitude, req.Location.Longitude,
		req.AccessInfo, "new", responseDeadline, arrivalDeadline,
		now, now,
	)

	if err != nil {
		s.logger.Error("Failed to create emergency", zap.Error(err))
		return nil, fmt.Errorf("failed to create emergency: %w", err)
	}

	// Cache emergency for real-time updates
	s.cacheEmergency(ctx, emergencyID, "new")

	s.logger.Info("Emergency created",
		zap.String("emergency_id", emergencyID.String()),
		zap.String("category", req.Category),
		zap.String("urgency", req.Urgency),
	)

	return &EmergencyRequest{
		ID:               emergencyID,
		UserID:           req.UserID,
		Category:         req.Category,
		Subcategory:      req.Subcategory,
		Urgency:          req.Urgency,
		Title:            req.Title,
		Description:      req.Description,
		Location:         req.Location,
		Status:           "new",
		ResponseDeadline: responseDeadline,
		ArrivalDeadline:  arrivalDeadline,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
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
		&status.EmergencyID,
		&status.Status,
		&status.AssignedTechID,
		&status.ResponseDeadline,
		&status.ArrivalDeadline,
		&estimatedArrival,
		&status.CreatedAt,
		&status.UpdatedAt,
		&techName,
		&techPhone,
	)

	if err != nil {
		s.logger.Error("Failed to get emergency status", zap.Error(err))
		return nil, fmt.Errorf("emergency not found")
	}

	if techName != nil {
		status.AssignedTechName = *techName
	}
	if techPhone != nil {
		status.AssignedTechPhone = *techPhone
	}
	status.EstimatedArrival = estimatedArrival

	return &status, nil
}

// GetEmergencyTracking retrieves real-time tracking information
func (s *Service) GetEmergencyTracking(ctx context.Context, emergencyID uuid.UUID) (*EmergencyTracking, error) {
	// Get emergency basic info
	query := `
		SELECT e.status, e.assigned_tech_id, e.latitude, e.longitude, e.estimated_arrival
		FROM emergencies e
		WHERE e.id = $1
	`

	var status string
	var techID *uuid.UUID
	var custLat, custLon float64
	var estimatedArrival *time.Time

	err := s.db.QueryRow(ctx, query, emergencyID).Scan(&status, &techID, &custLat, &custLon, &estimatedArrival)
	if err != nil {
		return nil, fmt.Errorf("emergency not found")
	}

	tracking := &EmergencyTracking{
		EmergencyID:      emergencyID,
		Status:           status,
		CustomerLocation: &GeoPoint{Latitude: custLat, Longitude: custLon},
		EstimatedArrival: estimatedArrival,
	}

	// If tech is assigned and en route, get their real-time location from cache
	if techID != nil && (status == "en_route" || status == "assigned") {
		techLoc, err := s.getTechLocation(ctx, *techID)
		if err == nil && techLoc != nil {
			tracking.TechLocation = techLoc

			// Calculate distance and time remaining
			distance := s.calculateDistance(
				techLoc.Latitude, techLoc.Longitude,
				custLat, custLon,
			)
			tracking.DistanceRemaining = &distance

			// Rough estimate: 40 km/h average in city
			timeMinutes := int((distance / 40.0) * 60)
			tracking.TimeRemaining = &timeMinutes
		}
	}

	return tracking, nil
}

// UpdateTechLocation updates a technician's current location
func (s *Service) UpdateTechLocation(ctx context.Context, techID uuid.UUID, lat, lon float64) error {
	// Store in Redis with 5-minute expiry (real-time data)
	key := fmt.Sprintf("tech:location:%s", techID.String())
	location := map[string]interface{}{
		"latitude":  lat,
		"longitude": lon,
		"timestamp": time.Now().Unix(),
	}

	err := s.cache.HSet(ctx, key, location).Err()
	if err != nil {
		return fmt.Errorf("failed to update tech location: %w", err)
	}

	// Set expiry
	s.cache.Expire(ctx, key, 5*time.Minute)

	return nil
}

// AcceptEmergency marks an emergency as accepted by a technician
func (s *Service) AcceptEmergency(ctx context.Context, emergencyID, techID uuid.UUID, estimatedArrival time.Time) error {
	query := `
		UPDATE emergencies
		SET assigned_tech_id = $1, status = $2, estimated_arrival = $3, updated_at = $4
		WHERE id = $5 AND status IN ('new', 'searching')
	`

	result, err := s.db.Exec(ctx, query, techID, "accepted", estimatedArrival, time.Now(), emergencyID)
	if err != nil {
		return fmt.Errorf("failed to accept emergency: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("emergency not available for acceptance")
	}

	// Update cache
	s.cacheEmergency(ctx, emergencyID, "accepted")

	s.logger.Info("Emergency accepted",
		zap.String("emergency_id", emergencyID.String()),
		zap.String("tech_id", techID.String()),
	)

	return nil
}

// CompleteEmergency marks an emergency as completed
func (s *Service) CompleteEmergency(ctx context.Context, emergencyID, techID uuid.UUID, workNotes string) error {
	now := time.Now()

	query := `
		UPDATE emergencies
		SET status = $1, work_performed = $2, completed_at = $3, updated_at = $4
		WHERE id = $5 AND assigned_tech_id = $6 AND status NOT IN ('completed', 'cancelled')
	`

	result, err := s.db.Exec(ctx, query, "completed", workNotes, now, now, emergencyID, techID)
	if err != nil {
		return fmt.Errorf("failed to complete emergency: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("emergency not found or already completed")
	}

	// Update cache
	s.cacheEmergency(ctx, emergencyID, "completed")

	s.logger.Info("Emergency completed",
		zap.String("emergency_id", emergencyID.String()),
		zap.String("tech_id", techID.String()),
	)

	return nil
}

// Helper functions

func (s *Service) cacheEmergency(ctx context.Context, emergencyID uuid.UUID, status string) {
	key := fmt.Sprintf("emergency:status:%s", emergencyID.String())
	s.cache.Set(ctx, key, status, 30*time.Minute)
}

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

// calculateDistance calculates the distance between two points using Haversine formula
func (s *Service) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	dLat := (lat2 - lat1) * (3.14159265359 / 180.0)
	dLon := (lon2 - lon1) * (3.14159265359 / 180.0)

	a := 0.5 - 0.5*((dLat)*(dLat)) +
		(1.0-((lat1)*(lat1)*(3.14159265359/180.0))) *
		(1.0-((lat2)*(lat2)*(3.14159265359/180.0))) *
		0.5 * (1.0 - ((dLon)*(dLon)))

	// Simplified Haversine
	return R * 2.0 * 3.14159265359 * (lat2 - lat1) / 360.0 * 111.0 // Rough approximation
}
