// Package payments provides HTTP handlers for payment processing
package payments

import (
	"fmt"
	"io"
	"net/http"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

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
		payments.POST("/initialize", h.InitializePayment)
		payments.GET("/:id", h.GetTransaction)
		payments.POST("/verify/:reference", h.VerifyPayment)
		payments.POST("/webhook/paystack", h.PaystackWebhook)
	}

	wallets := router.Group("/wallets")
	{
		wallets.GET("/:user_id", h.GetWallet)
	}

	payouts := router.Group("/payouts")
	{
		payouts.POST("", h.RequestPayout)
	}

	escrow := router.Group("/escrow")
	{
		escrow.POST("/:booking_id/release", h.ReleaseEscrow)
		escrow.POST("/:booking_id/refund", h.RefundEscrow)
	}
}

// InitializePayment handles payment initialization
func (h *Handler) InitializePayment(c *gin.Context) {
	var req payment.InitializePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid payment initialization request",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Validate required fields
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Amount must be greater than 0",
		})
		return
	}

	if req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email is required",
		})
		return
	}

	if req.Currency == "" {
		req.Currency = "NGN"
	}

	if req.Provider == "" {
		req.Provider = payment.ProviderPaystack
	}

	// Initialize payment
	ctx := c.Request.Context()
	resp, err := h.paymentService.InitializePayment(ctx, req)
	if err != nil {
		h.logger.Error("Failed to initialize payment",
			zap.Error(err),
			zap.String("user_id", req.UserID.String()),
			zap.Int64("amount", req.Amount),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to initialize payment",
		})
		return
	}

	h.logger.Info("Payment initialized",
		zap.String("transaction_id", resp.TransactionID.String()),
		zap.String("reference", resp.Reference),
		zap.String("provider", string(resp.Provider)),
	)

	c.JSON(http.StatusOK, resp)
}

// VerifyPayment verifies a payment with the provider
func (h *Handler) VerifyPayment(c *gin.Context) {
	reference := c.Param("reference")

	if reference == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Payment reference is required",
		})
		return
	}

	ctx := c.Request.Context()

	// Using Paystack verification
	txn, err := h.paymentService.VerifyPaystack(ctx, reference)
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
		payments.GET("/escrow/:booking_id", h.GetEscrowStatus)
	}

	// Webhook endpoints (public, no auth)
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
	// TODO: Get user_id from authenticated session
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	var req InitializePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get booking details to determine amount
	// TODO: Fetch booking from booking service
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify payment",
		})
		return
	}

	h.logger.Info("Payment verified",
		zap.String("transaction_id", txn.ID.String()),
		zap.String("reference", reference),
		zap.String("status", string(txn.Status)),
	)

	c.JSON(http.StatusOK, txn)
}

// PaystackWebhook handles Paystack webhook events
func (h *Handler) PaystackWebhook(c *gin.Context) {
	signature := c.GetHeader("x-paystack-signature")
	if signature == "" {
		h.logger.Error("Missing Paystack signature")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing signature",
		})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("Failed to read webhook body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read request body",
		})
		return
	}

	ctx := c.Request.Context()
	if err := h.paymentService.HandlePaystackWebhook(ctx, body, signature); err != nil {
		h.logger.Error("Failed to process webhook",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to process webhook",
		})
		return
	}

	h.logger.Info("Paystack webhook processed successfully")

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

// GetWallet retrieves a user's wallet
func (h *Handler) GetWallet(c *gin.Context) {
	userIDStr := c.Param("user_id")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify payment"})
		return
	}

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

	// TODO: Get user_id from authenticated session and verify ownership
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	// TODO: Implement GetTransactionByID in payment service
	_ = txnID
	_ = userID

	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// GetWallet retrieves user's wallet information
func (h *Handler) GetWallet(c *gin.Context) {
	// TODO: Get user_id from authenticated session
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	currency := c.DefaultQuery("currency", "NGN")

	ctx := c.Request.Context()
	wallet, err := h.paymentService.GetOrCreateWallet(ctx, userID, currency)
	wallet, err := h.paymentService.GetOrCreateWallet(c.Request.Context(), userID, currency)
	if err != nil {
		h.logger.Error("Failed to get wallet",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve wallet",
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get wallet"})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

// RequestPayout handles vendor payout requests
func (h *Handler) RequestPayout(c *gin.Context) {
	var req payment.PayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid payout request",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Validate required fields
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Amount must be greater than 0",
		})
		return
	}

	if req.AccountNumber == "" || req.BankCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Bank account details are required",
		})
		return
	}

	if req.Currency == "" {
		req.Currency = "NGN"
	}

	ctx := c.Request.Context()
	txn, err := h.paymentService.RequestPayout(ctx, req)
	if err != nil {
		h.logger.Error("Failed to request payout",
			zap.Error(err),
			zap.String("vendor_id", req.VendorID.String()),
			zap.Int64("amount", req.Amount),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to request payout: %v", err),
		})
		return
	}

	h.logger.Info("Payout requested",
		zap.String("transaction_id", txn.ID.String()),
		zap.String("vendor_id", req.VendorID.String()),
		zap.Int64("amount", req.Amount),
	)

	c.JSON(http.StatusOK, txn)
}

// ReleaseEscrow releases held funds to vendor
func (h *Handler) ReleaseEscrow(c *gin.Context) {
	bookingIDStr := c.Param("booking_id")

	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid booking ID",
		})
		return
	}

	ctx := c.Request.Context()
	if err := h.paymentService.ReleaseEscrow(ctx, bookingID); err != nil {
		h.logger.Error("Failed to release escrow",
			zap.Error(err),
			zap.String("booking_id", bookingID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to release escrow: %v", err),
		})
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
	bookingIDStr := c.Param("booking_id")

	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid booking ID",
		})
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

	ctx := c.Request.Context()
	if err := h.paymentService.RefundEscrow(ctx, bookingID, body.Reason); err != nil {
		h.logger.Error("Failed to refund escrow",
			zap.Error(err),
			zap.String("booking_id", bookingID.String()),
			zap.String("reason", body.Reason),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to refund escrow: %v", err),
		})
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

// GetTransaction retrieves a transaction by ID
func (h *Handler) GetTransaction(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid transaction ID",
		})
		return
	}

	h.logger.Warn("GetTransaction endpoint called but service method not fully implemented",
		zap.String("transaction_id", id.String()),
	)

	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Use /api/v1/payments/verify/:reference endpoint instead",
		"note":  "Transaction retrieval by ID requires additional service method",
	})
// GetWalletTransactions retrieves wallet transaction history
func (h *Handler) GetWalletTransactions(c *gin.Context) {
	// TODO: Get user_id from authenticated session
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
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
	// TODO: Get vendor_id from authenticated session
	vendorID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
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
		"payout_id":   txn.ID,
		"reference":   txn.Reference,
		"status":      txn.Status,
		"amount":      txn.Amount,
		"currency":    txn.Currency,
		"created_at":  txn.CreatedAt,
	})
}

// ListPayouts lists vendor's payout history
func (h *Handler) ListPayouts(c *gin.Context) {
	// TODO: Get vendor_id from authenticated session
	vendorID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
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

	// TODO: Get vendor_id from authenticated session and verify ownership
	vendorID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	_ = payoutID
	_ = vendorID

	// TODO: Implement GetPayout in payment service
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// GetEscrowStatus retrieves escrow status for a booking
func (h *Handler) GetEscrowStatus(c *gin.Context) {
	bookingID, err := uuid.Parse(c.Param("booking_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking ID"})
		return
	}

	// TODO: Get user_id from authenticated session and verify ownership
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	_ = bookingID
	_ = userID

	// TODO: Implement GetEscrowStatus in payment service
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
