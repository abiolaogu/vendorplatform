// Package eventgpt provides the service layer for conversational AI event planning
package eventgpt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Service handles EventGPT conversation business logic
type Service struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// NewService creates a new EventGPT service instance
func NewService(db *pgxpool.Pool, cache *redis.Client) *Service {
	return &Service{
		db:    db,
		cache: cache,
	}
}

// Conversation represents a chat session with EventGPT
type Conversation struct {
	ID                uuid.UUID              `json:"id"`
	UserID            uuid.UUID              `json:"user_id"`
	EventID           *uuid.UUID             `json:"event_id,omitempty"`
	SessionType       string                 `json:"session_type"`
	CurrentIntent     string                 `json:"current_intent,omitempty"`
	ConversationState string                 `json:"conversation_state"`
	SlotValues        map[string]interface{} `json:"slot_values,omitempty"`
	ShortTermMemory   map[string]interface{} `json:"short_term_memory,omitempty"`
	Channel           string                 `json:"channel"`
	LastMessageAt     *time.Time             `json:"last_message_at,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	Messages          []Message              `json:"messages,omitempty"`
}

// Message represents a single message in the conversation
type Message struct {
	ID             uuid.UUID   `json:"id"`
	ConversationID uuid.UUID   `json:"conversation_id"`
	Role           string      `json:"role"` // 'user', 'assistant', 'system'
	Content        string      `json:"content"`
	Intent         *string     `json:"intent,omitempty"`
	Entities       interface{} `json:"entities,omitempty"`
	Confidence     *float64    `json:"confidence,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}

// CreateConversationRequest represents a request to start a conversation
type CreateConversationRequest struct {
	UserID      uuid.UUID  `json:"user_id"`
	EventID     *uuid.UUID `json:"event_id,omitempty"`
	SessionType string     `json:"session_type,omitempty"` // defaults to 'general'
	Channel     string     `json:"channel,omitempty"`      // defaults to 'web'
	Language    string     `json:"language,omitempty"`     // defaults to 'en'
}

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	Content string                 `json:"content"`
	Intent  *string                `json:"intent,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// MessageResponse represents the response to a sent message
type MessageResponse struct {
	ConversationID uuid.UUID              `json:"conversation_id"`
	UserMessage    Message                `json:"user_message"`
	AssistantReply Message                `json:"assistant_reply"`
	SlotValues     map[string]interface{} `json:"slot_values,omitempty"`
	NextActions    []string               `json:"next_actions,omitempty"`
}

// CreateConversation starts a new conversation
func (s *Service) CreateConversation(ctx context.Context, req *CreateConversationRequest) (*Conversation, error) {
	// Default values
	if req.SessionType == "" {
		req.SessionType = "general_inquiry"
	}
	if req.Channel == "" {
		req.Channel = "web"
	}

	id := uuid.New()
	now := time.Now()

	// Insert conversation
	query := `
		INSERT INTO conversations (
			id, user_id, event_id, session_type, conversation_state,
			slot_values, short_term_memory, channel, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, user_id, event_id, session_type, current_intent,
				  conversation_state, slot_values, short_term_memory, channel,
				  last_message_at, created_at, updated_at
	`

	slotValuesJSON := "{}"
	memoryJSON := "{}"

	conv := &Conversation{}
	var slotValuesStr, memoryStr string

	err := s.db.QueryRow(ctx, query,
		id, req.UserID, req.EventID, req.SessionType, "welcome",
		slotValuesJSON, memoryJSON, req.Channel, now, now,
	).Scan(
		&conv.ID, &conv.UserID, &conv.EventID, &conv.SessionType, &conv.CurrentIntent,
		&conv.ConversationState, &slotValuesStr, &memoryStr, &conv.Channel,
		&conv.LastMessageAt, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	// Parse JSON fields
	if slotValuesStr != "" {
		json.Unmarshal([]byte(slotValuesStr), &conv.SlotValues)
	}
	if memoryStr != "" {
		json.Unmarshal([]byte(memoryStr), &conv.ShortTermMemory)
	}

	// Send initial welcome message
	welcomeMessage := s.generateWelcomeMessage(req.SessionType)
	_, err = s.addMessage(ctx, conv.ID, "assistant", welcomeMessage, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to add welcome message: %w", err)
	}

	return conv, nil
}

// GetConversation retrieves a conversation with its messages
func (s *Service) GetConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (*Conversation, error) {
	// Get conversation
	query := `
		SELECT id, user_id, event_id, session_type, current_intent,
			   conversation_state, slot_values, short_term_memory, channel,
			   last_message_at, created_at, updated_at
		FROM conversations
		WHERE id = $1 AND user_id = $2
	`

	conv := &Conversation{}
	var slotValuesStr, memoryStr string

	err := s.db.QueryRow(ctx, query, conversationID, userID).Scan(
		&conv.ID, &conv.UserID, &conv.EventID, &conv.SessionType, &conv.CurrentIntent,
		&conv.ConversationState, &slotValuesStr, &memoryStr, &conv.Channel,
		&conv.LastMessageAt, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// Parse JSON fields
	if slotValuesStr != "" {
		json.Unmarshal([]byte(slotValuesStr), &conv.SlotValues)
	}
	if memoryStr != "" {
		json.Unmarshal([]byte(memoryStr), &conv.ShortTermMemory)
	}

	// Get messages
	messagesQuery := `
		SELECT id, conversation_id, role, content, intent, entities, confidence, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`

	rows, err := s.db.Query(ctx, messagesQuery, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var entitiesStr *string

		err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content,
			&msg.Intent, &entitiesStr, &msg.Confidence, &msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if entitiesStr != nil {
			json.Unmarshal([]byte(*entitiesStr), &msg.Entities)
		}

		messages = append(messages, msg)
	}

	conv.Messages = messages
	return conv, nil
}

// SendMessage processes a user message and generates a response
func (s *Service) SendMessage(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, req *SendMessageRequest) (*MessageResponse, error) {
	// Verify conversation exists and belongs to user
	conv, err := s.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}

	// Add user message
	userMsg, err := s.addMessage(ctx, conversationID, "user", req.Content, req.Intent, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to add user message: %w", err)
	}

	// Process message and generate response
	// For now, this is a simple echo with basic intent detection
	// TODO: Integrate with Claude API for advanced NLU
	assistantContent := s.generateResponse(conv, req.Content)

	// Add assistant message
	assistantMsg, err := s.addMessage(ctx, conversationID, "assistant", assistantContent, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to add assistant message: %w", err)
	}

	// Update conversation state
	err = s.updateConversationState(ctx, conversationID, "gathering_info")
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation state: %w", err)
	}

	return &MessageResponse{
		ConversationID: conversationID,
		UserMessage:    *userMsg,
		AssistantReply: *assistantMsg,
		SlotValues:     conv.SlotValues,
		NextActions:    []string{"continue_conversation"},
	}, nil
}

// EndConversation marks a conversation as ended
func (s *Service) EndConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE conversations
		SET conversation_state = 'completed', updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`

	result, err := s.db.Exec(ctx, query, conversationID, userID)
	if err != nil {
		return fmt.Errorf("failed to end conversation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found or already ended")
	}

	return nil
}

// addMessage adds a message to a conversation
func (s *Service) addMessage(ctx context.Context, conversationID uuid.UUID, role, content string, intent *string, entities interface{}, confidence *float64) (*Message, error) {
	id := uuid.New()
	now := time.Now()

	var entitiesJSON *string
	if entities != nil {
		data, _ := json.Marshal(entities)
		str := string(data)
		entitiesJSON = &str
	}

	query := `
		INSERT INTO messages (id, conversation_id, role, content, intent, entities, confidence, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, conversation_id, role, content, intent, entities, confidence, created_at
	`

	msg := &Message{}
	var entitiesStr *string

	err := s.db.QueryRow(ctx, query,
		id, conversationID, role, content, intent, entitiesJSON, confidence, now,
	).Scan(
		&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content,
		&msg.Intent, &entitiesStr, &msg.Confidence, &msg.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if entitiesStr != nil {
		json.Unmarshal([]byte(*entitiesStr), &msg.Entities)
	}

	// Update last_message_at in conversation
	_, err = s.db.Exec(ctx, `UPDATE conversations SET last_message_at = $1, updated_at = $1 WHERE id = $2`, now, conversationID)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// updateConversationState updates the conversation state
func (s *Service) updateConversationState(ctx context.Context, conversationID uuid.UUID, state string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE conversations SET conversation_state = $1, updated_at = NOW() WHERE id = $2`,
		state, conversationID,
	)
	return err
}

// generateWelcomeMessage generates a welcome message based on session type
func (s *Service) generateWelcomeMessage(sessionType string) string {
	welcomeMessages := map[string]string{
		"new_event": "Hi! I'm EventGPT, your AI event planning assistant. I'm excited to help you plan your event! What kind of event are you planning?",
		"event_planning": "Welcome back! Let's continue planning your event. What would you like to work on today?",
		"vendor_search": "Hi! I can help you find the perfect vendors for your event. What type of vendor are you looking for?",
		"booking_help": "Hello! I'm here to help you with your booking. What questions do you have?",
		"general_inquiry": "Hi! I'm EventGPT, your AI event planning assistant. How can I help you today?",
		"support": "Hello! I'm here to help resolve your issue. Can you describe what you need assistance with?",
	}

	if msg, ok := welcomeMessages[sessionType]; ok {
		return msg
	}
	return "Hi! I'm EventGPT. How can I help you today?"
}

// generateResponse generates a response based on user input
// TODO: Replace with Claude API integration for advanced NLU
func (s *Service) generateResponse(conv *Conversation, userInput string) string {
	// Simple keyword-based response for MVP
	// This should be replaced with Claude API integration

	userInputLower := strings.ToLower(userInput)

	// Event type detection
	if strings.Contains(userInputLower, "wedding") {
		return "A wedding! How exciting! Can you tell me more about your wedding? When is the date, and roughly how many guests are you expecting?"
	}
	if strings.Contains(userInputLower, "birthday") {
		return "Planning a birthday party! That sounds fun. Who is the birthday for, and what kind of celebration are you thinking about?"
	}
	if strings.Contains(userInputLower, "corporate") || strings.Contains(userInputLower, "conference") {
		return "A corporate event - I can definitely help with that. What type of corporate event is this, and what are your main objectives?"
	}

	// Vendor search
	if strings.Contains(userInputLower, "photographer") || strings.Contains(userInputLower, "photo") {
		return "Looking for a photographer? I can help you find highly-rated photographers in your area. What's your event date and location?"
	}
	if strings.Contains(userInputLower, "caterer") || strings.Contains(userInputLower, "food") || strings.Contains(userInputLower, "catering") {
		return "Catering is such an important part of any event! What type of cuisine are you interested in, and how many guests will you be serving?"
	}

	// Budget questions
	if strings.Contains(userInputLower, "budget") || strings.Contains(userInputLower, "cost") || strings.Contains(userInputLower, "price") {
		return "Budget is definitely important to discuss. What's your overall budget range for this event? This will help me recommend vendors that match your needs."
	}

	// Default response
	return "I understand. Can you tell me more about what you're looking for? The more details you share, the better I can assist you."
}
