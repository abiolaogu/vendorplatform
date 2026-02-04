// Package vendornet provides HTTP handlers for VendorNet B2B partnership network
package vendornet

import (
	"net/http"
	"strconv"

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
		// Partnership routes
		vendornet.GET("/partners/matches", h.GetPartnerMatches)
		vendornet.POST("/partnerships", h.CreatePartnership)
		vendornet.GET("/partnerships/:id", h.GetPartnership)

		// Referral routes
		vendornet.POST("/referrals", h.CreateReferral)
		vendornet.GET("/referrals/:id", h.GetReferral)
		vendornet.PUT("/referrals/:id/status", h.UpdateReferralStatus)

		// Analytics routes
		vendornet.GET("/analytics", h.GetNetworkAnalytics)
	}
}

// GetPartnerMatches handles GET /api/v1/vendornet/partners/matches
func (h *Handler) GetPartnerMatches(c *gin.Context) {
	vendorIDStr := c.Query("vendor_id")
	if vendorIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "vendor_id query parameter is required",
		})
		return
	}

	vendorID, err := uuid.Parse(vendorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid vendor_id format",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	matches, err := h.service.GetPartnerMatches(c.Request.Context(), vendorID, limit)
	if err != nil {
		h.logger.Error("Failed to get partner matches",
			zap.Error(err),
			zap.String("vendor_id", vendorID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "fetch_failed",
			"message": "Failed to fetch partner matches",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"matches": matches,
			"count":   len(matches),
		},
	})
}

// CreatePartnership handles POST /api/v1/vendornet/partnerships
func (h *Handler) CreatePartnership(c *gin.Context) {
	var req vendornet.CreatePartnershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind create partnership request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// TODO: Validate that InitiatedBy is authenticated user or one of the vendors
	// For now, trust the request

	partnership, err := h.service.CreatePartnership(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create partnership",
			zap.Error(err),
			zap.String("vendor_a_id", req.VendorAID.String()),
			zap.String("vendor_b_id", req.VendorBID.String()),
		)

		statusCode := http.StatusInternalServerError
		errorCode := "creation_failed"
		message := "Failed to create partnership"

		switch err {
		case vendornet.ErrPartnershipExists:
			statusCode = http.StatusConflict
			errorCode = "partnership_exists"
			message = "Partnership already exists between these vendors"
		case vendornet.ErrSelfPartnership:
			statusCode = http.StatusBadRequest
			errorCode = "invalid_partnership"
			message = "Cannot create partnership with self"
		case vendornet.ErrInvalidPartnershipData:
			statusCode = http.StatusBadRequest
			errorCode = "invalid_data"
			message = err.Error()
		}

		c.JSON(statusCode, gin.H{
			"error":   errorCode,
			"message": message,
		})
		return
	}

	h.logger.Info("Partnership created",
		zap.String("partnership_id", partnership.ID.String()),
		zap.String("type", partnership.PartnershipType),
	)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"partnership": partnership,
		},
	})
}

// GetPartnership handles GET /api/v1/vendornet/partnerships/:id
func (h *Handler) GetPartnership(c *gin.Context) {
	partnershipIDStr := c.Param("id")
	partnershipID, err := uuid.Parse(partnershipIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid partnership ID format",
		})
		return
	}

	partnership, err := h.service.GetPartnership(c.Request.Context(), partnershipID)
	if err != nil {
		if err == vendornet.ErrPartnershipNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Partnership not found",
			})
			return
		}

		h.logger.Error("Failed to get partnership",
			zap.Error(err),
			zap.String("partnership_id", partnershipID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "fetch_failed",
			"message": "Failed to fetch partnership",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"partnership": partnership,
		},
	})
}

// CreateReferral handles POST /api/v1/vendornet/referrals
func (h *Handler) CreateReferral(c *gin.Context) {
	var req vendornet.CreateReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind create referral request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	referral, err := h.service.CreateReferral(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create referral",
			zap.Error(err),
			zap.String("source_vendor_id", req.SourceVendorID.String()),
			zap.String("dest_vendor_id", req.DestVendorID.String()),
		)

		statusCode := http.StatusInternalServerError
		errorCode := "creation_failed"
		message := "Failed to create referral"

		if err == vendornet.ErrInvalidReferralData {
			statusCode = http.StatusBadRequest
			errorCode = "invalid_data"
			message = err.Error()
		}

		c.JSON(statusCode, gin.H{
			"error":   errorCode,
			"message": message,
		})
		return
	}

	h.logger.Info("Referral created",
		zap.String("referral_id", referral.ID.String()),
		zap.String("tracking_code", referral.TrackingCode),
	)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"referral": referral,
		},
	})
}

// GetReferral handles GET /api/v1/vendornet/referrals/:id
func (h *Handler) GetReferral(c *gin.Context) {
	referralIDStr := c.Param("id")
	referralID, err := uuid.Parse(referralIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid referral ID format",
		})
		return
	}

	referral, err := h.service.GetReferral(c.Request.Context(), referralID)
	if err != nil {
		if err == vendornet.ErrReferralNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Referral not found",
			})
			return
		}

		h.logger.Error("Failed to get referral",
			zap.Error(err),
			zap.String("referral_id", referralID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "fetch_failed",
			"message": "Failed to fetch referral",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"referral": referral,
		},
	})
}

// UpdateReferralStatus handles PUT /api/v1/vendornet/referrals/:id/status
func (h *Handler) UpdateReferralStatus(c *gin.Context) {
	referralIDStr := c.Param("id")
	referralID, err := uuid.Parse(referralIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid referral ID format",
		})
		return
	}

	var req vendornet.UpdateReferralStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind update referral status request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	referral, err := h.service.UpdateReferralStatus(c.Request.Context(), referralID, &req)
	if err != nil {
		h.logger.Error("Failed to update referral status",
			zap.Error(err),
			zap.String("referral_id", referralID.String()),
			zap.String("new_status", req.Status),
		)

		statusCode := http.StatusInternalServerError
		errorCode := "update_failed"
		message := "Failed to update referral status"

		switch err {
		case vendornet.ErrReferralNotFound:
			statusCode = http.StatusNotFound
			errorCode = "not_found"
			message = "Referral not found"
		case vendornet.ErrInvalidReferralData:
			statusCode = http.StatusBadRequest
			errorCode = "invalid_data"
			message = err.Error()
		}

		c.JSON(statusCode, gin.H{
			"error":   errorCode,
			"message": message,
		})
		return
	}

	h.logger.Info("Referral status updated",
		zap.String("referral_id", referral.ID.String()),
		zap.String("status", referral.Status),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"referral": referral,
		},
	})
}

// GetNetworkAnalytics handles GET /api/v1/vendornet/analytics
func (h *Handler) GetNetworkAnalytics(c *gin.Context) {
	vendorIDStr := c.Query("vendor_id")
	if vendorIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "vendor_id query parameter is required",
		})
		return
	}

	vendorID, err := uuid.Parse(vendorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid vendor_id format",
		})
		return
	}

	analytics, err := h.service.GetNetworkAnalytics(c.Request.Context(), vendorID)
	if err != nil {
		h.logger.Error("Failed to get network analytics",
			zap.Error(err),
			zap.String("vendor_id", vendorID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "fetch_failed",
			"message": "Failed to fetch network analytics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"analytics": analytics,
		},
	})
}
