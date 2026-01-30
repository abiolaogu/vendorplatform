// Package homerescue provides HTTP handlers for emergency services
package homerescue

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/homerescue"
)

// Handler handles emergency HTTP requests
type Handler struct {
	service *homerescue.Service
	logger  *zap.Logger
}

// NewHandler creates a new emergency handler
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
		// Public emergency creation
		emergency.POST("/emergencies", h.CreateEmergency)

		// Emergency tracking and status
		emergency.GET("/emergencies/:id", h.GetEmergency)
		emergency.GET("/emergencies/:id/tracking", h.GetTracking)

		// Technician actions (requires auth)
		emergency.POST("/technicians/location", h.UpdateTechLocation)
		emergency.PUT("/emergencies/:id/accept", h.AcceptEmergency)
		emergency.PUT("/emergencies/:id/complete", h.CompleteEmergency)
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

	// Validate urgency level
	validUrgencies := map[string]bool{
		"critical":  true,
		"urgent":    true,
		"same_day":  true,
		"scheduled": true,
	}
	if !validUrgencies[req.Urgency] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid urgency level"})
		return
	}

	// Validate category
	validCategories := map[string]bool{
		"plumbing":   true,
		"electrical": true,
		"locksmith":  true,
		"hvac":       true,
		"glass":      true,
		"roofing":    true,
		"pest":       true,
		"security":   true,
		"general":    true,
	}
	if !validCategories[req.Category] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
		return
	}

	createReq := &homerescue.CreateEmergencyRequest{
		UserID:             userID,
		Category:           req.Category,
		Subcategory:        req.Subcategory,
		Urgency:            req.Urgency,
		Title:              req.Title,
		Description:        req.Description,
		Address:            req.Address,
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
		"message":   "Emergency created successfully. Finding nearby technicians...",
	})
}

// GetEmergency handles GET /homerescue/emergencies/:id
func (h *Handler) GetEmergency(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid emergency ID"})
		return
	}

	emergency, err := h.service.GetEmergency(c.Request.Context(), id)
	if err == homerescue.ErrEmergencyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Emergency not found"})
		return
	}
	if err != nil {
		h.logger.Error("Failed to get emergency", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get emergency"})
		return
	}

	c.JSON(http.StatusOK, emergency)
}

// GetTracking handles GET /homerescue/emergencies/:id/tracking
func (h *Handler) GetTracking(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid emergency ID"})
		return
	}

	tracking, err := h.service.GetTechnicianTracking(c.Request.Context(), id)
	if err == homerescue.ErrEmergencyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Emergency not found or no technician assigned yet"})
		return
	}
	if err != nil {
		h.logger.Error("Failed to get tracking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tracking"})
		return
	}

	c.JSON(http.StatusOK, tracking)
}

// UpdateTechLocation handles POST /homerescue/technicians/location
func (h *Handler) UpdateTechLocation(c *gin.Context) {
	var req struct {
		EmergencyID string  `json:"emergency_id" binding:"required"`
		Latitude    float64 `json:"latitude" binding:"required"`
		Longitude   float64 `json:"longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	emergencyID, err := uuid.Parse(req.EmergencyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid emergency ID"})
		return
	}

	err = h.service.UpdateTechnicianLocation(c.Request.Context(), emergencyID, req.Latitude, req.Longitude)
	if err == homerescue.ErrEmergencyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Emergency not found"})
		return
	}
	if err != nil {
		h.logger.Error("Failed to update location", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update location"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Location updated successfully"})
}

// AcceptEmergency handles PUT /homerescue/emergencies/:id/accept
func (h *Handler) AcceptEmergency(c *gin.Context) {
	idStr := c.Param("id")
	emergencyID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid emergency ID"})
		return
	}

	var req struct {
		TechnicianID string `json:"technician_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	techID, err := uuid.Parse(req.TechnicianID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid technician ID"})
		return
	}

	err = h.service.AcceptEmergency(c.Request.Context(), emergencyID, techID)
	if err == homerescue.ErrEmergencyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Emergency not found or already assigned"})
		return
	}
	if err != nil {
		h.logger.Error("Failed to accept emergency", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to accept emergency"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Emergency accepted successfully"})
}

// CompleteEmergency handles PUT /homerescue/emergencies/:id/complete
func (h *Handler) CompleteEmergency(c *gin.Context) {
	idStr := c.Param("id")
	emergencyID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid emergency ID"})
		return
	}

	var req struct {
		FinalCost float64 `json:"final_cost" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.FinalCost < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid final cost"})
		return
	}

	err = h.service.CompleteEmergency(c.Request.Context(), emergencyID, req.FinalCost)
	if err == homerescue.ErrEmergencyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Emergency not found or not in progress"})
		return
	}
	if err != nil {
		h.logger.Error("Failed to complete emergency", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete emergency"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Emergency completed successfully"})
}
