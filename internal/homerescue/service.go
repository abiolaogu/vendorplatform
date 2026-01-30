// Package homerescue provides emergency home services business logic
package homerescue

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
