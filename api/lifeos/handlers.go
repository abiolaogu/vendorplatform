// Package lifeos provides HTTP handlers for life event orchestration
package lifeos

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/lifeos"
)

// Handler handles LifeOS HTTP requests
type Handler struct {
	service *lifeos.Service
	logger  *zap.Logger
}

// NewHandler creates a new LifeOS handler
func NewHandler(service *lifeos.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers LifeOS routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	lifeos := router.Group("/lifeos")
	{
		lifeos.POST("/events", h.CreateLifeEvent)
		lifeos.GET("/events/:id", h.GetLifeEvent)
		lifeos.GET("/events/:id/plan", h.GetEventPlan)
		lifeos.POST("/events/:id/confirm", h.ConfirmDetectedEvent)
		lifeos.GET("/detected", h.GetDetectedEvents)

		// New endpoints for Phase 3 features
		lifeos.POST("/detect", h.DetectLifeEvents)
		lifeos.GET("/events/:id/bundles", h.GetBundleRecommendations)
		lifeos.GET("/events/:id/risks", h.AssessEventRisks)
		lifeos.POST("/events/:id/optimize", h.OptimizeBudgetAllocation)
	}
}

// CreateLifeEvent handles POST /api/v1/lifeos/events
func (h *Handler) CreateLifeEvent(c *gin.Context) {
	var req lifeos.CreateLifeEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Validate user_id
	if req.UserID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id is required",
		})
		return
	}

	// Validate event_type
	if req.EventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_type is required",
		})
		return
	}

	// Create the life event
	event, err := h.service.CreateLifeEvent(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create life event",
			zap.Error(err),
			zap.String("user_id", req.UserID.String()),
			zap.String("event_type", req.EventType),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create life event",
		})
		return
	}

	h.logger.Info("Life event created",
		zap.String("event_id", event.ID.String()),
		zap.String("user_id", event.UserID.String()),
		zap.String("event_type", event.EventType),
	)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    event,
	})
}

// GetLifeEvent handles GET /api/v1/lifeos/events/:id
func (h *Handler) GetLifeEvent(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
		})
		return
	}

	event, err := h.service.GetLifeEvent(c.Request.Context(), eventID)
	if err != nil {
		h.logger.Error("Failed to get life event",
			zap.Error(err),
			zap.String("event_id", eventIDStr),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Life event not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    event,
	})
}

// GetEventPlan handles GET /api/v1/lifeos/events/:id/plan
func (h *Handler) GetEventPlan(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
		})
		return
	}

	plan, err := h.service.GetEventPlan(c.Request.Context(), eventID)
	if err != nil {
		h.logger.Error("Failed to get event plan",
			zap.Error(err),
			zap.String("event_id", eventIDStr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate event plan",
		})
		return
	}

	h.logger.Info("Event plan generated",
		zap.String("event_id", eventIDStr),
		zap.Int("total_categories", plan.TotalCategories),
		zap.String("current_phase", plan.CurrentPhase),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    plan,
	})
}

// ConfirmDetectedEvent handles POST /api/v1/lifeos/events/:id/confirm
func (h *Handler) ConfirmDetectedEvent(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
		})
		return
	}

	// Get user_id from request body
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id is required",
		})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Confirm the event
	if err := h.service.ConfirmDetectedEvent(c.Request.Context(), eventID, userID); err != nil {
		h.logger.Error("Failed to confirm event",
			zap.Error(err),
			zap.String("event_id", eventIDStr),
			zap.String("user_id", req.UserID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to confirm event",
		})
		return
	}

	h.logger.Info("Event confirmed",
		zap.String("event_id", eventIDStr),
		zap.String("user_id", req.UserID),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Event confirmed successfully",
	})
}

// GetDetectedEvents handles GET /api/v1/lifeos/detected
func (h *Handler) GetDetectedEvents(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id query parameter is required",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	events, err := h.service.GetDetectedEvents(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get detected events",
			zap.Error(err),
			zap.String("user_id", userIDStr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch detected events",
		})
		return
	}

	h.logger.Info("Retrieved detected events",
		zap.String("user_id", userIDStr),
		zap.Int("count", len(events)),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    events,
		"count":   len(events),
	})
}

// DetectLifeEvents handles POST /api/v1/lifeos/detect
func (h *Handler) DetectLifeEvents(c *gin.Context) {
	var req struct {
		UserID       string `json:"user_id" binding:"required"`
		LookbackDays int    `json:"lookback_days"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id is required",
		})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	lookbackDays := req.LookbackDays
	if lookbackDays == 0 {
		lookbackDays = 30 // Default
	}

	result, err := h.service.DetectLifeEvents(c.Request.Context(), userID, lookbackDays)
	if err != nil {
		h.logger.Error("Failed to detect life events",
			zap.Error(err),
			zap.String("user_id", req.UserID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to detect life events",
		})
		return
	}

	h.logger.Info("Life events detected",
		zap.String("user_id", req.UserID),
		zap.Int("detected_count", len(result.DetectedEvents)),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetBundleRecommendations handles GET /api/v1/lifeos/events/:id/bundles
func (h *Handler) GetBundleRecommendations(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
		})
		return
	}

	bundles, err := h.service.GenerateBundleRecommendations(c.Request.Context(), eventID)
	if err != nil {
		h.logger.Error("Failed to generate bundle recommendations",
			zap.Error(err),
			zap.String("event_id", eventIDStr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate bundle recommendations",
		})
		return
	}

	h.logger.Info("Bundle recommendations generated",
		zap.String("event_id", eventIDStr),
		zap.Int("bundle_count", len(bundles)),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    bundles,
		"count":   len(bundles),
	})
}

// AssessEventRisks handles GET /api/v1/lifeos/events/:id/risks
func (h *Handler) AssessEventRisks(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
		})
		return
	}

	assessment, err := h.service.AssessEventRisks(c.Request.Context(), eventID)
	if err != nil {
		h.logger.Error("Failed to assess event risks",
			zap.Error(err),
			zap.String("event_id", eventIDStr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to assess event risks",
		})
		return
	}

	h.logger.Info("Event risk assessment completed",
		zap.String("event_id", eventIDStr),
		zap.String("overall_risk", assessment.OverallRisk),
		zap.Float64("risk_score", assessment.RiskScore),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    assessment,
	})
}

// OptimizeBudgetAllocation handles POST /api/v1/lifeos/events/:id/optimize
func (h *Handler) OptimizeBudgetAllocation(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
		})
		return
	}

	var req struct {
		TotalBudget float64 `json:"total_budget" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "total_budget is required",
		})
		return
	}

	if req.TotalBudget <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "total_budget must be positive",
		})
		return
	}

	optimization, err := h.service.OptimizeBudgetAllocation(c.Request.Context(), eventID, req.TotalBudget)
	if err != nil {
		h.logger.Error("Failed to optimize budget allocation",
			zap.Error(err),
			zap.String("event_id", eventIDStr),
			zap.Float64("total_budget", req.TotalBudget),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to optimize budget allocation",
		})
		return
	}

	h.logger.Info("Budget optimization completed",
		zap.String("event_id", eventIDStr),
		zap.Float64("total_budget", req.TotalBudget),
		zap.Float64("potential_savings", optimization.TotalPotentialSavings),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    optimization,
	})
}
