// Package api provides the HTTP REST API for the recommendation engine
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	recommendation "vendorplatform/recommendation-engine"
)

// =============================================================================
// SERVER
// =============================================================================

// Server is the HTTP API server
type Server struct {
	engine *recommendation.Engine
	router *chi.Mux
	logger *slog.Logger
}

// NewServer creates a new API server
func NewServer(engine *recommendation.Engine, logger *slog.Logger) *Server {
	s := &Server{
		engine: engine,
		router: chi.NewRouter(),
		logger: logger,
	}
	s.setupRoutes()
	return s
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) setupRoutes() {
	r := s.router

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", s.healthCheck)
	r.Get("/ready", s.readyCheck)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Recommendations
		r.Route("/recommendations", func(r chi.Router) {
			r.Post("/", s.getRecommendations)
			r.Post("/adjacent", s.getAdjacentRecommendations)
			r.Post("/event-based", s.getEventBasedRecommendations)
			r.Post("/trending", s.getTrendingRecommendations)
			r.Post("/personalized", s.getPersonalizedRecommendations)
			r.Post("/bundle", s.getBundleRecommendations)
		})

		// Feedback
		r.Route("/feedback", func(r chi.Router) {
			r.Post("/click", s.recordClick)
			r.Post("/conversion", s.recordConversion)
			r.Post("/dismiss", s.recordDismiss)
		})

		// Adjacency management (admin)
		r.Route("/admin/adjacencies", func(r chi.Router) {
			r.Get("/", s.listAdjacencies)
			r.Post("/", s.createAdjacency)
			r.Put("/{id}", s.updateAdjacency)
			r.Delete("/{id}", s.deleteAdjacency)
			r.Post("/refresh", s.refreshAdjacencyGraph)
		})

		// Events management (admin)
		r.Route("/admin/events", func(r chi.Router) {
			r.Get("/", s.listEventTriggers)
			r.Get("/{id}/categories", s.getEventCategories)
		})
	})
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// RecommendationAPIRequest is the API request for recommendations
type RecommendationAPIRequest struct {
	UserID           string   `json:"user_id,omitempty"`
	SessionID        string   `json:"session_id,omitempty"`
	ProjectID        string   `json:"project_id,omitempty"`
	CurrentServiceID string   `json:"current_service_id,omitempty"`
	CurrentVendorID  string   `json:"current_vendor_id,omitempty"`
	CurrentCategoryID string  `json:"current_category_id,omitempty"`
	EventType        string   `json:"event_type,omitempty"`
	Latitude         *float64 `json:"latitude,omitempty"`
	Longitude        *float64 `json:"longitude,omitempty"`
	BudgetMin        *float64 `json:"budget_min,omitempty"`
	BudgetMax        *float64 `json:"budget_max,omitempty"`
	Currency         string   `json:"currency,omitempty"`
	Limit            int      `json:"limit,omitempty"`
	ExcludeIDs       []string `json:"exclude_ids,omitempty"`
	DiversityFactor  float64  `json:"diversity_factor,omitempty"`
}

// RecommendationAPIResponse is the API response
type RecommendationAPIResponse struct {
	Success          bool                          `json:"success"`
	Data             *RecommendationDataResponse   `json:"data,omitempty"`
	Error            *ErrorResponse                `json:"error,omitempty"`
	Meta             *MetaResponse                 `json:"meta,omitempty"`
}

// RecommendationDataResponse contains the recommendation data
type RecommendationDataResponse struct {
	Recommendations []RecommendationItem `json:"recommendations"`
	TotalCandidates int                  `json:"total_candidates"`
}

// RecommendationItem is a single recommendation in the API response
type RecommendationItem struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	EntityType      string                 `json:"entity_type"`
	EntityID        string                 `json:"entity_id"`
	Score           float64                `json:"score"`
	Position        int                    `json:"position"`
	Explanation     string                 `json:"explanation"`
	Entity          *EntityDetails         `json:"entity,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// EntityDetails contains enriched entity information
type EntityDetails struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description,omitempty"`
	ImageURL    string   `json:"image_url,omitempty"`
	Rating      float64  `json:"rating,omitempty"`
	RatingCount int      `json:"rating_count,omitempty"`
	Price       *Price   `json:"price,omitempty"`
	Vendor      *Vendor  `json:"vendor,omitempty"`
	Category    *Category `json:"category,omitempty"`
}

// Price represents pricing information
type Price struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Unit     string  `json:"unit,omitempty"`
	Model    string  `json:"model"`
}

// Vendor represents vendor information
type Vendor struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Slug     string  `json:"slug"`
	Rating   float64 `json:"rating"`
	Verified bool    `json:"verified"`
}

// Category represents category information
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// MetaResponse contains metadata about the response
type MetaResponse struct {
	RequestID        string `json:"request_id"`
	ProcessingTimeMs int64  `json:"processing_time_ms"`
	AlgorithmVersion string `json:"algorithm_version"`
	ExperimentID     string `json:"experiment_id,omitempty"`
	Variant          string `json:"variant,omitempty"`
}

// FeedbackRequest is the request for feedback endpoints
type FeedbackRequest struct {
	UserID           string `json:"user_id,omitempty"`
	SessionID        string `json:"session_id"`
	RecommendationID string `json:"recommendation_id"`
	EntityType       string `json:"entity_type"`
	EntityID         string `json:"entity_id"`
	Position         int    `json:"position"`
	SourcePage       string `json:"source_page,omitempty"`
}

// =============================================================================
// HANDLERS
// =============================================================================

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func (s *Server) readyCheck(w http.ResponseWriter, r *http.Request) {
	// Could add database connectivity check here
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) getRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	var req RecommendationAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", err.Error())
		return
	}

	// Convert to internal request
	internalReq, err := s.convertToInternalRequest(&req)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_PARAMS", "Invalid parameters", err.Error())
		return
	}

	// Get recommendations
	resp, err := s.engine.GetRecommendations(ctx, internalReq)
	if err != nil {
		s.logger.Error("failed to get recommendations", "error", err, "request_id", requestID)
		s.errorResponse(w, http.StatusInternalServerError, "ENGINE_ERROR", "Failed to generate recommendations", "")
		return
	}

	// Enrich and convert response
	apiResp := s.convertToAPIResponse(ctx, resp, requestID)
	s.jsonResponse(w, http.StatusOK, apiResp)
}

func (s *Server) getAdjacentRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	var req RecommendationAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", err.Error())
		return
	}

	// Require current entity
	if req.CurrentServiceID == "" && req.CurrentCategoryID == "" && req.CurrentVendorID == "" {
		s.errorResponse(w, http.StatusBadRequest, "MISSING_PARAMS", "Must provide current_service_id, current_category_id, or current_vendor_id", "")
		return
	}

	internalReq, err := s.convertToInternalRequest(&req)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_PARAMS", "Invalid parameters", err.Error())
		return
	}

	// Limit to adjacent only
	internalReq.RequestedTypes = []recommendation.RecommendationType{recommendation.AdjacentService}

	resp, err := s.engine.GetRecommendations(ctx, internalReq)
	if err != nil {
		s.logger.Error("failed to get adjacent recommendations", "error", err, "request_id", requestID)
		s.errorResponse(w, http.StatusInternalServerError, "ENGINE_ERROR", "Failed to generate recommendations", "")
		return
	}

	apiResp := s.convertToAPIResponse(ctx, resp, requestID)
	s.jsonResponse(w, http.StatusOK, apiResp)
}

func (s *Server) getEventBasedRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	var req RecommendationAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", err.Error())
		return
	}

	// Require event type
	if req.EventType == "" {
		s.errorResponse(w, http.StatusBadRequest, "MISSING_PARAMS", "Must provide event_type", "")
		return
	}

	internalReq, err := s.convertToInternalRequest(&req)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_PARAMS", "Invalid parameters", err.Error())
		return
	}

	internalReq.RequestedTypes = []recommendation.RecommendationType{recommendation.EventBasedSuggest}

	resp, err := s.engine.GetRecommendations(ctx, internalReq)
	if err != nil {
		s.logger.Error("failed to get event-based recommendations", "error", err, "request_id", requestID)
		s.errorResponse(w, http.StatusInternalServerError, "ENGINE_ERROR", "Failed to generate recommendations", "")
		return
	}

	apiResp := s.convertToAPIResponse(ctx, resp, requestID)
	s.jsonResponse(w, http.StatusOK, apiResp)
}

func (s *Server) getTrendingRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	var req RecommendationAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", err.Error())
		return
	}

	internalReq, err := s.convertToInternalRequest(&req)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_PARAMS", "Invalid parameters", err.Error())
		return
	}

	internalReq.RequestedTypes = []recommendation.RecommendationType{recommendation.TrendingService}

	resp, err := s.engine.GetRecommendations(ctx, internalReq)
	if err != nil {
		s.logger.Error("failed to get trending recommendations", "error", err, "request_id", requestID)
		s.errorResponse(w, http.StatusInternalServerError, "ENGINE_ERROR", "Failed to generate recommendations", "")
		return
	}

	apiResp := s.convertToAPIResponse(ctx, resp, requestID)
	s.jsonResponse(w, http.StatusOK, apiResp)
}

func (s *Server) getPersonalizedRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	var req RecommendationAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", err.Error())
		return
	}

	// Require user ID for personalization
	if req.UserID == "" {
		s.errorResponse(w, http.StatusBadRequest, "MISSING_PARAMS", "Must provide user_id for personalized recommendations", "")
		return
	}

	internalReq, err := s.convertToInternalRequest(&req)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_PARAMS", "Invalid parameters", err.Error())
		return
	}

	internalReq.RequestedTypes = []recommendation.RecommendationType{
		recommendation.CollaborativeFilter,
		recommendation.PersonalizedPick,
	}

	resp, err := s.engine.GetRecommendations(ctx, internalReq)
	if err != nil {
		s.logger.Error("failed to get personalized recommendations", "error", err, "request_id", requestID)
		s.errorResponse(w, http.StatusInternalServerError, "ENGINE_ERROR", "Failed to generate recommendations", "")
		return
	}

	apiResp := s.convertToAPIResponse(ctx, resp, requestID)
	s.jsonResponse(w, http.StatusOK, apiResp)
}

func (s *Server) getBundleRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	var req RecommendationAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", err.Error())
		return
	}

	internalReq, err := s.convertToInternalRequest(&req)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_PARAMS", "Invalid parameters", err.Error())
		return
	}

	internalReq.RequestedTypes = []recommendation.RecommendationType{recommendation.BundleSuggestion}

	resp, err := s.engine.GetRecommendations(ctx, internalReq)
	if err != nil {
		s.logger.Error("failed to get bundle recommendations", "error", err, "request_id", requestID)
		s.errorResponse(w, http.StatusInternalServerError, "ENGINE_ERROR", "Failed to generate recommendations", "")
		return
	}

	apiResp := s.convertToAPIResponse(ctx, resp, requestID)
	s.jsonResponse(w, http.StatusOK, apiResp)
}

// Feedback handlers
func (s *Server) recordClick(w http.ResponseWriter, r *http.Request) {
	var req FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", err.Error())
		return
	}

	// Record click event (async)
	go s.recordFeedbackEvent(r.Context(), "click", &req)

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "recorded"})
}

func (s *Server) recordConversion(w http.ResponseWriter, r *http.Request) {
	var req FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", err.Error())
		return
	}

	go s.recordFeedbackEvent(r.Context(), "conversion", &req)

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "recorded"})
}

func (s *Server) recordDismiss(w http.ResponseWriter, r *http.Request) {
	var req FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", err.Error())
		return
	}

	go s.recordFeedbackEvent(r.Context(), "dismiss", &req)

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "recorded"})
}

func (s *Server) recordFeedbackEvent(ctx context.Context, eventType string, req *FeedbackRequest) {
	// Would insert into recommendation_events table
	s.logger.Info("feedback recorded",
		"event_type", eventType,
		"recommendation_id", req.RecommendationID,
		"entity_id", req.EntityID,
		"position", req.Position,
	)
}

// Admin handlers
func (s *Server) listAdjacencies(w http.ResponseWriter, r *http.Request) {
	// Implementation would query service_adjacencies table
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "not_implemented"})
}

func (s *Server) createAdjacency(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "not_implemented"})
}

func (s *Server) updateAdjacency(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "not_implemented"})
}

func (s *Server) deleteAdjacency(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "not_implemented"})
}

func (s *Server) refreshAdjacencyGraph(w http.ResponseWriter, r *http.Request) {
	// Would trigger engine.adjacencyGraph.Load()
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "refresh_triggered"})
}

func (s *Server) listEventTriggers(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "not_implemented"})
}

func (s *Server) getEventCategories(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "not_implemented"})
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (s *Server) convertToInternalRequest(req *RecommendationAPIRequest) (*recommendation.RecommendationRequest, error) {
	internal := &recommendation.RecommendationRequest{
		Limit:           req.Limit,
		DiversityFactor: req.DiversityFactor,
		EventType:       req.EventType,
	}

	if req.UserID != "" {
		id, err := uuid.Parse(req.UserID)
		if err != nil {
			return nil, err
		}
		internal.UserID = id
	}

	if req.SessionID != "" {
		id, err := uuid.Parse(req.SessionID)
		if err != nil {
			return nil, err
		}
		internal.SessionID = id
	}

	if req.ProjectID != "" {
		id, err := uuid.Parse(req.ProjectID)
		if err != nil {
			return nil, err
		}
		internal.ProjectID = id
	}

	// Determine current entity
	if req.CurrentServiceID != "" {
		id, err := uuid.Parse(req.CurrentServiceID)
		if err != nil {
			return nil, err
		}
		internal.CurrentEntityID = id
		internal.CurrentEntityType = recommendation.EntityService
	} else if req.CurrentCategoryID != "" {
		id, err := uuid.Parse(req.CurrentCategoryID)
		if err != nil {
			return nil, err
		}
		internal.CurrentEntityID = id
		internal.CurrentEntityType = recommendation.EntityCategory
	} else if req.CurrentVendorID != "" {
		id, err := uuid.Parse(req.CurrentVendorID)
		if err != nil {
			return nil, err
		}
		internal.CurrentEntityID = id
		internal.CurrentEntityType = recommendation.EntityVendor
	}

	// Location
	if req.Latitude != nil && req.Longitude != nil {
		internal.Location = &recommendation.GeoPoint{
			Latitude:  *req.Latitude,
			Longitude: *req.Longitude,
		}
	}

	// Budget
	if req.BudgetMin != nil || req.BudgetMax != nil {
		internal.Budget = &recommendation.BudgetRange{
			Currency: req.Currency,
		}
		if req.BudgetMin != nil {
			internal.Budget.Min = *req.BudgetMin
		}
		if req.BudgetMax != nil {
			internal.Budget.Max = *req.BudgetMax
		}
	}

	// Excludes
	for _, idStr := range req.ExcludeIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		internal.ExcludeIDs = append(internal.ExcludeIDs, id)
	}

	return internal, nil
}

func (s *Server) convertToAPIResponse(ctx context.Context, resp *recommendation.RecommendationResponse, requestID string) *RecommendationAPIResponse {
	items := make([]RecommendationItem, 0, len(resp.Recommendations))

	for _, rec := range resp.Recommendations {
		item := RecommendationItem{
			ID:          rec.ID.String(),
			Type:        string(rec.Type),
			EntityType:  string(rec.EntityType),
			EntityID:    rec.EntityID.String(),
			Score:       rec.Score,
			Position:    rec.Position,
			Explanation: rec.ExplanationCopy,
			Metadata:    rec.Metadata,
		}

		// Enrich with entity details (would query database)
		item.Entity = s.enrichEntity(ctx, rec.EntityType, rec.EntityID)

		items = append(items, item)
	}

	return &RecommendationAPIResponse{
		Success: true,
		Data: &RecommendationDataResponse{
			Recommendations: items,
			TotalCandidates: resp.TotalCandidates,
		},
		Meta: &MetaResponse{
			RequestID:        requestID,
			ProcessingTimeMs: resp.ProcessingTimeMs,
			AlgorithmVersion: resp.AlgorithmVersion,
			ExperimentID:     resp.ExperimentID.String(),
			Variant:          resp.Variant,
		},
	}
}

func (s *Server) enrichEntity(ctx context.Context, entityType recommendation.EntityType, entityID uuid.UUID) *EntityDetails {
	// Would query database to get full entity details
	// Simplified for now
	return &EntityDetails{
		ID: entityID.String(),
	}
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) errorResponse(w http.ResponseWriter, status int, code, message, details string) {
	resp := RecommendationAPIResponse{
		Success: false,
		Error: &ErrorResponse{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	s.jsonResponse(w, status, resp)
}
