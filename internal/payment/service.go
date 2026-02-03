// =============================================================================
// PAYMENT SERVICE
// Multi-provider payment processing for marketplace transactions
// Supports: Paystack, Flutterwave, Stripe, Escrow
// =============================================================================

package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// =============================================================================
// TYPES
// =============================================================================

// Transaction represents a payment transaction
type Transaction struct {
	ID              uuid.UUID           `json:"id"`
	Reference       string              `json:"reference"`
	UserID          uuid.UUID           `json:"user_id"`
	VendorID        *uuid.UUID          `json:"vendor_id,omitempty"`
	BookingID       *uuid.UUID          `json:"booking_id,omitempty"`
	
	Type            TransactionType     `json:"type"`
	Status          TransactionStatus   `json:"status"`
	Provider        PaymentProvider     `json:"provider"`
	
	Amount          int64               `json:"amount"` // In kobo/cents
	Currency        string              `json:"currency"`
	Fee             int64               `json:"fee"`
	NetAmount       int64               `json:"net_amount"`
	
	Description     string              `json:"description"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	
	ProviderRef     string              `json:"provider_reference,omitempty"`
	ProviderData    map[string]interface{} `json:"provider_data,omitempty"`
	
	PaidAt          *time.Time          `json:"paid_at,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

type TransactionType string
const (
	TypePayment       TransactionType = "payment"
	TypePayout        TransactionType = "payout"
	TypeRefund        TransactionType = "refund"
	TypeEscrowHold    TransactionType = "escrow_hold"
	TypeEscrowRelease TransactionType = "escrow_release"
	TypeSubscription  TransactionType = "subscription"
)

type TransactionStatus string
const (
	StatusPending   TransactionStatus = "pending"
	StatusProcessing TransactionStatus = "processing"
	StatusSuccess   TransactionStatus = "success"
	StatusFailed    TransactionStatus = "failed"
	StatusRefunded  TransactionStatus = "refunded"
	StatusCancelled TransactionStatus = "cancelled"
	StatusHeld      TransactionStatus = "held" // For escrow
)

type PaymentProvider string
const (
	ProviderPaystack    PaymentProvider = "paystack"
	ProviderFlutterwave PaymentProvider = "flutterwave"
	ProviderStripe      PaymentProvider = "stripe"
	ProviderInternal    PaymentProvider = "internal" // Wallet transfers
)

// Wallet represents a user's internal wallet
type Wallet struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	Balance         int64     `json:"balance"`
	PendingBalance  int64     `json:"pending_balance"` // Held in escrow
	Currency        string    `json:"currency"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// EscrowAccount for holding payments until service delivery
type EscrowAccount struct {
	ID              uuid.UUID     `json:"id"`
	TransactionID   uuid.UUID     `json:"transaction_id"`
	BookingID       uuid.UUID     `json:"booking_id"`
	CustomerID      uuid.UUID     `json:"customer_id"`
	VendorID        uuid.UUID     `json:"vendor_id"`
	Amount          int64         `json:"amount"`
	Currency        string        `json:"currency"`
	Status          EscrowStatus  `json:"status"`
	ReleaseCondition string       `json:"release_condition"`
	ReleasedAt      *time.Time    `json:"released_at,omitempty"`
	DisputeID       *uuid.UUID    `json:"dispute_id,omitempty"`
	ExpiresAt       time.Time     `json:"expires_at"`
	CreatedAt       time.Time     `json:"created_at"`
}

type EscrowStatus string
const (
	EscrowHeld     EscrowStatus = "held"
	EscrowReleased EscrowStatus = "released"
	EscrowDisputed EscrowStatus = "disputed"
	EscrowRefunded EscrowStatus = "refunded"
	EscrowExpired  EscrowStatus = "expired"
)

// =============================================================================
// SERVICE
// =============================================================================

// Config for payment service
type Config struct {
	PaystackSecretKey    string
	PaystackPublicKey    string
	FlutterwaveSecretKey string
	FlutterwavePublicKey string
	StripeSecretKey      string
	StripePublicKey      string
	WebhookSecret        string
	DefaultCurrency      string
	PlatformFeePercent   float64 // Platform fee percentage
	EscrowExpiryDays     int
}

// Service handles payments
type Service struct {
	db     *pgxpool.Pool
	cache  *redis.Client
	config *Config
	http   *http.Client
}

// NewService creates a new payment service
func NewService(db *pgxpool.Pool, cache *redis.Client, config *Config) *Service {
	return &Service{
		db:     db,
		cache:  cache,
		config: config,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

// =============================================================================
// PAYMENT INITIALIZATION
// =============================================================================

// InitializePaymentRequest for starting a payment
type InitializePaymentRequest struct {
	UserID      uuid.UUID              `json:"user_id"`
	VendorID    *uuid.UUID             `json:"vendor_id,omitempty"`
	BookingID   *uuid.UUID             `json:"booking_id,omitempty"`
	Amount      int64                  `json:"amount"` // In kobo/cents
	Currency    string                 `json:"currency"`
	Description string                 `json:"description"`
	Email       string                 `json:"email"`
	Provider    PaymentProvider        `json:"provider"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	UseEscrow   bool                   `json:"use_escrow"`
	CallbackURL string                 `json:"callback_url"`
}

// InitializePaymentResponse from payment initialization
type InitializePaymentResponse struct {
	TransactionID  uuid.UUID `json:"transaction_id"`
	Reference      string    `json:"reference"`
	AuthorizationURL string  `json:"authorization_url"`
	AccessCode     string    `json:"access_code,omitempty"`
	Provider       PaymentProvider `json:"provider"`
}

// InitializePayment starts a payment flow
func (s *Service) InitializePayment(ctx context.Context, req InitializePaymentRequest) (*InitializePaymentResponse, error) {
	// Generate unique reference
	reference := fmt.Sprintf("VND-%s-%d", uuid.New().String()[:8], time.Now().Unix())
	
	// Calculate fees
	platformFee := int64(float64(req.Amount) * s.config.PlatformFeePercent / 100)
	netAmount := req.Amount - platformFee
	
	// Create transaction record
	txn := &Transaction{
		ID:          uuid.New(),
		Reference:   reference,
		UserID:      req.UserID,
		VendorID:    req.VendorID,
		BookingID:   req.BookingID,
		Type:        TypePayment,
		Status:      StatusPending,
		Provider:    req.Provider,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Fee:         platformFee,
		NetAmount:   netAmount,
		Description: req.Description,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Save to database
	if err := s.saveTransaction(ctx, txn); err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}
	
	// Initialize with provider
	var authURL, accessCode string
	var err error
	
	switch req.Provider {
	case ProviderPaystack:
		authURL, accessCode, err = s.initializePaystack(ctx, reference, req)
	case ProviderFlutterwave:
		authURL, err = s.initializeFlutterwave(ctx, reference, req)
	default:
		return nil, errors.New("unsupported payment provider")
	}
	
	if err != nil {
		// Update transaction as failed
		txn.Status = StatusFailed
		s.saveTransaction(ctx, txn)
		return nil, err
	}
	
	// If escrow, create escrow account
	if req.UseEscrow && req.VendorID != nil && req.BookingID != nil {
		escrow := &EscrowAccount{
			ID:              uuid.New(),
			TransactionID:   txn.ID,
			BookingID:       *req.BookingID,
			CustomerID:      req.UserID,
			VendorID:        *req.VendorID,
			Amount:          netAmount,
			Currency:        req.Currency,
			Status:          EscrowHeld,
			ReleaseCondition: "service_completed",
			ExpiresAt:       time.Now().AddDate(0, 0, s.config.EscrowExpiryDays),
			CreatedAt:       time.Now(),
		}
		s.createEscrow(ctx, escrow)
	}
	
	return &InitializePaymentResponse{
		TransactionID:    txn.ID,
		Reference:        reference,
		AuthorizationURL: authURL,
		AccessCode:       accessCode,
		Provider:         req.Provider,
	}, nil
}

// =============================================================================
// PAYSTACK INTEGRATION
// =============================================================================

func (s *Service) initializePaystack(ctx context.Context, reference string, req InitializePaymentRequest) (string, string, error) {
	payload := map[string]interface{}{
		"email":       req.Email,
		"amount":      req.Amount,
		"reference":   reference,
		"currency":    req.Currency,
		"callback_url": req.CallbackURL,
		"metadata":    req.Metadata,
	}
	
	body, _ := json.Marshal(payload)
	
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.paystack.co/transaction/initialize", 
		strings.NewReader(string(body)))
	httpReq.Header.Set("Authorization", "Bearer "+s.config.PaystackSecretKey)
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := s.http.Do(httpReq)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	
	var result struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			AuthorizationURL string `json:"authorization_url"`
			AccessCode       string `json:"access_code"`
			Reference        string `json:"reference"`
		} `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	
	if !result.Status {
		return "", "", errors.New(result.Message)
	}
	
	return result.Data.AuthorizationURL, result.Data.AccessCode, nil
}

// VerifyPaystack verifies a Paystack payment
func (s *Service) VerifyPaystack(ctx context.Context, reference string) (*Transaction, error) {
	httpReq, _ := http.NewRequestWithContext(ctx, "GET", 
		fmt.Sprintf("https://api.paystack.co/transaction/verify/%s", reference), nil)
	httpReq.Header.Set("Authorization", "Bearer "+s.config.PaystackSecretKey)
	
	resp, err := s.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			ID          int    `json:"id"`
			Status      string `json:"status"`
			Reference   string `json:"reference"`
			Amount      int64  `json:"amount"`
			PaidAt      string `json:"paid_at"`
			Channel     string `json:"channel"`
			Currency    string `json:"currency"`
		} `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	// Get transaction from database
	txn, err := s.GetTransactionByReference(ctx, reference)
	if err != nil {
		return nil, err
	}
	
	// Update based on provider response
	if result.Data.Status == "success" {
		txn.Status = StatusSuccess
		paidAt, _ := time.Parse(time.RFC3339, result.Data.PaidAt)
		txn.PaidAt = &paidAt
		txn.ProviderRef = fmt.Sprintf("%d", result.Data.ID)
	} else {
		txn.Status = StatusFailed
	}
	
	txn.UpdatedAt = time.Now()
	s.saveTransaction(ctx, txn)
	
	// If successful and has escrow, update escrow status
	if txn.Status == StatusSuccess && txn.VendorID != nil {
		s.updateEscrowOnPayment(ctx, txn.ID)
	}
	
	return txn, nil
}

// =============================================================================
// FLUTTERWAVE INTEGRATION
// =============================================================================

func (s *Service) initializeFlutterwave(ctx context.Context, reference string, req InitializePaymentRequest) (string, error) {
	payload := map[string]interface{}{
		"tx_ref":         reference,
		"amount":         float64(req.Amount) / 100, // Flutterwave uses major units
		"currency":       req.Currency,
		"redirect_url":   req.CallbackURL,
		"customer": map[string]string{
			"email": req.Email,
		},
		"meta": req.Metadata,
	}
	
	body, _ := json.Marshal(payload)
	
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", 
		"https://api.flutterwave.com/v3/payments", strings.NewReader(string(body)))
	httpReq.Header.Set("Authorization", "Bearer "+s.config.FlutterwaveSecretKey)
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := s.http.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Link string `json:"link"`
		} `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	if result.Status != "success" {
		return "", errors.New(result.Message)
	}
	
	return result.Data.Link, nil
}

// =============================================================================
// ESCROW
// =============================================================================

func (s *Service) createEscrow(ctx context.Context, escrow *EscrowAccount) error {
	query := `
		INSERT INTO escrow_accounts (
			id, transaction_id, booking_id, customer_id, vendor_id,
			amount, currency, status, release_condition, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := s.db.Exec(ctx, query,
		escrow.ID, escrow.TransactionID, escrow.BookingID,
		escrow.CustomerID, escrow.VendorID, escrow.Amount,
		escrow.Currency, escrow.Status, escrow.ReleaseCondition,
		escrow.ExpiresAt, escrow.CreatedAt,
	)
	return err
}

func (s *Service) updateEscrowOnPayment(ctx context.Context, transactionID uuid.UUID) error {
	_, err := s.db.Exec(ctx, 
		"UPDATE escrow_accounts SET status = $1 WHERE transaction_id = $2",
		EscrowHeld, transactionID,
	)
	return err
}

// ReleaseEscrow releases held funds to vendor
func (s *Service) ReleaseEscrow(ctx context.Context, bookingID uuid.UUID) error {
	var escrow EscrowAccount
	err := s.db.QueryRow(ctx, `
		SELECT id, vendor_id, amount, currency, status 
		FROM escrow_accounts WHERE booking_id = $1
	`, bookingID).Scan(&escrow.ID, &escrow.VendorID, &escrow.Amount, &escrow.Currency, &escrow.Status)
	
	if err != nil {
		return errors.New("escrow not found")
	}
	
	if escrow.Status != EscrowHeld {
		return errors.New("escrow not in held status")
	}
	
	// Credit vendor wallet
	if err := s.creditWallet(ctx, escrow.VendorID, escrow.Amount, escrow.Currency); err != nil {
		return err
	}
	
	// Update escrow status
	now := time.Now()
	_, err = s.db.Exec(ctx, 
		"UPDATE escrow_accounts SET status = $1, released_at = $2 WHERE id = $3",
		EscrowReleased, now, escrow.ID,
	)
	
	return err
}

// RefundEscrow refunds held funds to customer
func (s *Service) RefundEscrow(ctx context.Context, bookingID uuid.UUID, reason string) error {
	var escrow EscrowAccount
	err := s.db.QueryRow(ctx, `
		SELECT id, customer_id, amount, currency, status, transaction_id 
		FROM escrow_accounts WHERE booking_id = $1
	`, bookingID).Scan(&escrow.ID, &escrow.CustomerID, &escrow.Amount, &escrow.Currency, &escrow.Status, &escrow.TransactionID)
	
	if err != nil {
		return errors.New("escrow not found")
	}
	
	if escrow.Status != EscrowHeld {
		return errors.New("escrow not in held status")
	}
	
	// Create refund transaction
	refund := &Transaction{
		ID:          uuid.New(),
		Reference:   fmt.Sprintf("REF-%s", uuid.New().String()[:8]),
		UserID:      escrow.CustomerID,
		Type:        TypeRefund,
		Status:      StatusSuccess,
		Provider:    ProviderInternal,
		Amount:      escrow.Amount,
		Currency:    escrow.Currency,
		Description: fmt.Sprintf("Refund: %s", reason),
		Metadata:    map[string]interface{}{"original_transaction_id": escrow.TransactionID.String()},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.saveTransaction(ctx, refund)
	
	// Credit customer wallet
	if err := s.creditWallet(ctx, escrow.CustomerID, escrow.Amount, escrow.Currency); err != nil {
		return err
	}
	
	// Update escrow status
	_, err = s.db.Exec(ctx, 
		"UPDATE escrow_accounts SET status = $1 WHERE id = $2",
		EscrowRefunded, escrow.ID,
	)
	
	return err
}

// =============================================================================
// WALLET
// =============================================================================

// GetOrCreateWallet gets or creates a user wallet
func (s *Service) GetOrCreateWallet(ctx context.Context, userID uuid.UUID, currency string) (*Wallet, error) {
	var wallet Wallet
	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, balance, pending_balance, currency, is_active, created_at, updated_at
		FROM wallets WHERE user_id = $1 AND currency = $2
	`, userID, currency).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.PendingBalance,
		&wallet.Currency, &wallet.IsActive, &wallet.CreatedAt, &wallet.UpdatedAt,
	)
	
	if err == nil {
		return &wallet, nil
	}
	
	// Create new wallet
	wallet = Wallet{
		ID:        uuid.New(),
		UserID:    userID,
		Balance:   0,
		Currency:  currency,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	_, err = s.db.Exec(ctx, `
		INSERT INTO wallets (id, user_id, balance, pending_balance, currency, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, wallet.ID, wallet.UserID, wallet.Balance, wallet.PendingBalance,
		wallet.Currency, wallet.IsActive, wallet.CreatedAt, wallet.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	
	return &wallet, nil
}

func (s *Service) creditWallet(ctx context.Context, userID uuid.UUID, amount int64, currency string) error {
	wallet, err := s.GetOrCreateWallet(ctx, userID, currency)
	if err != nil {
		return err
	}
	
	_, err = s.db.Exec(ctx, 
		"UPDATE wallets SET balance = balance + $1, updated_at = $2 WHERE id = $3",
		amount, time.Now(), wallet.ID,
	)
	return err
}

func (s *Service) debitWallet(ctx context.Context, userID uuid.UUID, amount int64, currency string) error {
	wallet, err := s.GetOrCreateWallet(ctx, userID, currency)
	if err != nil {
		return err
	}
	
	if wallet.Balance < amount {
		return errors.New("insufficient balance")
	}
	
	_, err = s.db.Exec(ctx, 
		"UPDATE wallets SET balance = balance - $1, updated_at = $2 WHERE id = $3",
		amount, time.Now(), wallet.ID,
	)
	return err
}

// =============================================================================
// PAYOUTS
// =============================================================================

// PayoutRequest for vendor withdrawals
type PayoutRequest struct {
	VendorID    uuid.UUID `json:"vendor_id"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	BankCode    string    `json:"bank_code"`
	AccountNumber string  `json:"account_number"`
	AccountName string    `json:"account_name"`
}

// RequestPayout initiates a vendor payout
func (s *Service) RequestPayout(ctx context.Context, req PayoutRequest) (*Transaction, error) {
	// Verify wallet balance
	wallet, err := s.GetOrCreateWallet(ctx, req.VendorID, req.Currency)
	if err != nil {
		return nil, err
	}
	
	if wallet.Balance < req.Amount {
		return nil, errors.New("insufficient balance")
	}
	
	// Create payout transaction
	txn := &Transaction{
		ID:          uuid.New(),
		Reference:   fmt.Sprintf("PAY-%s", uuid.New().String()[:8]),
		UserID:      req.VendorID,
		Type:        TypePayout,
		Status:      StatusProcessing,
		Provider:    ProviderPaystack, // Default to Paystack
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: "Wallet withdrawal",
		Metadata: map[string]interface{}{
			"bank_code":      req.BankCode,
			"account_number": req.AccountNumber,
			"account_name":   req.AccountName,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Debit wallet
	if err := s.debitWallet(ctx, req.VendorID, req.Amount, req.Currency); err != nil {
		return nil, err
	}
	
	// Save transaction
	if err := s.saveTransaction(ctx, txn); err != nil {
		// Rollback wallet debit
		s.creditWallet(ctx, req.VendorID, req.Amount, req.Currency)
		return nil, err
	}
	
	// Initiate transfer with provider (async)
	go s.processPaystackTransfer(context.Background(), txn, req)
	
	return txn, nil
}

func (s *Service) processPaystackTransfer(ctx context.Context, txn *Transaction, req PayoutRequest) {
	// First create transfer recipient
	recipientPayload := map[string]interface{}{
		"type":           "nuban",
		"name":           req.AccountName,
		"account_number": req.AccountNumber,
		"bank_code":      req.BankCode,
		"currency":       req.Currency,
	}
	
	body, _ := json.Marshal(recipientPayload)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", 
		"https://api.paystack.co/transferrecipient", strings.NewReader(string(body)))
	httpReq.Header.Set("Authorization", "Bearer "+s.config.PaystackSecretKey)
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := s.http.Do(httpReq)
	if err != nil {
		txn.Status = StatusFailed
		s.saveTransaction(ctx, txn)
		// Refund wallet
		s.creditWallet(ctx, req.VendorID, req.Amount, req.Currency)
		return
	}
	defer resp.Body.Close()
	
	var recipientResult struct {
		Status bool   `json:"status"`
		Data   struct {
			RecipientCode string `json:"recipient_code"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&recipientResult)
	
	if !recipientResult.Status {
		txn.Status = StatusFailed
		s.saveTransaction(ctx, txn)
		s.creditWallet(ctx, req.VendorID, req.Amount, req.Currency)
		return
	}
	
	// Initiate transfer
	transferPayload := map[string]interface{}{
		"source":    "balance",
		"amount":    req.Amount,
		"recipient": recipientResult.Data.RecipientCode,
		"reason":    "Vendor payout",
		"reference": txn.Reference,
	}
	
	body, _ = json.Marshal(transferPayload)
	httpReq, _ = http.NewRequestWithContext(ctx, "POST", 
		"https://api.paystack.co/transfer", strings.NewReader(string(body)))
	httpReq.Header.Set("Authorization", "Bearer "+s.config.PaystackSecretKey)
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err = s.http.Do(httpReq)
	if err != nil {
		txn.Status = StatusFailed
		s.saveTransaction(ctx, txn)
		s.creditWallet(ctx, req.VendorID, req.Amount, req.Currency)
		return
	}
	defer resp.Body.Close()
	
	var transferResult struct {
		Status bool   `json:"status"`
		Data   struct {
			TransferCode string `json:"transfer_code"`
			Status       string `json:"status"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&transferResult)
	
	if transferResult.Status && transferResult.Data.Status == "success" {
		txn.Status = StatusSuccess
		now := time.Now()
		txn.PaidAt = &now
	} else {
		txn.Status = StatusProcessing // Will be updated via webhook
	}
	txn.ProviderRef = transferResult.Data.TransferCode
	s.saveTransaction(ctx, txn)
}

// =============================================================================
// WEBHOOKS
// =============================================================================

// HandlePaystackWebhook processes Paystack webhooks
func (s *Service) HandlePaystackWebhook(ctx context.Context, payload []byte, signature string) error {
	// Verify signature
	mac := hmac.New(sha512.New, []byte(s.config.PaystackSecretKey))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	
	if signature != expectedSig {
		return errors.New("invalid signature")
	}
	
	var event struct {
		Event string `json:"event"`
		Data  struct {
			Reference string `json:"reference"`
			Status    string `json:"status"`
			Amount    int64  `json:"amount"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}
	
	switch event.Event {
	case "charge.success":
		return s.handleChargeSuccess(ctx, event.Data.Reference)
	case "transfer.success":
		return s.handleTransferSuccess(ctx, event.Data.Reference)
	case "transfer.failed":
		return s.handleTransferFailed(ctx, event.Data.Reference)
	}
	
	return nil
}

func (s *Service) handleChargeSuccess(ctx context.Context, reference string) error {
	_, err := s.VerifyPaystack(ctx, reference)
	return err
}

func (s *Service) handleTransferSuccess(ctx context.Context, reference string) error {
	now := time.Now()
	_, err := s.db.Exec(ctx, 
		"UPDATE transactions SET status = $1, paid_at = $2, updated_at = $2 WHERE reference = $3",
		StatusSuccess, now, reference,
	)
	return err
}

func (s *Service) handleTransferFailed(ctx context.Context, reference string) error {
	// Get transaction
	txn, err := s.GetTransactionByReference(ctx, reference)
	if err != nil {
		return err
	}
	
	// Update status
	txn.Status = StatusFailed
	s.saveTransaction(ctx, txn)
	
	// Refund wallet
	return s.creditWallet(ctx, txn.UserID, txn.Amount, txn.Currency)
}

// =============================================================================
// HELPERS
// =============================================================================

func (s *Service) saveTransaction(ctx context.Context, txn *Transaction) error {
	metadataJSON, _ := json.Marshal(txn.Metadata)
	providerDataJSON, _ := json.Marshal(txn.ProviderData)
	
	query := `
		INSERT INTO transactions (
			id, reference, user_id, vendor_id, booking_id,
			type, status, provider, amount, currency, fee, net_amount,
			description, metadata, provider_ref, provider_data,
			paid_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			provider_ref = EXCLUDED.provider_ref,
			provider_data = EXCLUDED.provider_data,
			paid_at = EXCLUDED.paid_at,
			updated_at = EXCLUDED.updated_at
	`
	
	_, err := s.db.Exec(ctx, query,
		txn.ID, txn.Reference, txn.UserID, txn.VendorID, txn.BookingID,
		txn.Type, txn.Status, txn.Provider, txn.Amount, txn.Currency,
		txn.Fee, txn.NetAmount, txn.Description, metadataJSON,
		txn.ProviderRef, providerDataJSON, txn.PaidAt, txn.CreatedAt, txn.UpdatedAt,
	)
	return err
}

// GetTransactionByReference retrieves a transaction by reference
func (s *Service) GetTransactionByReference(ctx context.Context, reference string) (*Transaction, error) {
	var txn Transaction
	var metadataJSON, providerDataJSON []byte
	
	query := `
		SELECT id, reference, user_id, vendor_id, booking_id,
		       type, status, provider, amount, currency, fee, net_amount,
		       description, metadata, provider_ref, provider_data,
		       paid_at, created_at, updated_at
		FROM transactions WHERE reference = $1
	`
	
	err := s.db.QueryRow(ctx, query, reference).Scan(
		&txn.ID, &txn.Reference, &txn.UserID, &txn.VendorID, &txn.BookingID,
		&txn.Type, &txn.Status, &txn.Provider, &txn.Amount, &txn.Currency,
		&txn.Fee, &txn.NetAmount, &txn.Description, &metadataJSON,
		&txn.ProviderRef, &providerDataJSON, &txn.PaidAt, &txn.CreatedAt, &txn.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(metadataJSON, &txn.Metadata)
	json.Unmarshal(providerDataJSON, &txn.ProviderData)

	return &txn, nil
}

// GetTransactionByID retrieves a transaction by ID
func (s *Service) GetTransactionByID(ctx context.Context, id uuid.UUID) (*Transaction, error) {
	var txn Transaction
	var metadataJSON, providerDataJSON []byte

	query := `
		SELECT id, reference, user_id, vendor_id, booking_id,
		       type, status, provider, amount, currency, fee, net_amount,
		       description, metadata, provider_ref, provider_data,
		       paid_at, created_at, updated_at
		FROM transactions WHERE id = $1
	`

	err := s.db.QueryRow(ctx, query, id).Scan(
		&txn.ID, &txn.Reference, &txn.UserID, &txn.VendorID, &txn.BookingID,
		&txn.Type, &txn.Status, &txn.Provider, &txn.Amount, &txn.Currency,
		&txn.Fee, &txn.NetAmount, &txn.Description, &metadataJSON,
		&txn.ProviderRef, &providerDataJSON, &txn.PaidAt, &txn.CreatedAt, &txn.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(metadataJSON, &txn.Metadata)
	json.Unmarshal(providerDataJSON, &txn.ProviderData)

	return &txn, nil
}

// GetWalletTransactions retrieves transaction history for a wallet
func (s *Service) GetWalletTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Transaction, error) {
	query := `
		SELECT id, reference, user_id, vendor_id, booking_id,
		       type, status, provider, amount, currency, fee, net_amount,
		       description, metadata, provider_ref, provider_data,
		       paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*Transaction
	for rows.Next() {
		var txn Transaction
		var metadataJSON, providerDataJSON []byte

		err := rows.Scan(
			&txn.ID, &txn.Reference, &txn.UserID, &txn.VendorID, &txn.BookingID,
			&txn.Type, &txn.Status, &txn.Provider, &txn.Amount, &txn.Currency,
			&txn.Fee, &txn.NetAmount, &txn.Description, &metadataJSON,
			&txn.ProviderRef, &providerDataJSON, &txn.PaidAt, &txn.CreatedAt, &txn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(metadataJSON, &txn.Metadata)
		json.Unmarshal(providerDataJSON, &txn.ProviderData)

		transactions = append(transactions, &txn)
	}

	return transactions, nil
}

// ListPayouts retrieves payout history for a vendor
func (s *Service) ListPayouts(ctx context.Context, vendorID uuid.UUID, limit, offset int) ([]*Transaction, error) {
	query := `
		SELECT id, reference, user_id, vendor_id, booking_id,
		       type, status, provider, amount, currency, fee, net_amount,
		       description, metadata, provider_ref, provider_data,
		       paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND type = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := s.db.Query(ctx, query, vendorID, TypePayout, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payouts []*Transaction
	for rows.Next() {
		var txn Transaction
		var metadataJSON, providerDataJSON []byte

		err := rows.Scan(
			&txn.ID, &txn.Reference, &txn.UserID, &txn.VendorID, &txn.BookingID,
			&txn.Type, &txn.Status, &txn.Provider, &txn.Amount, &txn.Currency,
			&txn.Fee, &txn.NetAmount, &txn.Description, &metadataJSON,
			&txn.ProviderRef, &providerDataJSON, &txn.PaidAt, &txn.CreatedAt, &txn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(metadataJSON, &txn.Metadata)
		json.Unmarshal(providerDataJSON, &txn.ProviderData)

		payouts = append(payouts, &txn)
	}

	return payouts, nil
}

// GetPayoutByID retrieves a specific payout with ownership verification
func (s *Service) GetPayoutByID(ctx context.Context, payoutID, vendorID uuid.UUID) (*Transaction, error) {
	var txn Transaction
	var metadataJSON, providerDataJSON []byte

	query := `
		SELECT id, reference, user_id, vendor_id, booking_id,
		       type, status, provider, amount, currency, fee, net_amount,
		       description, metadata, provider_ref, provider_data,
		       paid_at, created_at, updated_at
		FROM transactions
		WHERE id = $1 AND user_id = $2 AND type = $3
	`

	err := s.db.QueryRow(ctx, query, payoutID, vendorID, TypePayout).Scan(
		&txn.ID, &txn.Reference, &txn.UserID, &txn.VendorID, &txn.BookingID,
		&txn.Type, &txn.Status, &txn.Provider, &txn.Amount, &txn.Currency,
		&txn.Fee, &txn.NetAmount, &txn.Description, &metadataJSON,
		&txn.ProviderRef, &providerDataJSON, &txn.PaidAt, &txn.CreatedAt, &txn.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(metadataJSON, &txn.Metadata)
	json.Unmarshal(providerDataJSON, &txn.ProviderData)

	return &txn, nil
}

// GetEscrowByBookingID retrieves escrow status for a booking
func (s *Service) GetEscrowByBookingID(ctx context.Context, bookingID uuid.UUID) (*EscrowAccount, error) {
	var escrow EscrowAccount

	query := `
		SELECT id, transaction_id, booking_id, customer_id, vendor_id,
		       amount, currency, status, release_condition,
		       released_at, dispute_id, expires_at, created_at
		FROM escrow_accounts
		WHERE booking_id = $1
	`

	err := s.db.QueryRow(ctx, query, bookingID).Scan(
		&escrow.ID, &escrow.TransactionID, &escrow.BookingID,
		&escrow.CustomerID, &escrow.VendorID, &escrow.Amount,
		&escrow.Currency, &escrow.Status, &escrow.ReleaseCondition,
		&escrow.ReleasedAt, &escrow.DisputeID, &escrow.ExpiresAt, &escrow.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &escrow, nil
}

// VerifyFlutterwave verifies a Flutterwave payment
func (s *Service) VerifyFlutterwave(ctx context.Context, transactionID string) (*Transaction, error) {
	httpReq, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("https://api.flutterwave.com/v3/transactions/%s/verify", transactionID), nil)
	httpReq.Header.Set("Authorization", "Bearer "+s.config.FlutterwaveSecretKey)

	resp, err := s.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			ID            int    `json:"id"`
			TxRef         string `json:"tx_ref"`
			FlwRef        string `json:"flw_ref"`
			Amount        float64 `json:"amount"`
			Currency      string `json:"currency"`
			ChargedAmount float64 `json:"charged_amount"`
			Status        string `json:"status"`
			CreatedAt     string `json:"created_at"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, errors.New(result.Message)
	}

	// Get transaction from database by reference
	txn, err := s.GetTransactionByReference(ctx, result.Data.TxRef)
	if err != nil {
		return nil, err
	}

	// Update based on provider response
	if result.Data.Status == "successful" {
		txn.Status = StatusSuccess
		createdAt, _ := time.Parse(time.RFC3339, result.Data.CreatedAt)
		txn.PaidAt = &createdAt
		txn.ProviderRef = result.Data.FlwRef
	} else {
		txn.Status = StatusFailed
	}

	txn.UpdatedAt = time.Now()
	s.saveTransaction(ctx, txn)

	// If successful and has escrow, update escrow status
	if txn.Status == StatusSuccess && txn.VendorID != nil {
		s.updateEscrowOnPayment(ctx, txn.ID)
	}

	return txn, nil
}

// HandleFlutterwaveWebhook processes Flutterwave webhook events
func (s *Service) HandleFlutterwaveWebhook(ctx context.Context, payload []byte, secretHash string) error {
	// Verify secret hash
	if secretHash != s.config.WebhookSecret {
		return errors.New("invalid secret hash")
	}

	var event struct {
		Event string `json:"event"`
		Data  struct {
			ID     int     `json:"id"`
			TxRef  string  `json:"tx_ref"`
			FlwRef string  `json:"flw_ref"`
			Amount float64 `json:"amount"`
			Status string  `json:"status"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Handle different event types
	switch event.Event {
	case "charge.completed":
		if event.Data.Status == "successful" {
			return s.handleFlutterwaveChargeSuccess(ctx, event.Data.TxRef)
		}
	}

	return nil
}

func (s *Service) handleFlutterwaveChargeSuccess(ctx context.Context, txRef string) error {
	txn, err := s.GetTransactionByReference(ctx, txRef)
	if err != nil {
		return err
	}

	// Update transaction status
	txn.Status = StatusSuccess
	now := time.Now()
	txn.PaidAt = &now
	txn.UpdatedAt = now

	if err := s.saveTransaction(ctx, txn); err != nil {
		return err
	}

	// If has escrow, update escrow status
	if txn.VendorID != nil {
		return s.updateEscrowOnPayment(ctx, txn.ID)
	}

	return nil
}

