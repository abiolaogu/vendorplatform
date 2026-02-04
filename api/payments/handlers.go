// Package payments provides HTTP handlers for payment processing
package payments

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/auth"
	"github.com/BillyRonksGlobal/vendorplatform/internal/payment"
)

// Handler handles payment HTTP requests
type Handler struct {
	paymentService *payment.Service
	logger         *zap.Logger
}

// NewHandler creates a new payment handler
func NewHandler(paymentService *payment.Service, logger *zap.Logger) *Handler {
	return &Handler{
		paymentService: paymentService,
		logger:         logger,
	}
}

// RegisterRoutes registers payment routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	payments := router.Group("/payments")
	{
		// Payment initialization and verification
		payments.POST("/initialize", h.InitializePayment)
		payments.GET("/verify/:reference", h.VerifyPayment)
		payments.GET("/transactions/:id", h.GetTransaction)

		// Wallet management
		payments.GET("/wallet", h.GetWallet)
		payments.GET("/wallet/transactions", h.GetWalletTransactions)

		// Payout requests (for vendors)
		payments.POST("/payouts", h.RequestPayout)
		payments.GET("/payouts", h.ListPayouts)
		payments.GET("/payouts/:id", h.GetPayout)

		// Escrow management
		payments.POST("/escrow/:booking_id/release", h.ReleaseEscrow)
		payments.POST("/escrow/:booking_id/refund", h.RefundEscrow)
		payments.GET("/escrow/:booking_id", h.GetEscrowStatus)
	}

	// Webhook endpoints (public, no auth - should be registered separately)
	webhooks := router.Group("/webhooks")
	{
		webhooks.POST("/paystack", h.PaystackWebhook)
		webhooks.POST("/flutterwave", h.FlutterwaveWebhook)
	}
}

// InitializePaymentRequest represents payment initialization request
type InitializePaymentRequest struct {
	BookingID   uuid.UUID              `json:"booking_id" binding:"required"`
	Email       string                 `json:"email" binding:"required,email"`
	Provider    string                 `json:"provider" binding:"required,oneof=paystack flutterwave"`
	CallbackURL string                 `json:"callback_url"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// InitializePayment initializes a payment for a booking
func (h *Handler) InitializePayment(c *gin.Context) {
	// Get user_id from authenticated session
	userID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req InitializePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get booking details to determine amount
	// TODO: Fetch booking from booking service to get actual amount and validate ownership
	bookingAmount := int64(50000) // Placeholder - should come from booking
	bookingDescription := "Service booking payment"

	// Create payment initialization request
	paymentReq := payment.InitializePaymentRequest{
		UserID:      userID,
		BookingID:   &req.BookingID,
		Amount:      bookingAmount,
		Currency:    "NGN",
		Description: bookingDescription,
		Email:       req.Email,
		Provider:    payment.PaymentProvider(req.Provider),
		Metadata:    req.Metadata,
		UseEscrow:   true, // Always use escrow for bookings
		CallbackURL: req.CallbackURL,
	}

	// Initialize payment
	resp, err := h.paymentService.InitializePayment(c.Request.Context(), paymentReq)
	if err != nil {
		h.logger.Error("Failed to initialize payment",
			zap.Error(err),
			zap.String("user_id", userID.String()),
			zap.String("booking_id", req.BookingID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction_id":    resp.TransactionID,
		"reference":         resp.Reference,
		"authorization_url": resp.AuthorizationURL,
		"access_code":       resp.AccessCode,
		"provider":          resp.Provider,
	})
}

// VerifyPayment verifies a payment transaction
func (h *Handler) VerifyPayment(c *gin.Context) {
	reference := c.Param("reference")
	if reference == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reference is required"})
		return
	}

	// Verify with Paystack (default provider)
	txn, err := h.paymentService.VerifyPaystack(c.Request.Context(), reference)
	if err != nil {
		h.logger.Error("Failed to verify payment",
			zap.Error(err),
			zap.String("reference", reference),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify payment"})
		return
	}

	h.logger.Info("Payment verified",
		zap.String("transaction_id", txn.ID.String()),
		zap.String("reference", reference),
		zap.String("status", string(txn.Status)),
	)

	c.JSON(http.StatusOK, gin.H{
		"transaction": txn,
		"success":     txn.Status == payment.StatusSuccess,
	})
}

// GetTransaction retrieves a transaction by ID
func (h *Handler) GetTransaction(c *gin.Context) {
	txnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction ID"})
		return
	}

	// Get user_id from authenticated session and verify ownership
	userID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// TODO: Implement GetTransactionByID in payment service with ownership verification
	_ = txnID
	_ = userID

	h.logger.Warn("GetTransaction endpoint called but service method not fully implemented",
		zap.String("transaction_id", txnID.String()),
		zap.String("user_id", userID.String()),
	)

	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Use /api/v1/payments/verify/:reference endpoint instead",
		"note":  "Transaction retrieval by ID requires additional service method",
	})
}

// GetWallet retrieves user's wallet information
func (h *Handler) GetWallet(c *gin.Context) {
	// Get user_id from authenticated session
	userID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	currency := c.DefaultQuery("currency", "NGN")

	wallet, err := h.paymentService.GetOrCreateWallet(c.Request.Context(), userID, currency)
	if err != nil {
		h.logger.Error("Failed to get wallet",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get wallet"})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

// GetWalletTransactions retrieves wallet transaction history
func (h *Handler) GetWalletTransactions(c *gin.Context) {
	// Get user_id from authenticated session
	userID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	_ = userID
	_ = limit
	_ = offset

	// TODO: Implement GetWalletTransactions in payment service
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// PayoutRequest represents a payout request
type PayoutRequest struct {
	Amount        int64  `json:"amount" binding:"required,min=100"`
	Currency      string `json:"currency" binding:"required"`
	BankCode      string `json:"bank_code" binding:"required"`
	AccountNumber string `json:"account_number" binding:"required"`
	AccountName   string `json:"account_name" binding:"required"`
}

// RequestPayout initiates a vendor payout
func (h *Handler) RequestPayout(c *gin.Context) {
	// Get vendor_id from authenticated session
	vendorID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// TODO: Verify user has vendor role
	role, err := auth.GetRoleFromContext(c)
	if err == nil && role != auth.RoleVendor && role != auth.RoleAdmin && role != auth.RoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "vendor access required"})
		return
	}

	var req PayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payoutReq := payment.PayoutRequest{
		VendorID:      vendorID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		BankCode:      req.BankCode,
		AccountNumber: req.AccountNumber,
		AccountName:   req.AccountName,
	}

	txn, err := h.paymentService.RequestPayout(c.Request.Context(), payoutReq)
	if err != nil {
		h.logger.Error("Failed to request payout",
			zap.Error(err),
			zap.String("vendor_id", vendorID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payout_id":  txn.ID,
		"reference":  txn.Reference,
		"status":     txn.Status,
		"amount":     txn.Amount,
		"currency":   txn.Currency,
		"created_at": txn.CreatedAt,
	})
}

// ListPayouts lists vendor's payout history
func (h *Handler) ListPayouts(c *gin.Context) {
	// Get vendor_id from authenticated session
	vendorID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	_ = vendorID

	// TODO: Implement ListPayouts in payment service
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// GetPayout retrieves a specific payout
func (h *Handler) GetPayout(c *gin.Context) {
	payoutID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payout ID"})
		return
	}

	// Get vendor_id from authenticated session and verify ownership
	vendorID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	_ = payoutID
	_ = vendorID

	// TODO: Implement GetPayout in payment service with ownership verification
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// ReleaseEscrow releases held funds to vendor
func (h *Handler) ReleaseEscrow(c *gin.Context) {
	bookingID, err := uuid.Parse(c.Param("booking_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking ID"})
		return
	}

	// TODO: Verify user has permission to release escrow (admin or customer who owns booking)

	if err := h.paymentService.ReleaseEscrow(c.Request.Context(), bookingID); err != nil {
		h.logger.Error("Failed to release escrow",
			zap.Error(err),
			zap.String("booking_id", bookingID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to release escrow: %v", err)})
		return
	}

	h.logger.Info("Escrow released",
		zap.String("booking_id", bookingID.String()),
	)

	c.JSON(http.StatusOK, gin.H{
		"status":     "success",
		"message":    "Escrow funds released to vendor",
		"booking_id": bookingID.String(),
	})
}

// RefundEscrow refunds held funds to customer
func (h *Handler) RefundEscrow(c *gin.Context) {
	bookingID, err := uuid.Parse(c.Param("booking_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking ID"})
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		body.Reason = "Refund requested"
	}

	if body.Reason == "" {
		body.Reason = "Refund requested"
	}

	// TODO: Verify user has permission to refund escrow (admin or vendor who owns booking)

	if err := h.paymentService.RefundEscrow(c.Request.Context(), bookingID, body.Reason); err != nil {
		h.logger.Error("Failed to refund escrow",
			zap.Error(err),
			zap.String("booking_id", bookingID.String()),
			zap.String("reason", body.Reason),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to refund escrow: %v", err)})
		return
	}

	h.logger.Info("Escrow refunded",
		zap.String("booking_id", bookingID.String()),
		zap.String("reason", body.Reason),
	)

	c.JSON(http.StatusOK, gin.H{
		"status":     "success",
		"message":    "Escrow funds refunded to customer",
		"booking_id": bookingID.String(),
		"reason":     body.Reason,
	})
}

// GetEscrowStatus retrieves escrow status for a booking
func (h *Handler) GetEscrowStatus(c *gin.Context) {
	bookingID, err := uuid.Parse(c.Param("booking_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking ID"})
		return
	}

	// Get user_id from authenticated session and verify ownership
	userID, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	_ = bookingID
	_ = userID

	// TODO: Implement GetEscrowStatus in payment service with ownership verification
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// PaystackWebhook handles Paystack webhook events
func (h *Handler) PaystackWebhook(c *gin.Context) {
	signature := c.GetHeader("X-Paystack-Signature")
	if signature == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing signature"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.paymentService.HandlePaystackWebhook(c.Request.Context(), body, signature); err != nil {
		h.logger.Error("Failed to process Paystack webhook",
			zap.Error(err),
			zap.String("signature", signature),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// FlutterwaveWebhook handles Flutterwave webhook events
func (h *Handler) FlutterwaveWebhook(c *gin.Context) {
	// TODO: Implement Flutterwave webhook handling
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
