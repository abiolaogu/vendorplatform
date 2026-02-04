# Product Requirements Document: VendorPlatform

**Version:** 1.0
**Last Updated:** 2026-01-29
**Status:** Active Development
**Owner:** BillyRonks Global Limited

---

## Executive Summary

**VendorPlatform** is a contextual commerce orchestration platform that recognizes a fundamental truth: when someone needs one service, they typically need 5-15 related services. We become the orchestration layer that predicts adjacent needs, pre-qualifies vendors, reduces coordination friction, and captures the entire transaction value chain across 15 service clusters.

### Vision Statement
"When life happens, we handle it" - Transform how people discover, coordinate, and manage service vendors across major life events and daily needs.

### Business Objectives
- Capture 10-15% of multi-vendor transaction value chains instead of single 3-5% commissions
- Reduce service discovery friction from 2-3 weeks to 2-3 hours
- Enable vendor ecosystem growth through B2B partnerships and referrals
- Provide emergency response infrastructure with guaranteed SLAs

---

## Product Architecture

### Platform Products (4 Strategic Offerings)

#### 1. LifeOS - Intelligent Life Event Orchestration
**Tagline:** "Your Life's Operating System"

**Description:** Detects and orchestrates major life events (weddings, relocations, childbirth, renovations) by analyzing behavioral signals, creating comprehensive orchestration plans, and managing multi-vendor coordination.

**Core Features:**
- Event Detection Engine (analyzes search patterns, browse behavior, high-intent actions)
- Orchestration Engine (generates phase-based timelines, budgets, vendor assignments)
- Risk Assessment (identifies timeline, budget, and vendor risks)
- Smart Bundling (identifies bundle opportunities for savings)

**Target Events:** Weddings, relocations, home renovations, childbirth, business launches, graduations, retirements

**Revenue Model:**
- Transaction fees: 8-15%
- Vendor subscriptions: ₦10,000-30,000/month
- Consumer subscriptions: ₦5,000-12,000/month
- Financing and data insights

---

#### 2. EventGPT - Conversational AI Event Planner
**Tagline:** "Plan your perfect event through conversation"

**Description:** Natural language interface for event planning that transforms complex multi-step processes into conversational interactions.

**Core Features:**
- Natural Language Understanding (intent classification: 15+ intents)
- Entity Extraction (date, budget, location, event type, vendor type)
- Contextual Memory (remembers preferences across planning journey)
- Multi-Modal Responses (text, cards, comparisons, quick replies)
- Real-Time Vendor Matching

**Technical Capabilities:**
- Intent: create_event, find_vendor, get_quote, book_service, compare_options, check_availability
- Slot Filling: Progressive information gathering across conversation turns
- Dialog State Management

**Revenue Model:**
- Free tier: Basic vendor search, 3 comparisons/day
- Premium (₦3,500/mo): Unlimited comparisons, price negotiation, alerts
- Pro (₦10,000/mo): Concierge, multi-event, team collaboration, API access

---

#### 3. VendorNet - B2B Partnership Network
**Tagline:** "Grow together. Earn together"

**Description:** Connects vendors for mutual benefit through partnership matching, referral tracking, collaborative bidding, and revenue sharing.

**Core Features:**
- Partnership Matching (AI-powered suggestions based on complementarity)
- Automatic Referral Tracking (full lifecycle management)
- Revenue Sharing Infrastructure (built-in payment splitting)
- Collaborative Bidding (multiple vendors bid together on large projects)
- Network Analytics (connection stats, referral metrics)

**Partnership Types:**
- Referral: Simple referral exchange
- Preferred: Preferred partner status
- Exclusive: Exclusive in category
- Joint Venture: Joint business offering
- White Label: Resell services

**Revenue Model:**
- Subscriptions: Free / Professional (₦15,000/mo) / Business (₦50,000/mo)
- Transaction fees: 2.5% on referral payments
- Collaborative bids: 3% of won contracts

---

#### 4. HomeRescue - Emergency Home Services
**Tagline:** "Help arrives in minutes, not hours"

**Description:** Emergency response system for home crises with real-time dispatch, live tracking, and guaranteed response times.

**Core Features:**
- Real-Time Availability (see who's available NOW)
- Emergency-First Design (optimized for speed)
- Guaranteed Response Time (SLA with refund if missed)
- Live Tracking (GPS updates, ETA calculation)
- Dynamic Pricing (urgency premiums, after-hours rates)

**Response Time SLAs:**
- Critical (< 30 min): 100% refund if missed
- Urgent (< 2 hours): 50% refund if missed
- Same-Day (< 6 hours): 25% discount if missed

**Emergency Categories:** Plumbing, Electrical, Locksmith, HVAC, Glass, Roofing, Pest Control, Security

**Revenue Model:**
- Service fees: 15-20%
- Customer subscriptions: ₦5,000-10,000/month
- Technician subscriptions: ₦20,000-50,000/month
- Insurance partnerships

---

## Technical Architecture

### Technology Stack

**Backend:**
- Language: Go 1.21+
- Framework: Gin (HTTP routing and middleware)
- Database: PostgreSQL + PostGIS (geospatial)
- Cache: Redis (caching, queues, sessions)
- Search: Elasticsearch (full-text, geospatial, faceted search)
- Time-series: TimescaleDB (analytics, metrics)

**Frontend:**
- Mobile: Flutter 3.x (iOS/Android)
- Web: React (consumer web, admin panel)
- State Management: Context API / Redux

**AI/ML:**
- Recommendation Engine: Go (production) + Python (ML training)
- NLP: Claude API (EventGPT conversation)
- Event Detection: Pattern matching + ML models

**Infrastructure:**
- Containerization: Docker + Docker Compose
- Orchestration: Kubernetes (production)
- Monitoring: Prometheus + Grafana
- Payments: Paystack, Flutterwave

### Core Services (6 Microservices)

| Service | Purpose | Technology |
|---------|---------|------------|
| **Auth** | JWT authentication, RBAC, sessions, verification | Go, Redis |
| **Payment** | Payment processing, escrow, wallets, payouts | Go, Paystack, Flutterwave |
| **Notification** | Push, Email, SMS, In-App notifications | Go, FCM, SendGrid, Twilio |
| **Search** | Full-text search, geospatial queries, autocomplete | Elasticsearch |
| **Storage** | File storage, CDN, image processing | S3-compatible, CloudFront |
| **Worker** | Background jobs, cron tasks, retries | Go, Redis queues |

### Recommendation Engine

**Algorithms:**
- Adjacency Graph: Service relationships with affinity scores (35% weight)
- Collaborative Filtering: User-based recommendations (25% weight)
- Trending Services: Time-based popularity (15% weight)
- Personalization: User preferences and history (20% weight)
- Location: Geographic proximity (5% weight)

**Features:**
- MMR Diversification (prevents homogeneous results)
- Event Detection (pattern matching for life events)
- Real-time scoring and ranking

---

## Service Clusters (15 Categories)

| # | Cluster | Example Services |
|---|---------|------------------|
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

## User Roles & Permissions

### Consumer Roles
- **Guest**: Browse services, view vendors (read-only)
- **Customer**: Book services, make payments, leave reviews
- **Premium Customer**: Priority support, exclusive deals, advanced features

### Vendor Roles
- **Vendor Basic**: Profile, availability, receive bookings
- **Vendor Professional**: Analytics, marketing tools, priority placement
- **Vendor Business**: Multi-location, team management, API access

### Platform Roles
- **Admin**: Full platform access, user management, system configuration
- **Support**: Customer support, dispute resolution, vendor verification
- **Analyst**: Read-only access to analytics, reports, business intelligence

---

## Key User Flows

### 1. LifeOS Event Detection Flow
```
User searches "wedding venues Lagos" multiple times
  ↓
LifeOS detects life event signal (wedding)
  ↓
System generates orchestration plan (venue, catering, photography, etc.)
  ↓
User receives notification: "Planning a wedding? We've created a complete plan"
  ↓
User reviews plan, adjusts budget/timeline
  ↓
System recommends vendors for each service
  ↓
User books multiple services with bundled discount
```

### 2. EventGPT Conversation Flow
```
User: "I'm planning a wedding for 200 guests in Lagos next December"
  ↓
Intent: create_event | Entities: [wedding, 200, Lagos, December]
  ↓
EventGPT: "Great! I've noted your wedding for ~December in Lagos with 200 guests.
          What's your approximate budget?"
  ↓
User: "Around 5 million naira"
  ↓
EventGPT: "Perfect! Here are the top services you'll need: [Venue, Catering, Photography]
          Should we start with venue recommendations?"
  ↓
[Continues conversation until booking]
```

### 3. VendorNet Referral Flow
```
Photographer (Vendor A) completes wedding shoot
  ↓
Client needs catering for next event
  ↓
Vendor A creates referral → Caterer (Vendor B)
  ↓
System tracks: Pending → Accepted → Contacted → Quoted
  ↓
Client books Vendor B for ₦500,000
  ↓
Referral status: Converted
  ↓
System calculates fee: 10% × ₦500,000 = ₦50,000
  ↓
Vendor A receives ₦50,000 referral payment
```

### 4. HomeRescue Emergency Dispatch
```
User: Burst pipe at 2 AM
  ↓
User opens app → Selects "Plumbing" → "Critical Emergency"
  ↓
System finds nearest 5 available plumbers
  ↓
Dispatch to closest plumber (2.3km away, 4.8 rating)
  ↓
Plumber accepts within 30 seconds
  ↓
User sees live tracking: ETA 18 minutes
  ↓
Plumber arrives in 17 minutes (SLA met)
  ↓
Service completed, user charged ₦15,000 + ₦5,000 emergency fee
```

---

## Data Models (Core Entities)

### Vendors
```go
type Vendor struct {
    ID              uuid.UUID
    Name            string
    Description     string
    Categories      []string  // service categories
    Location        GeoPoint
    Rating          float64
    TotalBookings   int
    ResponseTime    time.Duration
    Availability    map[string]bool  // day: available
    Verified        bool
    Subscriptions   []string  // active subscriptions
}
```

### Services
```go
type Service struct {
    ID          uuid.UUID
    VendorID    uuid.UUID
    Name        string
    Category    string
    Price       PriceRange
    Duration    time.Duration
    Availability Schedule
    MaxBookings int
}
```

### Bookings
```go
type Booking struct {
    ID          uuid.UUID
    CustomerID  uuid.UUID
    VendorID    uuid.UUID
    ServiceID   uuid.UUID
    Status      string  // pending, confirmed, completed, cancelled
    Date        time.Time
    TotalAmount decimal.Decimal
    PaymentStatus string
    Reviews     []Review
}
```

### Events (LifeOS)
```go
type LifeEvent struct {
    ID              uuid.UUID
    CustomerID      uuid.UUID
    Type            string  // wedding, relocation, renovation, etc.
    DetectedAt      time.Time
    Phases          []Phase
    TotalBudget     decimal.Decimal
    Timeline        Timeline
    Services        []ServicePlan
    Status          string  // planning, in_progress, completed
}
```

---

## API Endpoints (High-Level)

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/verify` - Verify email/phone
- `POST /api/v1/auth/refresh` - Refresh JWT token

### Services
- `GET /api/v1/services` - List services (filterable, paginated)
- `GET /api/v1/services/:id` - Get service details
- `POST /api/v1/services` - Create service (vendor only)
- `GET /api/v1/services/search` - Search services

### Bookings
- `POST /api/v1/bookings` - Create booking
- `GET /api/v1/bookings/:id` - Get booking details
- `PATCH /api/v1/bookings/:id/status` - Update booking status
- `GET /api/v1/bookings/customer/:id` - Customer's bookings

### LifeOS
- `GET /api/v1/lifeos/events` - List detected events
- `POST /api/v1/lifeos/events/:id/plan` - Generate orchestration plan
- `GET /api/v1/lifeos/recommendations` - Get service recommendations

### EventGPT
- `POST /api/v1/eventgpt/message` - Send message to EventGPT
- `GET /api/v1/eventgpt/conversations/:id` - Get conversation history

### VendorNet
- `POST /api/v1/vendornet/partnerships` - Create partnership
- `POST /api/v1/vendornet/referrals` - Create referral
- `GET /api/v1/vendornet/analytics` - Get network analytics

### HomeRescue
- `POST /api/v1/homerescue/emergency` - Create emergency request
- `GET /api/v1/homerescue/track/:id` - Track emergency response
- `GET /api/v1/homerescue/available` - Get available techs

---

## Non-Functional Requirements

### Performance
- API response time: < 200ms (p95)
- Search response time: < 100ms (p95)
- Emergency dispatch: < 5 seconds
- Support 10,000 concurrent users
- Database query time: < 50ms (p95)

### Scalability
- Horizontal scaling for API servers
- Database read replicas for scaling reads
- Redis cluster for distributed caching
- Elasticsearch cluster for search scaling

### Security
- JWT-based authentication (15-minute access tokens)
- HTTPS/TLS for all API communication
- Rate limiting: 100 requests/minute per user
- Input validation and sanitization
- OWASP Top 10 compliance
- PCI DSS compliance for payment processing

### Availability
- 99.9% uptime SLA
- Automated failover for critical services
- Database backups: Daily full, hourly incremental
- Disaster recovery: RPO < 1 hour, RTO < 4 hours

### Monitoring
- Application metrics (Prometheus)
- Distributed tracing (Jaeger)
- Log aggregation (ELK Stack)
- Alerting (PagerDuty)
- Real-time dashboards (Grafana)

---

## Implementation Phases

### Phase 1: Foundation (Months 1-4)
**Goal:** Core infrastructure and MVP

**Deliverables:**
- Core data models (vendors, services, bookings)
- Authentication and authorization (JWT, RBAC)
- Payment integration (Paystack, Flutterwave)
- Basic recommendation engine
- HomeRescue MVP (highest urgency value)
- Mobile app (Flutter) - basic features
- Admin panel (React) - vendor management

**Success Metrics:**
- 100 vendors onboarded
- 500 bookings completed
- < 30 minute emergency response (HomeRescue)

---

### Phase 2: Growth (Months 5-8)
**Goal:** Enhanced features and user acquisition

**Deliverables:**
- ✅ EventGPT conversational interface (Completed: 2026-02-04)
- ✅ VendorNet referral tracking (Completed: 2026-02-04)
- Enhanced recommendation engine (collaborative filtering)
- Notification system (push, email, SMS)
- Search improvements (Elasticsearch, autocomplete)
- Customer reviews and ratings
- Vendor analytics dashboard

**Success Metrics:**
- 500 vendors
- 5,000 bookings/month
- 100 active partnerships (VendorNet)
- 70% EventGPT conversation-to-booking rate

**Implementation Status:**
- VendorNet: ✅ Fully implemented with partnership management, referral tracking, partner matching, and analytics
- EventGPT: ✅ Fully implemented with NLU, slot filling, dialog management, and conversation tracking

---

### Phase 3: Intelligence (Months 9-12)
**Goal:** AI-powered orchestration and automation

**Deliverables:**
- LifeOS event detection engine
- Full orchestration capabilities
- Advanced analytics and insights
- Multi-service bundling
- Budget optimization algorithms
- Predictive vendor matching
- Risk assessment engine

**Success Metrics:**
- 1,000 vendors
- 20,000 bookings/month
- 50 life events orchestrated/month
- 15% average bundle savings

---

### Phase 4: Scale (Year 2)
**Goal:** Market dominance and expansion

**Deliverables:**
- ML-powered predictions (churn, demand forecasting)
- Partner integrations (insurance, financing, logistics)
- Enterprise features (team accounts, API access)
- Geographic expansion (3 new cities)
- White-label solutions
- Advanced fraud detection

**Success Metrics:**
- 5,000 vendors
- 100,000 bookings/month
- 500 life events orchestrated/month
- 1,000 active partnerships

---

## Success Metrics & KPIs

### Platform Metrics
- **Total GMV (Gross Merchandise Value):** Target ₦500M in Year 1
- **Take Rate:** 8-15% average across all products
- **Active Vendors:** 1,000+ by end of Year 1
- **Monthly Active Users:** 50,000+ by end of Year 1

### Product-Specific Metrics

**LifeOS:**
- Life events detected/month
- Orchestration plan acceptance rate (> 60%)
- Average services per event (target: 8)
- Bundle adoption rate (> 40%)

**EventGPT:**
- Conversation-to-booking rate (> 70%)
- Average conversation length (target: < 5 minutes)
- User satisfaction score (> 4.5/5)
- Premium conversion rate (> 15%)

**VendorNet:**
- Active partnerships
- Referral conversion rate (> 30%)
- Average referral value
- Network revenue growth rate

**HomeRescue:**
- Average response time (< 30 min for critical)
- SLA compliance rate (> 95%)
- Emergency resolution rate (> 98%)
- Repeat customer rate (> 40%)

---

## Risk Mitigation

### Technical Risks
| Risk | Impact | Mitigation |
|------|--------|------------|
| Payment integration downtime | High | Multiple payment providers (Paystack, Flutterwave) |
| Database scalability | High | Read replicas, sharding strategy, caching |
| API performance degradation | Medium | Auto-scaling, CDN, database optimization |
| Security vulnerabilities | High | Regular security audits, OWASP compliance, penetration testing |

### Business Risks
| Risk | Impact | Mitigation |
|------|--------|------------|
| Vendor supply constraints | High | Aggressive vendor acquisition, referral incentives |
| Customer acquisition cost | Medium | Content marketing, vendor referrals, partnerships |
| Regulatory compliance | Medium | Legal review, data protection compliance, payment regulations |
| Competition | Medium | Differentiation via AI orchestration, superior UX |

---

## Dependencies & Integrations

### Required Integrations
- **Payments:** Paystack, Flutterwave
- **SMS:** Twilio, Africa's Talking
- **Email:** SendGrid, AWS SES
- **Push Notifications:** Firebase Cloud Messaging
- **Maps:** Google Maps API
- **Storage:** AWS S3 / DigitalOcean Spaces
- **AI/ML:** Anthropic Claude API (EventGPT)

### Optional Integrations (Phase 2+)
- **Analytics:** Mixpanel, Amplitude
- **CRM:** HubSpot, Salesforce
- **Support:** Intercom, Zendesk
- **Insurance:** Partner APIs
- **Financing:** Partner APIs

---

## Compliance & Legal

### Data Protection
- GDPR compliance (for international users)
- Nigeria Data Protection Regulation (NDPR)
- User consent for data processing
- Right to data deletion

### Payment Compliance
- PCI DSS Level 1 certification
- Anti-money laundering (AML) checks
- Know Your Customer (KYC) verification for vendors

### Terms of Service
- User terms and conditions
- Vendor terms and conditions
- Service-level agreements (SLAs)
- Privacy policy
- Refund and cancellation policy

---

## Implementation Progress

### Platform Products Status

| Product | Status | Completion | Last Updated | Notes |
|---------|--------|-----------|--------------|-------|
| **LifeOS** | 🟡 Advanced | 85% | 2026-02-04 | Event detection, bundling, risk assessment, budget optimization - Phase 3 intelligence features complete |
| **EventGPT** | ✅ Complete | 100% | 2026-02-04 | Full NLU, dialog management, conversation tracking |
| **VendorNet** | ✅ Complete | 100% | 2026-02-04 | Partnership management, referral tracking, analytics |
| **HomeRescue** | ✅ Complete | 95% | 2026-02-04 | SLA monitoring, smart dispatch, real-time tracking, availability management |

### Core Services Status

| Service | Status | Completion | Notes |
|---------|--------|-----------|-------|
| **Auth** | ✅ Complete | 100% | JWT, RBAC, verification |
| **Payment** | ✅ Complete | 100% | Paystack, Flutterwave, escrow |
| **Notification** | ✅ Complete | 100% | Email, SMS, push |
| **Search** | ✅ Complete | 100% | Elasticsearch, geospatial, autocomplete, faceted search |
| **Storage** | ✅ Complete | 100% | File upload and management |
| **Worker** | 🟡 Partial | 40% | Background jobs infrastructure |
| **Recommendation Engine** | ✅ Complete | 100% | Adjacency graph, collaborative filtering |

### Phase Completion

- **Phase 1 (Foundation)**: 92% Complete
  - ✅ Core data models
  - ✅ Authentication and authorization
  - ✅ Payment integration
  - ✅ Basic recommendation engine
  - ✅ HomeRescue MVP (95%)
  - 🔴 Mobile app (0%)
  - 🔴 Admin panel (0%)

- **Phase 2 (Growth)**: 55% Complete
  - ✅ EventGPT conversational interface
  - ✅ VendorNet referral tracking
  - ✅ Enhanced recommendation engine
  - ✅ Notification system
  - ✅ Search improvements (100%)
  - ✅ Customer reviews and ratings
  - 🔴 Vendor analytics dashboard (0%)

- **Phase 3 (Intelligence)**: 55% Complete
  - ✅ LifeOS event detection (100%)
  - 🟡 Full orchestration (70%)
  - ✅ Multi-service bundling (100%)
  - ✅ Risk assessment (100%)
  - ✅ Budget optimization (100%)
  - 🔴 Advanced analytics (0%)

- **Phase 4 (Scale)**: 0% Complete
  - Not yet started

### Recent Implementations (Last 7 Days)

**2026-02-04**
- ✅ Search Service Integration (Phase 2)
  - HTTP API handlers (search, autocomplete, reindex)
  - Server integration with Elasticsearch
  - Request validation and error handling
  - 3 API endpoints with comprehensive features
  - 50+ unit tests for validation logic

- ✅ LifeOS - Event Detection & Intelligent Orchestration (Phase 3)
  - Event detection engine with behavioral pattern analysis
  - Multi-service bundling (3 bundle types with 8-25% savings)
  - Risk assessment engine (4 risk types with automated mitigations)
  - Budget optimization with savings opportunities (up to 35%)
  - 4 new API endpoints for intelligence features

- ✅ EventGPT - Conversational AI Event Planner
  - Natural language understanding with 9 intent types
  - Entity extraction for event planning
  - Dialog state management with slot filling
  - Conversation memory and persistence
  - API endpoints for conversation management

- ✅ VendorNet - B2B Partnership Network
  - Partnership management (5 types)
  - Referral tracking with full lifecycle
  - Partner matching algorithm
  - Network analytics dashboard
  - Revenue sharing infrastructure

### Next Priorities

Based on PRD phases and business value:

1. **Advanced Analytics** (Phase 3, Business Intelligence)
   - Event orchestration metrics
   - Bundle adoption tracking
   - Risk mitigation effectiveness
   - Revenue optimization insights

2. **LifeOS ML Enhancement** (Phase 3, High Value)
   - ML-based event detection (replace pattern matching)
   - Recommendation engine integration for vendor bundling
   - Real-time pricing API integration
   - Predictive vendor availability

3. **Worker Service** (Core Infrastructure, Phase 1)
   - Background job processing
   - Cron task scheduling
   - Retry mechanisms
   - Job queue management

4. **Frontend Development** (Phase 1, User Access)
   - Flutter mobile app
   - React admin panel
   - Vendor dashboard

---

## Appendix

### Repository Structure
```
vendorplatform/
├── api/                          # Platform products
│   ├── lifeos/platform.go        (~1,800 lines)
│   ├── eventgpt/platform.go      (~1,600 lines)
│   ├── vendornet/platform.go     (~1,400 lines)
│   ├── homerescue/platform.go    (~1,500 lines)
│   ├── server.go                 (API server setup)
│   └── handlers.go               (HTTP handlers)
├── cmd/server/main.go            # Entry point
├── internal/                     # Core services
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
├── database/                     # SQL schemas
│   ├── 001_core_schema.sql
│   ├── 002_seed_data.sql
│   └── 003_services_schema.sql
├── recommendation-engine/        # ML recommendations
│   ├── engine.go                 (~2,000 lines)
│   └── ml_service.py
├── mobile/flutter/               # Mobile app
├── web/admin/                    # Admin dashboard
├── deployments/                  # Infrastructure
│   ├── docker/Dockerfile
│   └── terraform/main.tf
├── monitoring/prometheus.yml     # Observability
├── tests/                        # Test suites
├── docker-compose.yml
├── Makefile
└── go.mod
```

### Glossary
- **Adjacency:** Relationship score between two services (e.g., wedding photography → wedding catering)
- **Orchestration:** Coordinating multiple services across a life event timeline
- **GMV:** Gross Merchandise Value - total transaction volume
- **Take Rate:** Platform commission percentage
- **SLA:** Service Level Agreement - guaranteed response/completion time
- **MMR:** Maximal Marginal Relevance - diversity algorithm for recommendations
- **RBAC:** Role-Based Access Control

---

**Document End**
