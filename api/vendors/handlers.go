// Package vendors provides HTTP handlers for vendor management
package vendors

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/service"
	"github.com/BillyRonksGlobal/vendorplatform/internal/vendor"
)

// Handler handles vendor HTTP requests
type Handler struct {
	vendorService  *vendor.Service
	serviceManager *service.ServiceManager
	logger         *zap.Logger
}

// NewHandler creates a new vendor handler
func NewHandler(vendorService *vendor.Service, serviceManager *service.ServiceManager, logger *zap.Logger) *Handler {
	return &Handler{
		vendorService:  vendorService,
		serviceManager: serviceManager,
		logger:         logger,
	}
}

// RegisterRoutes registers vendor routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	vendors := router.Group("/vendors")
	{
		vendors.POST("", h.CreateVendor)
		vendors.GET("", h.ListVendors)
		vendors.GET("/:id", h.GetVendor)
		vendors.PUT("/:id", h.UpdateVendor)
		vendors.DELETE("/:id", h.DeleteVendor)
		vendors.POST("/:id/verify", h.VerifyVendor)
		vendors.GET("/:id/services", h.GetVendorServices)
	}
}

// CreateVendor handles POST /api/v1/vendors
func (h *Handler) CreateVendor(c *gin.Context) {
	var req vendor.CreateVendorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind create vendor request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// TODO: Get user_id from authenticated session
	// For now, expect it in the request
	if req.UserID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "user_id is required",
		})
		return
	}

	v, err := h.vendorService.Create(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create vendor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "creation_failed",
			"message": "Failed to create vendor",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Vendor created", zap.String("vendor_id", v.ID.String()))
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    v,
	})
}

// GetVendor handles GET /api/v1/vendors/:id
func (h *Handler) GetVendor(c *gin.Context) {
	idParam := c.Param("id")

	// Try parsing as UUID first
	id, err := uuid.Parse(idParam)
	var v *vendor.Vendor

	if err == nil {
		// Valid UUID, get by ID
		v, err = h.vendorService.GetByID(c.Request.Context(), id)
	} else {
		// Not a UUID, try as slug
		v, err = h.vendorService.GetBySlug(c.Request.Context(), idParam)
	}

	if err == vendor.ErrVendorNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Vendor not found",
		})
		return
	}

	if err != nil {
		h.logger.Error("Failed to get vendor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "fetch_failed",
			"message": "Failed to retrieve vendor",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    v,
	})
}

// ListVendors handles GET /api/v1/vendors
func (h *Handler) ListVendors(c *gin.Context) {
	opts := &vendor.VendorListOptions{
		Limit:  20,
		Offset: 0,
	}

	// Parse query parameters
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			opts.Offset = offset
		}
	}

	if categoryID := c.Query("category_id"); categoryID != "" {
		if id, err := uuid.Parse(categoryID); err == nil {
			opts.CategoryID = &id
		}
	}

	if city := c.Query("city"); city != "" {
		opts.City = &city
	}

	if state := c.Query("state"); state != "" {
		opts.State = &state
	}

	if status := c.Query("status"); status != "" {
		opts.Status = &status
	}

	if verifiedStr := c.Query("verified"); verifiedStr != "" {
		verified := verifiedStr == "true"
		opts.IsVerified = &verified
	}

	if minRatingStr := c.Query("min_rating"); minRatingStr != "" {
		if minRating, err := strconv.ParseFloat(minRatingStr, 64); err == nil {
			opts.MinRating = &minRating
		}
	}

	if searchQuery := c.Query("q"); searchQuery != "" {
		opts.SearchQuery = &searchQuery
	}

	if sortBy := c.Query("sort_by"); sortBy != "" {
		opts.SortBy = sortBy
	}

	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		opts.SortOrder = sortOrder
	}

	vendors, total, err := h.vendorService.List(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("Failed to list vendors", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "fetch_failed",
			"message": "Failed to retrieve vendors",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    vendors,
		"meta": gin.H{
			"total":  total,
			"limit":  opts.Limit,
			"offset": opts.Offset,
		},
	})
}

// UpdateVendor handles PUT /api/v1/vendors/:id
func (h *Handler) UpdateVendor(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid vendor ID",
		})
		return
	}

	var req vendor.UpdateVendorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// TODO: Verify user owns this vendor or is admin

	v, err := h.vendorService.Update(c.Request.Context(), id, &req)
	if err == vendor.ErrVendorNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Vendor not found",
		})
		return
	}

	if err != nil {
		h.logger.Error("Failed to update vendor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "update_failed",
			"message": "Failed to update vendor",
		})
		return
	}

	h.logger.Info("Vendor updated", zap.String("vendor_id", id.String()))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    v,
	})
}

// DeleteVendor handles DELETE /api/v1/vendors/:id
func (h *Handler) DeleteVendor(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid vendor ID",
		})
		return
	}

	// TODO: Verify user owns this vendor or is admin

	err = h.vendorService.Delete(c.Request.Context(), id)
	if err == vendor.ErrVendorNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Vendor not found",
		})
		return
	}

	if err != nil {
		h.logger.Error("Failed to delete vendor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "deletion_failed",
			"message": "Failed to delete vendor",
		})
		return
	}

	h.logger.Info("Vendor deleted", zap.String("vendor_id", id.String()))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Vendor deleted successfully",
	})
}

// VerifyVendor handles POST /api/v1/vendors/:id/verify
func (h *Handler) VerifyVendor(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid vendor ID",
		})
		return
	}

	// TODO: Verify user is admin

	err = h.vendorService.Verify(c.Request.Context(), id)
	if err == vendor.ErrVendorNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Vendor not found",
		})
		return
	}

	if err != nil {
		h.logger.Error("Failed to verify vendor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "verification_failed",
			"message": "Failed to verify vendor",
		})
		return
	}

	h.logger.Info("Vendor verified", zap.String("vendor_id", id.String()))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Vendor verified successfully",
	})
}

// GetVendorServices handles GET /api/v1/vendors/:id/services
func (h *Handler) GetVendorServices(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid vendor ID",
		})
		return
	}

	// Build service list options
	opts := &service.ServiceListOptions{
		VendorID: &id,
		Limit:    20,
		Offset:   0,
	}

	// Parse query parameters
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			opts.Offset = offset
		}
	}

	if status := c.Query("status"); status != "" {
		opts.Status = &status
	}

	if availableStr := c.Query("available"); availableStr != "" {
		available := availableStr == "true"
		opts.IsAvailable = &available
	}

	if featuredStr := c.Query("featured"); featuredStr != "" {
		featured := featuredStr == "true"
		opts.IsFeatured = &featured
	}

	if categoryID := c.Query("category_id"); categoryID != "" {
		if catID, err := uuid.Parse(categoryID); err == nil {
			opts.CategoryID = &catID
		}
	}

	if sortBy := c.Query("sort_by"); sortBy != "" {
		opts.SortBy = sortBy
	}

	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		opts.SortOrder = sortOrder
	}

	// Get services
	services, total, err := h.serviceManager.GetByVendorID(c.Request.Context(), id, opts)
	if err != nil {
		h.logger.Error("Failed to get vendor services",
			zap.Error(err),
			zap.String("vendor_id", id.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "fetch_failed",
			"message": "Failed to retrieve vendor services",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    services,
		"meta": gin.H{
			"vendor_id": id.String(),
			"total":     total,
			"limit":     opts.Limit,
			"offset":    opts.Offset,
		},
	})
}
