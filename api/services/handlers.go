// Package services provides HTTP handlers for service management
package services

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/service"
)

// Handler handles service HTTP requests
type Handler struct {
	serviceService *service.ServiceService
	logger         *zap.Logger
}

// NewHandler creates a new service handler
func NewHandler(serviceService *service.ServiceService, logger *zap.Logger) *Handler {
	return &Handler{
		serviceService: serviceService,
		logger:         logger,
	}
}

// RegisterRoutes registers service routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	services := router.Group("/services")
	{
		services.POST("", h.CreateService)
		services.GET("", h.ListServices)
		services.GET("/:id", h.GetService)
		services.PUT("/:id", h.UpdateService)
		services.DELETE("/:id", h.DeleteService)
	}
}

// CreateService creates a new service
func (h *Handler) CreateService(c *gin.Context) {
	var req service.CreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind create service request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// TODO: Extract vendor ID from authenticated user context
	// For now, expect it in the request body
	if req.VendorID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vendor_id is required"})
		return
	}

	createdService, err := h.serviceService.Create(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create service", zap.Error(err))

		if err == service.ErrServiceAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "Service with this name already exists for this vendor"})
			return
		}

		if err == service.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service"})
		return
	}

	h.logger.Info("Service created successfully",
		zap.String("service_id", createdService.ID.String()),
		zap.String("vendor_id", createdService.VendorID.String()),
		zap.String("name", createdService.Name),
	)

	c.JSON(http.StatusCreated, gin.H{
		"service": createdService,
	})
}

// GetService retrieves a service by ID
func (h *Handler) GetService(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	svc, err := h.serviceService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrServiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
			return
		}

		h.logger.Error("Failed to get service", zap.Error(err), zap.String("service_id", id.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"service": svc,
	})
}

// ListServices retrieves services with filters
func (h *Handler) ListServices(c *gin.Context) {
	filter := &service.ServiceFilter{
		Limit:  50,
		Offset: 0,
	}

	// Parse query parameters
	if vendorID := c.Query("vendor_id"); vendorID != "" {
		if id, err := uuid.Parse(vendorID); err == nil {
			filter.VendorID = &id
		}
	}

	if categoryID := c.Query("category_id"); categoryID != "" {
		if id, err := uuid.Parse(categoryID); err == nil {
			filter.CategoryID = &id
		}
	}

	if pricingModel := c.Query("pricing_model"); pricingModel != "" {
		filter.PricingModel = &pricingModel
	}

	if minPrice := c.Query("min_price"); minPrice != "" {
		var price float64
		if _, err := fmt.Sscanf(minPrice, "%f", &price); err == nil {
			filter.MinPrice = &price
		}
	}

	if maxPrice := c.Query("max_price"); maxPrice != "" {
		var price float64
		if _, err := fmt.Sscanf(maxPrice, "%f", &price); err == nil {
			filter.MaxPrice = &price
		}
	}

	if available := c.Query("is_available"); available != "" {
		isAvail := available == "true"
		filter.IsAvailable = &isAvail
	}

	if featured := c.Query("is_featured"); featured != "" {
		isFeat := featured == "true"
		filter.IsFeatured = &isFeat
	}

	if search := c.Query("search"); search != "" {
		filter.SearchQuery = &search
	}

	if limit := c.Query("limit"); limit != "" {
		var l int
		if _, err := fmt.Sscanf(limit, "%d", &l); err == nil && l > 0 {
			filter.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		var o int
		if _, err := fmt.Sscanf(offset, "%d", &o); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	services, err := h.serviceService.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list services", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve services"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"services": services,
		"count":    len(services),
	})
}

// UpdateService updates a service
func (h *Handler) UpdateService(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	var req service.UpdateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind update service request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// TODO: Extract vendor ID from authenticated user context
	// For now, expect it in the request header or body
	vendorIDStr := c.GetHeader("X-Vendor-ID")
	if vendorIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Vendor-ID header is required"})
		return
	}

	vendorID, err := uuid.Parse(vendorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor ID"})
		return
	}

	updatedService, err := h.serviceService.Update(c.Request.Context(), id, vendorID, &req)
	if err != nil {
		if err == service.ErrServiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
			return
		}

		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this service"})
			return
		}

		h.logger.Error("Failed to update service", zap.Error(err), zap.String("service_id", id.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service"})
		return
	}

	h.logger.Info("Service updated successfully", zap.String("service_id", id.String()))

	c.JSON(http.StatusOK, gin.H{
		"service": updatedService,
	})
}

// DeleteService deletes a service
func (h *Handler) DeleteService(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	// TODO: Extract vendor ID from authenticated user context
	vendorIDStr := c.GetHeader("X-Vendor-ID")
	if vendorIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Vendor-ID header is required"})
		return
	}

	vendorID, err := uuid.Parse(vendorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor ID"})
		return
	}

	err = h.serviceService.Delete(c.Request.Context(), id, vendorID)
	if err != nil {
		if err == service.ErrServiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
			return
		}

		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this service"})
			return
		}

		h.logger.Error("Failed to delete service", zap.Error(err), zap.String("service_id", id.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete service"})
		return
	}

	h.logger.Info("Service deleted successfully", zap.String("service_id", id.String()))

	c.JSON(http.StatusOK, gin.H{
		"message": "Service deleted successfully",
	})
}
