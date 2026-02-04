// EventGPT - Conversational AI Event Planner
// Copyright (c) 2024 BillyRonks Global Limited. All rights reserved.

package eventgpt

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// =============================================================================
// TYPES & ENUMS
// =============================================================================

// Intent represents the user's intention in the conversation
type Intent string

const (
	IntentCreateEvent    Intent = "create_event"
	IntentFindVendor     Intent = "find_vendor"
	IntentGetQuote       Intent = "get_quote"
	IntentBookService    Intent = "book_service"
	IntentCompareOptions Intent = "compare_options"
	IntentCheckAvailability Intent = "check_availability"
	IntentModifyEvent    Intent = "modify_event"
	IntentAskQuestion    Intent = "ask_question"
	IntentUnknown        Intent = "unknown"
)

// ConversationState represents the current state of the conversation
type ConversationState string

const (
	StateInitial          ConversationState = "initial"
	StateGatheringDetails ConversationState = "gathering_details"
	StateShowingOptions   ConversationState = "showing_options"
	StateConfirming       ConversationState = "confirming"
	StateCompleted        ConversationState = "completed"
	StateEnded            ConversationState = "ended"
)

// Slot represents a piece of information we need to gather
type Slot string

const (
	SlotEventType    Slot = "event_type"
	SlotEventDate    Slot = "event_date"
	SlotLocation     Slot = "location"
	SlotGuestCount   Slot = "guest_count"
	SlotBudget       Slot = "budget"
	SlotVendorType   Slot = "vendor_type"
	SlotPreferences  Slot = "preferences"
)

// Message represents a single message in the conversation
type Message struct {
	ID        uuid.UUID              `json:"id"`
	Role      string                 `json:"role"` // "user" or "assistant"
	Content   string                 `json:"content"`
	Intent    Intent                 `json:"intent,omitempty"`
	Slots     map[Slot]interface{}   `json:"slots,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Conversation represents a chat session
type Conversation struct {
	ID            uuid.UUID              `json:"id"`
	UserID        uuid.UUID              `json:"user_id"`
	State         ConversationState      `json:"state"`
	Messages      []Message              `json:"messages"`
	Slots         map[Slot]interface{}   `json:"slots"`
	Context       map[string]interface{} `json:"context"`
	TurnCount     int                    `json:"turn_count"`
	StartedAt     time.Time              `json:"started_at"`
	LastMessageAt time.Time              `json:"last_message_at"`
	EndedAt       *time.Time             `json:"ended_at,omitempty"`
}

// Config holds EventGPT configuration
type Config struct {
	ClaudeAPIKey      string
	ClaudeModel       string
	MaxTokens         int
	Temperature       float64
	ConversationTTL   time.Duration
}

// Service handles EventGPT business logic
type Service struct {
	db     *pgxpool.Pool
	cache  *redis.Client
	config *Config
	logger *zap.Logger
}

// =============================================================================
// CONSTRUCTOR
// =============================================================================

// NewService creates a new EventGPT service
func NewService(db *pgxpool.Pool, cache *redis.Client, config *Config, logger *zap.Logger) *Service {
	if config == nil {
		config = &Config{
			ClaudeModel:     "claude-3-5-sonnet-20241022",
			MaxTokens:       1024,
			Temperature:     0.7,
			ConversationTTL: 24 * time.Hour,
		}
	}

	return &Service{
		db:     db,
		cache:  cache,
		config: config,
		logger: logger,
	}
}

// =============================================================================
// CONVERSATION MANAGEMENT
// =============================================================================

// StartConversation creates a new conversation
func (s *Service) StartConversation(ctx context.Context, userID uuid.UUID) (*Conversation, error) {
	conversation := &Conversation{
		ID:            uuid.New(),
		UserID:        userID,
		State:         StateInitial,
		Messages:      []Message{},
		Slots:         make(map[Slot]interface{}),
		Context:       make(map[string]interface{}),
		TurnCount:     0,
		StartedAt:     time.Now(),
		LastMessageAt: time.Now(),
	}

	// Save to database
	messagesJSON, _ := json.Marshal(conversation.Messages)
	slotsJSON, _ := json.Marshal(conversation.Slots)
	contextJSON, _ := json.Marshal(conversation.Context)

	query := `
		INSERT INTO conversations (
			id, user_id, conversation_state, messages, slots, context,
			turn_count, started_at, last_message_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := s.db.Exec(ctx, query,
		conversation.ID,
		conversation.UserID,
		conversation.State,
		messagesJSON,
		slotsJSON,
		contextJSON,
		conversation.TurnCount,
		conversation.StartedAt,
		conversation.LastMessageAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	// Add welcome message
	welcomeMsg := s.generateWelcomeMessage()
	conversation.Messages = append(conversation.Messages, welcomeMsg)

	return conversation, nil
}

// ProcessMessage handles a user message and generates a response
func (s *Service) ProcessMessage(ctx context.Context, conversationID uuid.UUID, userMessage string) (*Message, error) {
	// Get conversation from database
	conversation, err := s.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if conversation.State == StateEnded {
		return nil, fmt.Errorf("conversation has ended")
	}

	// Create user message
	userMsg := Message{
		ID:        uuid.New(),
		Role:      "user",
		Content:   userMessage,
		Timestamp: time.Now(),
	}

	// Classify intent
	intent := s.classifyIntent(userMessage)
	userMsg.Intent = intent

	// Extract entities/slots
	extractedSlots := s.extractSlots(userMessage, intent)
	userMsg.Slots = extractedSlots

	// Update conversation slots
	for slot, value := range extractedSlots {
		conversation.Slots[slot] = value
	}

	// Add user message to conversation
	conversation.Messages = append(conversation.Messages, userMsg)
	conversation.TurnCount++

	// Generate assistant response
	assistantMsg, err := s.generateResponse(ctx, conversation, userMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// Add assistant message to conversation
	conversation.Messages = append(conversation.Messages, *assistantMsg)
	conversation.LastMessageAt = time.Now()

	// Update conversation state
	conversation.State = s.determineNextState(conversation)

	// Save updated conversation
	if err := s.updateConversation(ctx, conversation); err != nil {
		s.logger.Error("Failed to update conversation", zap.Error(err))
	}

	return assistantMsg, nil
}

// GetConversation retrieves a conversation by ID
func (s *Service) GetConversation(ctx context.Context, conversationID uuid.UUID) (*Conversation, error) {
	query := `
		SELECT id, user_id, conversation_state, messages, slots, context,
		       turn_count, started_at, last_message_at, ended_at
		FROM conversations
		WHERE id = $1
	`

	var conversation Conversation
	var messagesJSON, slotsJSON, contextJSON []byte

	err := s.db.QueryRow(ctx, query, conversationID).Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.State,
		&messagesJSON,
		&slotsJSON,
		&contextJSON,
		&conversation.TurnCount,
		&conversation.StartedAt,
		&conversation.LastMessageAt,
		&conversation.EndedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// Unmarshal JSON fields
	json.Unmarshal(messagesJSON, &conversation.Messages)
	json.Unmarshal(slotsJSON, &conversation.Slots)
	json.Unmarshal(contextJSON, &conversation.Context)

	return &conversation, nil
}

// EndConversation marks a conversation as ended
func (s *Service) EndConversation(ctx context.Context, conversationID uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE conversations
		SET conversation_state = $1, ended_at = $2
		WHERE id = $3
	`

	_, err := s.db.Exec(ctx, query, StateEnded, now, conversationID)
	if err != nil {
		return fmt.Errorf("failed to end conversation: %w", err)
	}

	return nil
}

// =============================================================================
// INTENT CLASSIFICATION
// =============================================================================

// classifyIntent determines the user's intent from their message
func (s *Service) classifyIntent(message string) Intent {
	messageLower := strings.ToLower(message)

	// Intent patterns
	intentPatterns := map[Intent][]string{
		IntentCreateEvent: {
			"planning", "organize", "want to", "need to", "i'm planning",
			"help me plan", "create event", "new event",
		},
		IntentFindVendor: {
			"find", "looking for", "need a", "recommend", "suggest",
			"vendor", "photographer", "caterer", "DJ", "decorator",
		},
		IntentGetQuote: {
			"quote", "cost", "price", "how much", "budget", "estimate",
		},
		IntentBookService: {
			"book", "reserve", "hire", "schedule", "confirm",
		},
		IntentCompareOptions: {
			"compare", "difference", "which is better", "versus", "vs",
		},
		IntentCheckAvailability: {
			"available", "availability", "free", "open", "can you",
		},
		IntentModifyEvent: {
			"change", "update", "modify", "edit", "reschedule",
		},
	}

	// Check patterns
	for intent, patterns := range intentPatterns {
		for _, pattern := range patterns {
			if strings.Contains(messageLower, pattern) {
				return intent
			}
		}
	}

	// Check for questions
	if strings.HasSuffix(messageLower, "?") || strings.HasPrefix(messageLower, "what") ||
		strings.HasPrefix(messageLower, "how") || strings.HasPrefix(messageLower, "when") ||
		strings.HasPrefix(messageLower, "where") || strings.HasPrefix(messageLower, "why") {
		return IntentAskQuestion
	}

	return IntentUnknown
}

// =============================================================================
// SLOT EXTRACTION
// =============================================================================

// extractSlots extracts entities from the user message
func (s *Service) extractSlots(message string, intent Intent) map[Slot]interface{} {
	slots := make(map[Slot]interface{})

	messageLower := strings.ToLower(message)

	// Extract event type
	eventTypes := map[string]string{
		"wedding":     "wedding",
		"birthday":    "birthday",
		"corporate":   "corporate_event",
		"conference":  "conference",
		"party":       "party",
		"anniversary": "anniversary",
		"graduation":  "graduation",
		"baby shower": "baby_shower",
	}

	for keyword, eventType := range eventTypes {
		if strings.Contains(messageLower, keyword) {
			slots[SlotEventType] = eventType
			break
		}
	}

	// Extract guest count
	guestCountRegex := regexp.MustCompile(`(\d+)\s*(guests?|people|persons?)`)
	if matches := guestCountRegex.FindStringSubmatch(messageLower); len(matches) > 1 {
		slots[SlotGuestCount] = matches[1]
	}

	// Extract budget
	budgetRegex := regexp.MustCompile(`(?:₦|naira|ngn)?\s*(\d+(?:,\d{3})*(?:\.\d{2})?)\s*(?:million|k|thousand)?`)
	if matches := budgetRegex.FindStringSubmatch(messageLower); len(matches) > 1 {
		slots[SlotBudget] = matches[1]
	}

	// Extract location (Nigerian cities)
	locations := []string{
		"lagos", "abuja", "ibadan", "port harcourt", "kano", "kaduna",
		"benin", "enugu", "jos", "ilorin", "abeokuta", "oyo", "warri",
	}

	for _, location := range locations {
		if strings.Contains(messageLower, location) {
			slots[SlotLocation] = strings.Title(location)
			break
		}
	}

	// Extract vendor type
	vendorTypes := map[string]string{
		"photographer": "photography",
		"caterer":      "catering",
		"dj":           "entertainment",
		"decorator":    "decoration",
		"venue":        "venue",
		"makeup":       "makeup",
		"planner":      "event_planning",
	}

	for keyword, vendorType := range vendorTypes {
		if strings.Contains(messageLower, keyword) {
			slots[SlotVendorType] = vendorType
			break
		}
	}

	// Extract date patterns (simple)
	dateRegex := regexp.MustCompile(`(?:january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{1,2}`)
	if matches := dateRegex.FindString(messageLower); matches != "" {
		slots[SlotEventDate] = matches
	}

	// Month extraction
	monthRegex := regexp.MustCompile(`(?:next|in)\s+(january|february|march|april|may|june|july|august|september|october|november|december)`)
	if matches := monthRegex.FindStringSubmatch(messageLower); len(matches) > 1 {
		slots[SlotEventDate] = matches[1]
	}

	return slots
}

// =============================================================================
// RESPONSE GENERATION
// =============================================================================

// generateResponse creates an assistant response
func (s *Service) generateResponse(ctx context.Context, conversation *Conversation, userMsg Message) (*Message, error) {
	response := &Message{
		ID:        uuid.New(),
		Role:      "assistant",
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Generate response based on intent and state
	switch userMsg.Intent {
	case IntentCreateEvent:
		response.Content = s.handleCreateEvent(conversation, userMsg)
	case IntentFindVendor:
		response.Content = s.handleFindVendor(conversation, userMsg)
	case IntentGetQuote:
		response.Content = s.handleGetQuote(conversation, userMsg)
	case IntentCompareOptions:
		response.Content = s.handleCompareOptions(conversation, userMsg)
	case IntentAskQuestion:
		response.Content = s.handleQuestion(conversation, userMsg)
	default:
		response.Content = s.handleUnknown(conversation, userMsg)
	}

	// Add quick replies based on state
	response.Metadata["quick_replies"] = s.generateQuickReplies(conversation)

	return response, nil
}

// handleCreateEvent handles event creation intent
func (s *Service) handleCreateEvent(conversation *Conversation, userMsg Message) string {
	// Check what slots we have
	missingSlots := s.getMissingSlots(conversation.Slots)

	if len(missingSlots) == 0 {
		return s.summarizeEvent(conversation)
	}

	// Ask for the next missing slot
	nextSlot := missingSlots[0]
	return s.askForSlot(nextSlot, conversation)
}

// handleFindVendor handles vendor search intent
func (s *Service) handleFindVendor(conversation *Conversation, userMsg Message) string {
	vendorType, hasVendorType := conversation.Slots[SlotVendorType].(string)
	location, hasLocation := conversation.Slots[SlotLocation].(string)

	if !hasVendorType {
		return "What type of vendor are you looking for? For example: photographer, caterer, DJ, decorator, or venue?"
	}

	if !hasLocation {
		return fmt.Sprintf("Great! I can help you find a %s. What city or area are you looking in?", vendorType)
	}

	return fmt.Sprintf("Perfect! I'm searching for the best %s vendors in %s. Based on your requirements, here are my top recommendations:\n\n"+
		"(This will integrate with the vendor search service to show real results)", vendorType, location)
}

// handleGetQuote handles quote request intent
func (s *Service) handleGetQuote(conversation *Conversation, userMsg Message) string {
	return "I can help you get quotes! To provide accurate estimates, I'll need a few details:\n\n" +
		"1. What service do you need?\n" +
		"2. What's your event date?\n" +
		"3. How many guests are you expecting?"
}

// handleCompareOptions handles comparison intent
func (s *Service) handleCompareOptions(conversation *Conversation, userMsg Message) string {
	return "I'd be happy to help you compare options! What would you like to compare?\n\n" +
		"• Different vendors\n" +
		"• Service packages\n" +
		"• Price points\n" +
		"• Availability"
}

// handleQuestion handles general questions
func (s *Service) handleQuestion(conversation *Conversation, userMsg Message) string {
	return "That's a great question! Let me help you with that. " +
		"Our platform offers comprehensive event planning services including vendor matching, " +
		"price comparison, and booking management. What specific aspect would you like to know more about?"
}

// handleUnknown handles unknown intents
func (s *Service) handleUnknown(conversation *Conversation, userMsg Message) string {
	return "I'm here to help you plan your event! I can assist with:\n\n" +
		"• Finding and booking vendors\n" +
		"• Getting price quotes\n" +
		"• Comparing service options\n" +
		"• Planning your event timeline\n\n" +
		"What would you like help with today?"
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// generateWelcomeMessage creates the initial welcome message
func (s *Service) generateWelcomeMessage() Message {
	return Message{
		ID:        uuid.New(),
		Role:      "assistant",
		Content:   "👋 Welcome to EventGPT! I'm your AI event planning assistant.\n\n" +
			"I can help you plan your perfect event by:\n" +
			"• Finding the right vendors\n" +
			"• Getting competitive quotes\n" +
			"• Managing your budget\n" +
			"• Coordinating timelines\n\n" +
			"Tell me about your event and let's get started!",
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{
			"quick_replies": []string{"Plan a wedding", "Find vendors", "Get quotes", "Just browsing"},
		},
	}
}

// getMissingSlots returns slots that haven't been filled yet
func (s *Service) getMissingSlots(slots map[Slot]interface{}) []Slot {
	requiredSlots := []Slot{SlotEventType, SlotEventDate, SlotLocation, SlotGuestCount, SlotBudget}
	missing := []Slot{}

	for _, slot := range requiredSlots {
		if _, exists := slots[slot]; !exists {
			missing = append(missing, slot)
		}
	}

	return missing
}

// askForSlot generates a question for a missing slot
func (s *Service) askForSlot(slot Slot, conversation *Conversation) string {
	questions := map[Slot]string{
		SlotEventType:  "What type of event are you planning? (e.g., wedding, birthday, corporate event, conference)",
		SlotEventDate:  "When is your event scheduled? Please provide a date or month.",
		SlotLocation:   "Where will your event take place? Which city or area?",
		SlotGuestCount: "How many guests are you expecting?",
		SlotBudget:     "What's your budget for this event?",
		SlotVendorType: "What type of vendor are you looking for?",
	}

	return questions[slot]
}

// summarizeEvent creates a summary of the event details
func (s *Service) summarizeEvent(conversation *Conversation) string {
	eventType, _ := conversation.Slots[SlotEventType].(string)
	date, _ := conversation.Slots[SlotEventDate].(string)
	location, _ := conversation.Slots[SlotLocation].(string)
	guestCount, _ := conversation.Slots[SlotGuestCount].(string)
	budget, _ := conversation.Slots[SlotBudget].(string)

	return fmt.Sprintf("Perfect! Let me summarize your event:\n\n"+
		"• Event Type: %s\n"+
		"• Date: %s\n"+
		"• Location: %s\n"+
		"• Guests: %s\n"+
		"• Budget: ₦%s\n\n"+
		"Would you like me to find vendors for your event?",
		eventType, date, location, guestCount, budget)
}

// generateQuickReplies creates contextual quick reply suggestions
func (s *Service) generateQuickReplies(conversation *Conversation) []string {
	switch conversation.State {
	case StateInitial:
		return []string{"Plan an event", "Find vendors", "Get quotes"}
	case StateGatheringDetails:
		return []string{"Yes", "No", "Tell me more", "Skip"}
	case StateShowingOptions:
		return []string{"Show more", "Compare", "Book now", "Get quote"}
	default:
		return []string{"Yes", "No", "Help"}
	}
}

// determineNextState calculates the next conversation state
func (s *Service) determineNextState(conversation *Conversation) ConversationState {
	missingSlots := s.getMissingSlots(conversation.Slots)

	if len(missingSlots) > 0 {
		return StateGatheringDetails
	}

	if conversation.State == StateGatheringDetails {
		return StateShowingOptions
	}

	return conversation.State
}

// updateConversation saves conversation changes to database
func (s *Service) updateConversation(ctx context.Context, conversation *Conversation) error {
	messagesJSON, _ := json.Marshal(conversation.Messages)
	slotsJSON, _ := json.Marshal(conversation.Slots)
	contextJSON, _ := json.Marshal(conversation.Context)

	query := `
		UPDATE conversations
		SET conversation_state = $1, messages = $2, slots = $3, context = $4,
		    turn_count = $5, last_message_at = $6
		WHERE id = $7
	`

	_, err := s.db.Exec(ctx, query,
		conversation.State,
		messagesJSON,
		slotsJSON,
		contextJSON,
		conversation.TurnCount,
		conversation.LastMessageAt,
		conversation.ID,
	)

	return err
}
