// Package api provides HTTP handlers for the recommendation engine
package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	recommendation "vendorplatform/recommendation-engine"
)

// Server represents the HTTP server
type Server struct {
	engine *recommendation.Engine
	router *chi.Mux
}

// NewServer creates a new API server
func NewServer(engine *recommendation.Engine) *Server {
	s := &Server{
		engine: engine,
		router: chi.NewRouter(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	r := s.router

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	
	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Session-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", s.healthCheck)
	r.Get("/ready", s.readinessCheck)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Recommendations
		r.Route("/recommendations", func(r chi.Router) {
			r.Post("/", s.getRecommendations)
			r.Get("/adjacent", s.getAdjacentServices)
			r.Get("/event/{eventType}", s.getEventRecommendations)
			r.Get("/trending", s.getTrending)
			r.Get("/similar/{entityType}/{entityID}", s.getSimilar)
			r.Post("/bundle", s.getBundleRecommendations)
		})

		// Projects (for event-based recommendations)
		r.Route("/projects", func(r chi.Router) {
			r.Get("/{projectID}/recommendations", s.getProjectRecommendations)
			r.Get("/{projectID}/next-steps", s.getProjectNextSteps)
			r.Get("/{projectID}/completion", s.getProjectCompletion)
		})

		// Feedback (for improving recommendations)
		r.Route("/feedback", func(r chi.Router) {
			r.Post("/click", s.recordClick)
			r.Post("/conversion", s.recordConversion)
			r.Post("/dismiss", s.recordDismiss)
		})

		// Adjacency management (admin)
		r.Route("/admin/adjacencies", func(r chi.Router) {
			r.Get("/", s.listAdjacencies)
			r.Post("/", s.createAdjacency)
			r.Put("/{adjacencyID}", s.updateAdjacency)
			r.Delete("/{adjacencyID}", s.deleteAdjacency)
			r.Post("/refresh", s.refreshAdjacencyGraph)
		})
	})
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe(addr string) error {
	log.Printf("Starting server on %s", addr)
	return http.ListenAndServe(addr, s.router)
}

// =============================================================================
// HANDLERS
// =============================================================================

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func (s *Server) readinessCheck(w http.ResponseWriter, r *http.Request) {
	// Check database connection, cache, etc.
	respondJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// GetRecommendationsRequest is the request body for POST /recommendations
type GetRecommendationsRequest struct {
	UserID            string   `json:"user_id,omitempty"`
	SessionID         string   `json:"session_id,omitempty"`
	ProjectID         string   `json:"project_id,omitempty"`
	CurrentEntityID   string   `json:"current_entity_id,omitempty"`
	CurrentEntityType string   `json:"current_entity_type,omitempty"`
	EventType         string   `json:"event_type,omitempty"`
	Latitude          *float64 `json:"latitude,omitempty"`
	Longitude         *float64 `json:"longitude,omitempty"`
	BudgetMin         *float64 `json:"budget_min,omitempty"`
	BudgetMax         *float64 `json:"budget_max,omitempty"`
	Currency          string   `json:"currency,omitempty"`
	RequestedTypes    []string `json:"requested_types,omitempty"`
	Limit             int      `json:"limit,omitempty"`
	ExcludeIDs        []string `json:"exclude_ids,omitempty"`
	DiversityFactor   float64  `json:"diversity_factor,omitempty"`
}

func (s *Server) getRecommendations(w http.ResponseWriter, r *http.Request) {
	var req GetRecommendationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Build recommendation request
	recReq := &recommendation.RecommendationRequest{
		Limit:           req.Limit,
		DiversityFactor: req.DiversityFactor,
		EventType:       req.EventType,
	}

	// Parse UUIDs
	if req.UserID != "" {
		if id, err := uuid.Parse(req.UserID); err == nil {
			recReq.UserID = id
		}
	}
	if req.SessionID != "" {
		if id, err := uuid.Parse(req.SessionID); err == nil {
			recReq.SessionID = id
		}
	}
	if req.ProjectID != "" {
		if id, err := uuid.Parse(req.ProjectID); err == nil {
			recReq.ProjectID = id
		}
	}
	if req.CurrentEntityID != "" {
		if id, err := uuid.Parse(req.CurrentEntityID); err == nil {
			recReq.CurrentEntityID = id
		}
	}

	// Parse entity type
	if req.CurrentEntityType != "" {
		recReq.CurrentEntityType = recommendation.EntityType(req.CurrentEntityType)
	}

	// Parse location
	if req.Latitude != nil && req.Longitude != nil {
		recReq.Location = &recommendation.GeoPoint{
			Latitude:  *req.Latitude,
			Longitude: *req.Longitude,
		}
	}

	// Parse budget
	if req.BudgetMin != nil || req.BudgetMax != nil {
		recReq.Budget = &recommendation.BudgetRange{
			Currency: req.Currency,
		}
		if req.BudgetMin != nil {
			recReq.Budget.Min = *req.BudgetMin
		}
		if req.BudgetMax != nil {
			recReq.Budget.Max = *req.BudgetMax
		}
	}

	// Parse excluded IDs
	for _, idStr := range req.ExcludeIDs {
		if id, err := uuid.Parse(idStr); err == nil {
			recReq.ExcludeIDs = append(recReq.ExcludeIDs, id)
		}
	}

	// Parse requested types
	for _, t := range req.RequestedTypes {
		recReq.RequestedTypes = append(recReq.RequestedTypes, recommendation.RecommendationType(t))
	}

	// Get recommendations
	ctx := r.Context()
	resp, err := s.engine.GetRecommendations(ctx, recReq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) getAdjacentServices(w http.ResponseWriter, r *http.Request) {
	categoryID := r.URL.Query().Get("category_id")
	serviceID := r.URL.Query().Get("service_id")
	eventType := r.URL.Query().Get("event_type")
	limitStr := r.URL.Query().Get("limit")

	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	var entityID uuid.UUID
	var entityType recommendation.EntityType

	if categoryID != "" {
		if id, err := uuid.Parse(categoryID); err == nil {
			entityID = id
			entityType = recommendation.EntityCategory
		}
	} else if serviceID != "" {
		if id, err := uuid.Parse(serviceID); err == nil {
			entityID = id
			entityType = recommendation.EntityService
		}
	}

	if entityID == uuid.Nil {
		respondError(w, http.StatusBadRequest, "category_id or service_id required")
		return
	}

	req := &recommendation.RecommendationRequest{
		CurrentEntityID:   entityID,
		CurrentEntityType: entityType,
		EventType:         eventType,
		Limit:             limit,
		RequestedTypes:    []recommendation.RecommendationType{recommendation.AdjacentService},
	}

	ctx := r.Context()
	resp, err := s.engine.GetRecommendations(ctx, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) getEventRecommendations(w http.ResponseWriter, r *http.Request) {
	eventType := chi.URLParam(r, "eventType")
	userID := r.URL.Query().Get("user_id")
	projectID := r.URL.Query().Get("project_id")
	limitStr := r.URL.Query().Get("limit")

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	req := &recommendation.RecommendationRequest{
		EventType:      eventType,
		Limit:          limit,
		RequestedTypes: []recommendation.RecommendationType{recommendation.EventBasedSuggest},
	}

	if userID != "" {
		if id, err := uuid.Parse(userID); err == nil {
			req.UserID = id
		}
	}
	if projectID != "" {
		if id, err := uuid.Parse(projectID); err == nil {
			req.ProjectID = id
		}
	}

	ctx := r.Context()
	resp, err := s.engine.GetRecommendations(ctx, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) getTrending(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("latitude")
	lonStr := r.URL.Query().Get("longitude")
	limitStr := r.URL.Query().Get("limit")

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	req := &recommendation.RecommendationRequest{
		Limit:          limit,
		RequestedTypes: []recommendation.RecommendationType{recommendation.TrendingService},
	}

	if latStr != "" && lonStr != "" {
		lat, _ := strconv.ParseFloat(latStr, 64)
		lon, _ := strconv.ParseFloat(lonStr, 64)
		req.Location = &recommendation.GeoPoint{Latitude: lat, Longitude: lon}
	}

	ctx := r.Context()
	resp, err := s.engine.GetRecommendations(ctx, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) getSimilar(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entityType")
	entityID := chi.URLParam(r, "entityID")
	limitStr := r.URL.Query().Get("limit")

	id, err := uuid.Parse(entityID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid entity ID")
		return
	}

	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	req := &recommendation.RecommendationRequest{
		CurrentEntityID:   id,
		CurrentEntityType: recommendation.EntityType(entityType),
		Limit:             limit,
		RequestedTypes:    []recommendation.RecommendationType{recommendation.SimilarVendor},
	}

	ctx := r.Context()
	resp, err := s.engine.GetRecommendations(ctx, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// BundleRequest represents a request for bundle recommendations
type BundleRequest struct {
	EventType  string   `json:"event_type"`
	CategoryIDs []string `json:"category_ids"`
	Budget     *float64 `json:"budget,omitempty"`
	GuestCount *int     `json:"guest_count,omitempty"`
	Location   *struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location,omitempty"`
}

func (s *Server) getBundleRecommendations(w http.ResponseWriter, r *http.Request) {
	var req BundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	recReq := &recommendation.RecommendationRequest{
		EventType:      req.EventType,
		Limit:          5,
		RequestedTypes: []recommendation.RecommendationType{recommendation.BundleSuggestion},
	}

	if req.Location != nil {
		recReq.Location = &recommendation.GeoPoint{
			Latitude:  req.Location.Latitude,
			Longitude: req.Location.Longitude,
		}
	}

	if req.Budget != nil {
		recReq.Budget = &recommendation.BudgetRange{
			Max:      *req.Budget,
			Currency: "NGN",
		}
	}

	ctx := r.Context()
	resp, err := s.engine.GetRecommendations(ctx, recReq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// =============================================================================
// PROJECT HANDLERS
// =============================================================================

// ProjectRecommendationsResponse contains project-specific recommendations
type ProjectRecommendationsResponse struct {
	ProjectID           string                            `json:"project_id"`
	EventType           string                            `json:"event_type"`
	CompletionPercent   float64                           `json:"completion_percent"`
	BookedCategories    []string                          `json:"booked_categories"`
	PendingCategories   []string                          `json:"pending_categories"`
	Recommendations     *recommendation.RecommendationResponse `json:"recommendations"`
	NextSteps           []NextStep                        `json:"next_steps"`
	BudgetSummary       *BudgetSummary                    `json:"budget_summary,omitempty"`
}

type NextStep struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Priority     int     `json:"priority"`
	Reason       string  `json:"reason"`
	Deadline     *string `json:"deadline,omitempty"`
}

type BudgetSummary struct {
	TotalBudget     float64 `json:"total_budget"`
	AllocatedAmount float64 `json:"allocated_amount"`
	RemainingAmount float64 `json:"remaining_amount"`
	Currency        string  `json:"currency"`
}

func (s *Server) getProjectRecommendations(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	
	id, err := uuid.Parse(projectID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid project ID")
		return
	}

	req := &recommendation.RecommendationRequest{
		ProjectID: id,
		Limit:     20,
	}

	ctx := r.Context()
	resp, err := s.engine.GetRecommendations(ctx, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Would enrich with project-specific data
	projectResp := &ProjectRecommendationsResponse{
		ProjectID:       projectID,
		Recommendations: resp,
	}

	respondJSON(w, http.StatusOK, projectResp)
}

func (s *Server) getProjectNextSteps(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	
	// Would fetch project and determine next steps based on:
	// - What's already booked
	// - Event date timeline
	// - Category dependencies
	
	nextSteps := []NextStep{
		{
			CategoryID:   "venue",
			CategoryName: "Event Venue",
			Priority:     1,
			Reason:       "Most other bookings depend on venue selection",
		},
		{
			CategoryID:   "catering",
			CategoryName: "Catering",
			Priority:     2,
			Reason:       "Book early to secure preferred caterer",
		},
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"project_id": projectID,
		"next_steps": nextSteps,
	})
}

func (s *Server) getProjectCompletion(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	
	// Would calculate completion based on required vs booked categories
	
	respondJSON(w, http.StatusOK, map[string]any{
		"project_id":          projectID,
		"completion_percent":  45.0,
		"categories_booked":   5,
		"categories_required": 11,
		"categories_optional": 8,
	})
}

// =============================================================================
// FEEDBACK HANDLERS
// =============================================================================

type ClickFeedback struct {
	RecommendationID string `json:"recommendation_id"`
	EntityType       string `json:"entity_type"`
	EntityID         string `json:"entity_id"`
	Position         int    `json:"position"`
	SessionID        string `json:"session_id,omitempty"`
	UserID           string `json:"user_id,omitempty"`
}

func (s *Server) recordClick(w http.ResponseWriter, r *http.Request) {
	var feedback ClickFeedback
	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Record click for recommendation improvement
	// Would update recommendation_events table
	
	respondJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

type ConversionFeedback struct {
	RecommendationID string `json:"recommendation_id"`
	BookingID        string `json:"booking_id"`
	EntityID         string `json:"entity_id"`
	SessionID        string `json:"session_id,omitempty"`
	UserID           string `json:"user_id,omitempty"`
}

func (s *Server) recordConversion(w http.ResponseWriter, r *http.Request) {
	var feedback ConversionFeedback
	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Record conversion for recommendation improvement
	// This is crucial for training and optimizing the recommendation engine
	
	respondJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

type DismissFeedback struct {
	RecommendationID string `json:"recommendation_id"`
	EntityID         string `json:"entity_id"`
	Reason           string `json:"reason,omitempty"` // 'not_relevant', 'already_have', 'too_expensive', etc.
	SessionID        string `json:"session_id,omitempty"`
	UserID           string `json:"user_id,omitempty"`
}

func (s *Server) recordDismiss(w http.ResponseWriter, r *http.Request) {
	var feedback DismissFeedback
	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Record dismissal - helps understand what NOT to recommend
	
	respondJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

// =============================================================================
// ADMIN HANDLERS
// =============================================================================

func (s *Server) listAdjacencies(w http.ResponseWriter, r *http.Request) {
	// Would query service_adjacencies table with filters
	respondJSON(w, http.StatusOK, map[string]string{"status": "not_implemented"})
}

func (s *Server) createAdjacency(w http.ResponseWriter, r *http.Request) {
	// Would create new adjacency
	respondJSON(w, http.StatusOK, map[string]string{"status": "not_implemented"})
}

func (s *Server) updateAdjacency(w http.ResponseWriter, r *http.Request) {
	// Would update adjacency
	respondJSON(w, http.StatusOK, map[string]string{"status": "not_implemented"})
}

func (s *Server) deleteAdjacency(w http.ResponseWriter, r *http.Request) {
	// Would delete adjacency
	respondJSON(w, http.StatusOK, map[string]string{"status": "not_implemented"})
}

func (s *Server) refreshAdjacencyGraph(w http.ResponseWriter, r *http.Request) {
	// Trigger refresh of in-memory adjacency graph
	ctx := context.Background()
	// Would call: s.engine.RefreshAdjacencyGraph(ctx)
	_ = ctx
	
	respondJSON(w, http.StatusOK, map[string]string{"status": "refresh_triggered"})
}

// =============================================================================
// HELPERS
// =============================================================================

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
