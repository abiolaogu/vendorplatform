// Package vendornet provides HTTP handlers for vendor network and partnerships
package vendornet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Handler handles VendorNet HTTP requests
type Handler struct {
	matchingEngine  *PartnershipMatchingEngine
	referralEngine  *ReferralEngine
	analytics       *NetworkAnalytics
	db              *pgxpool.Pool
	cache           *redis.Client
	logger          *zap.Logger
}

// NewHandler creates a new VendorNet handler
func NewHandler(db *pgxpool.Pool, cache *redis.Client, logger *zap.Logger) *Handler {
	adjacencyService := &AdjacencyService{db: db}

	return &Handler{
		matchingEngine: &PartnershipMatchingEngine{
			db:               db,
			cache:            cache,
			adjacencyService: adjacencyService,
		},
		referralEngine: &ReferralEngine{
			db:              db,
			cache:           cache,
			notificationSvc: &NotificationService{},
			paymentSvc:      &PaymentService{},
		},
		analytics: &NetworkAnalytics{
			db:    db,
			cache: cache,
		},
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

// RegisterRoutes registers VendorNet routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	vendornet := router.Group("/vendornet")
	{
		// Partner matching
		vendornet.GET("/matches/:vendor_id", h.GetPartnerMatches)

		// Partnerships
		vendornet.POST("/partnerships", h.CreatePartnership)
		vendornet.GET("/partnerships/:id", h.GetPartnership)
		vendornet.PUT("/partnerships/:id/status", h.UpdatePartnershipStatus)

		// Referrals
		vendornet.POST("/referrals", h.CreateReferral)
		vendornet.GET("/referrals/:id", h.GetReferral)
		vendornet.PUT("/referrals/:id/status", h.UpdateReferralStatus)
		vendornet.GET("/referrals/vendor/:vendor_id", h.ListReferrals)

		// Analytics
		vendornet.GET("/analytics/:vendor_id", h.GetNetworkAnalytics)

		// Connections
		vendornet.POST("/connections", h.CreateConnection)
		vendornet.GET("/connections/:vendor_id", h.ListConnections)
	}
}

// GetPartnerMatches handles GET /vendornet/matches/:vendor_id
func (h *Handler) GetPartnerMatches(c *gin.Context) {
	vendorIDStr := c.Param("vendor_id")
	vendorID, err := uuid.Parse(vendorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor ID"})
		return
	}

	// Get limit from query params, default to 10
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	matches, err := h.matchingEngine.FindPartnerMatches(c.Request.Context(), vendorID, limit)
	if err != nil {
		h.logger.Error("Failed to find partner matches", zap.Error(err), zap.String("vendor_id", vendorIDStr))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find matches"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vendor_id": vendorID,
		"matches":   matches,
		"count":     len(matches),
	})
}

// CreatePartnership handles POST /vendornet/partnerships
func (h *Handler) CreatePartnership(c *gin.Context) {
	var req struct {
		VendorAID       string           `json:"vendor_a_id" binding:"required"`
		VendorBID       string           `json:"vendor_b_id" binding:"required"`
		PartnershipType PartnershipType  `json:"partnership_type" binding:"required"`
		Name            string           `json:"name" binding:"required"`
		Description     string           `json:"description"`
		Terms           PartnershipTerms `json:"terms" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	vendorAID, err := uuid.Parse(req.VendorAID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor_a_id"})
		return
	}

	vendorBID, err := uuid.Parse(req.VendorBID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor_b_id"})
		return
	}

	// Validate partnership type
	validTypes := map[PartnershipType]bool{
		PartnershipReferral:     true,
		PartnershipPreferred:    true,
		PartnershipExclusive:    true,
		PartnershipJointVenture: true,
		PartnershipWhiteLabel:   true,
	}
	if !validTypes[req.PartnershipType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid partnership type"})
		return
	}

	// TODO: Get proposer from authenticated session
	proposedBy := vendorAID

	partnership := &Partnership{
		ID:              uuid.New(),
		VendorAID:       vendorAID,
		VendorBID:       vendorBID,
		PartnershipType: req.PartnershipType,
		Name:            req.Name,
		Description:     req.Description,
		Terms:           req.Terms,
		Status:          PartnershipProposed,
		ProposedBy:      proposedBy,
	}

	if err := h.savePartnership(c.Request.Context(), partnership); err != nil {
		h.logger.Error("Failed to create partnership", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create partnership"})
		return
	}

	c.JSON(http.StatusCreated, partnership)
}

// GetPartnership handles GET /vendornet/partnerships/:id
func (h *Handler) GetPartnership(c *gin.Context) {
	partnershipIDStr := c.Param("id")
	partnershipID, err := uuid.Parse(partnershipIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid partnership ID"})
		return
	}

	partnership, err := h.getPartnership(c.Request.Context(), partnershipID)
	if err != nil {
		h.logger.Error("Failed to get partnership", zap.Error(err), zap.String("partnership_id", partnershipIDStr))
		c.JSON(http.StatusNotFound, gin.H{"error": "Partnership not found"})
		return
	}

	c.JSON(http.StatusOK, partnership)
}

// UpdatePartnershipStatus handles PUT /vendornet/partnerships/:id/status
func (h *Handler) UpdatePartnershipStatus(c *gin.Context) {
	partnershipIDStr := c.Param("id")
	partnershipID, err := uuid.Parse(partnershipIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid partnership ID"})
		return
	}

	var req struct {
		Status PartnershipStatus `json:"status" binding:"required"`
		Reason string            `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate status
	validStatuses := map[PartnershipStatus]bool{
		PartnershipProposed:    true,
		PartnershipNegotiating: true,
		PartnershipActive:      true,
		PartnershipPaused:      true,
		PartnershipExpired:     true,
		PartnershipTerminated:  true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	if err := h.updatePartnershipStatus(c.Request.Context(), partnershipID, req.Status, req.Reason); err != nil {
		h.logger.Error("Failed to update partnership status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"partnership_id": partnershipID,
		"status":         req.Status,
		"message":        "Status updated successfully",
	})
}

// CreateReferral handles POST /vendornet/referrals
func (h *Handler) CreateReferral(c *gin.Context) {
	var req struct {
		SourceVendorID  string     `json:"source_vendor_id" binding:"required"`
		DestVendorID    string     `json:"dest_vendor_id" binding:"required"`
		ClientName      string     `json:"client_name" binding:"required"`
		ClientEmail     string     `json:"client_email" binding:"required,email"`
		ClientPhone     string     `json:"client_phone" binding:"required"`
		EventType       string     `json:"event_type" binding:"required"`
		EventDate       *string    `json:"event_date"`
		ServiceCategory string     `json:"service_category_id" binding:"required"`
		EstimatedValue  float64    `json:"estimated_value" binding:"required,gt=0"`
		Notes           string     `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	sourceVendorID, err := uuid.Parse(req.SourceVendorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source_vendor_id"})
		return
	}

	destVendorID, err := uuid.Parse(req.DestVendorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid dest_vendor_id"})
		return
	}

	serviceCategoryID, err := uuid.Parse(req.ServiceCategory)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service_category_id"})
		return
	}

	createReq := CreateReferralRequest{
		SourceVendorID:  sourceVendorID,
		DestVendorID:    destVendorID,
		ClientName:      req.ClientName,
		ClientEmail:     req.ClientEmail,
		ClientPhone:     req.ClientPhone,
		EventType:       req.EventType,
		ServiceCategory: serviceCategoryID,
		EstimatedValue:  req.EstimatedValue,
		Notes:           req.Notes,
	}

	referral, err := h.referralEngine.CreateReferral(c.Request.Context(), createReq)
	if err != nil {
		h.logger.Error("Failed to create referral", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create referral"})
		return
	}

	c.JSON(http.StatusCreated, referral)
}

// GetReferral handles GET /vendornet/referrals/:id
func (h *Handler) GetReferral(c *gin.Context) {
	referralIDStr := c.Param("id")
	referralID, err := uuid.Parse(referralIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid referral ID"})
		return
	}

	referral, err := h.referralEngine.getReferral(c.Request.Context(), referralID)
	if err != nil {
		h.logger.Error("Failed to get referral", zap.Error(err), zap.String("referral_id", referralIDStr))
		c.JSON(http.StatusNotFound, gin.H{"error": "Referral not found"})
		return
	}

	// TODO: Verify ownership/permission

	c.JSON(http.StatusOK, referral)
}

// UpdateReferralStatus handles PUT /vendornet/referrals/:id/status
func (h *Handler) UpdateReferralStatus(c *gin.Context) {
	referralIDStr := c.Param("id")
	referralID, err := uuid.Parse(referralIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid referral ID"})
		return
	}

	var req struct {
		Status      ReferralStatus `json:"status" binding:"required"`
		Notes       string         `json:"notes"`
		ActualValue float64        `json:"actual_value"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate status
	validStatuses := map[ReferralStatus]bool{
		ReferralPending:   true,
		ReferralAccepted:  true,
		ReferralDeclined:  true,
		ReferralContacted: true,
		ReferralQuoted:    true,
		ReferralConverted: true,
		ReferralLost:      true,
		ReferralExpired:   true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	// TODO: Get vendor from authenticated session
	// For now, we'll use a placeholder
	vendorID := uuid.New()

	// Update actual value if converting
	if req.Status == ReferralConverted && req.ActualValue > 0 {
		if err := h.updateReferralValue(c.Request.Context(), referralID, req.ActualValue); err != nil {
			h.logger.Error("Failed to update referral value", zap.Error(err))
		}
	}

	if err := h.referralEngine.UpdateReferralStatus(c.Request.Context(), referralID, req.Status, vendorID, req.Notes); err != nil {
		h.logger.Error("Failed to update referral status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"referral_id": referralID,
		"status":      req.Status,
		"message":     "Status updated successfully",
	})
}

// ListReferrals handles GET /vendornet/referrals/vendor/:vendor_id
func (h *Handler) ListReferrals(c *gin.Context) {
	vendorIDStr := c.Param("vendor_id")
	vendorID, err := uuid.Parse(vendorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor ID"})
		return
	}

	// Get filter parameters
	direction := c.DefaultQuery("direction", "all") // sent, received, all
	status := c.Query("status")

	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 100 {
		limit = 20
	}

	referrals, err := h.listReferrals(c.Request.Context(), vendorID, direction, status, limit)
	if err != nil {
		h.logger.Error("Failed to list referrals", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list referrals"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vendor_id":  vendorID,
		"direction":  direction,
		"referrals":  referrals,
		"count":      len(referrals),
	})
}

// GetNetworkAnalytics handles GET /vendornet/analytics/:vendor_id
func (h *Handler) GetNetworkAnalytics(c *gin.Context) {
	vendorIDStr := c.Param("vendor_id")
	vendorID, err := uuid.Parse(vendorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor ID"})
		return
	}

	stats, err := h.analytics.GetVendorStats(c.Request.Context(), vendorID)
	if err != nil {
		h.logger.Error("Failed to get network analytics", zap.Error(err), zap.String("vendor_id", vendorIDStr))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vendor_id": vendorID,
		"stats":     stats,
	})
}

// CreateConnection handles POST /vendornet/connections
func (h *Handler) CreateConnection(c *gin.Context) {
	var req struct {
		VendorAID        string         `json:"vendor_a_id" binding:"required"`
		VendorBID        string         `json:"vendor_b_id" binding:"required"`
		ConnectionType   ConnectionType `json:"connection_type" binding:"required"`
		RelationshipNote string         `json:"relationship_note"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	vendorAID, err := uuid.Parse(req.VendorAID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor_a_id"})
		return
	}

	vendorBID, err := uuid.Parse(req.VendorBID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor_b_id"})
		return
	}

	connection := &Connection{
		ID:               uuid.New(),
		VendorAID:        vendorAID,
		VendorBID:        vendorBID,
		ConnectionType:   req.ConnectionType,
		RelationshipNote: req.RelationshipNote,
		Status:           ConnectionPending,
		InitiatedBy:      vendorAID,
	}

	if err := h.saveConnection(c.Request.Context(), connection); err != nil {
		h.logger.Error("Failed to create connection", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create connection"})
		return
	}

	c.JSON(http.StatusCreated, connection)
}

// ListConnections handles GET /vendornet/connections/:vendor_id
func (h *Handler) ListConnections(c *gin.Context) {
	vendorIDStr := c.Param("vendor_id")
	vendorID, err := uuid.Parse(vendorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor ID"})
		return
	}

	status := c.DefaultQuery("status", "accepted")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 100 {
		limit = 50
	}

	connections, err := h.listConnections(c.Request.Context(), vendorID, status, limit)
	if err != nil {
		h.logger.Error("Failed to list connections", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list connections"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vendor_id":   vendorID,
		"connections": connections,
		"count":       len(connections),
	})
}

// =============================================================================
// DATABASE HELPER METHODS
// =============================================================================

func (h *Handler) savePartnership(ctx context.Context, p *Partnership) error {
	query := `
		INSERT INTO partnerships (
			id, vendor_a_id, vendor_b_id, partnership_type, name, description,
			terms, status, proposed_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	termsJSON, _ := json.Marshal(p.Terms)

	_, err := h.db.Exec(ctx, query,
		p.ID, p.VendorAID, p.VendorBID, p.PartnershipType, p.Name, p.Description,
		termsJSON, p.Status, p.ProposedBy,
	)

	return err
}

func (h *Handler) getPartnership(ctx context.Context, partnershipID uuid.UUID) (*Partnership, error) {
	query := `
		SELECT id, vendor_a_id, vendor_b_id, partnership_type, name, description,
		       terms, status, proposed_by, activated_at, expires_at
		FROM partnerships
		WHERE id = $1
	`

	var p Partnership
	var termsJSON []byte

	err := h.db.QueryRow(ctx, query, partnershipID).Scan(
		&p.ID, &p.VendorAID, &p.VendorBID, &p.PartnershipType, &p.Name, &p.Description,
		&termsJSON, &p.Status, &p.ProposedBy, &p.ActivatedAt, &p.ExpiresAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(termsJSON, &p.Terms)
	return &p, nil
}

func (h *Handler) updatePartnershipStatus(ctx context.Context, partnershipID uuid.UUID, status PartnershipStatus, reason string) error {
	query := `
		UPDATE partnerships
		SET status = $2, termination_reason = $3
		WHERE id = $1
	`

	_, err := h.db.Exec(ctx, query, partnershipID, status, reason)
	return err
}

func (h *Handler) updateReferralValue(ctx context.Context, referralID uuid.UUID, actualValue float64) error {
	query := `UPDATE referrals SET actual_value = $2 WHERE id = $1`
	_, err := h.db.Exec(ctx, query, referralID, actualValue)
	return err
}

func (h *Handler) listReferrals(ctx context.Context, vendorID uuid.UUID, direction string, status string, limit int) ([]Referral, error) {
	query := `
		SELECT id, source_vendor_id, dest_vendor_id, client_name, client_email,
		       event_type, estimated_value, status, tracking_code, created_at
		FROM referrals
		WHERE
	`

	args := []interface{}{vendorID}
	argPos := 1

	switch direction {
	case "sent":
		query += fmt.Sprintf("source_vendor_id = $%d", argPos)
	case "received":
		query += fmt.Sprintf("dest_vendor_id = $%d", argPos)
	default:
		query += fmt.Sprintf("(source_vendor_id = $%d OR dest_vendor_id = $%d)", argPos, argPos)
	}

	if status != "" {
		argPos++
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, status)
	}

	argPos++
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argPos)
	args = append(args, limit)

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var referrals []Referral
	for rows.Next() {
		var r Referral
		if err := rows.Scan(
			&r.ID, &r.SourceVendorID, &r.DestVendorID, &r.ClientName, &r.ClientEmail,
			&r.EventType, &r.EstimatedValue, &r.Status, &r.TrackingCode, &r.CreatedAt,
		); err != nil {
			continue
		}
		referrals = append(referrals, r)
	}

	return referrals, nil
}

func (h *Handler) saveConnection(ctx context.Context, c *Connection) error {
	query := `
		INSERT INTO connections (
			id, vendor_a_id, vendor_b_id, connection_type, relationship_note,
			status, initiated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := h.db.Exec(ctx, query,
		c.ID, c.VendorAID, c.VendorBID, c.ConnectionType, c.RelationshipNote,
		c.Status, c.InitiatedBy,
	)

	return err
}

func (h *Handler) listConnections(ctx context.Context, vendorID uuid.UUID, status string, limit int) ([]Connection, error) {
	query := `
		SELECT id, vendor_a_id, vendor_b_id, connection_type, relationship_note,
		       status, initiated_by, requested_at, accepted_at
		FROM connections
		WHERE (vendor_a_id = $1 OR vendor_b_id = $1)
		  AND status = $2
		ORDER BY requested_at DESC
		LIMIT $3
	`

	rows, err := h.db.Query(ctx, query, vendorID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []Connection
	for rows.Next() {
		var c Connection
		if err := rows.Scan(
			&c.ID, &c.VendorAID, &c.VendorBID, &c.ConnectionType, &c.RelationshipNote,
			&c.Status, &c.InitiatedBy, &c.RequestedAt, &c.AcceptedAt,
		); err != nil {
			continue
		}
		connections = append(connections, c)
	}

	return connections, nil
}
