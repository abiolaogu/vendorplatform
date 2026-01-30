// Package lifeos provides HTTP handlers for the LifeOS platform
package lifeos

import (
	"net/http"
	"time"

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

// CreateLifeEvent handles POST /api/v1/lifeos/events
func (h *Handler) CreateLifeEvent(c *gin.Context) {
	var req CreateEventRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Validate required fields
	if req.EventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event_type is required"})
		return
	}

	// Get user ID from context (assuming auth middleware sets it)
	userID, exists := c.Get("user_id")
	if !exists {
		// For now, use a demo user ID if not authenticated
		userID = uuid.New()
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		if userIDStr, ok := userID.(string); ok {
			parsedID, err := uuid.Parse(userIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
				return
			}
			userUUID = parsedID
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID type"})
			return
		}
	}

	// Convert DTO to service request
	serviceReq := &lifeos.CreateEventRequest{
		UserID:          userUUID,
		EventType:       EventType(req.EventType),
		EventSubtype:    req.EventSubtype,
		EventDate:       req.EventDate,
		DateFlexibility: DateFlexibility(req.DateFlexibility),
		Scale:           EventScale(req.Scale),
		GuestCount:      req.GuestCount,
	}

	if req.Location != nil {
		serviceReq.Location = &Location{
			Address:    req.Location.Address,
			City:       req.Location.City,
			State:      req.Location.State,
			Country:    req.Location.Country,
			PostalCode: req.Location.PostalCode,
			Latitude:   req.Location.Latitude,
			Longitude:  req.Location.Longitude,
		}
	}

	if req.Budget != nil {
		serviceReq.Budget = &Budget{
			TotalAmount: req.Budget.TotalAmount,
			Currency:    req.Budget.Currency,
			Flexibility: BudgetFlex(req.Budget.Flexibility),
		}
	}

	event, err := h.service.CreateLifeEvent(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Failed to create life event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create life event"})
		return
	}

	c.JSON(http.StatusCreated, event)
}

// GetLifeEvent handles GET /api/v1/lifeos/events/:id
func (h *Handler) GetLifeEvent(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	event, err := h.service.GetLifeEvent(c.Request.Context(), eventID)
	if err != nil {
		h.logger.Error("Failed to get life event", zap.Error(err), zap.String("event_id", eventIDStr))
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	c.JSON(http.StatusOK, event)
}

// GetEventPlan handles GET /api/v1/lifeos/events/:id/plan
func (h *Handler) GetEventPlan(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	plan, err := h.service.GetEventPlan(c.Request.Context(), eventID)
	if err != nil {
		h.logger.Error("Failed to get event plan", zap.Error(err), zap.String("event_id", eventIDStr))
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	c.JSON(http.StatusOK, plan)
}

// ConfirmDetectedEvent handles POST /api/v1/lifeos/events/:id/confirm
func (h *Handler) ConfirmDetectedEvent(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var req ConfirmEventRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err = h.service.ConfirmDetectedEvent(c.Request.Context(), eventID, req.Confirmed)
	if err != nil {
		h.logger.Error("Failed to confirm event", zap.Error(err), zap.String("event_id", eventIDStr))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Event confirmation updated",
		"confirmed": req.Confirmed,
	})
}

// GetDetectedEvents handles GET /api/v1/lifeos/detected
func (h *Handler) GetDetectedEvents(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		if userIDStr, ok := userID.(string); ok {
			parsedID, err := uuid.Parse(userIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
				return
			}
			userUUID = parsedID
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID type"})
			return
		}
	}

	events, err := h.service.GetDetectedEvents(c.Request.Context(), userUUID)
	if err != nil {
		h.logger.Error("Failed to get detected events", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get detected events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
	})
}

// DTOs (Data Transfer Objects)

type CreateEventRequestDTO struct {
	EventType       string          `json:"event_type" binding:"required"`
	EventSubtype    string          `json:"event_subtype,omitempty"`
	EventDate       *time.Time      `json:"event_date,omitempty"`
	DateFlexibility string          `json:"date_flexibility,omitempty"`
	Scale           string          `json:"scale,omitempty"`
	GuestCount      *int            `json:"guest_count,omitempty"`
	Location        *LocationDTO    `json:"location,omitempty"`
	Budget          *BudgetDTO      `json:"budget,omitempty"`
}

type LocationDTO struct {
	Address    string  `json:"address,omitempty"`
	City       string  `json:"city,omitempty"`
	State      string  `json:"state,omitempty"`
	Country    string  `json:"country,omitempty"`
	PostalCode string  `json:"postal_code,omitempty"`
	Latitude   float64 `json:"latitude,omitempty"`
	Longitude  float64 `json:"longitude,omitempty"`
}

type BudgetDTO struct {
	TotalAmount float64 `json:"total_amount"`
	Currency    string  `json:"currency"`
	Flexibility string  `json:"flexibility,omitempty"`
}

type ConfirmEventRequestDTO struct {
	Confirmed bool `json:"confirmed"`
}
