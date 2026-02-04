// Package homerescue provides HTTP handlers for emergency home services
package homerescue

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/homerescue"
)

// Handler handles HomeRescue HTTP requests
type Handler struct {
	service *homerescue.Service
	logger  *zap.Logger
}

// NewHandler creates a new HomeRescue handler
func NewHandler(service *homerescue.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers emergency routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	emergency := router.Group("/homerescue")
	{
		// Emergency creation and management
		emergency.POST("/emergencies", h.CreateEmergency)
		emergency.GET("/emergencies/:id", h.GetEmergency)
		emergency.GET("/emergencies/:id/status", h.GetEmergencyStatus)
		emergency.GET("/emergencies/:id/tracking", h.GetTracking)
		emergency.GET("/emergencies/:id/sla", h.GetSLAMetrics)

		// Technician actions (in production, requires auth)
		emergency.POST("/technicians/location", h.UpdateTechLocation)
		emergency.PUT("/emergencies/:id/accept", h.AcceptEmergency)
		emergency.PUT("/emergencies/:id/complete", h.CompleteEmergency)

		// Technician availability management
		emergency.PUT("/technicians/:id/availability", h.UpdateTechAvailability)
	}
}

// CreateEmergency handles POST /homerescue/emergencies
func (h *Handler) CreateEmergency(c *gin.Context) {
	var req struct {
		UserID             string  `json:"user_id" binding:"required"`
		Category           string  `json:"category" binding:"required"`
		Subcategory        string  `json:"subcategory"`
		Urgency            string  `json:"urgency" binding:"required"`
		Title              string  `json:"title" binding:"required"`
		Description        string  `json:"description" binding:"required"`
		Address            string  `json:"address" binding:"required"`
		Unit               string  `json:"unit"`
		City               string  `json:"city" binding:"required"`
		State              string  `json:"state" binding:"required"`
		PostalCode         string  `json:"postal_code" binding:"required"`
		Latitude           float64 `json:"latitude" binding:"required"`
		Longitude          float64 `json:"longitude" binding:"required"`
		AccessInstructions string  `json:"access_instructions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Validate category
	validCategories := map[string]bool{
		"plumbing": true, "electrical": true, "locksmith": true, "hvac": true,
		"glass": true, "roofing": true, "pest": true, "security": true, "general": true,
	}
	if !validCategories[req.Category] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
		return
	}

	// Validate urgency
	validUrgencies := map[string]bool{
		"critical": true, "urgent": true, "same_day": true, "scheduled": true,
	}
	if !validUrgencies[req.Urgency] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid urgency level"})
		return
	}

	// Create emergency request
	createReq := &homerescue.CreateEmergencyRequest{
		UserID:             userID,
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
	}

	emergency, err := h.service.CreateEmergency(c.Request.Context(), createReq)
	if err != nil {
		h.logger.Error("Failed to create emergency", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create emergency"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"emergency": emergency,
		"message":   "Emergency created. Searching for available technicians...",
	})
}

// GetEmergency handles GET /homerescue/emergencies/:id
func (h *Handler) GetEmergency(c *gin.Context) {
	emergencyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid emergency ID"})
		return
	}

	emergency, err := h.service.GetEmergency(c.Request.Context(), emergencyID)
	if err != nil {
		if err == homerescue.ErrEmergencyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Emergency not found"})
			return
		}
		h.logger.Error("Failed to get emergency", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve emergency"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"emergency": emergency})
}

// GetEmergencyStatus handles GET /homerescue/emergencies/:id/status
func (h *Handler) GetEmergencyStatus(c *gin.Context) {
	emergencyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid emergency ID"})
		return
	}

	status, err := h.service.GetEmergencyStatus(c.Request.Context(), emergencyID)
	if err != nil {
		if err == homerescue.ErrEmergencyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Emergency not found"})
			return
		}
		h.logger.Error("Failed to get emergency status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}

// GetTracking handles GET /homerescue/emergencies/:id/tracking
func (h *Handler) GetTracking(c *gin.Context) {
	emergencyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid emergency ID"})
		return
	}

	tracking, err := h.service.GetEmergencyTracking(c.Request.Context(), emergencyID)
	if err != nil {
		if err == homerescue.ErrEmergencyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Emergency not found"})
			return
		}
		h.logger.Error("Failed to get tracking info", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tracking"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tracking": tracking})
}

// GetSLAMetrics handles GET /homerescue/emergencies/:id/sla
func (h *Handler) GetSLAMetrics(c *gin.Context) {
	emergencyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid emergency ID"})
		return
	}

	metrics, err := h.service.GetSLAMetrics(c.Request.Context(), emergencyID)
	if err != nil {
		if err == homerescue.ErrEmergencyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Emergency not found"})
			return
		}
		h.logger.Error("Failed to get SLA metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve SLA metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sla_metrics": metrics})
}

// UpdateTechLocation handles POST /homerescue/technicians/location
func (h *Handler) UpdateTechLocation(c *gin.Context) {
	var req struct {
		EmergencyID string  `json:"emergency_id" binding:"required"`
		Latitude    float64 `json:"latitude" binding:"required"`
		Longitude   float64 `json:"longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	emergencyID, err := uuid.Parse(req.EmergencyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid emergency ID"})
		return
	}

	err = h.service.UpdateTechnicianLocation(c.Request.Context(), emergencyID, req.Latitude, req.Longitude)
	if err != nil {
		if err == homerescue.ErrEmergencyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Emergency not found"})
			return
		}
		h.logger.Error("Failed to update tech location", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update location"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Location updated successfully"})
}

// AcceptEmergency handles PUT /homerescue/emergencies/:id/accept
func (h *Handler) AcceptEmergency(c *gin.Context) {
	emergencyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid emergency ID"})
		return
	}

	var req struct {
		TechnicianID     string `json:"technician_id" binding:"required"`
		EstimatedArrival string `json:"estimated_arrival" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	techID, err := uuid.Parse(req.TechnicianID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid technician ID"})
		return
	}

	estimatedArrival, err := time.Parse(time.RFC3339, req.EstimatedArrival)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid estimated arrival time format (use RFC3339)"})
		return
	}

	err = h.service.AcceptEmergency(c.Request.Context(), emergencyID, techID, estimatedArrival)
	if err != nil {
		h.logger.Error("Failed to accept emergency", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to accept emergency"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Emergency accepted",
		"estimated_arrival": estimatedArrival,
	})
}

// CompleteEmergency handles PUT /homerescue/emergencies/:id/complete
func (h *Handler) CompleteEmergency(c *gin.Context) {
	emergencyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid emergency ID"})
		return
	}

	var req struct {
		TechnicianID string  `json:"technician_id" binding:"required"`
		WorkNotes    string  `json:"work_notes" binding:"required"`
		FinalCost    float64 `json:"final_cost" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	techID, err := uuid.Parse(req.TechnicianID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid technician ID"})
		return
	}

	if req.FinalCost < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Final cost must be non-negative"})
		return
	}

	err = h.service.CompleteEmergency(c.Request.Context(), emergencyID, techID, req.WorkNotes, req.FinalCost)
	if err != nil {
		h.logger.Error("Failed to complete emergency", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete emergency"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Emergency completed successfully",
		"final_cost": req.FinalCost,
	})
}

// UpdateTechAvailability handles PUT /homerescue/technicians/:id/availability
func (h *Handler) UpdateTechAvailability(c *gin.Context) {
	techID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid technician ID"})
		return
	}

	var req struct {
		IsAvailable bool `json:"is_available"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	err = h.service.UpdateTechnicianAvailability(c.Request.Context(), techID, req.IsAvailable)
	if err != nil {
		h.logger.Error("Failed to update tech availability", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update availability"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Availability updated successfully",
		"is_available": req.IsAvailable,
	})
}
