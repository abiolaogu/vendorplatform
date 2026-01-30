// Package bookings provides HTTP handlers for booking management
package bookings

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/booking"
)

// Handler handles booking HTTP requests
type Handler struct {
	bookingService *booking.Service
	logger         *zap.Logger
}

// NewHandler creates a new booking handler
func NewHandler(bookingService *booking.Service, logger *zap.Logger) *Handler {
	return &Handler{
		bookingService: bookingService,
		logger:         logger,
	}
}

// RegisterRoutes registers booking routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	bookings := router.Group("/bookings")
	{
		bookings.POST("", h.CreateBooking)
		bookings.GET("", h.ListBookings)
		bookings.GET("/:id", h.GetBooking)
		bookings.GET("/code/:code", h.GetBookingByCode)
		bookings.PUT("/:id", h.UpdateBooking)
		bookings.PUT("/:id/status", h.UpdateBookingStatus)
		bookings.PUT("/:id/cancel", h.CancelBooking)
		bookings.PUT("/:id/payment", h.UpdatePaymentStatus)
		bookings.POST("/:id/rating", h.AddRating)
	}
}

// CreateBooking handles POST /bookings
func (h *Handler) CreateBooking(c *gin.Context) {
	var req booking.CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	created, err := h.bookingService.Create(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetBooking handles GET /bookings/:id
func (h *Handler) GetBooking(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	booking, err := h.bookingService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == booking.ErrBookingNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}
		h.logger.Error("Failed to get booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, booking)
}

// GetBookingByCode handles GET /bookings/code/:code
func (h *Handler) GetBookingByCode(c *gin.Context) {
	code := c.Param("code")

	booking, err := h.bookingService.GetByCode(c.Request.Context(), code)
	if err != nil {
		if err == booking.ErrBookingNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}
		h.logger.Error("Failed to get booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, booking)
}

// ListBookings handles GET /bookings
func (h *Handler) ListBookings(c *gin.Context) {
	opts := &booking.BookingListOptions{
		Limit:     20,
		Offset:    0,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	// Parse query parameters
	if userID := c.Query("user_id"); userID != "" {
		if id, err := uuid.Parse(userID); err == nil {
			opts.UserID = &id
		}
	}
	if vendorID := c.Query("vendor_id"); vendorID != "" {
		if id, err := uuid.Parse(vendorID); err == nil {
			opts.VendorID = &id
		}
	}
	if serviceID := c.Query("service_id"); serviceID != "" {
		if id, err := uuid.Parse(serviceID); err == nil {
			opts.ServiceID = &id
		}
	}
	if projectID := c.Query("project_id"); projectID != "" {
		if id, err := uuid.Parse(projectID); err == nil {
			opts.ProjectID = &id
		}
	}
	if status := c.Query("status"); status != "" {
		bookingStatus := booking.BookingStatus(status)
		opts.Status = &bookingStatus
	}
	if paymentStatus := c.Query("payment_status"); paymentStatus != "" {
		opts.PaymentStatus = &paymentStatus
	}
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			opts.Limit = l
		}
	}
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			opts.Offset = o
		}
	}
	if sortBy := c.Query("sort_by"); sortBy != "" {
		opts.SortBy = sortBy
	}
	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		opts.SortOrder = sortOrder
	}

	bookings, total, err := h.bookingService.List(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("Failed to list bookings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bookings": bookings,
		"total":    total,
		"limit":    opts.Limit,
		"offset":   opts.Offset,
	})
}

// UpdateBooking handles PUT /bookings/:id
func (h *Handler) UpdateBooking(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var req booking.UpdateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updated, err := h.bookingService.Update(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.Error("Failed to update booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// UpdateBookingStatus handles PUT /bookings/:id/status
func (h *Handler) UpdateBookingStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err = h.bookingService.UpdateStatus(c.Request.Context(), id, booking.BookingStatus(req.Status))
	if err != nil {
		if err == booking.ErrInvalidStatus {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status transition"})
			return
		}
		h.logger.Error("Failed to update booking status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

// CancelBooking handles PUT /bookings/:id/cancel
func (h *Handler) CancelBooking(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err = h.bookingService.Cancel(c.Request.Context(), id, req.Reason)
	if err != nil {
		h.logger.Error("Failed to cancel booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Booking cancelled successfully"})
}

// UpdatePaymentStatus handles PUT /bookings/:id/payment
func (h *Handler) UpdatePaymentStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var req struct {
		Status         string  `json:"status" binding:"required"`
		TransactionRef *string `json:"transaction_ref"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err = h.bookingService.UpdatePaymentStatus(c.Request.Context(), id, req.Status, req.TransactionRef)
	if err != nil {
		h.logger.Error("Failed to update payment status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment status updated successfully"})
}

// AddRating handles POST /bookings/:id/rating
func (h *Handler) AddRating(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var req struct {
		Rating float64 `json:"rating" binding:"required,min=1,max=5"`
		Review string  `json:"review"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err = h.bookingService.AddRating(c.Request.Context(), id, req.Rating, req.Review)
	if err != nil {
		h.logger.Error("Failed to add rating", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rating added successfully"})
}
