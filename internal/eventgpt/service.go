// Package eventgpt provides the service layer for EventGPT conversational AI
package eventgpt

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/api/eventgpt"
)

// Service provides EventGPT functionality
type Service struct {
	db            *pgxpool.Pool
	cache         *redis.Client
	logger        *zap.Logger
	dialogManager *eventgpt.DialogManager
	api           *eventgpt.EventGPTAPI
}

// Config holds service configuration
type Config struct {
	MaxConversationTurns int
	CacheExpiryMinutes   int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxConversationTurns: 100,
		CacheExpiryMinutes:   30,
	}
}

// NewService creates a new EventGPT service
func NewService(db *pgxpool.Pool, cache *redis.Client, logger *zap.Logger) *Service {
	// Initialize NLU components
	intentClassifier := &eventgpt.IntentClassifier{}
	intentClassifier = &eventgpt.IntentClassifier{} // Will set fallback rules

	entityExtractor := eventgpt.NewEntityExtractor()

	slotFiller := &eventgpt.SlotFiller{}

	nlu := &eventgpt.NLUEngine{}

	// Initialize Dialog Manager components
	responseGen := eventgpt.NewResponseGenerator(db)

	actionExecutor := &eventgpt.ActionExecutor{}

	memoryManager := &eventgpt.MemoryManager{}

	contextManager := &eventgpt.ContextManager{}

	// Create Dialog Manager
	dialogManager := &eventgpt.DialogManager{}

	// Create API
	api := &eventgpt.EventGPTAPI{}

	service := &Service{
		db:            db,
		cache:         cache,
		logger:        logger,
		dialogManager: dialogManager,
		api:           api,
	}

	logger.Info("EventGPT service initialized successfully")

	return service
}

// StartConversation creates a new conversation
func (s *Service) StartConversation(ctx context.Context, userID uuid.UUID, channel eventgpt.Channel) (*eventgpt.Conversation, error) {
	conv := &eventgpt.Conversation{
		ID:                uuid.New(),
		UserID:            userID,
		SessionType:       eventgpt.SessionGeneralInquiry,
		ConversationState: eventgpt.StateWelcome,
		SlotValues:        make(map[string]eventgpt.SlotValue),
		Messages:          []eventgpt.Message{},
		TurnCount:         0,
		ShortTermMemory:   make(map[string]interface{}),
		Language:          "en",
		Channel:           channel,
	}

	s.logger.Info("Started new conversation",
		zap.String("conversation_id", conv.ID.String()),
		zap.String("user_id", userID.String()),
	)

	return conv, nil
}

// SendMessage sends a message to an existing conversation
func (s *Service) SendMessage(ctx context.Context, userID uuid.UUID, req eventgpt.ChatRequest) (*eventgpt.ChatResponse, error) {
	s.logger.Info("Processing chat message",
		zap.String("user_id", userID.String()),
		zap.Int("message_length", len(req.Message)),
	)

	response, err := s.api.Chat(ctx, userID, req)
	if err != nil {
		s.logger.Error("Failed to process message",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		return nil, err
	}

	return response, nil
}

// GetConversation retrieves a conversation by ID
func (s *Service) GetConversation(ctx context.Context, conversationID uuid.UUID) (*eventgpt.Conversation, error) {
	s.logger.Info("Fetching conversation",
		zap.String("conversation_id", conversationID.String()),
	)

	conv, err := s.api.LoadConversation(ctx, conversationID)
	if err != nil {
		s.logger.Error("Failed to load conversation",
			zap.Error(err),
			zap.String("conversation_id", conversationID.String()),
		)
		return nil, err
	}

	return conv, nil
}

// EndConversation marks a conversation as ended
func (s *Service) EndConversation(ctx context.Context, conversationID uuid.UUID) error {
	s.logger.Info("Ending conversation",
		zap.String("conversation_id", conversationID.String()),
	)

	// Load conversation
	conv, err := s.api.LoadConversation(ctx, conversationID)
	if err != nil {
		return err
	}

	// Mark as completed
	conv.ConversationState = eventgpt.StateCompleted

	// Save
	// This would call the dialog manager's save method

	return nil
}

// GetUserConversations retrieves all conversations for a user
func (s *Service) GetUserConversations(ctx context.Context, userID uuid.UUID, limit int) ([]eventgpt.Conversation, error) {
	s.logger.Info("Fetching user conversations",
		zap.String("user_id", userID.String()),
		zap.Int("limit", limit),
	)

	query := `
		SELECT id, user_id, event_id, session_type,
		       conversation_state, started_at, last_message_at
		FROM conversations
		WHERE user_id = $1
		ORDER BY last_message_at DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []eventgpt.Conversation
	for rows.Next() {
		var conv eventgpt.Conversation
		if err := rows.Scan(
			&conv.ID, &conv.UserID, &conv.EventID, &conv.SessionType,
			&conv.ConversationState, &conv.StartedAt, &conv.LastMessageAt,
		); err != nil {
			s.logger.Error("Failed to scan conversation", zap.Error(err))
			continue
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}
