-- ============================================================================
-- VENDOR & ARTISANS PLATFORM - CORE DATABASE SCHEMA
-- Version: 1.0.0
-- Database: PostgreSQL 15+ with extensions
-- Purpose: Multi-cluster vendor marketplace with adjacency-powered recommendations
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";           -- Fuzzy text search
CREATE EXTENSION IF NOT EXISTS "btree_gin";         -- Composite indexes
CREATE EXTENSION IF NOT EXISTS "postgis";           -- Geospatial queries
CREATE EXTENSION IF NOT EXISTS "timescaledb";       -- Time-series analytics

-- ============================================================================
-- SECTION 1: CORE ENTITY TABLES
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1.1 Users (Consumers/Customers)
-- ----------------------------------------------------------------------------
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Identity
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    
    -- Profile
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200),
    avatar_url TEXT,
    date_of_birth DATE,
    gender VARCHAR(20),
    
    -- Location
    primary_address_id UUID,
    current_location GEOGRAPHY(POINT, 4326),
    preferred_timezone VARCHAR(50) DEFAULT 'Africa/Lagos',
    
    -- Preferences (JSONB for flexibility)
    preferences JSONB DEFAULT '{
        "communication": {"email": true, "sms": true, "push": true, "whatsapp": true},
        "language": "en",
        "currency": "NGN",
        "notification_frequency": "instant"
    }'::jsonb,
    
    -- Behavioral Data
    interests TEXT[],
    life_stage VARCHAR(50), -- 'single', 'engaged', 'married', 'parent', 'empty_nester'
    household_size INTEGER DEFAULT 1,
    
    -- Engagement Metrics
    total_bookings INTEGER DEFAULT 0,
    total_spend DECIMAL(15, 2) DEFAULT 0,
    loyalty_points INTEGER DEFAULT 0,
    lifetime_value DECIMAL(15, 2) DEFAULT 0,
    
    -- Trust & Safety
    is_verified BOOLEAN DEFAULT FALSE,
    verification_level INTEGER DEFAULT 0, -- 0-5 scale
    trust_score DECIMAL(3, 2) DEFAULT 0.50,
    is_active BOOLEAN DEFAULT TRUE,
    is_suspended BOOLEAN DEFAULT FALSE,
    suspension_reason TEXT,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_active_at TIMESTAMPTZ DEFAULT NOW(),
    email_verified_at TIMESTAMPTZ,
    phone_verified_at TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_location ON users USING GIST(current_location);
CREATE INDEX idx_users_life_stage ON users(life_stage);
CREATE INDEX idx_users_interests ON users USING GIN(interests);
CREATE INDEX idx_users_preferences ON users USING GIN(preferences);

-- ----------------------------------------------------------------------------
-- 1.2 Vendors/Artisans
-- ----------------------------------------------------------------------------
CREATE TABLE vendors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Identity
    user_id UUID REFERENCES users(id), -- If vendor has user account
    business_name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    legal_name VARCHAR(255),
    
    -- Contact
    primary_email VARCHAR(255) NOT NULL,
    primary_phone VARCHAR(20) NOT NULL,
    whatsapp_number VARCHAR(20),
    website_url TEXT,
    
    -- Business Info
    business_type VARCHAR(50) NOT NULL, -- 'individual', 'registered_business', 'enterprise'
    registration_number VARCHAR(100),
    tax_id VARCHAR(50),
    year_established INTEGER,
    employee_count_range VARCHAR(20), -- '1', '2-5', '6-10', '11-50', '51+'
    
    -- Location & Coverage
    headquarters_address_id UUID,
    service_location GEOGRAPHY(POINT, 4326),
    service_radius_km DECIMAL(6, 2) DEFAULT 50,
    service_areas GEOGRAPHY(MULTIPOLYGON, 4326), -- Complex service boundaries
    covers_nationwide BOOLEAN DEFAULT FALSE,
    
    -- Description
    short_description VARCHAR(500),
    full_description TEXT,
    tagline VARCHAR(200),
    
    -- Media
    logo_url TEXT,
    cover_image_url TEXT,
    gallery_urls TEXT[],
    video_urls TEXT[],
    
    -- Capacity & Availability
    max_concurrent_bookings INTEGER DEFAULT 5,
    lead_time_hours INTEGER DEFAULT 24, -- Minimum notice required
    advance_booking_days INTEGER DEFAULT 90, -- Max days ahead
    instant_booking_enabled BOOLEAN DEFAULT FALSE,
    
    -- Pricing
    currency VARCHAR(3) DEFAULT 'NGN',
    minimum_order_value DECIMAL(12, 2),
    accepts_installments BOOLEAN DEFAULT FALSE,
    payment_methods TEXT[], -- ['cash', 'card', 'transfer', 'ussd']
    
    -- Quality Metrics
    rating_average DECIMAL(2, 1) DEFAULT 0.0,
    rating_count INTEGER DEFAULT 0,
    completion_rate DECIMAL(5, 2) DEFAULT 100.0,
    response_rate DECIMAL(5, 2) DEFAULT 100.0,
    response_time_minutes INTEGER DEFAULT 60,
    repeat_customer_rate DECIMAL(5, 2) DEFAULT 0.0,
    
    -- Trust & Verification
    is_verified BOOLEAN DEFAULT FALSE,
    verification_level INTEGER DEFAULT 0, -- 0-5 scale
    verification_badges TEXT[], -- ['identity', 'business', 'insurance', 'background_check']
    insurance_verified BOOLEAN DEFAULT FALSE,
    background_check_passed BOOLEAN DEFAULT FALSE,
    
    -- Platform Status
    is_active BOOLEAN DEFAULT TRUE,
    is_featured BOOLEAN DEFAULT FALSE,
    is_premium BOOLEAN DEFAULT FALSE,
    subscription_tier VARCHAR(20) DEFAULT 'free', -- 'free', 'basic', 'pro', 'enterprise'
    
    -- Analytics
    profile_views INTEGER DEFAULT 0,
    total_bookings INTEGER DEFAULT 0,
    total_revenue DECIMAL(15, 2) DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_active_at TIMESTAMPTZ DEFAULT NOW(),
    verified_at TIMESTAMPTZ
);

CREATE INDEX idx_vendors_slug ON vendors(slug);
CREATE INDEX idx_vendors_location ON vendors USING GIST(service_location);
CREATE INDEX idx_vendors_service_areas ON vendors USING GIST(service_areas);
CREATE INDEX idx_vendors_rating ON vendors(rating_average DESC, rating_count DESC);
CREATE INDEX idx_vendors_active ON vendors(is_active, is_verified);

-- ----------------------------------------------------------------------------
-- 1.3 Service Categories (Hierarchical)
-- ----------------------------------------------------------------------------
CREATE TABLE service_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Hierarchy
    parent_id UUID REFERENCES service_categories(id),
    level INTEGER NOT NULL DEFAULT 0, -- 0=cluster, 1=category, 2=subcategory, 3=service
    path LTREE NOT NULL, -- Materialized path for efficient queries
    
    -- Identity
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(200) NOT NULL,
    code VARCHAR(50) UNIQUE, -- Internal reference code
    
    -- Description
    short_description VARCHAR(500),
    full_description TEXT,
    icon_name VARCHAR(100),
    icon_url TEXT,
    cover_image_url TEXT,
    
    -- Classification
    cluster_type VARCHAR(50), -- 'celebrations', 'home', 'travel', 'horeca', etc.
    is_seasonal BOOLEAN DEFAULT FALSE,
    peak_seasons TEXT[], -- ['december', 'easter', 'summer']
    
    -- Behavior
    requires_verification BOOLEAN DEFAULT FALSE,
    requires_insurance BOOLEAN DEFAULT FALSE,
    requires_license BOOLEAN DEFAULT FALSE,
    license_types TEXT[],
    
    -- Display
    display_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    is_featured BOOLEAN DEFAULT FALSE,
    
    -- Metrics
    vendor_count INTEGER DEFAULT 0,
    booking_count INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_categories_parent ON service_categories(parent_id);
CREATE INDEX idx_categories_path ON service_categories USING GIST(path);
CREATE INDEX idx_categories_slug ON service_categories(slug);
CREATE INDEX idx_categories_cluster ON service_categories(cluster_type);

-- ----------------------------------------------------------------------------
-- 1.4 Services (Vendor Offerings)
-- ----------------------------------------------------------------------------
CREATE TABLE services (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES service_categories(id),
    
    -- Identity
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    sku VARCHAR(100), -- Stock keeping unit for inventory
    
    -- Description
    short_description VARCHAR(500),
    full_description TEXT,
    highlights TEXT[], -- Key selling points
    includes TEXT[], -- What's included
    excludes TEXT[], -- What's not included
    
    -- Pricing
    pricing_model VARCHAR(30) NOT NULL, -- 'fixed', 'hourly', 'daily', 'per_unit', 'quote', 'package'
    base_price DECIMAL(12, 2),
    price_unit VARCHAR(50), -- 'hour', 'day', 'person', 'item', 'sqm'
    min_price DECIMAL(12, 2),
    max_price DECIMAL(12, 2),
    currency VARCHAR(3) DEFAULT 'NGN',
    
    -- Capacity
    min_quantity INTEGER DEFAULT 1,
    max_quantity INTEGER,
    min_guests INTEGER, -- For event-related services
    max_guests INTEGER,
    
    -- Duration
    duration_minutes INTEGER,
    min_duration_minutes INTEGER,
    max_duration_minutes INTEGER,
    setup_time_minutes INTEGER DEFAULT 0,
    cleanup_time_minutes INTEGER DEFAULT 0,
    
    -- Availability
    is_available BOOLEAN DEFAULT TRUE,
    availability_type VARCHAR(20) DEFAULT 'always', -- 'always', 'scheduled', 'on_request'
    lead_time_hours INTEGER,
    
    -- Media
    images TEXT[],
    videos TEXT[],
    
    -- Variations/Options (JSONB for flexibility)
    variations JSONB DEFAULT '[]'::jsonb,
    addons JSONB DEFAULT '[]'::jsonb,
    
    -- Quality
    rating_average DECIMAL(2, 1) DEFAULT 0.0,
    rating_count INTEGER DEFAULT 0,
    booking_count INTEGER DEFAULT 0,
    
    -- Display
    display_order INTEGER DEFAULT 0,
    is_featured BOOLEAN DEFAULT FALSE,
    tags TEXT[],
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_services_vendor ON services(vendor_id);
CREATE INDEX idx_services_category ON services(category_id);
CREATE INDEX idx_services_pricing ON services(pricing_model, base_price);
CREATE INDEX idx_services_rating ON services(rating_average DESC);
CREATE INDEX idx_services_tags ON services USING GIN(tags);
CREATE UNIQUE INDEX idx_services_vendor_slug ON services(vendor_id, slug);

-- ============================================================================
-- SECTION 2: ADJACENCY & RELATIONSHIP TABLES
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 2.1 Service Adjacency Matrix (Core of Recommendation System)
-- ----------------------------------------------------------------------------
CREATE TABLE service_adjacencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Relationship
    source_category_id UUID NOT NULL REFERENCES service_categories(id),
    target_category_id UUID NOT NULL REFERENCES service_categories(id),
    
    -- Adjacency Type
    adjacency_type VARCHAR(50) NOT NULL, -- 'complementary', 'alternative', 'prerequisite', 'follow_up'
    relationship_direction VARCHAR(20) DEFAULT 'bidirectional', -- 'unidirectional', 'bidirectional'
    
    -- Context
    trigger_context VARCHAR(100), -- 'wedding', 'home_renovation', 'travel', etc.
    trigger_phase VARCHAR(50), -- 'planning', 'execution', 'post_event'
    
    -- Strength Scores
    base_affinity_score DECIMAL(5, 4) NOT NULL DEFAULT 0.5, -- 0-1 scale
    co_purchase_frequency DECIMAL(5, 4) DEFAULT 0, -- From behavioral data
    manual_boost DECIMAL(5, 4) DEFAULT 0, -- Editorial override
    seasonal_factor DECIMAL(5, 4) DEFAULT 1.0, -- Seasonal adjustment
    
    -- Computed Score (updated by triggers/jobs)
    computed_score DECIMAL(5, 4) GENERATED ALWAYS AS (
        LEAST(1.0, base_affinity_score + co_purchase_frequency * 0.3 + manual_boost)
    ) STORED,
    
    -- Business Logic
    bundle_discount_eligible BOOLEAN DEFAULT FALSE,
    suggested_bundle_discount DECIMAL(5, 2), -- Percentage
    cross_sell_priority INTEGER DEFAULT 50, -- 1-100
    
    -- Timing
    typical_time_gap_hours INTEGER, -- How long after source is target typically needed
    time_gap_flexibility VARCHAR(20) DEFAULT 'flexible', -- 'strict', 'flexible', 'any'
    
    -- Display
    recommendation_copy TEXT, -- "Customers who booked X also booked Y"
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Analytics
    impression_count INTEGER DEFAULT 0,
    click_count INTEGER DEFAULT 0,
    conversion_count INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_adjacency UNIQUE(source_category_id, target_category_id, trigger_context)
);

CREATE INDEX idx_adjacency_source ON service_adjacencies(source_category_id);
CREATE INDEX idx_adjacency_target ON service_adjacencies(target_category_id);
CREATE INDEX idx_adjacency_context ON service_adjacencies(trigger_context);
CREATE INDEX idx_adjacency_score ON service_adjacencies(computed_score DESC);
CREATE INDEX idx_adjacency_active ON service_adjacencies(is_active, computed_score DESC);

-- ----------------------------------------------------------------------------
-- 2.2 Life Event Triggers (What initiates demand cascades)
-- ----------------------------------------------------------------------------
CREATE TABLE life_event_triggers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Identity
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(200) UNIQUE NOT NULL,
    code VARCHAR(50) UNIQUE, -- 'WEDDING', 'CHILDBIRTH', 'RELOCATION'
    
    -- Classification
    event_type VARCHAR(50) NOT NULL, -- 'celebration', 'transition', 'emergency', 'milestone', 'routine'
    cluster_type VARCHAR(50) NOT NULL,
    
    -- Description
    description TEXT,
    typical_timeline_days INTEGER, -- How long is the typical planning/execution period
    
    -- Associated Categories
    primary_category_ids UUID[], -- Main services for this event
    
    -- Seasonality
    peak_months INTEGER[], -- 1-12
    peak_days_of_week INTEGER[], -- 0-6 (Sunday = 0)
    
    -- User Signals (How we detect this event)
    detection_keywords TEXT[],
    detection_patterns JSONB, -- Complex pattern matching rules
    
    -- Demand Modeling
    avg_services_booked DECIMAL(4, 1),
    avg_spend DECIMAL(12, 2),
    avg_lead_time_days INTEGER,
    
    -- Display
    icon_name VARCHAR(100),
    display_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_triggers_type ON life_event_triggers(event_type);
CREATE INDEX idx_triggers_cluster ON life_event_triggers(cluster_type);
CREATE INDEX idx_triggers_keywords ON life_event_triggers USING GIN(detection_keywords);

-- ----------------------------------------------------------------------------
-- 2.3 Event Category Mapping (Links events to service categories)
-- ----------------------------------------------------------------------------
CREATE TABLE event_category_mappings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    event_trigger_id UUID NOT NULL REFERENCES life_event_triggers(id),
    category_id UUID NOT NULL REFERENCES service_categories(id),
    
    -- Role in Event
    role_type VARCHAR(50) NOT NULL, -- 'primary', 'secondary', 'optional', 'luxury'
    phase VARCHAR(50), -- 'planning', 'pre_event', 'event_day', 'post_event'
    
    -- Timing
    typical_booking_offset_days INTEGER, -- Days before event this is typically booked
    deadline_offset_days INTEGER, -- Latest this can be booked
    
    -- Importance
    necessity_score DECIMAL(3, 2) DEFAULT 0.5, -- 0-1, how essential
    popularity_score DECIMAL(3, 2) DEFAULT 0.5, -- 0-1, how often booked
    
    -- Budget Allocation
    typical_budget_percentage DECIMAL(5, 2), -- What % of event budget
    
    -- Display
    display_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_event_category UNIQUE(event_trigger_id, category_id)
);

CREATE INDEX idx_event_mapping_trigger ON event_category_mappings(event_trigger_id);
CREATE INDEX idx_event_mapping_category ON event_category_mappings(category_id);

-- ----------------------------------------------------------------------------
-- 2.4 Vendor Partnerships (Referral networks)
-- ----------------------------------------------------------------------------
CREATE TABLE vendor_partnerships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Partners
    vendor_a_id UUID NOT NULL REFERENCES vendors(id),
    vendor_b_id UUID NOT NULL REFERENCES vendors(id),
    
    -- Partnership Type
    partnership_type VARCHAR(50) NOT NULL, -- 'referral', 'bundle', 'exclusive', 'preferred'
    
    -- Terms
    referral_fee_type VARCHAR(20), -- 'percentage', 'fixed', 'none'
    referral_fee_value DECIMAL(8, 2),
    is_bidirectional BOOLEAN DEFAULT TRUE,
    
    -- Performance
    total_referrals INTEGER DEFAULT 0,
    successful_referrals INTEGER DEFAULT 0,
    total_revenue_generated DECIMAL(15, 2) DEFAULT 0,
    
    -- Status
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'active', 'paused', 'terminated'
    initiated_by UUID REFERENCES vendors(id),
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    
    CONSTRAINT different_vendors CHECK (vendor_a_id != vendor_b_id),
    CONSTRAINT unique_partnership UNIQUE(vendor_a_id, vendor_b_id)
);

CREATE INDEX idx_partnerships_vendor_a ON vendor_partnerships(vendor_a_id);
CREATE INDEX idx_partnerships_vendor_b ON vendor_partnerships(vendor_b_id);
CREATE INDEX idx_partnerships_status ON vendor_partnerships(status);

-- ============================================================================
-- SECTION 3: BOOKING & TRANSACTION TABLES
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 3.1 Projects (Group of related bookings for an event)
-- ----------------------------------------------------------------------------
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Owner
    user_id UUID NOT NULL REFERENCES users(id),
    
    -- Identity
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255),
    
    -- Event Details
    event_trigger_id UUID REFERENCES life_event_triggers(id),
    event_type VARCHAR(100),
    event_date DATE,
    event_location GEOGRAPHY(POINT, 4326),
    event_venue_id UUID,
    
    -- Guest/Scale
    expected_guests INTEGER,
    guest_range_min INTEGER,
    guest_range_max INTEGER,
    
    -- Budget
    total_budget DECIMAL(15, 2),
    budget_flexibility VARCHAR(20) DEFAULT 'moderate', -- 'strict', 'moderate', 'flexible'
    currency VARCHAR(3) DEFAULT 'NGN',
    
    -- Progress Tracking
    status VARCHAR(30) DEFAULT 'planning', -- 'planning', 'booking', 'confirmed', 'in_progress', 'completed', 'cancelled'
    completion_percentage DECIMAL(5, 2) DEFAULT 0,
    
    -- Services
    required_category_ids UUID[],
    booked_category_ids UUID[],
    pending_category_ids UUID[],
    
    -- Financial Summary
    total_quoted DECIMAL(15, 2) DEFAULT 0,
    total_booked DECIMAL(15, 2) DEFAULT 0,
    total_paid DECIMAL(15, 2) DEFAULT 0,
    
    -- Collaboration
    collaborator_ids UUID[],
    is_public BOOLEAN DEFAULT FALSE, -- Allow vendor discovery
    
    -- Notes
    notes TEXT,
    preferences JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    event_completed_at TIMESTAMPTZ
);

CREATE INDEX idx_projects_user ON projects(user_id);
CREATE INDEX idx_projects_event ON projects(event_trigger_id);
CREATE INDEX idx_projects_date ON projects(event_date);
CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_projects_location ON projects USING GIST(event_location);

-- ----------------------------------------------------------------------------
-- 3.2 Bookings
-- ----------------------------------------------------------------------------
CREATE TABLE bookings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- References
    user_id UUID NOT NULL REFERENCES users(id),
    vendor_id UUID NOT NULL REFERENCES vendors(id),
    service_id UUID NOT NULL REFERENCES services(id),
    project_id UUID REFERENCES projects(id),
    
    -- Booking Number
    booking_number VARCHAR(20) UNIQUE NOT NULL,
    
    -- Schedule
    scheduled_date DATE NOT NULL,
    scheduled_start_time TIME,
    scheduled_end_time TIME,
    duration_minutes INTEGER,
    timezone VARCHAR(50) DEFAULT 'Africa/Lagos',
    
    -- Location
    service_location_type VARCHAR(20), -- 'vendor', 'customer', 'venue', 'online'
    service_address_id UUID,
    service_location GEOGRAPHY(POINT, 4326),
    
    -- Quantity/Details
    quantity INTEGER DEFAULT 1,
    guest_count INTEGER,
    
    -- Pricing
    unit_price DECIMAL(12, 2) NOT NULL,
    quantity_multiplier DECIMAL(8, 4) DEFAULT 1,
    subtotal DECIMAL(12, 2) NOT NULL,
    discount_amount DECIMAL(12, 2) DEFAULT 0,
    discount_reason VARCHAR(100),
    tax_amount DECIMAL(12, 2) DEFAULT 0,
    service_fee DECIMAL(12, 2) DEFAULT 0,
    total_amount DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'NGN',
    
    -- Payment
    payment_status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'partial', 'paid', 'refunded'
    amount_paid DECIMAL(12, 2) DEFAULT 0,
    payment_due_date DATE,
    
    -- Status
    status VARCHAR(30) DEFAULT 'pending', 
    -- 'pending', 'confirmed', 'in_progress', 'completed', 'cancelled', 'no_show', 'disputed'
    
    -- Customer Notes
    customer_notes TEXT,
    special_requests TEXT,
    
    -- Vendor Notes
    vendor_notes TEXT,
    internal_notes TEXT,
    
    -- Variations/Addons Selected
    selected_variations JSONB,
    selected_addons JSONB,
    
    -- Attribution (for adjacency tracking)
    source_booking_id UUID REFERENCES bookings(id), -- If came from recommendation
    source_type VARCHAR(30), -- 'direct', 'recommendation', 'bundle', 'referral', 'search'
    recommendation_position INTEGER,
    
    -- Ratings
    customer_rating DECIMAL(2, 1),
    customer_review TEXT,
    vendor_rating DECIMAL(2, 1),
    vendor_review TEXT,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancellation_reason TEXT
);

CREATE INDEX idx_bookings_user ON bookings(user_id);
CREATE INDEX idx_bookings_vendor ON bookings(vendor_id);
CREATE INDEX idx_bookings_service ON bookings(service_id);
CREATE INDEX idx_bookings_project ON bookings(project_id);
CREATE INDEX idx_bookings_date ON bookings(scheduled_date);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_bookings_source ON bookings(source_type);
CREATE INDEX idx_bookings_number ON bookings(booking_number);

-- ============================================================================
-- SECTION 4: USER BEHAVIOR & ANALYTICS TABLES
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 4.1 User Service Interactions (For ML training)
-- ----------------------------------------------------------------------------
CREATE TABLE user_interactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    user_id UUID NOT NULL REFERENCES users(id),
    
    -- What was interacted with
    entity_type VARCHAR(30) NOT NULL, -- 'vendor', 'service', 'category', 'bundle'
    entity_id UUID NOT NULL,
    
    -- Context
    session_id UUID,
    project_id UUID REFERENCES projects(id),
    source_page VARCHAR(100),
    
    -- Interaction Type
    interaction_type VARCHAR(30) NOT NULL,
    -- 'view', 'click', 'save', 'share', 'inquire', 'add_to_cart', 'book', 'review'
    
    -- Duration
    duration_seconds INTEGER,
    scroll_depth_percentage DECIMAL(5, 2),
    
    -- Device/Location
    device_type VARCHAR(20),
    platform VARCHAR(20),
    user_location GEOGRAPHY(POINT, 4326),
    
    -- Timestamp
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Use TimescaleDB hypertable for efficient time-series queries
SELECT create_hypertable('user_interactions', 'created_at', if_not_exists => TRUE);

CREATE INDEX idx_interactions_user ON user_interactions(user_id, created_at DESC);
CREATE INDEX idx_interactions_entity ON user_interactions(entity_type, entity_id);
CREATE INDEX idx_interactions_type ON user_interactions(interaction_type);

-- ----------------------------------------------------------------------------
-- 4.2 Search History (For understanding intent)
-- ----------------------------------------------------------------------------
CREATE TABLE search_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    user_id UUID REFERENCES users(id), -- Can be null for anonymous
    session_id UUID,
    
    -- Query
    search_query TEXT NOT NULL,
    normalized_query TEXT, -- Cleaned version
    detected_intent VARCHAR(50), -- ML-classified intent
    detected_event_type VARCHAR(50),
    detected_categories UUID[],
    
    -- Context
    search_location GEOGRAPHY(POINT, 4326),
    filters_applied JSONB,
    
    -- Results
    result_count INTEGER,
    clicked_results UUID[], -- Entity IDs clicked
    first_click_position INTEGER,
    
    -- Timestamp
    created_at TIMESTAMPTZ DEFAULT NOW()
);

SELECT create_hypertable('search_history', 'created_at', if_not_exists => TRUE);

CREATE INDEX idx_search_user ON search_history(user_id, created_at DESC);
CREATE INDEX idx_search_query ON search_history USING GIN(to_tsvector('english', search_query));
CREATE INDEX idx_search_intent ON search_history(detected_intent);

-- ----------------------------------------------------------------------------
-- 4.3 Co-Purchase Patterns (Aggregated for recommendations)
-- ----------------------------------------------------------------------------
CREATE TABLE co_purchase_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Category Pair
    category_a_id UUID NOT NULL REFERENCES service_categories(id),
    category_b_id UUID NOT NULL REFERENCES service_categories(id),
    
    -- Context
    event_trigger_id UUID REFERENCES life_event_triggers(id),
    time_window_days INTEGER, -- Within how many days
    
    -- Metrics
    co_purchase_count INTEGER DEFAULT 0,
    category_a_first_count INTEGER DEFAULT 0, -- How often A was booked first
    category_b_first_count INTEGER DEFAULT 0,
    avg_time_between_hours DECIMAL(10, 2),
    
    -- Calculated Scores
    lift_score DECIMAL(8, 4), -- Statistical lift
    confidence_score DECIMAL(5, 4), -- Confidence
    support_score DECIMAL(5, 4), -- Support
    
    -- Timestamps
    calculated_at TIMESTAMPTZ DEFAULT NOW(),
    period_start DATE,
    period_end DATE,
    
    CONSTRAINT unique_copurchase UNIQUE(category_a_id, category_b_id, event_trigger_id)
);

CREATE INDEX idx_copurchase_category_a ON co_purchase_patterns(category_a_id);
CREATE INDEX idx_copurchase_category_b ON co_purchase_patterns(category_b_id);
CREATE INDEX idx_copurchase_event ON co_purchase_patterns(event_trigger_id);
CREATE INDEX idx_copurchase_lift ON co_purchase_patterns(lift_score DESC);

-- ============================================================================
-- SECTION 5: BUNDLE & PACKAGE TABLES
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 5.1 Service Bundles
-- ----------------------------------------------------------------------------
CREATE TABLE service_bundles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Identity
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    
    -- Context
    event_trigger_id UUID REFERENCES life_event_triggers(id),
    bundle_type VARCHAR(30) NOT NULL, -- 'curated', 'dynamic', 'vendor_created'
    
    -- Description
    short_description VARCHAR(500),
    full_description TEXT,
    highlights TEXT[],
    
    -- Components
    category_ids UUID[] NOT NULL,
    min_vendors INTEGER DEFAULT 2,
    max_vendors INTEGER,
    
    -- Pricing
    pricing_strategy VARCHAR(30) DEFAULT 'sum_discount',
    -- 'fixed', 'sum_discount', 'tiered', 'negotiated'
    fixed_price DECIMAL(12, 2),
    discount_percentage DECIMAL(5, 2),
    min_savings_percentage DECIMAL(5, 2),
    
    -- Targeting
    target_budget_min DECIMAL(12, 2),
    target_budget_max DECIMAL(12, 2),
    target_guest_min INTEGER,
    target_guest_max INTEGER,
    
    -- Media
    cover_image_url TEXT,
    gallery_urls TEXT[],
    
    -- Display
    display_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    is_featured BOOLEAN DEFAULT FALSE,
    
    -- Performance
    view_count INTEGER DEFAULT 0,
    purchase_count INTEGER DEFAULT 0,
    conversion_rate DECIMAL(5, 4) DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_bundles_event ON service_bundles(event_trigger_id);
CREATE INDEX idx_bundles_categories ON service_bundles USING GIN(category_ids);
CREATE INDEX idx_bundles_active ON service_bundles(is_active, is_featured);

-- ----------------------------------------------------------------------------
-- 5.2 Bundle Vendor Assignments
-- ----------------------------------------------------------------------------
CREATE TABLE bundle_vendor_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    bundle_id UUID NOT NULL REFERENCES service_bundles(id),
    vendor_id UUID NOT NULL REFERENCES vendors(id),
    category_id UUID NOT NULL REFERENCES service_categories(id),
    service_id UUID REFERENCES services(id),
    
    -- Role
    is_required BOOLEAN DEFAULT FALSE,
    is_default BOOLEAN DEFAULT FALSE, -- Pre-selected option
    
    -- Pricing
    bundle_price DECIMAL(12, 2), -- Special price for this bundle
    regular_price DECIMAL(12, 2),
    
    -- Display
    display_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_bundle_vendor_category UNIQUE(bundle_id, vendor_id, category_id)
);

CREATE INDEX idx_bundle_assignments_bundle ON bundle_vendor_assignments(bundle_id);
CREATE INDEX idx_bundle_assignments_vendor ON bundle_vendor_assignments(vendor_id);

-- ============================================================================
-- SECTION 6: RECOMMENDATION TRACKING TABLES
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 6.1 Recommendation Events (For A/B testing and optimization)
-- ----------------------------------------------------------------------------
CREATE TABLE recommendation_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Context
    user_id UUID REFERENCES users(id),
    session_id UUID,
    
    -- Recommendation Details
    recommendation_type VARCHAR(50) NOT NULL,
    -- 'adjacent_service', 'similar_vendor', 'bundle', 'trending', 'personalized'
    algorithm_version VARCHAR(20),
    
    -- What was recommended
    recommended_entity_type VARCHAR(30), -- 'vendor', 'service', 'category', 'bundle'
    recommended_entity_id UUID,
    
    -- Source context
    source_entity_type VARCHAR(30),
    source_entity_id UUID,
    source_page VARCHAR(100),
    
    -- Position
    position INTEGER,
    total_recommendations INTEGER,
    
    -- Scores
    relevance_score DECIMAL(5, 4),
    diversity_score DECIMAL(5, 4),
    
    -- Outcome
    was_impressed BOOLEAN DEFAULT TRUE,
    was_clicked BOOLEAN DEFAULT FALSE,
    was_converted BOOLEAN DEFAULT FALSE,
    
    -- A/B Testing
    experiment_id UUID,
    variant VARCHAR(50),
    
    -- Timestamp
    created_at TIMESTAMPTZ DEFAULT NOW(),
    clicked_at TIMESTAMPTZ,
    converted_at TIMESTAMPTZ
);

SELECT create_hypertable('recommendation_events', 'created_at', if_not_exists => TRUE);

CREATE INDEX idx_rec_events_user ON recommendation_events(user_id, created_at DESC);
CREATE INDEX idx_rec_events_type ON recommendation_events(recommendation_type);
CREATE INDEX idx_rec_events_entity ON recommendation_events(recommended_entity_type, recommended_entity_id);
CREATE INDEX idx_rec_events_experiment ON recommendation_events(experiment_id, variant);

-- ============================================================================
-- SECTION 7: HELPER FUNCTIONS AND TRIGGERS
-- ============================================================================

-- Function to update adjacency scores based on co-purchase data
CREATE OR REPLACE FUNCTION update_adjacency_scores()
RETURNS VOID AS $$
BEGIN
    UPDATE service_adjacencies sa
    SET co_purchase_frequency = COALESCE(
        (SELECT cp.confidence_score 
         FROM co_purchase_patterns cp 
         WHERE (cp.category_a_id = sa.source_category_id AND cp.category_b_id = sa.target_category_id)
            OR (cp.category_a_id = sa.target_category_id AND cp.category_b_id = sa.source_category_id 
                AND sa.relationship_direction = 'bidirectional')
         ORDER BY cp.calculated_at DESC
         LIMIT 1
        ), 0
    ),
    updated_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- Function to calculate user lifetime value
CREATE OR REPLACE FUNCTION calculate_user_ltv(p_user_id UUID)
RETURNS DECIMAL AS $$
DECLARE
    v_ltv DECIMAL;
BEGIN
    SELECT COALESCE(SUM(total_amount), 0)
    INTO v_ltv
    FROM bookings
    WHERE user_id = p_user_id
    AND status IN ('completed', 'confirmed');
    
    RETURN v_ltv;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update user stats after booking
CREATE OR REPLACE FUNCTION update_user_stats_after_booking()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR (TG_OP = 'UPDATE' AND NEW.status != OLD.status) THEN
        UPDATE users
        SET 
            total_bookings = (SELECT COUNT(*) FROM bookings WHERE user_id = NEW.user_id AND status NOT IN ('cancelled')),
            total_spend = (SELECT COALESCE(SUM(total_amount), 0) FROM bookings WHERE user_id = NEW.user_id AND status IN ('completed', 'confirmed')),
            lifetime_value = calculate_user_ltv(NEW.user_id),
            updated_at = NOW()
        WHERE id = NEW.user_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_user_stats
AFTER INSERT OR UPDATE ON bookings
FOR EACH ROW
EXECUTE FUNCTION update_user_stats_after_booking();

-- Trigger to update vendor stats after booking
CREATE OR REPLACE FUNCTION update_vendor_stats_after_booking()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR (TG_OP = 'UPDATE' AND NEW.status != OLD.status) THEN
        UPDATE vendors
        SET 
            total_bookings = (SELECT COUNT(*) FROM bookings WHERE vendor_id = NEW.vendor_id AND status NOT IN ('cancelled')),
            total_revenue = (SELECT COALESCE(SUM(total_amount), 0) FROM bookings WHERE vendor_id = NEW.vendor_id AND status IN ('completed', 'confirmed')),
            updated_at = NOW()
        WHERE id = NEW.vendor_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_vendor_stats
AFTER INSERT OR UPDATE ON bookings
FOR EACH ROW
EXECUTE FUNCTION update_vendor_stats_after_booking();

-- Function to get adjacent categories for a given category
CREATE OR REPLACE FUNCTION get_adjacent_categories(
    p_category_id UUID,
    p_context VARCHAR DEFAULT NULL,
    p_limit INTEGER DEFAULT 10
)
RETURNS TABLE (
    category_id UUID,
    category_name VARCHAR,
    adjacency_type VARCHAR,
    score DECIMAL,
    recommendation_copy TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        sc.id,
        sc.name,
        sa.adjacency_type,
        sa.computed_score,
        sa.recommendation_copy
    FROM service_adjacencies sa
    JOIN service_categories sc ON sc.id = sa.target_category_id
    WHERE sa.source_category_id = p_category_id
    AND sa.is_active = TRUE
    AND (p_context IS NULL OR sa.trigger_context = p_context OR sa.trigger_context IS NULL)
    ORDER BY sa.computed_score DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SECTION 8: VIEWS FOR COMMON QUERIES
-- ============================================================================

-- View: Active vendors with services
CREATE OR REPLACE VIEW v_active_vendors AS
SELECT 
    v.*,
    COUNT(DISTINCT s.id) AS service_count,
    ARRAY_AGG(DISTINCT s.category_id) AS category_ids
FROM vendors v
LEFT JOIN services s ON s.vendor_id = v.id AND s.is_available = TRUE
WHERE v.is_active = TRUE
GROUP BY v.id;

-- View: Category hierarchy with counts
CREATE OR REPLACE VIEW v_category_hierarchy AS
WITH RECURSIVE category_tree AS (
    SELECT 
        id, parent_id, name, slug, level, path,
        ARRAY[name] AS path_names
    FROM service_categories
    WHERE parent_id IS NULL
    
    UNION ALL
    
    SELECT 
        sc.id, sc.parent_id, sc.name, sc.slug, sc.level, sc.path,
        ct.path_names || sc.name
    FROM service_categories sc
    JOIN category_tree ct ON sc.parent_id = ct.id
)
SELECT 
    ct.*,
    (SELECT COUNT(*) FROM services s WHERE s.category_id = ct.id AND s.is_available = TRUE) AS active_service_count,
    (SELECT COUNT(DISTINCT v.id) FROM vendors v 
     JOIN services s ON s.vendor_id = v.id 
     WHERE s.category_id = ct.id AND v.is_active = TRUE) AS active_vendor_count
FROM category_tree ct;

-- View: Adjacency recommendations with full details
CREATE OR REPLACE VIEW v_adjacency_recommendations AS
SELECT 
    sa.id AS adjacency_id,
    sa.trigger_context,
    sa.adjacency_type,
    sa.computed_score,
    sa.recommendation_copy,
    source_cat.id AS source_category_id,
    source_cat.name AS source_category_name,
    source_cat.slug AS source_category_slug,
    target_cat.id AS target_category_id,
    target_cat.name AS target_category_name,
    target_cat.slug AS target_category_slug,
    sa.bundle_discount_eligible,
    sa.suggested_bundle_discount
FROM service_adjacencies sa
JOIN service_categories source_cat ON source_cat.id = sa.source_category_id
JOIN service_categories target_cat ON target_cat.id = sa.target_category_id
WHERE sa.is_active = TRUE
ORDER BY sa.computed_score DESC;

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================
