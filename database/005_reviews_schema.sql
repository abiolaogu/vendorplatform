-- ============================================================================
-- REVIEWS & RATINGS SCHEMA
-- Purpose: Enable customers to review vendors and services
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Reviews Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Relations
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    booking_id UUID REFERENCES bookings(id) ON DELETE SET NULL, -- Links to verified booking

    -- Rating (1-5 stars)
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),

    -- Content
    title VARCHAR(200),
    comment TEXT,

    -- Detailed ratings (optional breakdowns)
    quality_rating INTEGER CHECK (quality_rating IS NULL OR (quality_rating >= 1 AND quality_rating <= 5)),
    communication_rating INTEGER CHECK (communication_rating IS NULL OR (communication_rating >= 1 AND communication_rating <= 5)),
    timeliness_rating INTEGER CHECK (timeliness_rating IS NULL OR (timeliness_rating <= 1 AND timeliness_rating <= 5)),
    value_rating INTEGER CHECK (value_rating IS NULL OR (value_rating >= 1 AND value_rating <= 5)),

    -- Media
    image_urls TEXT[],

    -- Status
    is_verified BOOLEAN DEFAULT FALSE, -- Verified purchase/booking
    is_published BOOLEAN DEFAULT TRUE,
    is_flagged BOOLEAN DEFAULT FALSE,
    flag_reason TEXT,

    -- Engagement
    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0,

    -- Vendor Response
    vendor_response TEXT,
    vendor_responded_at TIMESTAMPTZ,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_reviews_vendor ON reviews(vendor_id, is_published) WHERE is_published = TRUE;
CREATE INDEX idx_reviews_user ON reviews(user_id);
CREATE INDEX idx_reviews_booking ON reviews(booking_id);
CREATE INDEX idx_reviews_rating ON reviews(vendor_id, rating DESC);
CREATE INDEX idx_reviews_created ON reviews(created_at DESC);
CREATE INDEX idx_reviews_verified ON reviews(vendor_id, is_verified) WHERE is_verified = TRUE;

-- Unique constraint: One review per user per booking
CREATE UNIQUE INDEX idx_reviews_user_booking ON reviews(user_id, booking_id) WHERE booking_id IS NOT NULL;

-- ----------------------------------------------------------------------------
-- Review Helpfulness Votes
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS review_votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    is_helpful BOOLEAN NOT NULL, -- true = helpful, false = not helpful

    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- One vote per user per review
    UNIQUE(review_id, user_id)
);

CREATE INDEX idx_review_votes_review ON review_votes(review_id);
CREATE INDEX idx_review_votes_user ON review_votes(user_id);

-- ----------------------------------------------------------------------------
-- Function to update vendor ratings
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION update_vendor_ratings()
RETURNS TRIGGER AS $$
BEGIN
    -- Recalculate vendor's average rating and count
    UPDATE vendors
    SET
        rating_average = (
            SELECT COALESCE(ROUND(AVG(rating)::numeric, 1), 0)
            FROM reviews
            WHERE vendor_id = COALESCE(NEW.vendor_id, OLD.vendor_id)
            AND is_published = TRUE
        ),
        rating_count = (
            SELECT COUNT(*)
            FROM reviews
            WHERE vendor_id = COALESCE(NEW.vendor_id, OLD.vendor_id)
            AND is_published = TRUE
        ),
        updated_at = NOW()
    WHERE id = COALESCE(NEW.vendor_id, OLD.vendor_id);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update vendor ratings
DROP TRIGGER IF EXISTS trigger_update_vendor_ratings ON reviews;
CREATE TRIGGER trigger_update_vendor_ratings
    AFTER INSERT OR UPDATE OF rating, is_published OR DELETE ON reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_vendor_ratings();

-- ----------------------------------------------------------------------------
-- Function to update review helpful counts
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION update_review_helpful_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.is_helpful THEN
            UPDATE reviews SET helpful_count = helpful_count + 1 WHERE id = NEW.review_id;
        ELSE
            UPDATE reviews SET not_helpful_count = not_helpful_count + 1 WHERE id = NEW.review_id;
        END IF;
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.is_helpful <> NEW.is_helpful THEN
            IF NEW.is_helpful THEN
                UPDATE reviews
                SET helpful_count = helpful_count + 1,
                    not_helpful_count = not_helpful_count - 1
                WHERE id = NEW.review_id;
            ELSE
                UPDATE reviews
                SET helpful_count = helpful_count - 1,
                    not_helpful_count = not_helpful_count + 1
                WHERE id = NEW.review_id;
            END IF;
        END IF;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.is_helpful THEN
            UPDATE reviews SET helpful_count = helpful_count - 1 WHERE id = OLD.review_id;
        ELSE
            UPDATE reviews SET not_helpful_count = not_helpful_count - 1 WHERE id = OLD.review_id;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update review helpful counts
DROP TRIGGER IF EXISTS trigger_update_review_helpful_counts ON review_votes;
CREATE TRIGGER trigger_update_review_helpful_counts
    AFTER INSERT OR UPDATE OR DELETE ON review_votes
    FOR EACH ROW
    EXECUTE FUNCTION update_review_helpful_counts();
