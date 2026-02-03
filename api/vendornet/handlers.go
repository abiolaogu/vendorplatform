// Package vendornet provides HTTP handlers for VendorNet B2B partnership system
package vendornet

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/vendornet"
)

// Handler handles VendorNet HTTP requests
type Handler struct {
	service *vendornet.Service
	logger  *zap.Logger
}

// NewHandler creates a new VendorNet handler
func NewHandler(service *vendornet.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers VendorNet routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	vendornet := router.Group("/vendornet")
	{
		// Partner Matching
		vendornet.GET("/partners/matches", h.GetPartnerMatches)

		// Partnerships
		vendornet.POST("/partnerships", h.CreatePartnership)
		vendornet.GET("/partnerships/:id", h.GetPartnership)

		// Referrals
		vendornet.POST("/referrals", h.CreateReferral)
		vendornet.GET("/referrals/:id", h.GetReferral)
		vendornet.PUT("/referrals/:id/status", h.UpdateReferralStatus)

		// Analytics
		vendornet.GET("/analytics", h.GetNetworkAnalytics)
	}
}

// GetPartnerMatches handles GET /api/v1/vendornet/partners/matches
func (h *Handler) GetPartnerMatches(c *gin.Context) {
	// Get vendor ID from query or auth context
	vendorIDStr := c.Query("vendor_id")
	if vendorIDStr == "" {
		vendorIDStr = c.GetString("vendor_id")
	}

	if vendorIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor_id is required"})
		return
	}

	vendorID, err := uuid.Parse(vendorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vendor_id"})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := parseLimit(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	matches, err := h.service.GetPartnerMatches(c.Request.Context(), vendorID, limit)
	if err != nil {
		h.logger.Error("Failed to get partner matches", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find partner matches"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    matches,
		"count":   len(matches),
	})
}

// CreatePartnership handles POST /api/v1/vendornet/partnerships
func (h *Handler) CreatePartnership(c *gin.Context) {
	var req struct {
		VendorAID   string `json:"vendor_a_id" binding:"required"`
		VendorBID   string `json:"vendor_b_id" binding:"required"`
		Type        string `json:"type" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Terms       string `json:"terms"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vendorAID, err := uuid.Parse(req.VendorAID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vendor_a_id"})
		return
	}

	vendorBID, err := uuid.Parse(req.VendorBID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vendor_b_id"})
		return
	}

	partnership, err := h.service.CreatePartnership(c.Request.Context(), vendornet.CreatePartnershipRequest{
		VendorAID:   vendorAID,
		VendorBID:   vendorBID,
		Type:        req.Type,
		Name:        req.Name,
		Description: req.Description,
		Terms:       req.Terms,
	})

	if err != nil {
		h.logger.Error("Failed to create partnership", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create partnership"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    partnership,
	})
}

// GetPartnership handles GET /api/v1/vendornet/partnerships/:id
func (h *Handler) GetPartnership(c *gin.Context) {
	partnershipID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid partnership id"})
		return
	}

	partnership, err := h.service.GetPartnership(c.Request.Context(), partnershipID)
	if err != nil {
		if err == vendornet.ErrPartnershipNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "partnership not found"})
			return
		}
		h.logger.Error("Failed to get partnership", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get partnership"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    partnership,
	})
}

// CreateReferral handles POST /api/v1/vendornet/referrals
func (h *Handler) CreateReferral(c *gin.Context) {
	var req struct {
		SourceVendorID  string  `json:"source_vendor_id" binding:"required"`
		DestVendorID    string  `json:"dest_vendor_id" binding:"required"`
		ClientName      string  `json:"client_name" binding:"required"`
		ClientEmail     string  `json:"client_email" binding:"required"`
		ClientPhone     string  `json:"client_phone" binding:"required"`
		EventType       string  `json:"event_type" binding:"required"`
		ServiceCategory string  `json:"service_category_id" binding:"required"`
		EstimatedValue  float64 `json:"estimated_value"`
		Notes           string  `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sourceVendorID, err := uuid.Parse(req.SourceVendorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source_vendor_id"})
		return
	}

	destVendorID, err := uuid.Parse(req.DestVendorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dest_vendor_id"})
		return
	}

	serviceCategoryID, err := uuid.Parse(req.ServiceCategory)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service_category_id"})
		return
	}

	referral, err := h.service.CreateReferral(c.Request.Context(), vendornet.CreateReferralRequest{
		SourceVendorID:  sourceVendorID,
		DestVendorID:    destVendorID,
		ClientName:      req.ClientName,
		ClientEmail:     req.ClientEmail,
		ClientPhone:     req.ClientPhone,
		EventType:       req.EventType,
		ServiceCategory: serviceCategoryID,
		EstimatedValue:  req.EstimatedValue,
		Notes:           req.Notes,
	})

	if err != nil {
		h.logger.Error("Failed to create referral", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create referral"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    referral,
	})
}

// GetReferral handles GET /api/v1/vendornet/referrals/:id
func (h *Handler) GetReferral(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid referral id"})
		return
	}

	referral, err := h.service.GetReferral(c.Request.Context(), referralID)
	if err != nil {
		if err == vendornet.ErrReferralNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "referral not found"})
			return
		}
		h.logger.Error("Failed to get referral", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get referral"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    referral,
	})
}

// UpdateReferralStatus handles PUT /api/v1/vendornet/referrals/:id/status
func (h *Handler) UpdateReferralStatus(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid referral id"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}

	err = h.service.UpdateReferralStatus(c.Request.Context(), referralID, req.Status)
	if err != nil {
		if err == vendornet.ErrReferralNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "referral not found"})
			return
		}
		h.logger.Error("Failed to update referral status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update referral status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "referral status updated successfully",
	})
}

// GetNetworkAnalytics handles GET /api/v1/vendornet/analytics
func (h *Handler) GetNetworkAnalytics(c *gin.Context) {
	// Get vendor ID from query or auth context
	vendorIDStr := c.Query("vendor_id")
	if vendorIDStr == "" {
		vendorIDStr = c.GetString("vendor_id")
	}

	if vendorIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor_id is required"})
		return
	}

	vendorID, err := uuid.Parse(vendorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vendor_id"})
		return
	}

	analytics, err := h.service.GetNetworkAnalytics(c.Request.Context(), vendorID)
	if err != nil {
		h.logger.Error("Failed to get network analytics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    analytics,
	})
}

// Helper function to parse limit parameter
func parseLimit(limitStr string) (int, error) {
	var limit int
	_, err := fmt.Sscanf(limitStr, "%d", &limit)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("invalid limit")
	}
	return limit, nil
}
