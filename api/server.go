// Package api provides the HTTP REST API for the recommendation engine
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	
	"vendorplatform/recommendation"
)

// =============================================================================
// SERVER SETUP
// =============================================================================

// Server represents the API server
type Server struct {
	router     *chi.Mux
	engine     *recommendation.Engine
	db         *pgxpool.Pool
	cache      *redis.Client
	logger     *slog.Logger
	config     *ServerConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	EnableMetrics   bool
	EnableProfiling bool
	RateLimitRPS    int
}

// DefaultServerConfig returns sensible defaults
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:            8080,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		EnableMetrics:   true,
		EnableProfiling: false,
		RateLimitRPS:    100,
	}
}

// NewServer creates a new API server
func NewServer(db *pgxpool.Pool, cache *redis.Client, config *ServerConfig, logger *slog.Logger) (*Server, error) {
	if config == nil {
		config = DefaultServerConfig()
	}
	
	// Initialize recommendation engine
	engine, err := recommendation.NewEngine(db, cache, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create recommendation engine: %w", err)
	}
	
	s := &Server{
		router: chi.NewRouter(),
		engine: engine,
		db:     db,
		cache:  cache,
		logger: logger,
		config: config,
	}
	
	s.setupMiddleware()
	s.setupRoutes()
	
	return s, nil
}

// setupMiddleware configures middleware
func (s *Server) setupMiddleware() {
	// Request ID
	s.router.Use(middleware.RequestID)
	
	// Real IP
	s.router.Use(middleware.RealIP)
	
	// Logging
	s.router.Use(middleware.Logger)
	
	// Recover from panics
	s.router.Use(middleware.Recoverer)
	
	// CORS
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	
	// Timeout
	s.router.Use(middleware.Timeout(s.config.WriteTimeout))
	
	// Compression
	s.router.Use(middleware.Compress(5))
}

// setupRoutes configures API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.Get("/health", s.handleHealth)
	s.router.Get("/ready", s.handleReady)
	
	// API v1
	s.router.Route("/api/v1", func(r chi.Router) {
		// Recommendations
		r.Route("/recommendations", func(r chi.Router) {
			r.Post("/", s.handleGetRecommendations)
			r.Get("/adjacent/{entityType}/{entityID}", s.handleGetAdjacentRecommendations)
			r.Get("/event/{eventType}", s.handleGetEventRecommendations)
			r.Get("/trending", s.handleGetTrending)
			r.Get("/personalized/{userID}", s.handleGetPersonalized)
			r.Post("/bundle-suggestions", s.handleGetBundleSuggestions)
		})
		
		// Adjacencies (for admin/debugging)
		r.Route("/adjacencies", func(r chi.Router) {
			r.Get("/", s.handleListAdjacencies)
			r.Get("/category/{categoryID}", s.handleGetCategoryAdjacencies)
			r.Get("/graph", s.handleGetAdjacencyGraph)
		})
		
		// Events
		r.Route("/events", func(r chi.Router) {
			r.Get("/", s.handleListEventTriggers)
			r.Get("/{eventSlug}/categories", s.handleGetEventCategories)
			r.Get("/{eventSlug}/checklist", s.handleGetEventChecklist)
		})
		
		// Tracking
		r.Route("/tracking", func(r chi.Router) {
			r.Post("/impression", s.handleTrackImpression)
			r.Post("/click", s.handleTrackClick)
			r.Post("/conversion", s.handleTrackConversion)
		})
		
		// Projects (user event planning)
		r.Route("/projects", func(r chi.Router) {
			r.Post("/", s.handleCreateProject)
			r.Get("/{projectID}", s.handleGetProject)
			r.Get("/{projectID}/recommendations", s.handleGetProjectRecommendations)
			r.Get("/{projectID}/progress", s.handleGetProjectProgress)
		})
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	
	server := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}
	
	s.logger.Info("Starting API server", "addr", addr)
	return server.ListenAndServe()
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// API request types
type GetRecommendationsRequest struct {
	UserID          string   `json:"user_id,omitempty"`
	SessionID       string   `json:"session_id,omitempty"`
	ProjectID       string   `json:"project_id,omitempty"`
	CurrentEntityID string   `json:"current_entity_id,omitempty"`
	CurrentEntityType string `json:"current_entity_type,omitempty"`
	EventType       string   `json:"event_type,omitempty"`
	Latitude        *float64 `json:"latitude,omitempty"`
	Longitude       *float64 `json:"longitude,omitempty"`
	BudgetMin       *float64 `json:"budget_min,omitempty"`
	BudgetMax       *float64 `json:"budget_max,omitempty"`
	Currency        string   `json:"currency,omitempty"`
	Limit           int      `json:"limit,omitempty"`
	DiversityFactor float64  `json:"diversity_factor,omitempty"`
	ExcludeIDs      []string `json:"exclude_ids,omitempty"`
}

type BundleSuggestionRequest struct {
	EventType    string   `json:"event_type"`
	CategoryIDs  []string `json:"category_ids,omitempty"`
	Budget       float64  `json:"budget,omitempty"`
	GuestCount   int      `json:"guest_count,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
}

type CreateProjectRequest struct {
	UserID       string   `json:"user_id"`
	Name         string   `json:"name"`
	EventType    string   `json:"event_type"`
	EventDate    string   `json:"event_date,omitempty"`
	Budget       float64  `json:"budget,omitempty"`
	GuestCount   int      `json:"guest_count,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
}

type TrackEventRequest struct {
	UserID           string `json:"user_id,omitempty"`
	SessionID        string `json:"session_id"`
	RecommendationID string `json:"recommendation_id"`
	EntityType       string `json:"entity_type"`
	EntityID         string `json:"entity_id"`
	Position         int    `json:"position,omitempty"`
	Source           string `json:"source,omitempty"`
}

// API response types
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type APIMeta struct {
	RequestID      string `json:"request_id,omitempty"`
	ProcessingTime int64  `json:"processing_time_ms,omitempty"`
	TotalCount     int    `json:"total_count,omitempty"`
	Page           int    `json:"page,omitempty"`
	PageSize       int    `json:"page_size,omitempty"`
}

type RecommendationItem struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	EntityType      string                 `json:"entity_type"`
	EntityID        string                 `json:"entity_id"`
	Score           float64                `json:"score"`
	Position        int                    `json:"position"`
	Explanation     string                 `json:"explanation"`
	Entity          interface{}            `json:"entity,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type AdjacencyItem struct {
	SourceCategoryID   string  `json:"source_category_id"`
	SourceCategoryName string  `json:"source_category_name"`
	TargetCategoryID   string  `json:"target_category_id"`
	TargetCategoryName string  `json:"target_category_name"`
	AdjacencyType      string  `json:"adjacency_type"`
	Score              float64 `json:"score"`
	RecommendationCopy string  `json:"recommendation_copy"`
	Context            string  `json:"context,omitempty"`
}

type EventTrigger struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	EventType   string   `json:"event_type"`
	ClusterType string   `json:"cluster_type"`
	Description string   `json:"description"`
	PeakMonths  []int    `json:"peak_months,omitempty"`
	AvgServices float64  `json:"avg_services_booked"`
	AvgSpend    float64  `json:"avg_spend"`
}

type EventCategory struct {
	CategoryID       string  `json:"category_id"`
	CategoryName     string  `json:"category_name"`
	CategorySlug     string  `json:"category_slug"`
	RoleType         string  `json:"role_type"`
	Phase            string  `json:"phase"`
	NecessityScore   float64 `json:"necessity_score"`
	PopularityScore  float64 `json:"popularity_score"`
	BudgetPercentage float64 `json:"budget_percentage"`
	BookingOffset    int     `json:"typical_booking_offset_days"`
}

type ProjectProgress struct {
	ProjectID           string            `json:"project_id"`
	TotalCategories     int               `json:"total_categories"`
	BookedCategories    int               `json:"booked_categories"`
	CompletionPercent   float64           `json:"completion_percentage"`
	TotalBudget         float64           `json:"total_budget"`
	SpentAmount         float64           `json:"spent_amount"`
	RemainingBudget     float64           `json:"remaining_budget"`
	PendingCategories   []EventCategory   `json:"pending_categories"`
	UpcomingDeadlines   []CategoryDeadline `json:"upcoming_deadlines"`
}

type CategoryDeadline struct {
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	DeadlineDate string `json:"deadline_date"`
	DaysRemaining int   `json:"days_remaining"`
	IsUrgent     bool   `json:"is_urgent"`
}

// =============================================================================
// HANDLERS
// =============================================================================

// Health & Ready checks
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"status": "healthy"}})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	
	if err := s.db.Ping(ctx); err != nil {
		s.respondError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Database not ready")
		return
	}
	
	s.respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"status": "ready"}})
}

// Main recommendations endpoint
func (s *Server) handleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	var req GetRecommendationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body")
		return
	}
	
	// Convert to engine request
	engineReq := s.convertToEngineRequest(&req)
	
	// Get recommendations
	resp, err := s.engine.GetRecommendations(r.Context(), engineReq)
	if err != nil {
		s.logger.Error("Failed to get recommendations", "error", err)
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get recommendations")
		return
	}
	
	// Convert to API response
	items := make([]RecommendationItem, len(resp.Recommendations))
	for i, rec := range resp.Recommendations {
		items[i] = RecommendationItem{
			ID:          rec.ID.String(),
			Type:        string(rec.Type),
			EntityType:  string(rec.EntityType),
			EntityID:    rec.EntityID.String(),
			Score:       rec.Score,
			Position:    rec.Position,
			Explanation: rec.ExplanationCopy,
			Metadata:    rec.Metadata,
		}
		
		// Enrich with entity details
		items[i].Entity = s.getEntityDetails(r.Context(), rec.EntityType, rec.EntityID)
	}
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    items,
		Meta: &APIMeta{
			RequestID:      middleware.GetReqID(r.Context()),
			ProcessingTime: resp.ProcessingTimeMs,
			TotalCount:     resp.TotalCandidates,
		},
	})
}

// Adjacent recommendations for a specific entity
func (s *Server) handleGetAdjacentRecommendations(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entityType")
	entityIDStr := chi.URLParam(r, "entityID")
	
	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid entity ID")
		return
	}
	
	limit := s.getIntParam(r, "limit", 10)
	context := r.URL.Query().Get("context")
	
	engineReq := &recommendation.RecommendationRequest{
		CurrentEntityID:   entityID,
		CurrentEntityType: recommendation.EntityType(entityType),
		EventType:         context,
		Limit:             limit,
	}
	
	resp, err := s.engine.GetRecommendations(r.Context(), engineReq)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get recommendations")
		return
	}
	
	items := s.convertRecommendations(r.Context(), resp.Recommendations)
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    items,
		Meta: &APIMeta{
			ProcessingTime: resp.ProcessingTimeMs,
			TotalCount:     len(items),
		},
	})
}

// Event-based recommendations
func (s *Server) handleGetEventRecommendations(w http.ResponseWriter, r *http.Request) {
	eventType := chi.URLParam(r, "eventType")
	limit := s.getIntParam(r, "limit", 20)
	
	userIDStr := r.URL.Query().Get("user_id")
	var userID uuid.UUID
	if userIDStr != "" {
		userID, _ = uuid.Parse(userIDStr)
	}
	
	engineReq := &recommendation.RecommendationRequest{
		UserID:    userID,
		EventType: eventType,
		Limit:     limit,
	}
	
	// Parse location if provided
	if lat := r.URL.Query().Get("latitude"); lat != "" {
		if lon := r.URL.Query().Get("longitude"); lon != "" {
			latF, _ := strconv.ParseFloat(lat, 64)
			lonF, _ := strconv.ParseFloat(lon, 64)
			engineReq.Location = &recommendation.GeoPoint{Latitude: latF, Longitude: lonF}
		}
	}
	
	resp, err := s.engine.GetRecommendations(r.Context(), engineReq)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get recommendations")
		return
	}
	
	items := s.convertRecommendations(r.Context(), resp.Recommendations)
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    items,
	})
}

// Trending services
func (s *Server) handleGetTrending(w http.ResponseWriter, r *http.Request) {
	limit := s.getIntParam(r, "limit", 20)
	
	engineReq := &recommendation.RecommendationRequest{
		RequestedTypes: []recommendation.RecommendationType{recommendation.TrendingService},
		Limit:          limit,
	}
	
	resp, err := s.engine.GetRecommendations(r.Context(), engineReq)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get trending")
		return
	}
	
	items := s.convertRecommendations(r.Context(), resp.Recommendations)
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    items,
	})
}

// Personalized recommendations for a user
func (s *Server) handleGetPersonalized(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}
	
	limit := s.getIntParam(r, "limit", 20)
	
	engineReq := &recommendation.RecommendationRequest{
		UserID:          userID,
		Limit:           limit,
		DiversityFactor: 0.4,
	}
	
	resp, err := s.engine.GetRecommendations(r.Context(), engineReq)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get recommendations")
		return
	}
	
	items := s.convertRecommendations(r.Context(), resp.Recommendations)
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    items,
	})
}

// Bundle suggestions
func (s *Server) handleGetBundleSuggestions(w http.ResponseWriter, r *http.Request) {
	var req BundleSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body")
		return
	}
	
	// Get bundle suggestions based on event type and already selected categories
	ctx := r.Context()
	
	query := `
		SELECT sb.id, sb.name, sb.slug, sb.short_description,
		       sb.discount_percentage, sb.min_savings_percentage,
		       sb.category_ids
		FROM service_bundles sb
		JOIN life_event_triggers let ON let.id = sb.event_trigger_id
		WHERE let.slug = $1
		  AND sb.is_active = TRUE
		  AND ($2::decimal IS NULL OR sb.target_budget_max >= $2)
		  AND ($3::int IS NULL OR sb.target_guest_max >= $3)
		ORDER BY sb.purchase_count DESC
		LIMIT 5
	`
	
	rows, err := s.db.Query(ctx, query, req.EventType, req.Budget, req.GuestCount)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get bundles")
		return
	}
	defer rows.Close()
	
	var bundles []map[string]interface{}
	for rows.Next() {
		var id, name, slug, description string
		var discount, minSavings *float64
		var categoryIDs []uuid.UUID
		
		if err := rows.Scan(&id, &name, &slug, &description, &discount, &minSavings, &categoryIDs); err != nil {
			continue
		}
		
		bundles = append(bundles, map[string]interface{}{
			"id":                  id,
			"name":                name,
			"slug":                slug,
			"description":         description,
			"discount_percentage": discount,
			"min_savings":         minSavings,
			"category_count":      len(categoryIDs),
		})
	}
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    bundles,
	})
}

// List all adjacencies
func (s *Server) handleListAdjacencies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	context := r.URL.Query().Get("context")
	limit := s.getIntParam(r, "limit", 100)
	
	query := `
		SELECT sa.source_category_id, sc1.name, sa.target_category_id, sc2.name,
		       sa.adjacency_type, sa.computed_score, sa.recommendation_copy,
		       COALESCE(sa.trigger_context, '')
		FROM service_adjacencies sa
		JOIN service_categories sc1 ON sc1.id = sa.source_category_id
		JOIN service_categories sc2 ON sc2.id = sa.target_category_id
		WHERE sa.is_active = TRUE
		  AND ($1 = '' OR sa.trigger_context = $1 OR sa.trigger_context IS NULL)
		ORDER BY sa.computed_score DESC
		LIMIT $2
	`
	
	rows, err := s.db.Query(ctx, query, context, limit)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list adjacencies")
		return
	}
	defer rows.Close()
	
	var adjacencies []AdjacencyItem
	for rows.Next() {
		var adj AdjacencyItem
		if err := rows.Scan(&adj.SourceCategoryID, &adj.SourceCategoryName,
			&adj.TargetCategoryID, &adj.TargetCategoryName,
			&adj.AdjacencyType, &adj.Score, &adj.RecommendationCopy, &adj.Context); err != nil {
			continue
		}
		adjacencies = append(adjacencies, adj)
	}
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    adjacencies,
		Meta:    &APIMeta{TotalCount: len(adjacencies)},
	})
}

// Get adjacencies for a specific category
func (s *Server) handleGetCategoryAdjacencies(w http.ResponseWriter, r *http.Request) {
	categoryIDStr := chi.URLParam(r, "categoryID")
	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid category ID")
		return
	}
	
	ctx := r.Context()
	
	query := `
		SELECT sa.target_category_id, sc.name, sc.slug,
		       sa.adjacency_type, sa.computed_score, sa.recommendation_copy
		FROM service_adjacencies sa
		JOIN service_categories sc ON sc.id = sa.target_category_id
		WHERE sa.source_category_id = $1 AND sa.is_active = TRUE
		ORDER BY sa.computed_score DESC
	`
	
	rows, err := s.db.Query(ctx, query, categoryID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get adjacencies")
		return
	}
	defer rows.Close()
	
	var adjacencies []map[string]interface{}
	for rows.Next() {
		var targetID, name, slug, adjType, copy string
		var score float64
		
		if err := rows.Scan(&targetID, &name, &slug, &adjType, &score, &copy); err != nil {
			continue
		}
		
		adjacencies = append(adjacencies, map[string]interface{}{
			"category_id":        targetID,
			"category_name":      name,
			"category_slug":      slug,
			"adjacency_type":     adjType,
			"score":              score,
			"recommendation_copy": copy,
		})
	}
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    adjacencies,
	})
}

// Get full adjacency graph (for visualization)
func (s *Server) handleGetAdjacencyGraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	context := r.URL.Query().Get("context")
	
	// Get nodes (categories)
	nodesQuery := `
		SELECT DISTINCT id, name, slug, cluster_type
		FROM service_categories
		WHERE level = 1 AND is_active = TRUE
	`
	
	nodeRows, err := s.db.Query(ctx, nodesQuery)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get graph")
		return
	}
	defer nodeRows.Close()
	
	var nodes []map[string]interface{}
	for nodeRows.Next() {
		var id, name, slug, cluster string
		if err := nodeRows.Scan(&id, &name, &slug, &cluster); err != nil {
			continue
		}
		nodes = append(nodes, map[string]interface{}{
			"id":      id,
			"name":    name,
			"slug":    slug,
			"cluster": cluster,
		})
	}
	
	// Get edges (adjacencies)
	edgesQuery := `
		SELECT source_category_id, target_category_id, adjacency_type, computed_score
		FROM service_adjacencies
		WHERE is_active = TRUE
		  AND ($1 = '' OR trigger_context = $1 OR trigger_context IS NULL)
	`
	
	edgeRows, err := s.db.Query(ctx, edgesQuery, context)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get graph")
		return
	}
	defer edgeRows.Close()
	
	var edges []map[string]interface{}
	for edgeRows.Next() {
		var source, target, adjType string
		var score float64
		if err := edgeRows.Scan(&source, &target, &adjType, &score); err != nil {
			continue
		}
		edges = append(edges, map[string]interface{}{
			"source": source,
			"target": target,
			"type":   adjType,
			"weight": score,
		})
	}
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"nodes": nodes,
			"edges": edges,
		},
	})
}

// List all event triggers
func (s *Server) handleListEventTriggers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clusterType := r.URL.Query().Get("cluster")
	
	query := `
		SELECT id, name, slug, event_type, cluster_type, description,
		       peak_months, avg_services_booked, avg_spend
		FROM life_event_triggers
		WHERE is_active = TRUE
		  AND ($1 = '' OR cluster_type = $1)
		ORDER BY display_order, name
	`
	
	rows, err := s.db.Query(ctx, query, clusterType)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list events")
		return
	}
	defer rows.Close()
	
	var events []EventTrigger
	for rows.Next() {
		var e EventTrigger
		var peakMonths []int
		
		if err := rows.Scan(&e.ID, &e.Name, &e.Slug, &e.EventType, &e.ClusterType,
			&e.Description, &peakMonths, &e.AvgServices, &e.AvgSpend); err != nil {
			continue
		}
		e.PeakMonths = peakMonths
		events = append(events, e)
	}
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    events,
	})
}

// Get categories for an event
func (s *Server) handleGetEventCategories(w http.ResponseWriter, r *http.Request) {
	eventSlug := chi.URLParam(r, "eventSlug")
	ctx := r.Context()
	
	query := `
		SELECT sc.id, sc.name, sc.slug, ecm.role_type, ecm.phase,
		       ecm.necessity_score, ecm.popularity_score, 
		       ecm.typical_budget_percentage, ecm.typical_booking_offset_days
		FROM event_category_mappings ecm
		JOIN life_event_triggers let ON let.id = ecm.event_trigger_id
		JOIN service_categories sc ON sc.id = ecm.category_id
		WHERE let.slug = $1 AND ecm.is_active = TRUE
		ORDER BY ecm.necessity_score DESC, ecm.popularity_score DESC
	`
	
	rows, err := s.db.Query(ctx, query, eventSlug)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get categories")
		return
	}
	defer rows.Close()
	
	var categories []EventCategory
	for rows.Next() {
		var c EventCategory
		if err := rows.Scan(&c.CategoryID, &c.CategoryName, &c.CategorySlug,
			&c.RoleType, &c.Phase, &c.NecessityScore, &c.PopularityScore,
			&c.BudgetPercentage, &c.BookingOffset); err != nil {
			continue
		}
		categories = append(categories, c)
	}
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    categories,
	})
}

// Get event checklist (organized by phase)
func (s *Server) handleGetEventChecklist(w http.ResponseWriter, r *http.Request) {
	eventSlug := chi.URLParam(r, "eventSlug")
	ctx := r.Context()
	
	// Get event date if provided
	eventDateStr := r.URL.Query().Get("event_date")
	var eventDate *time.Time
	if eventDateStr != "" {
		t, err := time.Parse("2006-01-02", eventDateStr)
		if err == nil {
			eventDate = &t
		}
	}
	
	query := `
		SELECT sc.id, sc.name, sc.slug, ecm.role_type, ecm.phase,
		       ecm.necessity_score, ecm.typical_booking_offset_days
		FROM event_category_mappings ecm
		JOIN life_event_triggers let ON let.id = ecm.event_trigger_id
		JOIN service_categories sc ON sc.id = ecm.category_id
		WHERE let.slug = $1 AND ecm.is_active = TRUE
		ORDER BY ecm.typical_booking_offset_days DESC
	`
	
	rows, err := s.db.Query(ctx, query, eventSlug)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get checklist")
		return
	}
	defer rows.Close()
	
	checklist := make(map[string][]map[string]interface{})
	
	for rows.Next() {
		var catID, name, slug, roleType, phase string
		var necessityScore float64
		var bookingOffset int
		
		if err := rows.Scan(&catID, &name, &slug, &roleType, &phase, &necessityScore, &bookingOffset); err != nil {
			continue
		}
		
		item := map[string]interface{}{
			"category_id":     catID,
			"category_name":   name,
			"category_slug":   slug,
			"role_type":       roleType,
			"necessity_score": necessityScore,
			"booking_offset":  bookingOffset,
		}
		
		if eventDate != nil {
			deadline := eventDate.AddDate(0, 0, -bookingOffset)
			item["deadline"] = deadline.Format("2006-01-02")
			item["days_until_deadline"] = int(time.Until(deadline).Hours() / 24)
		}
		
		checklist[phase] = append(checklist[phase], item)
	}
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    checklist,
	})
}

// Tracking handlers
func (s *Server) handleTrackImpression(w http.ResponseWriter, r *http.Request) {
	var req TrackEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body")
		return
	}
	
	// Log impression asynchronously
	go func() {
		ctx := context.Background()
		_, _ = s.db.Exec(ctx, `
			UPDATE recommendation_events 
			SET was_impressed = TRUE 
			WHERE id = $1
		`, req.RecommendationID)
	}()
	
	s.respondJSON(w, http.StatusOK, APIResponse{Success: true})
}

func (s *Server) handleTrackClick(w http.ResponseWriter, r *http.Request) {
	var req TrackEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body")
		return
	}
	
	go func() {
		ctx := context.Background()
		_, _ = s.db.Exec(ctx, `
			UPDATE recommendation_events 
			SET was_clicked = TRUE, clicked_at = NOW() 
			WHERE id = $1
		`, req.RecommendationID)
		
		// Also log to user_interactions
		_, _ = s.db.Exec(ctx, `
			INSERT INTO user_interactions (user_id, entity_type, entity_id, interaction_type, session_id)
			VALUES ($1, $2, $3, 'click', $4)
		`, req.UserID, req.EntityType, req.EntityID, req.SessionID)
	}()
	
	s.respondJSON(w, http.StatusOK, APIResponse{Success: true})
}

func (s *Server) handleTrackConversion(w http.ResponseWriter, r *http.Request) {
	var req TrackEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body")
		return
	}
	
	go func() {
		ctx := context.Background()
		_, _ = s.db.Exec(ctx, `
			UPDATE recommendation_events 
			SET was_converted = TRUE, converted_at = NOW() 
			WHERE id = $1
		`, req.RecommendationID)
		
		// Update adjacency analytics
		_, _ = s.db.Exec(ctx, `
			UPDATE service_adjacencies
			SET conversion_count = conversion_count + 1
			WHERE id IN (
				SELECT sa.id FROM service_adjacencies sa
				JOIN recommendation_events re ON re.id = $1
				WHERE sa.source_category_id = (
					SELECT category_id FROM services WHERE id = re.source_entity_id
				)
				AND sa.target_category_id = (
					SELECT category_id FROM services WHERE id = re.recommended_entity_id
				)
			)
		`, req.RecommendationID)
	}()
	
	s.respondJSON(w, http.StatusOK, APIResponse{Success: true})
}

// Project handlers
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body")
		return
	}
	
	ctx := r.Context()
	
	// Get event trigger ID
	var eventTriggerID uuid.UUID
	err := s.db.QueryRow(ctx, `SELECT id FROM life_event_triggers WHERE slug = $1`, req.EventType).Scan(&eventTriggerID)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_EVENT", "Invalid event type")
		return
	}
	
	// Create project
	var projectID uuid.UUID
	var eventDate *time.Time
	if req.EventDate != "" {
		t, _ := time.Parse("2006-01-02", req.EventDate)
		eventDate = &t
	}
	
	err = s.db.QueryRow(ctx, `
		INSERT INTO projects (user_id, name, event_trigger_id, event_type, event_date, 
		                      total_budget, expected_guests, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'planning')
		RETURNING id
	`, req.UserID, req.Name, eventTriggerID, req.EventType, eventDate, req.Budget, req.GuestCount).Scan(&projectID)
	
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create project")
		return
	}
	
	s.respondJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data: map[string]string{
			"project_id": projectID.String(),
		},
	})
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid project ID")
		return
	}
	
	ctx := r.Context()
	
	var project map[string]interface{}
	row := s.db.QueryRow(ctx, `
		SELECT p.id, p.name, p.event_type, p.event_date, p.status,
		       p.total_budget, p.expected_guests, p.completion_percentage,
		       let.name as event_name
		FROM projects p
		LEFT JOIN life_event_triggers let ON let.id = p.event_trigger_id
		WHERE p.id = $1
	`, projectID)
	
	var id, name, eventType, status, eventName string
	var eventDate *time.Time
	var budget *float64
	var guests *int
	var completion float64
	
	if err := row.Scan(&id, &name, &eventType, &eventDate, &status, &budget, &guests, &completion, &eventName); err != nil {
		s.respondError(w, http.StatusNotFound, "NOT_FOUND", "Project not found")
		return
	}
	
	project = map[string]interface{}{
		"id":                    id,
		"name":                  name,
		"event_type":            eventType,
		"event_name":            eventName,
		"status":                status,
		"total_budget":          budget,
		"expected_guests":       guests,
		"completion_percentage": completion,
	}
	if eventDate != nil {
		project["event_date"] = eventDate.Format("2006-01-02")
	}
	
	s.respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: project})
}

func (s *Server) handleGetProjectRecommendations(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid project ID")
		return
	}
	
	// Get project details
	ctx := r.Context()
	var userID uuid.UUID
	var eventType string
	var budget *float64
	
	err = s.db.QueryRow(ctx, `
		SELECT user_id, event_type, total_budget FROM projects WHERE id = $1
	`, projectID).Scan(&userID, &eventType, &budget)
	
	if err != nil {
		s.respondError(w, http.StatusNotFound, "NOT_FOUND", "Project not found")
		return
	}
	
	// Build recommendation request
	engineReq := &recommendation.RecommendationRequest{
		UserID:    userID,
		ProjectID: projectID,
		EventType: eventType,
		Limit:     20,
	}
	
	if budget != nil {
		engineReq.Budget = &recommendation.BudgetRange{Max: *budget}
	}
	
	resp, err := s.engine.GetRecommendations(ctx, engineReq)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get recommendations")
		return
	}
	
	items := s.convertRecommendations(ctx, resp.Recommendations)
	
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    items,
	})
}

func (s *Server) handleGetProjectProgress(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid project ID")
		return
	}
	
	ctx := r.Context()
	
	// Get project and calculate progress
	var eventType string
	var eventDate *time.Time
	var totalBudget, spentAmount float64
	
	err = s.db.QueryRow(ctx, `
		SELECT p.event_type, p.event_date, COALESCE(p.total_budget, 0),
		       COALESCE(p.total_booked, 0)
		FROM projects p WHERE p.id = $1
	`, projectID).Scan(&eventType, &eventDate, &totalBudget, &spentAmount)
	
	if err != nil {
		s.respondError(w, http.StatusNotFound, "NOT_FOUND", "Project not found")
		return
	}
	
	// Get booked categories
	var bookedCats []uuid.UUID
	rows, _ := s.db.Query(ctx, `
		SELECT DISTINCT s.category_id
		FROM bookings b
		JOIN services s ON s.id = b.service_id
		WHERE b.project_id = $1 AND b.status != 'cancelled'
	`, projectID)
	for rows.Next() {
		var catID uuid.UUID
		rows.Scan(&catID)
		bookedCats = append(bookedCats, catID)
	}
	rows.Close()
	
	// Get total required categories
	var totalCats int
	s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM event_category_mappings ecm
		JOIN life_event_triggers let ON let.id = ecm.event_trigger_id
		WHERE let.slug = $1 AND ecm.role_type IN ('primary', 'secondary')
	`, eventType).Scan(&totalCats)
	
	progress := ProjectProgress{
		ProjectID:         projectID.String(),
		TotalCategories:   totalCats,
		BookedCategories:  len(bookedCats),
		CompletionPercent: float64(len(bookedCats)) / float64(totalCats) * 100,
		TotalBudget:       totalBudget,
		SpentAmount:       spentAmount,
		RemainingBudget:   totalBudget - spentAmount,
	}
	
	s.respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: progress})
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (s *Server) convertToEngineRequest(req *GetRecommendationsRequest) *recommendation.RecommendationRequest {
	engineReq := &recommendation.RecommendationRequest{
		EventType:       req.EventType,
		Limit:           req.Limit,
		DiversityFactor: req.DiversityFactor,
	}
	
	if req.UserID != "" {
		engineReq.UserID, _ = uuid.Parse(req.UserID)
	}
	if req.SessionID != "" {
		engineReq.SessionID, _ = uuid.Parse(req.SessionID)
	}
	if req.ProjectID != "" {
		engineReq.ProjectID, _ = uuid.Parse(req.ProjectID)
	}
	if req.CurrentEntityID != "" {
		engineReq.CurrentEntityID, _ = uuid.Parse(req.CurrentEntityID)
	}
	if req.CurrentEntityType != "" {
		engineReq.CurrentEntityType = recommendation.EntityType(req.CurrentEntityType)
	}
	if req.Latitude != nil && req.Longitude != nil {
		engineReq.Location = &recommendation.GeoPoint{
			Latitude:  *req.Latitude,
			Longitude: *req.Longitude,
		}
	}
	if req.BudgetMin != nil || req.BudgetMax != nil {
		engineReq.Budget = &recommendation.BudgetRange{Currency: req.Currency}
		if req.BudgetMin != nil {
			engineReq.Budget.Min = *req.BudgetMin
		}
		if req.BudgetMax != nil {
			engineReq.Budget.Max = *req.BudgetMax
		}
	}
	for _, id := range req.ExcludeIDs {
		if parsed, err := uuid.Parse(id); err == nil {
			engineReq.ExcludeIDs = append(engineReq.ExcludeIDs, parsed)
		}
	}
	
	return engineReq
}

func (s *Server) convertRecommendations(ctx context.Context, recs []recommendation.Recommendation) []RecommendationItem {
	items := make([]RecommendationItem, len(recs))
	for i, rec := range recs {
		items[i] = RecommendationItem{
			ID:          rec.ID.String(),
			Type:        string(rec.Type),
			EntityType:  string(rec.EntityType),
			EntityID:    rec.EntityID.String(),
			Score:       rec.Score,
			Position:    rec.Position,
			Explanation: rec.ExplanationCopy,
			Metadata:    rec.Metadata,
		}
		items[i].Entity = s.getEntityDetails(ctx, rec.EntityType, rec.EntityID)
	}
	return items
}

func (s *Server) getEntityDetails(ctx context.Context, entityType recommendation.EntityType, entityID uuid.UUID) interface{} {
	switch entityType {
	case recommendation.EntityService:
		var service map[string]interface{}
		row := s.db.QueryRow(ctx, `
			SELECT s.id, s.name, s.short_description, s.base_price, s.rating_average,
			       v.business_name, v.slug as vendor_slug, sc.name as category_name
			FROM services s
			JOIN vendors v ON v.id = s.vendor_id
			JOIN service_categories sc ON sc.id = s.category_id
			WHERE s.id = $1
		`, entityID)
		
		var id, name, desc, vendorName, vendorSlug, catName string
		var price, rating *float64
		if err := row.Scan(&id, &name, &desc, &price, &rating, &vendorName, &vendorSlug, &catName); err == nil {
			service = map[string]interface{}{
				"id":            id,
				"name":          name,
				"description":   desc,
				"price":         price,
				"rating":        rating,
				"vendor_name":   vendorName,
				"vendor_slug":   vendorSlug,
				"category_name": catName,
			}
		}
		return service
	
	case recommendation.EntityVendor:
		var vendor map[string]interface{}
		row := s.db.QueryRow(ctx, `
			SELECT id, business_name, slug, short_description, rating_average, rating_count
			FROM vendors WHERE id = $1
		`, entityID)
		
		var id, name, slug, desc string
		var rating *float64
		var ratingCount *int
		if err := row.Scan(&id, &name, &slug, &desc, &rating, &ratingCount); err == nil {
			vendor = map[string]interface{}{
				"id":           id,
				"name":         name,
				"slug":         slug,
				"description":  desc,
				"rating":       rating,
				"rating_count": ratingCount,
			}
		}
		return vendor
	
	case recommendation.EntityCategory:
		var category map[string]interface{}
		row := s.db.QueryRow(ctx, `
			SELECT id, name, slug, short_description, icon_url
			FROM service_categories WHERE id = $1
		`, entityID)
		
		var id, name, slug, desc string
		var iconURL *string
		if err := row.Scan(&id, &name, &slug, &desc, &iconURL); err == nil {
			category = map[string]interface{}{
				"id":          id,
				"name":        name,
				"slug":        slug,
				"description": desc,
				"icon_url":    iconURL,
			}
		}
		return category
	}
	
	return nil
}

func (s *Server) getIntParam(r *http.Request, name string, defaultVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	if i, err := strconv.Atoi(val); err == nil {
		return i
	}
	return defaultVal
}

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) respondError(w http.ResponseWriter, status int, code, message string) {
	s.respondJSON(w, status, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}
