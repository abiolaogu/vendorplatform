// Package vendornet provides HTTP handlers for B2B vendor partnership network
package vendornet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Handler handles VendorNet HTTP requests
type Handler struct {
	db               *pgxpool.Pool
	cache            *redis.Client
	matchingEngine   *PartnershipMatchingEngine
	referralEngine   *ReferralEngine
	analytics        *NetworkAnalytics
	logger           *zap.Logger
}

// NewHandler creates a new VendorNet handler
func NewHandler(db *pgxpool.Pool, cache *redis.Client, logger *zap.Logger) *Handler {
	// Initialize services
	adjacencyService := &AdjacencyService{db: db}
	matchingEngine := &PartnershipMatchingEngine{
		db:               db,
		cache:            cache,
		adjacencyService: adjacencyService,
	}

	notificationService := &NotificationService{}
	paymentService := &PaymentService{}

	referralEngine := &ReferralEngine{
		db:              db,
		cache:           cache,
		notificationSvc: notificationService,
		paymentSvc:      paymentService,
	}

	analytics := &NetworkAnalytics{
		db:    db,
		cache: cache,
	}

	return &Handler{
		db:             db,
		cache:          cache,
		matchingEngine: matchingEngine,
		referralEngine: referralEngine,
		analytics:      analytics,
		logger:         logger,
	}
}

// RegisterRoutes registers VendorNet routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	vendornet := router.Group("/vendornet")
	{
		vendornet.GET("/partners/matches", h.GetPartnerMatches)
		vendornet.POST("/partnerships", h.CreatePartnership)
		vendornet.GET("/partnerships/:id", h.GetPartnership)
		vendornet.POST("/referrals", h.CreateReferral)
		vendornet.PUT("/referrals/:id/status", h.UpdateReferralStatus)
		vendornet.GET("/analytics", h.GetNetworkAnalytics)
	}
}

// GetPartnerMatches handles GET /api/v1/vendornet/partners/matches
// Returns potential partnership matches for a vendor
func (h *Handler) GetPartnerMatches(c *gin.Context) {
	// Get vendor ID from query or authenticated user
	vendorIDStr := c.Query("vendor_id")
	if vendorIDStr == "" {
		// In production, this would come from auth middleware
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "vendor_id parameter is required",
		})
		return
	}

	vendorID, err := uuid.Parse(vendorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid vendor_id format",
		})
		return
	}

	// Parse limit parameter
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := parseIntParam(limitStr, 1, 50); err == nil {
			limit = l
		}
	}

	// Find partner matches
	matches, err := h.matchingEngine.FindPartnerMatches(c.Request.Context(), vendorID, limit)
	if err != nil {
		h.logger.Error("Failed to find partner matches",
			zap.Error(err),
			zap.String("vendor_id", vendorID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to find partner matches",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vendor_id":      vendorID.String(),
		"matches":        matches,
		"total_matches":  len(matches),
		"generated_at":   time.Now(),
	})
}

// CreatePartnership handles POST /api/v1/vendornet/partnerships
// Creates a new partnership between two vendors
func (h *Handler) CreatePartnership(c *gin.Context) {
	var req struct {
		VendorAID       string            `json:"vendor_a_id" binding:"required"`
		VendorBID       string            `json:"vendor_b_id" binding:"required"`
		PartnershipType string            `json:"partnership_type" binding:"required"`
		Name            string            `json:"name" binding:"required"`
		Description     string            `json:"description"`
		Terms           PartnershipTerms  `json:"terms" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// Parse vendor IDs
	vendorAID, err := uuid.Parse(req.VendorAID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid vendor_a_id format",
		})
		return
	}

	vendorBID, err := uuid.Parse(req.VendorBID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid vendor_b_id format",
		})
		return
	}

	// Validate vendors are different
	if vendorAID == vendorBID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot create partnership with yourself",
		})
		return
	}

	// Validate partnership type
	partnershipType := PartnershipType(req.PartnershipType)
	validTypes := []PartnershipType{
		PartnershipReferral,
		PartnershipPreferred,
		PartnershipExclusive,
		PartnershipJointVenture,
		PartnershipWhiteLabel,
	}
	isValid := false
	for _, t := range validTypes {
		if partnershipType == t {
			isValid = true
			break
		}
	}
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid partnership_type. Must be one of: referral, preferred, exclusive, joint_venture, white_label",
		})
		return
	}

	// Check if partnership already exists
	existingPartnership, _ := h.getActivePartnership(c.Request.Context(), vendorAID, vendorBID)
	if existingPartnership != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Active partnership already exists between these vendors",
			"partnership_id": existingPartnership.ID.String(),
		})
		return
	}

	// Create partnership
	now := time.Now()
	partnership := &Partnership{
		ID:              uuid.New(),
		VendorAID:       vendorAID,
		VendorBID:       vendorBID,
		PartnershipType: partnershipType,
		Name:            req.Name,
		Description:     req.Description,
		Terms:           req.Terms,
		Status:          PartnershipProposed,
		ProposedAt:      now,
		ProposedBy:      vendorAID, // In production, get from auth context
		SignedByA:       false,
		SignedByB:       false,
	}

	// Calculate expiry if duration is set
	if req.Terms.DurationMonths > 0 {
		expiresAt := now.AddDate(0, req.Terms.DurationMonths, 0)
		partnership.ExpiresAt = &expiresAt
	}

	// Save to database
	if err := h.savePartnership(c.Request.Context(), partnership); err != nil {
		h.logger.Error("Failed to create partnership",
			zap.Error(err),
			zap.String("vendor_a", vendorAID.String()),
			zap.String("vendor_b", vendorBID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create partnership",
		})
		return
	}

	h.logger.Info("Partnership created",
		zap.String("partnership_id", partnership.ID.String()),
		zap.String("type", string(partnershipType)),
	)

	c.JSON(http.StatusCreated, gin.H{
		"partnership": partnership,
		"message": "Partnership proposal created. Awaiting acceptance from partner.",
	})
}

// GetPartnership handles GET /api/v1/vendornet/partnerships/:id
// Returns partnership details
func (h *Handler) GetPartnership(c *gin.Context) {
	partnershipIDStr := c.Param("id")
	partnershipID, err := uuid.Parse(partnershipIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid partnership ID format",
		})
		return
	}

	// Get partnership from database
	partnership, err := h.getPartnership(c.Request.Context(), partnershipID)
	if err != nil {
		h.logger.Error("Failed to get partnership",
			zap.Error(err),
			zap.String("partnership_id", partnershipID.String()),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Partnership not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"partnership": partnership,
	})
}

// CreateReferral handles POST /api/v1/vendornet/referrals
// Creates a new referral from one vendor to another
func (h *Handler) CreateReferral(c *gin.Context) {
	var req struct {
		SourceVendorID  string     `json:"source_vendor_id" binding:"required"`
		DestVendorID    string     `json:"dest_vendor_id" binding:"required"`
		ClientName      string     `json:"client_name" binding:"required"`
		ClientEmail     string     `json:"client_email" binding:"required"`
		ClientPhone     string     `json:"client_phone" binding:"required"`
		EventType       string     `json:"event_type" binding:"required"`
		EventDate       *time.Time `json:"event_date,omitempty"`
		ServiceCategory string     `json:"service_category_id" binding:"required"`
		EstimatedValue  float64    `json:"estimated_value" binding:"required"`
		Notes           string     `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// Parse vendor IDs
	sourceVendorID, err := uuid.Parse(req.SourceVendorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid source_vendor_id format",
		})
		return
	}

	destVendorID, err := uuid.Parse(req.DestVendorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid dest_vendor_id format",
		})
		return
	}

	serviceCategoryID, err := uuid.Parse(req.ServiceCategory)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid service_category_id format",
		})
		return
	}

	// Validate vendors are different
	if sourceVendorID == destVendorID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot refer to yourself",
		})
		return
	}

	// Validate estimated value
	if req.EstimatedValue <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Estimated value must be greater than 0",
		})
		return
	}

	// Create referral request
	createReq := CreateReferralRequest{
		SourceVendorID:  sourceVendorID,
		DestVendorID:    destVendorID,
		ClientName:      req.ClientName,
		ClientEmail:     req.ClientEmail,
		ClientPhone:     req.ClientPhone,
		EventType:       req.EventType,
		EventDate:       req.EventDate,
		ServiceCategory: serviceCategoryID,
		EstimatedValue:  req.EstimatedValue,
		Notes:           req.Notes,
	}

	// Create referral using the engine
	referral, err := h.referralEngine.CreateReferral(c.Request.Context(), createReq)
	if err != nil {
		h.logger.Error("Failed to create referral",
			zap.Error(err),
			zap.String("source_vendor", sourceVendorID.String()),
			zap.String("dest_vendor", destVendorID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create referral",
		})
		return
	}

	h.logger.Info("Referral created",
		zap.String("referral_id", referral.ID.String()),
		zap.String("tracking_code", referral.TrackingCode),
	)

	c.JSON(http.StatusCreated, gin.H{
		"referral": referral,
		"message": "Referral sent successfully. Destination vendor will be notified.",
	})
}

// UpdateReferralStatus handles PUT /api/v1/vendornet/referrals/:id/status
// Updates the status of a referral
func (h *Handler) UpdateReferralStatus(c *gin.Context) {
	referralIDStr := c.Param("id")
	referralID, err := uuid.Parse(referralIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid referral ID format",
		})
		return
	}

	var req struct {
		Status      string  `json:"status" binding:"required"`
		VendorID    string  `json:"vendor_id" binding:"required"`
		Notes       string  `json:"notes"`
		ActualValue float64 `json:"actual_value,omitempty"` // For converted status
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// Parse vendor ID
	vendorID, err := uuid.Parse(req.VendorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid vendor_id format",
		})
		return
	}

	// Validate status
	newStatus := ReferralStatus(req.Status)
	validStatuses := []ReferralStatus{
		ReferralAccepted,
		ReferralDeclined,
		ReferralContacted,
		ReferralQuoted,
		ReferralConverted,
		ReferralLost,
	}
	isValid := false
	for _, s := range validStatuses {
		if newStatus == s {
			isValid = true
			break
		}
	}
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid status. Must be one of: accepted, declined, contacted, quoted, converted, lost",
		})
		return
	}

	// If status is converted, get the referral and update actual value
	if newStatus == ReferralConverted {
		if req.ActualValue <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "actual_value is required for converted status",
			})
			return
		}

		// Get referral and update actual value
		referral, err := h.referralEngine.getReferral(c.Request.Context(), referralID)
		if err == nil {
			referral.ActualValue = req.ActualValue
			h.referralEngine.updateReferral(c.Request.Context(), referral)
		}
	}

	// Update referral status
	if err := h.referralEngine.UpdateReferralStatus(c.Request.Context(), referralID, newStatus, vendorID, req.Notes); err != nil {
		h.logger.Error("Failed to update referral status",
			zap.Error(err),
			zap.String("referral_id", referralID.String()),
			zap.String("new_status", string(newStatus)),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update referral status: " + err.Error(),
		})
		return
	}

	h.logger.Info("Referral status updated",
		zap.String("referral_id", referralID.String()),
		zap.String("status", string(newStatus)),
	)

	// Get updated referral
	updatedReferral, _ := h.referralEngine.getReferral(c.Request.Context(), referralID)

	c.JSON(http.StatusOK, gin.H{
		"referral": updatedReferral,
		"message": "Referral status updated successfully",
	})
}

// GetNetworkAnalytics handles GET /api/v1/vendornet/analytics
// Returns network analytics for a vendor
func (h *Handler) GetNetworkAnalytics(c *gin.Context) {
	// Get vendor ID from query or authenticated user
	vendorIDStr := c.Query("vendor_id")
	if vendorIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "vendor_id parameter is required",
		})
		return
	}

	vendorID, err := uuid.Parse(vendorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid vendor_id format",
		})
		return
	}

	// Get analytics from analytics service
	stats, err := h.analytics.GetVendorStats(c.Request.Context(), vendorID)
	if err != nil {
		h.logger.Error("Failed to get network analytics",
			zap.Error(err),
			zap.String("vendor_id", vendorID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get network analytics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vendor_id":   vendorID.String(),
		"analytics":   stats,
		"generated_at": time.Now(),
	})
}

// Helper functions

func parseIntParam(s string, min, max int) (int, error) {
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return 0, err
	}
	if val < min || val > max {
		return 0, fmt.Errorf("value must be between %d and %d", min, max)
	}
	return val, nil
}

func (h *Handler) getActivePartnership(ctx context.Context, vendorA, vendorB uuid.UUID) (*Partnership, error) {
	query := `
		SELECT id, vendor_a_id, vendor_b_id, partnership_type, status,
		       terms, signed_at, expires_at, created_at
		FROM partnerships
		WHERE ((vendor_a_id = $1 AND vendor_b_id = $2) OR (vendor_a_id = $2 AND vendor_b_id = $1))
		  AND status = 'active'
		LIMIT 1
	`

	var p Partnership
	var termsJSON []byte
	err := h.db.QueryRow(ctx, query, vendorA, vendorB).Scan(
		&p.ID, &p.VendorAID, &p.VendorBID, &p.PartnershipType, &p.Status,
		&termsJSON, &p.ActivatedAt, &p.ExpiresAt, &p.ProposedAt,
	)

	if err != nil {
		return nil, err
	}

	// Unmarshal terms JSON
	if len(termsJSON) > 0 {
		json.Unmarshal(termsJSON, &p.Terms)
	}

	return &p, nil
}

func (h *Handler) getPartnership(ctx context.Context, partnershipID uuid.UUID) (*Partnership, error) {
	query := `
		SELECT id, vendor_a_id, vendor_b_id, partnership_type, status,
		       terms, performance, signed_at, expires_at, created_at
		FROM partnerships
		WHERE id = $1
	`

	var p Partnership
	var termsJSON, performanceJSON []byte

	err := h.db.QueryRow(ctx, query, partnershipID).Scan(
		&p.ID, &p.VendorAID, &p.VendorBID, &p.PartnershipType, &p.Status,
		&termsJSON, &performanceJSON, &p.ActivatedAt, &p.ExpiresAt, &p.ProposedAt,
	)

	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	if len(termsJSON) > 0 {
		json.Unmarshal(termsJSON, &p.Terms)
	}
	if len(performanceJSON) > 0 {
		// Parse performance metrics into Partnership fields
		var perf map[string]interface{}
		json.Unmarshal(performanceJSON, &perf)
		if totalReferrals, ok := perf["total_referrals"].(float64); ok {
			p.TotalReferrals = int(totalReferrals)
		}
		if successfulReferrals, ok := perf["successful_referrals"].(float64); ok {
			p.SuccessfulReferrals = int(successfulReferrals)
		}
		if totalRevenue, ok := perf["total_revenue"].(float64); ok {
			p.TotalRevenue = totalRevenue
		}
	}

	return &p, nil
}

func (h *Handler) savePartnership(ctx context.Context, p *Partnership) error {
	termsJSON, err := json.Marshal(p.Terms)
	if err != nil {
		return err
	}

	// Store partnership metadata in performance JSONB field
	performance := map[string]interface{}{
		"name":        p.Name,
		"description": p.Description,
		"proposed_by": p.ProposedBy.String(),
	}
	performanceJSON, err := json.Marshal(performance)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO partnerships (
			id, vendor_a_id, vendor_b_id, partnership_type, status,
			terms, performance, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`

	_, err = h.db.Exec(ctx, query,
		p.ID, p.VendorAID, p.VendorBID, p.PartnershipType, p.Status,
		termsJSON, performanceJSON, p.ExpiresAt,
	)

	return err
}
