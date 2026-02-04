// Package reviews provides HTTP handlers for review management
package reviews

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/auth"
	"github.com/BillyRonksGlobal/vendorplatform/internal/review"
)

// Handler handles review HTTP requests
type Handler struct {
	reviewService *review.Service
	logger        *zap.Logger
}

// NewHandler creates a new review handler
func NewHandler(reviewService *review.Service, logger *zap.Logger) *Handler {
	return &Handler{
		reviewService: reviewService,
		logger:        logger,
	}
}

// RegisterRoutes registers review routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	reviews := router.Group("/reviews")
	{
		reviews.POST("", h.CreateReview)
		reviews.GET("", h.ListReviews)
		reviews.GET("/:id", h.GetReview)
		reviews.PUT("/:id", h.UpdateReview)
		reviews.DELETE("/:id", h.DeleteReview)
		reviews.POST("/:id/response", h.AddVendorResponse)
		reviews.POST("/:id/vote", h.VoteHelpful)
	}

	// Vendor-specific review routes
	router.GET("/vendors/:vendor_id/reviews", h.GetVendorReviews)
}

// CreateReview handles POST /api/v1/reviews
func (h *Handler) CreateReview(c *gin.Context) {
	var req review.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind create review request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get user_id from authenticated session
	userID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Override the user_id from request with authenticated user_id
	req.UserID = userID

	r, err := h.reviewService.Create(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create review", zap.Error(err))

		// Handle specific errors
		switch err {
		case review.ErrDuplicateReview:
			c.JSON(http.StatusConflict, gin.H{
				"error":   "duplicate_review",
				"message": "You have already reviewed this booking",
			})
		case review.ErrBookingNotCompleted:
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "booking_not_completed",
				"message": "Booking must be completed before reviewing",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "creation_failed",
				"message": "Failed to create review",
				"details": err.Error(),
			})
		}
		return
	}

	h.logger.Info("Review created", zap.String("review_id", r.ID.String()))
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    r,
	})
}

// GetReview handles GET /api/v1/reviews/:id
func (h *Handler) GetReview(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid review ID",
		})
		return
	}

	r, err := h.reviewService.GetByID(c.Request.Context(), id)
	if err == review.ErrReviewNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Review not found",
		})
		return
	}

	if err != nil {
		h.logger.Error("Failed to get review", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "fetch_failed",
			"message": "Failed to retrieve review",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    r,
	})
}

// ListReviews handles GET /api/v1/reviews
func (h *Handler) ListReviews(c *gin.Context) {
	opts := &review.ReviewListOptions{
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

	if vendorID := c.Query("vendor_id"); vendorID != "" {
		if id, err := uuid.Parse(vendorID); err == nil {
			opts.VendorID = &id
		}
	}

	if userID := c.Query("user_id"); userID != "" {
		if id, err := uuid.Parse(userID); err == nil {
			opts.UserID = &id
		}
	}

	if minRatingStr := c.Query("min_rating"); minRatingStr != "" {
		if minRating, err := strconv.Atoi(minRatingStr); err == nil {
			opts.MinRating = &minRating
		}
	}

	if verifiedStr := c.Query("verified"); verifiedStr != "" {
		verified := verifiedStr == "true"
		opts.IsVerified = &verified
	}

	if withResponseStr := c.Query("with_response"); withResponseStr != "" {
		withResponse := withResponseStr == "true"
		opts.WithResponse = &withResponse
	}

	if sortBy := c.Query("sort_by"); sortBy != "" {
		opts.SortBy = sortBy
	}

	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		opts.SortOrder = sortOrder
	}

	reviews, total, err := h.reviewService.List(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("Failed to list reviews", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "fetch_failed",
			"message": "Failed to retrieve reviews",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    reviews,
		"meta": gin.H{
			"total":  total,
			"limit":  opts.Limit,
			"offset": opts.Offset,
		},
	})
}

// GetVendorReviews handles GET /api/v1/vendors/:vendor_id/reviews
func (h *Handler) GetVendorReviews(c *gin.Context) {
	vendorIDParam := c.Param("vendor_id")
	vendorID, err := uuid.Parse(vendorIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid vendor ID",
		})
		return
	}

	opts := &review.ReviewListOptions{
		VendorID: &vendorID,
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

	if minRatingStr := c.Query("min_rating"); minRatingStr != "" {
		if minRating, err := strconv.Atoi(minRatingStr); err == nil {
			opts.MinRating = &minRating
		}
	}

	if sortBy := c.Query("sort_by"); sortBy != "" {
		opts.SortBy = sortBy
	}

	reviews, total, err := h.reviewService.List(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("Failed to list vendor reviews", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "fetch_failed",
			"message": "Failed to retrieve reviews",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    reviews,
		"meta": gin.H{
			"vendor_id": vendorID.String(),
			"total":     total,
			"limit":     opts.Limit,
			"offset":    opts.Offset,
		},
	})
}

// UpdateReview handles PUT /api/v1/reviews/:id
func (h *Handler) UpdateReview(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid review ID",
		})
		return
	}

	var req review.UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get user_id from authenticated session
	userID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	r, err := h.reviewService.Update(c.Request.Context(), id, userID, &req)
	if err == review.ErrReviewNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Review not found",
		})
		return
	}

	if err == review.ErrUnauthorized {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "You can only edit your own reviews",
		})
		return
	}

	if err != nil {
		h.logger.Error("Failed to update review", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "update_failed",
			"message": "Failed to update review",
		})
		return
	}

	h.logger.Info("Review updated", zap.String("review_id", id.String()))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    r,
	})
}

// DeleteReview handles DELETE /api/v1/reviews/:id
func (h *Handler) DeleteReview(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid review ID",
		})
		return
	}

	// Get user_id from authenticated session
	userID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	err = h.reviewService.Delete(c.Request.Context(), id, userID)
	if err == review.ErrReviewNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Review not found",
		})
		return
	}

	if err == review.ErrUnauthorized {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "You can only delete your own reviews",
		})
		return
	}

	if err != nil {
		h.logger.Error("Failed to delete review", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "deletion_failed",
			"message": "Failed to delete review",
		})
		return
	}

	h.logger.Info("Review deleted", zap.String("review_id", id.String()))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Review deleted successfully",
	})
}

// AddVendorResponse handles POST /api/v1/reviews/:id/response
func (h *Handler) AddVendorResponse(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid review ID",
		})
		return
	}

	var req struct {
		Response string `json:"response" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Response text is required",
		})
		return
	}

	// Get vendor user_id from authenticated session
	vendorUserID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	err = h.reviewService.AddVendorResponse(c.Request.Context(), id, vendorUserID, req.Response)
	if err == review.ErrReviewNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Review not found",
		})
		return
	}

	if err == review.ErrUnauthorized {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "You can only respond to reviews for your vendor",
		})
		return
	}

	if err != nil {
		h.logger.Error("Failed to add vendor response", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "response_failed",
			"message": "Failed to add vendor response",
		})
		return
	}

	h.logger.Info("Vendor response added", zap.String("review_id", id.String()))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Response added successfully",
	})
}

// VoteHelpful handles POST /api/v1/reviews/:id/vote
func (h *Handler) VoteHelpful(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid review ID",
		})
		return
	}

	var req struct {
		IsHelpful bool `json:"is_helpful"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
		return
	}

	// Get user_id from authenticated session
	userID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	err = h.reviewService.VoteHelpful(c.Request.Context(), id, userID, req.IsHelpful)
	if err != nil {
		h.logger.Error("Failed to record vote", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "vote_failed",
			"message": "Failed to record vote",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Vote recorded successfully",
	})
}
