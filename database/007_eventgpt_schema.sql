-- =============================================================================
-- EVENTGPT DATABASE SCHEMA
-- Conversational AI Event Planning System
-- =============================================================================

-- Conversations table: Stores EventGPT conversation sessions
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id UUID REFERENCES life_events(id) ON DELETE SET NULL,

    -- Session metadata
    session_type VARCHAR(50) NOT NULL DEFAULT 'general_inquiry',
    -- session_type: 'new_event', 'event_planning', 'vendor_search', 'booking_help', 'general_inquiry', 'support'

    -- Conversation state
    current_intent JSONB NOT NULL DEFAULT '{}',
    conversation_state VARCHAR(50) NOT NULL DEFAULT 'welcome',
    -- conversation_state: 'welcome', 'gathering_info', 'confirming', 'recommending', 'comparing', 'booking', 'completed', 'handoff'
    slot_values JSONB NOT NULL DEFAULT '{}',

    -- Message history
    messages JSONB NOT NULL DEFAULT '[]',
    turn_count INT NOT NULL DEFAULT 0,

    -- Memory and context
    short_term_memory JSONB NOT NULL DEFAULT '{}',
    long_term_memory JSONB DEFAULT '{}',

    -- Communication preferences
    language VARCHAR(10) NOT NULL DEFAULT 'en',
    channel VARCHAR(20) NOT NULL DEFAULT 'web',
    -- channel: 'web', 'mobile', 'whatsapp', 'voice', 'api'

    -- Timestamps
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP,

    -- Indexes
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for conversations
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_event_id ON conversations(event_id) WHERE event_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_conversations_last_message ON conversations(last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversations_state ON conversations(conversation_state);
CREATE INDEX IF NOT EXISTS idx_conversations_session_type ON conversations(session_type);

-- Conversation analytics table for tracking metrics
CREATE TABLE IF NOT EXISTS conversation_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,

    -- Conversation metrics
    total_turns INT NOT NULL DEFAULT 0,
    user_messages_count INT NOT NULL DEFAULT 0,
    assistant_messages_count INT NOT NULL DEFAULT 0,
    average_response_time_ms INT,

    -- Intent tracking
    primary_intent VARCHAR(50),
    intent_changes INT NOT NULL DEFAULT 0,
    intents_used JSONB DEFAULT '[]',

    -- Slot filling metrics
    slots_required INT NOT NULL DEFAULT 0,
    slots_filled INT NOT NULL DEFAULT 0,
    slot_filling_turns INT NOT NULL DEFAULT 0,

    -- Outcome tracking
    outcome VARCHAR(50),
    -- outcome: 'event_created', 'vendor_found', 'booking_made', 'abandoned', 'escalated', 'completed'
    outcome_value DECIMAL(12,2),

    -- User satisfaction (if collected)
    satisfaction_score INT CHECK (satisfaction_score BETWEEN 1 AND 5),
    feedback_text TEXT,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conv_analytics_conversation ON conversation_analytics(conversation_id);
CREATE INDEX IF NOT EXISTS idx_conv_analytics_outcome ON conversation_analytics(outcome);
CREATE INDEX IF NOT EXISTS idx_conv_analytics_satisfaction ON conversation_analytics(satisfaction_score);

-- Intent classification logs for ML training
CREATE TABLE IF NOT EXISTS intent_classification_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,

    -- Input
    user_message TEXT NOT NULL,
    context_slots JSONB DEFAULT '{}',

    -- Classification result
    classified_intent VARCHAR(50) NOT NULL,
    confidence DECIMAL(3,2) NOT NULL,
    alternative_intents JSONB DEFAULT '[]',

    -- Validation
    is_correct BOOLEAN,
    correct_intent VARCHAR(50),
    validated_by UUID REFERENCES users(id),
    validated_at TIMESTAMP,

    -- Metadata
    model_version VARCHAR(50),
    processing_time_ms INT,

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_intent_logs_conversation ON intent_classification_logs(conversation_id);
CREATE INDEX IF NOT EXISTS idx_intent_logs_intent ON intent_classification_logs(classified_intent);
CREATE INDEX IF NOT EXISTS idx_intent_logs_validation ON intent_classification_logs(is_correct) WHERE is_correct IS NOT NULL;

-- Entity extraction logs for training
CREATE TABLE IF NOT EXISTS entity_extraction_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    intent_log_id UUID REFERENCES intent_classification_logs(id) ON DELETE CASCADE,

    -- Input
    user_message TEXT NOT NULL,

    -- Extracted entities
    entity_type VARCHAR(50) NOT NULL,
    entity_value TEXT NOT NULL,
    text_span VARCHAR(200),
    start_pos INT,
    end_pos INT,
    confidence DECIMAL(3,2) NOT NULL,

    -- Validation
    is_correct BOOLEAN,
    correct_entity_type VARCHAR(50),
    correct_entity_value TEXT,
    validated_by UUID REFERENCES users(id),
    validated_at TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_entity_logs_conversation ON entity_extraction_logs(conversation_id);
CREATE INDEX IF NOT EXISTS idx_entity_logs_type ON entity_extraction_logs(entity_type);
CREATE INDEX IF NOT EXISTS idx_entity_logs_validation ON entity_extraction_logs(is_correct) WHERE is_correct IS NOT NULL;

-- User conversation preferences
CREATE TABLE IF NOT EXISTS user_conversation_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

    -- Communication style
    preferred_language VARCHAR(10) NOT NULL DEFAULT 'en',
    preferred_channel VARCHAR(20) NOT NULL DEFAULT 'web',
    response_length VARCHAR(20) DEFAULT 'balanced', -- 'brief', 'balanced', 'detailed'

    -- Content preferences
    show_prices BOOLEAN DEFAULT TRUE,
    show_ratings BOOLEAN DEFAULT TRUE,
    show_comparisons BOOLEAN DEFAULT TRUE,
    max_vendor_suggestions INT DEFAULT 5,

    -- Privacy preferences
    save_conversation_history BOOLEAN DEFAULT TRUE,
    allow_analytics BOOLEAN DEFAULT TRUE,

    -- Notification preferences
    notify_on_vendor_response BOOLEAN DEFAULT TRUE,
    notify_on_price_drops BOOLEAN DEFAULT TRUE,
    notify_on_availability BOOLEAN DEFAULT TRUE,

    -- Custom preferences
    custom_preferences JSONB DEFAULT '{}',

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Conversation templates for common scenarios
CREATE TABLE IF NOT EXISTS conversation_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Template content
    event_type VARCHAR(50),
    initial_slots JSONB DEFAULT '{}',
    suggested_questions JSONB DEFAULT '[]',

    -- Usage tracking
    usage_count INT NOT NULL DEFAULT 0,
    success_rate DECIMAL(3,2),

    -- Status
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_templates_event_type ON conversation_templates(event_type);
CREATE INDEX IF NOT EXISTS idx_templates_active ON conversation_templates(is_active) WHERE is_active = TRUE;

-- Quick replies library
CREATE TABLE IF NOT EXISTS quick_replies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    intent VARCHAR(50) NOT NULL,
    context_state VARCHAR(50),

    -- Reply content
    title VARCHAR(100) NOT NULL,
    payload VARCHAR(200) NOT NULL,
    icon VARCHAR(50),

    -- Targeting
    target_audience VARCHAR(50) DEFAULT 'all', -- 'all', 'new_users', 'returning_users'
    priority INT DEFAULT 50,

    -- Performance tracking
    display_count INT DEFAULT 0,
    click_count INT DEFAULT 0,
    click_through_rate DECIMAL(5,2),

    -- Status
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quick_replies_intent ON quick_replies(intent);
CREATE INDEX IF NOT EXISTS idx_quick_replies_state ON quick_replies(context_state);
CREATE INDEX IF NOT EXISTS idx_quick_replies_active ON quick_replies(is_active) WHERE is_active = TRUE;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update triggers
CREATE TRIGGER update_conversations_updated_at BEFORE UPDATE ON conversations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_conversation_analytics_updated_at BEFORE UPDATE ON conversation_analytics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_conv_prefs_updated_at BEFORE UPDATE ON user_conversation_preferences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_conv_templates_updated_at BEFORE UPDATE ON conversation_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_quick_replies_updated_at BEFORE UPDATE ON quick_replies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE conversations IS 'EventGPT conversation sessions with full history and state';
COMMENT ON TABLE conversation_analytics IS 'Analytics and metrics for conversation performance';
COMMENT ON TABLE intent_classification_logs IS 'Intent classification results for ML training';
COMMENT ON TABLE entity_extraction_logs IS 'Entity extraction results for ML training';
COMMENT ON TABLE user_conversation_preferences IS 'User preferences for EventGPT interactions';
COMMENT ON TABLE conversation_templates IS 'Pre-defined templates for common event types';
COMMENT ON TABLE quick_replies IS 'Quick reply suggestions for different conversation contexts';
