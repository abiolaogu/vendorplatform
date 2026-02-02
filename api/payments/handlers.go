// Package payments provides HTTP handlers for payment processing
package payments

import (
	"fmt"
	"io"
	"net/http"

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
		return
	}

	currency := c.DefaultQuery("currency", "NGN")

	ctx := c.Request.Context()
	wallet, err := h.paymentService.GetOrCreateWallet(ctx, userID, currency)
	if err != nil {
		h.logger.Error("Failed to get wallet",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve wallet",
		})
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
}
