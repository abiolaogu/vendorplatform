# Payment System Documentation

## Overview

The VendorPlatform payment system provides comprehensive payment processing, escrow management, and wallet functionality for the marketplace. It supports multiple payment providers and ensures secure transactions between customers and vendors.

## Features

### 1. Multi-Provider Support
- **Paystack** (Primary for Nigerian market)
- **Flutterwave** (Alternative provider)
- **Stripe** (International payments - future)
- **Internal Wallet** (For wallet-to-wallet transfers)

### 2. Escrow System
- Holds customer payments until service delivery
- Automatic release on booking completion
- Refund support for cancelled bookings
- Dispute management capability

### 3. Wallet Management
- Internal wallet for users and vendors
- Real-time balance tracking
- Pending balance (held in escrow)
- Multi-currency support

### 4. Vendor Payouts
- Bank transfer payouts for vendors
- Withdrawal request tracking
- Automatic processing with Paystack Transfer API
- Failed payout handling with balance refund

### 5. Webhook Integration
- Real-time payment status updates
- Signature verification for security
- Idempotent webhook processing
- Event logging for audit trails

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│      Payment API Handlers           │
│  - Initialize Payment               │
│  - Verify Payment                   │
│  - Request Payout                   │
│  - Get Wallet Info                  │
└──────────┬──────────────────────────┘
           │
           ▼
┌──────────────────────────────────────┐
│      Payment Service                 │
│  - Provider Integration              │
│  - Escrow Management                 │
│  - Wallet Operations                 │
│  - Transaction Tracking              │
└──────┬───────────────────────────────┘
       │
       ├──────────┐
       │          │
       ▼          ▼
┌──────────┐  ┌──────────┐
│ Paystack │  │PostgreSQL│
│    API   │  │    DB    │
└──────────┘  └──────────┘
```

## API Endpoints

### Payment Initialization

**POST** `/api/v1/payments/initialize`

Initialize a payment for a booking.

**Request:**
```json
{
  "booking_id": "uuid",
  "email": "customer@example.com",
  "provider": "paystack",
  "callback_url": "https://yourapp.com/payment/callback",
  "metadata": {
    "custom_field": "value"
  }
}
```

**Response:**
```json
{
  "transaction_id": "uuid",
  "reference": "VND-abc123-1234567890",
  "authorization_url": "https://checkout.paystack.com/abc123",
  "access_code": "abc123def456",
  "provider": "paystack"
}
```

### Payment Verification

**GET** `/api/v1/payments/verify/:reference`

Verify a payment transaction.

**Response:**
```json
{
  "transaction": {
    "id": "uuid",
    "reference": "VND-abc123-1234567890",
    "status": "success",
    "amount": 50000,
    "currency": "NGN",
    "paid_at": "2024-01-15T10:30:00Z"
  },
  "success": true
}
```

### Get Wallet

**GET** `/api/v1/payments/wallet?currency=NGN`

Get user's wallet information.

**Response:**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "balance": 150000,
  "pending_balance": 50000,
  "currency": "NGN",
  "is_active": true,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### Request Payout

**POST** `/api/v1/payments/payouts`

Request a payout (vendor withdrawal).

**Request:**
```json
{
  "amount": 100000,
  "currency": "NGN",
  "bank_code": "058",
  "account_number": "0123456789",
  "account_name": "John Doe"
}
```

**Response:**
```json
{
  "payout_id": "uuid",
  "reference": "PAY-xyz789",
  "status": "processing",
  "amount": 100000,
  "currency": "NGN",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Webhook Endpoints

**POST** `/api/v1/webhooks/paystack`

Receive Paystack webhook events (no authentication required, signature verified).

**POST** `/api/v1/webhooks/flutterwave`

Receive Flutterwave webhook events.

## Payment Flow

### 1. Booking Payment Flow

```
Customer Creates Booking
         │
         ▼
Initialize Payment (Customer)
         │
         ▼
Redirect to Payment Provider
         │
         ▼
Customer Completes Payment
         │
         ▼
Provider Sends Webhook
         │
         ▼
Verify Payment Status
         │
         ▼
Create Escrow Account (Hold Funds)
         │
         ▼
Booking Status: Confirmed
         │
         ▼
[Service Delivery]
         │
         ▼
Booking Status: Completed
         │
         ▼
Release Escrow to Vendor Wallet
         │
         ▼
Vendor Requests Payout
         │
         ▼
Transfer to Vendor Bank Account
```

### 2. Escrow Management

**When Payment Succeeds:**
1. Create escrow account linked to booking
2. Hold funds (net amount after platform fee)
3. Set expiry date (30 days default)
4. Update booking status to "confirmed"

**When Booking Completes:**
1. Verify booking completion
2. Release escrow funds
3. Credit vendor wallet
4. Create escrow release transaction

**When Booking Cancels:**
1. Check cancellation policy
2. Calculate refund amount
3. Release escrow
4. Credit customer wallet/refund original payment method

### 3. Vendor Payout Flow

**Initiate Payout:**
1. Vendor requests payout
2. Verify wallet balance
3. Debit wallet
4. Create payout transaction

**Process Payout:**
1. Create transfer recipient (Paystack)
2. Initiate bank transfer
3. Update transaction status
4. Handle failures (refund wallet)

## Database Schema

### Transactions Table
```sql
CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    reference VARCHAR(100) UNIQUE NOT NULL,
    user_id UUID NOT NULL,
    vendor_id UUID,
    booking_id UUID,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    fee BIGINT DEFAULT 0,
    net_amount BIGINT NOT NULL,
    description TEXT,
    metadata JSONB,
    provider_ref VARCHAR(255),
    provider_data JSONB,
    paid_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Wallets Table
```sql
CREATE TABLE wallets (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    pending_balance BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, currency)
);
```

### Escrow Accounts Table
```sql
CREATE TABLE escrow_accounts (
    id UUID PRIMARY KEY,
    transaction_id UUID NOT NULL,
    booking_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    vendor_id UUID NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    status VARCHAR(50) NOT NULL,
    release_condition VARCHAR(255),
    released_at TIMESTAMP,
    dispute_id UUID,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## Configuration

Set the following environment variables:

```bash
# Paystack Configuration
PAYSTACK_SECRET_KEY=sk_test_your_secret_key
PAYSTACK_PUBLIC_KEY=pk_test_your_public_key

# Flutterwave Configuration (optional)
FLUTTERWAVE_SECRET_KEY=FLWSECK_TEST-your_secret_key
FLUTTERWAVE_PUBLIC_KEY=FLWPUBK_TEST-your_public_key

# Payment Settings
DEFAULT_CURRENCY=NGN
PLATFORM_FEE_PERCENT=5.0
ESCROW_EXPIRY_DAYS=30
```

## Security Considerations

### 1. Webhook Security
- **Signature Verification**: All webhooks are verified using HMAC-SHA512
- **IP Whitelisting**: Configure firewall to only accept webhooks from provider IPs
- **Idempotency**: Duplicate webhook events are ignored

### 2. Transaction Security
- All amounts stored in smallest unit (kobo/cents) to avoid floating-point errors
- Transaction references are unique and unguessable
- All database operations are wrapped in transactions for atomicity

### 3. API Security
- Payment endpoints require authentication (JWT token)
- User can only access their own transactions and wallet
- Vendor-only endpoints restricted by role check
- Webhook endpoints are public but signature-verified

## Testing

### Test Payment Flow

1. **Initialize Payment:**
```bash
curl -X POST http://localhost:8080/api/v1/payments/initialize \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user-uuid" \
  -d '{
    "booking_id": "booking-uuid",
    "email": "test@example.com",
    "provider": "paystack"
  }'
```

2. **Verify Payment:**
```bash
curl http://localhost:8080/api/v1/payments/verify/VND-abc123-1234567890 \
  -H "X-User-ID: user-uuid"
```

3. **Get Wallet:**
```bash
curl http://localhost:8080/api/v1/payments/wallet \
  -H "X-User-ID: user-uuid"
```

### Test Webhook

```bash
# Simulate Paystack webhook
curl -X POST http://localhost:8080/api/v1/webhooks/paystack \
  -H "Content-Type: application/json" \
  -H "X-Paystack-Signature: computed-signature" \
  -d '{
    "event": "charge.success",
    "data": {
      "reference": "VND-abc123-1234567890",
      "status": "success",
      "amount": 5000000
    }
  }'
```

## Integration with Booking System

### On Booking Creation
```go
// Customer creates booking
booking := createBooking(...)

// Initialize payment
payment := paymentService.InitializePayment(ctx, InitializePaymentRequest{
    UserID:      booking.UserID,
    VendorID:    &booking.VendorID,
    BookingID:   &booking.ID,
    Amount:      booking.TotalAmount * 100, // Convert to kobo
    Currency:    "NGN",
    Email:       customer.Email,
    Provider:    ProviderPaystack,
    UseEscrow:   true,
})

// Redirect customer to payment.AuthorizationURL
```

### On Booking Completion
```go
// Vendor marks service as completed
bookingService.CompleteBooking(ctx, bookingID)

// Release escrow to vendor
paymentService.ReleaseEscrow(ctx, bookingID)
```

### On Booking Cancellation
```go
// Customer/Vendor cancels booking
bookingService.CancelBooking(ctx, bookingID, reason)

// Refund escrow to customer
paymentService.RefundEscrow(ctx, bookingID, reason)
```

## Monitoring & Observability

### Key Metrics to Track

1. **Payment Metrics:**
   - Payment success rate
   - Average transaction value
   - Payment processing time
   - Failed payment reasons

2. **Escrow Metrics:**
   - Average escrow hold time
   - Escrow release rate
   - Disputed escrow count
   - Expired escrow count

3. **Payout Metrics:**
   - Payout success rate
   - Average payout amount
   - Payout processing time
   - Failed payout rate

### Logging

All payment operations are logged with structured logging:
```go
logger.Info("Payment initialized",
    zap.String("user_id", userID.String()),
    zap.String("booking_id", bookingID.String()),
    zap.Int64("amount", amount),
    zap.String("provider", provider),
)
```

## Troubleshooting

### Common Issues

**1. Payment fails but customer was charged**
- Check transaction status in database
- Verify with payment provider API
- If charge confirmed, manually update transaction status
- Release escrow if payment successful

**2. Webhook not received**
- Check webhook URL configuration in provider dashboard
- Verify webhook endpoint is accessible from internet
- Check webhook event logs in provider dashboard
- Manually verify payment if needed

**3. Escrow not released**
- Check booking completion status
- Verify escrow account status
- Check for disputes or holds
- Manually release if booking is confirmed complete

**4. Payout fails**
- Check vendor wallet balance
- Verify bank account details
- Check Paystack transfer logs
- Balance is automatically refunded on failure

## Future Enhancements

1. **Subscription Payments**
   - Recurring billing for subscription services
   - Auto-renewal with saved payment methods

2. **Installment Payments**
   - Split payments for high-value bookings
   - Payment plans with scheduled charges

3. **Multi-Currency Support**
   - International payments with Stripe
   - Automatic currency conversion
   - Multi-currency wallets

4. **Advanced Dispute Resolution**
   - In-app dispute management
   - Evidence submission system
   - Automated resolution workflow

5. **Payment Analytics Dashboard**
   - Real-time payment metrics
   - Revenue forecasting
   - Vendor payout schedules

## Support

For issues or questions about the payment system:
- Check the [troubleshooting guide](#troubleshooting)
- Review payment logs in the database
- Contact payment provider support for provider-specific issues
- Raise an issue in the GitHub repository
