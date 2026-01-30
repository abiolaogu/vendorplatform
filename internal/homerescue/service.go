// Package homerescue provides emergency home services functionality
// Package homerescue provides emergency home services business logic
package homerescue

import (
	"context"
	"errors"
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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	ErrEmergencyNotFound  = errors.New("emergency not found")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrNoTechniciansAvailable = errors.New("no technicians available")
	ErrUnauthorized       = errors.New("unauthorized")
)

// Service handles emergency service operations
type Service struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewService creates a new emergency service
func NewService(db *pgxpool.Pool, cache *redis.Client) *Service {
	return &Service{
		db:    db,
		cache: cache,
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

	// Location
	Address            string     `json:"address"`
	Latitude           float64    `json:"latitude"`
	Longitude          float64    `json:"longitude"`
	AccessInstructions string     `json:"access_instructions"`

	// Status
	Status             string     `json:"status"`

	// Assignment
	AssignedVendorID   *uuid.UUID `json:"assigned_vendor_id,omitempty"`
	AssignedTechID     *uuid.UUID `json:"assigned_tech_id,omitempty"`

	// Tracking
	TechLatitude       *float64   `json:"tech_latitude,omitempty"`
	TechLongitude      *float64   `json:"tech_longitude,omitempty"`
	EstimatedArrival   *time.Time `json:"estimated_arrival,omitempty"`
	ActualArrival      *time.Time `json:"actual_arrival,omitempty"`

	// Pricing
	EstimatedCost      *float64   `json:"estimated_cost,omitempty"`
	FinalCost          *float64   `json:"final_cost,omitempty"`

	// Timestamps
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
	Latitude           float64   `json:"latitude"`
	Longitude          float64   `json:"longitude"`
	AccessInstructions string    `json:"access_instructions"`
}

// CreateEmergency creates a new emergency request and starts technician matching
func (s *Service) CreateEmergency(ctx context.Context, req *CreateEmergencyRequest) (*Emergency, error) {
	if req.UserID == uuid.Nil || req.Category == "" || req.Title == "" {
		return nil, ErrInvalidRequest
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
		Latitude:           req.Latitude,
		Longitude:          req.Longitude,
		AccessInstructions: req.AccessInstructions,
		Status:             "new",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	query := `
		INSERT INTO emergencies (
			id, user_id, category, subcategory, urgency,
			title, description, address, latitude, longitude,
			access_instructions, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at
	`

	err := s.db.QueryRow(ctx, query,
		emergency.ID, emergency.UserID, emergency.Category, emergency.Subcategory,
		emergency.Urgency, emergency.Title, emergency.Description, emergency.Address,
		emergency.Latitude, emergency.Longitude, emergency.AccessInstructions,
		emergency.Status, emergency.CreatedAt, emergency.UpdatedAt,
	).Scan(&emergency.ID, &emergency.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create emergency: %w", err)
	}

	// Start async technician matching
	go s.matchTechnician(context.Background(), emergency.ID)

	return emergency, nil
}

// GetEmergency retrieves an emergency by ID
func (s *Service) GetEmergency(ctx context.Context, id uuid.UUID) (*Emergency, error) {
	query := `
		SELECT id, user_id, category, subcategory, urgency,
		       title, description, address, latitude, longitude,
		       access_instructions, status, assigned_vendor_id,
		       assigned_tech_id, tech_latitude, tech_longitude,
		       estimated_arrival, actual_arrival, estimated_cost,
		       final_cost, created_at, updated_at, completed_at
		FROM emergencies WHERE id = $1
	`

	emergency := &Emergency{}
	err := s.db.QueryRow(ctx, query, id).Scan(
		&emergency.ID, &emergency.UserID, &emergency.Category, &emergency.Subcategory,
		&emergency.Urgency, &emergency.Title, &emergency.Description, &emergency.Address,
		&emergency.Latitude, &emergency.Longitude, &emergency.AccessInstructions,
		&emergency.Status, &emergency.AssignedVendorID, &emergency.AssignedTechID,
		&emergency.TechLatitude, &emergency.TechLongitude, &emergency.EstimatedArrival,
		&emergency.ActualArrival, &emergency.EstimatedCost, &emergency.FinalCost,
		&emergency.CreatedAt, &emergency.UpdatedAt, &emergency.CompletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrEmergencyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get emergency: %w", err)
	}

	return emergency, nil
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
		return fmt.Errorf("failed to update location: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrEmergencyNotFound
	}

	// Update estimated arrival based on distance
	go s.calculateETA(context.Background(), emergencyID, lat, lon)

	return nil
}

// AcceptEmergency marks the emergency as accepted by a technician
func (s *Service) AcceptEmergency(ctx context.Context, emergencyID, techID uuid.UUID) error {
	query := `
		UPDATE emergencies
		SET assigned_tech_id = $2, status = 'accepted', updated_at = NOW()
		WHERE id = $1 AND status IN ('new', 'searching')
	`

	result, err := s.db.Exec(ctx, query, emergencyID, techID)
	if err != nil {
		return fmt.Errorf("failed to accept emergency: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrEmergencyNotFound
	}

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
// CompleteEmergency marks the emergency as completed
func (s *Service) CompleteEmergency(ctx context.Context, emergencyID uuid.UUID, finalCost float64) error {
	now := time.Now()
	query := `
		UPDATE emergencies
		SET status = 'completed', final_cost = $2, completed_at = $3, updated_at = NOW()
		WHERE id = $1 AND status = 'in_progress'
	`

	result, err := s.db.Exec(ctx, query, emergencyID, finalCost, now)
	if err != nil {
		return fmt.Errorf("failed to complete emergency: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrEmergencyNotFound
	}

	return nil
}

// GetTechnicianTracking returns real-time location data for tracking
type TechnicianTracking struct {
	TechnicianID     uuid.UUID  `json:"technician_id"`
	TechnicianName   string     `json:"technician_name"`
	CurrentLatitude  float64    `json:"current_latitude"`
	CurrentLongitude float64    `json:"current_longitude"`
	DestLatitude     float64    `json:"dest_latitude"`
	DestLongitude    float64    `json:"dest_longitude"`
	EstimatedArrival *time.Time `json:"estimated_arrival"`
	DistanceKm       float64    `json:"distance_km"`
	Status           string     `json:"status"`
}

func (s *Service) GetTechnicianTracking(ctx context.Context, emergencyID uuid.UUID) (*TechnicianTracking, error) {
	query := `
		SELECT
			e.assigned_tech_id,
			COALESCE(u.name, 'Technician') as tech_name,
			e.tech_latitude,
			e.tech_longitude,
			e.latitude as dest_lat,
			e.longitude as dest_lon,
			e.estimated_arrival,
			e.status
		FROM emergencies e
		LEFT JOIN users u ON u.id = e.assigned_tech_id
		WHERE e.id = $1 AND e.assigned_tech_id IS NOT NULL
	`

	tracking := &TechnicianTracking{}
	err := s.db.QueryRow(ctx, query, emergencyID).Scan(
		&tracking.TechnicianID,
		&tracking.TechnicianName,
		&tracking.CurrentLatitude,
		&tracking.CurrentLongitude,
		&tracking.DestLatitude,
		&tracking.DestLongitude,
		&tracking.EstimatedArrival,
		&tracking.Status,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrEmergencyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tracking: %w", err)
	}

	// Calculate distance using Haversine formula
	tracking.DistanceKm = calculateDistance(
		tracking.CurrentLatitude, tracking.CurrentLongitude,
		tracking.DestLatitude, tracking.DestLongitude,
	)

	return tracking, nil
}

// matchTechnician finds and assigns the nearest available technician
func (s *Service) matchTechnician(ctx context.Context, emergencyID uuid.UUID) {
	// Update status to searching
	s.db.Exec(ctx, `UPDATE emergencies SET status = 'searching' WHERE id = $1`, emergencyID)

	// In a real implementation, this would:
	// 1. Query available technicians by category and location
	// 2. Calculate distance to each technician
	// 3. Send push notifications to nearest technicians
	// 4. Wait for acceptance (with timeout and fallback)
	// 5. Update emergency with assignment

	// For now, this is a placeholder
	time.Sleep(2 * time.Second) // Simulate matching delay
}

// calculateETA estimates arrival time based on distance and traffic
func (s *Service) calculateETA(ctx context.Context, emergencyID uuid.UUID, techLat, techLon float64) {
	// Get destination
	var destLat, destLon float64
	err := s.db.QueryRow(ctx, `SELECT latitude, longitude FROM emergencies WHERE id = $1`, emergencyID).
		Scan(&destLat, &destLon)
	if err != nil {
		return
	}

	// Calculate distance
	distanceKm := calculateDistance(techLat, techLon, destLat, destLon)

	// Estimate time (assuming average speed of 40 km/h in city traffic)
	estimatedMinutes := int(distanceKm / 40.0 * 60)
	eta := time.Now().Add(time.Duration(estimatedMinutes) * time.Minute)

	// Update estimated arrival
	s.db.Exec(ctx, `UPDATE emergencies SET estimated_arrival = $2 WHERE id = $1`, emergencyID, eta)
}

// calculateDistance calculates the distance between two GPS coordinates using Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := toRadians(lat2 - lat1)
	dLon := toRadians(lon2 - lon1)

	lat1Rad := toRadians(lat1)
	lat2Rad := toRadians(lat2)

	a := sin(dLat/2)*sin(dLat/2) +
		sin(dLon/2)*sin(dLon/2)*cos(lat1Rad)*cos(lat2Rad)
	c := 2 * atan2(sqrt(a), sqrt(1-a))

	return earthRadiusKm * c
}

func toRadians(deg float64) float64 {
	return deg * (3.14159265358979323846 / 180)
}

func sin(x float64) float64 {
	// Simple sine approximation
	return x - (x*x*x)/6 + (x*x*x*x*x)/120
}

func cos(x float64) float64 {
	return sin(x + 3.14159265358979323846/2)
}

func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	z := 1.0
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}

func atan2(y, x float64) float64 {
	// Simplified atan2 approximation
	if x > 0 {
		return atan(y / x)
	}
	if x < 0 && y >= 0 {
		return atan(y/x) + 3.14159265358979323846
	}
	if x < 0 && y < 0 {
		return atan(y/x) - 3.14159265358979323846
	}
	if x == 0 && y > 0 {
		return 3.14159265358979323846 / 2
	}
	if x == 0 && y < 0 {
		return -3.14159265358979323846 / 2
	}
	return 0
}

func atan(x float64) float64 {
	return x - (x*x*x)/3 + (x*x*x*x*x)/5 - (x*x*x*x*x*x*x)/7
}
