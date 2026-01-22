# VendorPlatform - Contextual Commerce Orchestration

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Python Version](https://img.shields.io/badge/Python-3.11+-3776AB?style=flat&logo=python)](https://python.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql)](https://postgresql.org)
[![License](https://img.shields.io/badge/License-Proprietary-red.svg)](LICENSE)

> **"When someone needs one service, they typically need 5-15 related services."**

VendorPlatform is a multi-product platform that captures entire transaction value chains through contextual commerce orchestration. Instead of discrete vendor discovery, we predict adjacent needs, pre-qualify vendors, and reduce coordination friction.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     SHARED DATA LAYER                           │
├─────────────────────────────────────────────────────────────────┤
│  Vendors  │  Services  │  Categories  │  Adjacencies  │  Users  │
├─────────────────────────────────────────────────────────────────┤
│                   RECOMMENDATION ENGINE                         │
│  Adjacent Services │ Collaborative Filtering │ Event-Based     │
├─────────────────────────────────────────────────────────────────┤
│                    BOOKING & PAYMENT                            │
│  Reservations │ Payments │ Escrow │ Invoicing │ Refunds        │
└─────────────────────────────────────────────────────────────────┘
          ↑              ↑              ↑              ↑
    ┌─────┴─────┐  ┌─────┴─────┐  ┌─────┴─────┐  ┌─────┴─────┐
    │  LifeOS   │  │ EventGPT  │  │ VendorNet │  │HomeRescue │
    └───────────┘  └───────────┘  └───────────┘  └───────────┘
```

## 📦 Platform Products

### 1. LifeOS - Intelligent Life Event Orchestration
**"Your Life's Operating System"**

When life events happen (weddings, relocations, childbirth), LifeOS detects them from behavioral signals and orchestrates the entire service cascade.

- 🔍 **Event Detection**: AI detects life events before explicit user action
- 📋 **Predictive Planning**: Complete service plans with timelines & budgets
- 🎯 **Full Orchestration**: Manages entire vendor coordination
- 💰 **Smart Bundling**: Bundle opportunities for savings

### 2. EventGPT - Conversational AI Event Planner
**"Plan your perfect event through conversation"**

Natural language interface that understands intent, asks clarifying questions, generates plans, and coordinates everything through chat.

- 💬 **Natural Language**: No forms, just conversation
- 🧠 **Contextual Memory**: Remembers preferences across planning
- ⚡ **Real-Time Matching**: Instant vendor recommendations
- 🎨 **Rich Responses**: Cards, comparisons, quick replies

### 3. VendorNet - B2B Partnership Network
**"Grow together. Earn together."**

Professional network for vendors to discover partners, share referrals, and collaborate on projects.

- 🤝 **Partnership Matching**: AI-powered complementary business matching
- 📊 **Referral Tracking**: Automatic tracking & fee calculation
- 💸 **Revenue Sharing**: Built-in payment splitting
- 🏆 **Collaborative Bidding**: Team up for large projects

### 4. HomeRescue - Emergency Home Services
**"Help arrives in minutes, not hours."**

Emergency response system connecting homeowners with verified professionals for immediate response.

- ⚡ **Real-Time Availability**: See who's available NOW
- 🎯 **Guaranteed Response**: SLA with refund if missed
- 📍 **Live Tracking**: Know exactly when help arrives
- 📄 **Instant Documentation**: Photos, receipts for insurance

## 🗂️ Project Structure

```
vendorplatform/
├── api/
│   ├── lifeos/           # Life event orchestration platform
│   │   └── platform.go   # Core LifeOS implementation (~1,800 lines)
│   ├── eventgpt/         # Conversational AI planner
│   │   └── platform.go   # EventGPT implementation (~1,600 lines)
│   ├── vendornet/        # B2B partnership network
│   │   └── platform.go   # VendorNet implementation (~1,400 lines)
│   ├── homerescue/       # Emergency services platform
│   │   └── platform.go   # HomeRescue implementation (~1,500 lines)
│   ├── server.go         # Main API server
│   └── handlers.go       # Shared API handlers
├── recommendation-engine/
│   ├── engine.go         # Core recommendation engine (Go)
│   ├── ml_service.py     # ML models for predictions (Python)
│   └── api/              # Recommendation API handlers
├── database/
│   ├── 001_core_schema.sql    # PostgreSQL schema with TimescaleDB, PostGIS
│   └── 002_seed_data.sql      # Seed data for 15 service clusters
├── docs/
│   ├── PLATFORM_CONCEPTS_SUMMARY.md  # Executive summary
│   ├── cluster_deep_dive_part1.md    # Service clusters 1-8
│   ├── cluster_deep_dive_part2.md    # Service clusters 9-15
│   └── Vendor_Platform_Strategy_Document.docx
├── business-models/
│   └── business_model_canvases.md    # Business model documentation
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 15+ with extensions:
  - TimescaleDB
  - PostGIS
  - pg_trgm
  - ltree
- Redis 7+
- Python 3.11+ (for ML services)

### Installation

```bash
# Clone the repository
git clone https://github.com/BillyRonksGlobal/vendorplatform.git
cd vendorplatform

# Install Go dependencies
go mod download

# Install Python dependencies
pip install -r requirements.txt

# Setup database
createdb vendorplatform
psql vendorplatform < database/001_core_schema.sql
psql vendorplatform < database/002_seed_data.sql

# Run the server
make run
```

### Configuration

Create a `.env` file:

```env
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/vendorplatform
REDIS_URL=redis://localhost:6379

# API
PORT=8080
ENV=development

# Services
NOTIFICATION_SERVICE_URL=http://localhost:8081
PAYMENT_SERVICE_URL=http://localhost:8082
```

## 🛠️ Development

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test ./api/lifeos/... -v
```

### Code Generation

```bash
# Generate mocks
make generate

# Generate API documentation
make docs
```

### Linting

```bash
# Run linters
make lint

# Auto-fix issues
make lint-fix
```

## 📊 Database Schema

### Core Tables

| Table | Description |
|-------|-------------|
| `users` | Platform users (customers, vendors, admins) |
| `vendors` | Vendor profiles with verification status |
| `service_categories` | Hierarchical category tree (LTREE) |
| `services` | Individual services offered by vendors |
| `service_adjacencies` | Service recommendation graph |
| `life_event_triggers` | Detectable life events |
| `bookings` | Service bookings and reservations |
| `user_interactions` | User activity (TimescaleDB hypertable) |

### Key Features

- **LTREE** for hierarchical categories (e.g., `events.weddings.photography`)
- **PostGIS** for geospatial queries (location-based search)
- **TimescaleDB** for time-series interaction data
- **JSONB** for flexible metadata storage

## 🔌 API Reference

### LifeOS Endpoints

```
POST   /api/v1/lifeos/events              # Create event
GET    /api/v1/lifeos/events/:id          # Get event
GET    /api/v1/lifeos/events/:id/plan     # Get orchestration plan
POST   /api/v1/lifeos/events/:id/confirm  # Confirm detected event
GET    /api/v1/lifeos/detected            # Get detected events
```

### EventGPT Endpoints

```
POST   /api/v1/eventgpt/conversations         # Start conversation
POST   /api/v1/eventgpt/conversations/:id/messages  # Send message
GET    /api/v1/eventgpt/conversations/:id     # Get conversation
DELETE /api/v1/eventgpt/conversations/:id     # End conversation
```

### VendorNet Endpoints

```
GET    /api/v1/vendornet/partners/matches     # Get partner recommendations
POST   /api/v1/vendornet/partnerships         # Create partnership
POST   /api/v1/vendornet/referrals           # Create referral
PUT    /api/v1/vendornet/referrals/:id/status # Update referral status
GET    /api/v1/vendornet/analytics           # Get network analytics
```

### HomeRescue Endpoints

```
POST   /api/v1/homerescue/emergencies        # Create emergency request
GET    /api/v1/homerescue/emergencies/:id    # Get emergency status
GET    /api/v1/homerescue/emergencies/:id/tracking  # Real-time tracking
POST   /api/v1/homerescue/technicians/location      # Update tech location
PUT    /api/v1/homerescue/emergencies/:id/accept    # Tech accepts request
```

## 💰 Business Model

### Revenue Streams

| Stream | LifeOS | EventGPT | VendorNet | HomeRescue |
|--------|--------|----------|-----------|------------|
| Transaction Fees | 8-15% | 10-12% | 2.5% | 15-20% |
| Subscriptions | ✅ | ✅ | ✅ | ✅ |
| Premium Features | ✅ | ✅ | ✅ | - |
| Insurance/Partners | - | - | - | ✅ |

### Target Markets

- **Primary**: Nigeria (Lagos, Abuja, Port Harcourt)
- **Expansion**: West Africa, then Pan-African
- **Long-term**: Global emerging markets

## 🗺️ Roadmap

### Phase 1: Foundation (Q1-Q2)
- [x] Core database schema
- [x] Recommendation engine
- [x] Platform specifications
- [ ] HomeRescue MVP launch
- [ ] Basic vendor onboarding

### Phase 2: Growth (Q3-Q4)
- [ ] EventGPT conversational interface
- [ ] VendorNet referral system
- [ ] Mobile apps (iOS/Android)
- [ ] Payment integration

### Phase 3: Intelligence (Year 2)
- [ ] LifeOS event detection
- [ ] ML-powered recommendations
- [ ] Advanced analytics
- [ ] Partner API

### Phase 4: Scale (Year 2+)
- [ ] Geographic expansion
- [ ] Enterprise features
- [ ] White-label solutions
- [ ] International markets

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

Proprietary - BillyRonks Global Limited. All rights reserved.

## 📞 Contact

- **Company**: BillyRonks Global Limited
- **CEO**: Abiola Ogunsakin
- **Email**: [contact@billyronks.com](mailto:abiolaog@billyronks.net)
- **Website**: [https://billyronks.com](https://billyronks.net)

---

Built with ❤️ by BillyRonks Global Limited
