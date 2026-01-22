# VendorPlatform

> **Contextual Commerce Orchestration Platform** - When life happens, we handle it.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Python Version](https://img.shields.io/badge/Python-3.11+-3776AB?style=flat&logo=python)](https://python.org/)
[![Flutter](https://img.shields.io/badge/Flutter-3.x-02569B?style=flat&logo=flutter)](https://flutter.dev/)
[![License](https://img.shields.io/badge/License-Proprietary-red)](LICENSE)

---

## 🎯 Vision

**VendorPlatform** is a comprehensive marketplace that recognizes a fundamental truth: when someone needs one service, they typically need 5-15 related services. We become the orchestration layer that predicts adjacent needs, pre-qualifies vendors, reduces coordination friction, and captures the entire transaction value chain across **15 service clusters**.

---

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              VENDORPLATFORM                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│  FRONTEND CLIENTS                                                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ Mobile App  │  │  Web Client │  │Admin Panel  │  │ Vendor App  │         │
│  │  (Flutter)  │  │   (React)   │  │   (React)   │  │  (Flutter)  │         │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘         │
├─────────┴────────────────┴────────────────┴────────────────┴────────────────┤
│  API GATEWAY (Gin/Go)                                                        │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │  Auth │ Rate Limiting │ Request Logging │ CORS │ Metrics │ Tracing  │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────────────────┤
│  PLATFORM PRODUCTS                                                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   LifeOS    │  │  EventGPT   │  │ VendorNet   │  │ HomeRescue  │         │
│  │  Life Event │  │   AI Chat   │  │  B2B Network│  │  Emergency  │         │
│  │Orchestration│  │   Planner   │  │  Referrals  │  │  Response   │         │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘         │
├─────────┴────────────────┴────────────────┴────────────────┴────────────────┤
│  CORE SERVICES                                                               │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐  │
│  │   Auth    │  │  Payment  │  │Notification│  │  Search   │  │  Storage  │  │
│  │   JWT     │  │ Paystack  │  │Push/Email │  │Elasticsearch│ │    S3    │  │
│  │  RBAC     │  │Flutterwave│  │   SMS     │  │Full-text  │  │   CDN    │  │
│  └───────────┘  └───────────┘  └───────────┘  └───────────┘  └───────────┘  │
├─────────────────────────────────────────────────────────────────────────────┤
│  RECOMMENDATION ENGINE                                                       │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │  Adjacency Graph │ Collaborative Filtering │ Event Detection │ ML  │     │
│  └────────────────────────────────────────────────────────────────────┘     │
├─────────────────────────────────────────────────────────────────────────────┤
│  DATA LAYER                                                                  │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐                 │
│  │PostgreSQL │  │   Redis   │  │Elasticsearch│ │ TimescaleDB│                │
│  │ + PostGIS │  │Cache/Queue│  │   Search   │  │Time-series │                │
│  └───────────┘  └───────────┘  └───────────┘  └───────────┘                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 📦 Platform Products

### 1. LifeOS - Intelligent Life Event Orchestration
**"When life happens, LifeOS handles it."**

Detects and orchestrates major life events (weddings, relocations, childbirth, renovations) by:
- Analyzing behavioral signals to detect life events
- Creating comprehensive orchestration plans with 7 phases
- Coordinating multiple vendors automatically
- Managing budgets and timelines

📁 `api/lifeos/platform.go` (~1,800 lines)

---

### 2. EventGPT - Conversational AI Event Planner
**"Plan your perfect event through conversation."**

Natural language interface for event planning with:
- Intent classification (13 intents)
- Entity extraction (date, budget, location, event type)
- Dialog state management
- Rich responses (cards, quick replies, actions)

📁 `api/eventgpt/platform.go` (~1,600 lines)

---

### 3. VendorNet - B2B Partnership Network
**"Grow together. Earn together."**

Connects vendors for mutual benefit through:
- Partnership matching with multi-factor scoring
- Referral tracking with full lifecycle management
- Collaborative bidding on large projects
- Revenue sharing and fee management

📁 `api/vendornet/platform.go` (~1,400 lines)

---

### 4. HomeRescue - Emergency Home Services
**"Help arrives in minutes, not hours."**

Uber-like emergency dispatch for home crises with SLA guarantees:

| Urgency | Response Time | Refund if Missed |
|---------|--------------|-----------------|
| Critical | < 30 min | 100% |
| Urgent | < 2 hours | 50% |
| Same-Day | < 6 hours | 25% discount |

📁 `api/homerescue/platform.go` (~1,500 lines)

---

## 🛠️ Core Services

| Service | Description | Location |
|---------|-------------|----------|
| **Auth** | JWT authentication, RBAC, sessions, verification | `internal/auth/` |
| **Payment** | Paystack, Flutterwave, escrow, wallets, payouts | `internal/payment/` |
| **Notification** | Push, Email, SMS, In-App with preferences | `internal/notification/` |
| **Search** | Elasticsearch full-text, geo, facets, autocomplete | `internal/search/` |
| **Storage** | S3-compatible file storage with CDN | `internal/storage/` |
| **Worker** | Background jobs, cron, retries, monitoring | `internal/worker/` |

---

## 🧠 Recommendation Engine

Production Go implementation with:
- **Adjacency Graph**: Service relationships with affinity scores
- **Collaborative Filtering**: User-based recommendations
- **Event Detection**: Life event pattern matching
- **Multi-factor Scoring**: Adjacency (35%), Collaborative (25%), Trending (15%), Personalization (20%), Location (5%)
- **MMR Diversification**: Prevents homogeneous results

📁 `recommendation-engine/engine.go` (~2,000 lines)

---

## 📊 Service Clusters (15 Categories)

| # | Cluster | Example Services |
|---|---------|-----------------|
| 1 | **Celebrations** | Weddings, birthdays, corporate events |
| 2 | **Home Services** | Cleaning, repairs, renovations |
| 3 | **Travel** | Hotels, flights, car rentals |
| 4 | **HORECA** | Catering, restaurants, hospitality |
| 5 | **Fashion** | Tailoring, styling, accessories |
| 6 | **Business** | Legal, accounting, consulting |
| 7 | **Education** | Tutoring, training, certifications |
| 8 | **Health** | Medical, wellness, fitness |
| 9 | **Automotive** | Repairs, rentals, sales |
| 10 | **Creative** | Photography, video, design |
| 11 | **Agriculture** | Farming, equipment, processing |
| 12 | **Pets** | Veterinary, grooming, supplies |
| 13 | **Construction** | Building, architecture, engineering |
| 14 | **Energy** | Solar, generators, electrical |
| 15 | **Security** | Guards, CCTV, cyber security |

---

## 🚀 Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone repository
git clone https://github.com/BillyRonksGlobal/vendorplatform.git
cd vendorplatform

# Start all services
docker-compose up -d

# With development tools (Adminer, Mailhog)
docker-compose --profile dev up -d

# View logs
docker-compose logs -f api
```

**Services Started:**
- API Server: `http://localhost:8080`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`
- Elasticsearch: `localhost:9200`
- Grafana: `http://localhost:3000`
- Prometheus: `http://localhost:9090`

### Manual Setup

```bash
# 1. Install dependencies
go mod download

# 2. Set up database
psql $DATABASE_URL -f database/001_core_schema.sql
psql $DATABASE_URL -f database/002_seed_data.sql
psql $DATABASE_URL -f database/003_services_schema.sql

# 3. Configure environment
cp .env.example .env

# 4. Run server
make run
```

---

## 📁 Project Structure

```
vendorplatform/
├── api/                          # Platform products (4 products)
│   ├── lifeos/platform.go
│   ├── eventgpt/platform.go
│   ├── vendornet/platform.go
│   ├── homerescue/platform.go
│   ├── server.go
│   └── handlers.go
├── cmd/server/main.go            # Entry point
├── internal/                     # Core services (6 services)
│   ├── auth/service.go
│   ├── payment/service.go
│   ├── notification/service.go
│   ├── search/service.go
│   ├── storage/service.go
│   └── worker/service.go
├── pkg/                          # Shared utilities
│   ├── config/
│   ├── logger/
│   └── middleware/
├── database/                     # SQL schemas (3 files)
│   ├── 001_core_schema.sql
│   ├── 002_seed_data.sql
│   └── 003_services_schema.sql
├── recommendation-engine/        # ML recommendations
│   ├── engine.go
│   └── ml_service.py
├── deployments/
│   ├── docker/Dockerfile
│   └── terraform/main.tf
├── mobile/flutter/               # Mobile app scaffold
├── web/admin/                    # Admin dashboard scaffold
├── monitoring/prometheus.yml     # Observability
├── docs/                         # Documentation
├── business-models/              # Business canvases
├── tests/                        # Test suites
├── docker-compose.yml
├── Makefile
├── go.mod
└── requirements.txt
```

---

## 📈 Business Model Summary

| Revenue Stream | LifeOS | EventGPT | VendorNet | HomeRescue |
|----------------|--------|----------|-----------|------------|
| Transaction Fee | 8-15% | 8-12% | 2.5-3% | 15-20% |
| Vendor Subscription | ₦10-30K/mo | ₦15-50K/mo | ₦15-50K/mo | ₦20-50K/mo |
| Consumer Subscription | ₦5-12K/mo | $10-30/mo | - | ₦5-10K/mo |

---

## 🔧 Make Commands

```bash
make build          # Build the binary
make run            # Run the server
make test           # Run tests
make lint           # Run linter
make docker-build   # Build Docker image
make db-migrate     # Run migrations
make db-seed        # Seed database
make clean          # Clean build artifacts
```

---

## 📄 License

Proprietary - © 2025 BillyRonks Global Limited. All rights reserved.

---

## 📞 Contact

- **Website:** vendorplatform.com
- **Email:** support@vendorplatform.com
