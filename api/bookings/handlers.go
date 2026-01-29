// Package bookings provides HTTP handlers for booking management
package bookings

import (
	"fmt"
	"net/http"
	"time"

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
		bookings.PUT("/:id", h.UpdateBooking)
		bookings.DELETE("/:id", h.CancelBooking)
		bookings.POST("/:id/confirm", h.ConfirmBooking)
		bookings.POST("/:id/start", h.StartBooking)
		bookings.POST("/:id/complete", h.CompleteBooking)
		bookings.POST("/:id/review", h.AddReview)
	}
}

// CreateBookingRequest represents the request body for creating a booking
type CreateBookingRequest struct {
	ServiceID       string  `json:"service_id" binding:"required"`
	ProjectID       *string `json:"project_id,omitempty"`
	ScheduledDate   string  `json:"scheduled_date" binding:"required"`
	ScheduledStart  *string `json:"scheduled_start_time,omitempty"`
	ScheduledEnd    *string `json:"scheduled_end_time,omitempty"`
	DurationMinutes *int    `json:"duration_minutes,omitempty"`
	LocationType    string  `json:"service_location_type" binding:"required"`
	AddressID       *string `json:"service_address_id,omitempty"`
	Quantity        int     `json:"quantity" binding:"min=1"`
	GuestCount      *int    `json:"guest_count,omitempty"`
	CustomerNotes   string  `json:"customer_notes,omitempty"`
	SpecialRequests string  `json:"special_requests,omitempty"`
	SourceType      string  `json:"source_type,omitempty"`
}

// UpdateBookingRequest represents the request body for updating a booking
type UpdateBookingRequest struct {
	ScheduledDate   *string `json:"scheduled_date,omitempty"`
	ScheduledStart  *string `json:"scheduled_start_time,omitempty"`
	ScheduledEnd    *string `json:"scheduled_end_time,omitempty"`
	LocationType    *string `json:"service_location_type,omitempty"`
	AddressID       *string `json:"service_address_id,omitempty"`
	Quantity        *int    `json:"quantity,omitempty"`
	GuestCount      *int    `json:"guest_count,omitempty"`
	CustomerNotes   *string `json:"customer_notes,omitempty"`
	SpecialRequests *string `json:"special_requests,omitempty"`
}

// CancelBookingRequest represents the request body for cancelling a booking
type CancelBookingRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// AddReviewRequest represents the request body for adding a review
type AddReviewRequest struct {
	Rating float64 `json:"rating" binding:"required,min=1,max=5"`
	Review string  `json:"review" binding:"required"`
}

// CreateBooking handles POST /api/v1/bookings
func (h *Handler) CreateBooking(c *gin.Context) {
	var req CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (would normally come from auth middleware)
	userID := c.GetString("user_id")
	if userID == "" {
		// For now, use a header or query param
		userID = c.GetHeader("X-User-ID")
		if userID == "" {
			userID = c.Query("user_id")
		}
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id is required"})
			return
		}
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// Parse service ID
	serviceID, err := uuid.Parse(req.ServiceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service_id"})
		return
	}

	// Parse scheduled date
	scheduledDate, err := time.Parse("2006-01-02", req.ScheduledDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scheduled_date format (use YYYY-MM-DD)"})
		return
	}

	// Build service request
	serviceReq := &booking.CreateBookingRequest{
		UserID:          userUUID,
		ServiceID:       serviceID,
		ScheduledDate:   scheduledDate,
		DurationMinutes: req.DurationMinutes,
		LocationType:    req.LocationType,
		Quantity:        req.Quantity,
		GuestCount:      req.GuestCount,
		CustomerNotes:   req.CustomerNotes,
		SpecialRequests: req.SpecialRequests,
		SourceType:      req.SourceType,
	}

	// Parse optional fields
	if req.ProjectID != nil {
		projectID, err := uuid.Parse(*req.ProjectID)
		if err == nil {
			serviceReq.ProjectID = &projectID
		}
	}

	if req.AddressID != nil {
		addressID, err := uuid.Parse(*req.AddressID)
		if err == nil {
			serviceReq.AddressID = &addressID
		}
	}

	if req.ScheduledStart != nil {
		startTime, err := time.Parse("15:04", *req.ScheduledStart)
		if err == nil {
			serviceReq.ScheduledStart = &startTime
		}
	}

	if req.ScheduledEnd != nil {
		endTime, err := time.Parse("15:04", *req.ScheduledEnd)
		if err == nil {
			serviceReq.ScheduledEnd = &endTime
		}
	}

	// Create booking
	bookingResult, err := h.bookingService.CreateBooking(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Failed to create booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    bookingResult,
	})
}

// GetBooking handles GET /api/v1/bookings/:id
func (h *Handler) GetBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	bookingResult, err := h.bookingService.GetBooking(c.Request.Context(), id)
	if err != nil {
		if err == booking.ErrBookingNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
			return
		}
		h.logger.Error("Failed to get booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get booking"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    bookingResult,
	})
}

// ListBookings handles GET /api/v1/bookings
func (h *Handler) ListBookings(c *gin.Context) {
	filter := &booking.ListBookingsFilter{
		Limit:  20,
		Offset: 0,
	}

	// Parse query parameters
	if userID := c.Query("user_id"); userID != "" {
		id, err := uuid.Parse(userID)
		if err == nil {
			filter.UserID = &id
		}
	}

	if vendorID := c.Query("vendor_id"); vendorID != "" {
		id, err := uuid.Parse(vendorID)
		if err == nil {
			filter.VendorID = &id
		}
	}

	if serviceID := c.Query("service_id"); serviceID != "" {
		id, err := uuid.Parse(serviceID)
		if err == nil {
			filter.ServiceID = &id
		}
	}

	if projectID := c.Query("project_id"); projectID != "" {
		id, err := uuid.Parse(projectID)
		if err == nil {
			filter.ProjectID = &id
		}
	}

	if status := c.Query("status"); status != "" {
		filter.Status = []string{status}
	}

	if fromDate := c.Query("from_date"); fromDate != "" {
		date, err := time.Parse("2006-01-02", fromDate)
		if err == nil {
			filter.FromDate = &date
		}
	}

	if toDate := c.Query("to_date"); toDate != "" {
		date, err := time.Parse("2006-01-02", toDate)
		if err == nil {
			filter.ToDate = &date
		}
	}

	// Get limit and offset
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

	bookings, err := h.bookingService.ListBookings(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list bookings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list bookings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    bookings,
		"count":   len(bookings),
	})
}

// UpdateBooking handles PUT /api/v1/bookings/:id
func (h *Handler) UpdateBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	var req UpdateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build service request
	serviceReq := &booking.UpdateBookingRequest{
		LocationType:    req.LocationType,
		Quantity:        req.Quantity,
		GuestCount:      req.GuestCount,
		CustomerNotes:   req.CustomerNotes,
		SpecialRequests: req.SpecialRequests,
	}

	// Parse optional date/time fields
	if req.ScheduledDate != nil {
		date, err := time.Parse("2006-01-02", *req.ScheduledDate)
		if err == nil {
			serviceReq.ScheduledDate = &date
		}
	}

	if req.ScheduledStart != nil {
		startTime, err := time.Parse("15:04", *req.ScheduledStart)
		if err == nil {
			serviceReq.ScheduledStart = &startTime
		}
	}

	if req.ScheduledEnd != nil {
		endTime, err := time.Parse("15:04", *req.ScheduledEnd)
		if err == nil {
			serviceReq.ScheduledEnd = &endTime
		}
	}

	if req.AddressID != nil {
		addressID, err := uuid.Parse(*req.AddressID)
		if err == nil {
			serviceReq.AddressID = &addressID
		}
	}

	bookingResult, err := h.bookingService.UpdateBooking(c.Request.Context(), id, serviceReq)
	if err != nil {
		h.logger.Error("Failed to update booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    bookingResult,
	})
}

// CancelBooking handles DELETE /api/v1/bookings/:id
func (h *Handler) CancelBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	var req CancelBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.bookingService.CancelBooking(c.Request.Context(), id, req.Reason)
	if err != nil {
		h.logger.Error("Failed to cancel booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "booking cancelled successfully",
	})
}

// ConfirmBooking handles POST /api/v1/bookings/:id/confirm
func (h *Handler) ConfirmBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	err = h.bookingService.ConfirmBooking(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to confirm booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "booking confirmed successfully",
	})
}

// StartBooking handles POST /api/v1/bookings/:id/start
func (h *Handler) StartBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	err = h.bookingService.StartBooking(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to start booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "booking started successfully",
	})
}

// CompleteBooking handles POST /api/v1/bookings/:id/complete
func (h *Handler) CompleteBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	err = h.bookingService.CompleteBooking(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to complete booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "booking completed successfully",
	})
}

// AddReview handles POST /api/v1/bookings/:id/review
func (h *Handler) AddReview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	var req AddReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.bookingService.AddReview(c.Request.Context(), id, req.Rating, req.Review)
	if err != nil {
		h.logger.Error("Failed to add review", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "review added successfully",
	})
}
