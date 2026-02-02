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
