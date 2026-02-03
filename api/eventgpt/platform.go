// =============================================================================
// EVENTGPT - CONVERSATIONAL AI EVENT PLANNER
// Comprehensive Technical & Business Specification
// Version: 1.0.0
// =============================================================================

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
)

/*
================================================================================
SECTION 1: PRODUCT VISION & POSITIONING
================================================================================

EVENTGPT: Your AI Event Planning Partner

TAGLINE: "Plan your perfect event through conversation"

VISION:
EventGPT transforms event planning from a complex, multi-step process into a 
natural conversation. Users describe their event in plain language, and EventGPT
understands intent, asks clarifying questions, generates plans, finds vendors,
and coordinates everything—all through chat.

CORE VALUE PROPOSITION:
"Tell us about your event. We'll handle the rest."

TARGET USERS:
1. First-time event planners overwhelmed by complexity
2. Busy professionals who want efficiency
3. Budget-conscious users who need guidance
4. Users who prefer conversational interfaces over forms

KEY DIFFERENTIATORS:
1. Natural Language Understanding: No forms, just conversation
2. Contextual Memory: Remembers preferences across the planning journey
3. Proactive Suggestions: Anticipates needs before being asked
4. Real-Time Vendor Matching: Instant recommendations based on availability
5. Multi-Modal: Text, voice, and visual responses

INTERACTION PRINCIPLES:
1. Be Helpful, Not Pushy: Guide without overwhelming
2. Ask Smart Questions: Minimize back-and-forth with intelligent questions
3. Show, Don't Tell: Use visuals, comparisons, and examples
4. Explain Reasoning: Be transparent about recommendations
5. Graceful Handoff: Know when to involve humans

================================================================================
SECTION 2: CONVERSATION ARCHITECTURE
================================================================================
*/

// =============================================================================
// 2.1 CORE CONVERSATION TYPES
// =============================================================================

// Conversation represents a chat session with EventGPT
type Conversation struct {
	ID              uuid.UUID              `json:"id"`
	UserID          uuid.UUID              `json:"user_id"`
	
	// Context
	EventID         *uuid.UUID             `json:"event_id,omitempty"`
	SessionType     SessionType            `json:"session_type"`
	
	// State
	CurrentIntent   Intent                 `json:"current_intent"`
	ConversationState ConversationState    `json:"conversation_state"`
	SlotValues      map[string]SlotValue   `json:"slot_values"`
	
	// History
	Messages        []Message              `json:"messages"`
	TurnCount       int                    `json:"turn_count"`
	
	// Memory
	ShortTermMemory map[string]interface{} `json:"short_term_memory"`
	
	// Metadata
	Language        string                 `json:"language"`
	Channel         Channel                `json:"channel"`
	
	// Timestamps
	StartedAt       time.Time              `json:"started_at"`
	LastMessageAt   time.Time              `json:"last_message_at"`
	EndedAt         *time.Time             `json:"ended_at,omitempty"`
}

type SessionType string
const (
	SessionNewEvent       SessionType = "new_event"
	SessionEventPlanning  SessionType = "event_planning"
	SessionVendorSearch   SessionType = "vendor_search"
	SessionBookingHelp    SessionType = "booking_help"
	SessionGeneralInquiry SessionType = "general_inquiry"
	SessionSupport        SessionType = "support"
)

type ConversationState string
const (
	StateWelcome           ConversationState = "welcome"
	StateGatheringInfo     ConversationState = "gathering_info"
	StateConfirming        ConversationState = "confirming"
	StateRecommending      ConversationState = "recommending"
	StateComparing         ConversationState = "comparing"
	StateBooking           ConversationState = "booking"
	StateCompleted         ConversationState = "completed"
	StateHandoff           ConversationState = "handoff"
)

type Channel string
const (
	ChannelWeb      Channel = "web"
	ChannelMobile   Channel = "mobile"
	ChannelWhatsApp Channel = "whatsapp"
	ChannelVoice    Channel = "voice"
	ChannelAPI      Channel = "api"
)

// Message represents a single message in the conversation
type Message struct {
	ID              uuid.UUID              `json:"id"`
	Role            MessageRole            `json:"role"`
	Content         string                 `json:"content"`
	
	// Rich Content
	Attachments     []Attachment           `json:"attachments,omitempty"`
	QuickReplies    []QuickReply           `json:"quick_replies,omitempty"`
	Cards           []Card                 `json:"cards,omitempty"`
	Actions         []ActionButton         `json:"actions,omitempty"`
	
	// Metadata
	Intent          *Intent                `json:"intent,omitempty"`
	Entities        []Entity               `json:"entities,omitempty"`
	Confidence      float64                `json:"confidence,omitempty"`
	
	// Processing
	ProcessingTime  int64                  `json:"processing_time_ms,omitempty"`
	ModelUsed       string                 `json:"model_used,omitempty"`
	
	Timestamp       time.Time              `json:"timestamp"`
}

type MessageRole string
const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
)

// Attachment for images, documents, etc.
type Attachment struct {
	Type     string `json:"type"` // 'image', 'document', 'audio', 'video'
	URL      string `json:"url"`
	MimeType string `json:"mime_type,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

// QuickReply for suggested responses
type QuickReply struct {
	Title   string `json:"title"`
	Payload string `json:"payload"`
	Icon    string `json:"icon,omitempty"`
}

// Card for rich vendor/service displays
type Card struct {
	Type        string            `json:"type"` // 'vendor', 'service', 'bundle', 'comparison'
	Title       string            `json:"title"`
	Subtitle    string            `json:"subtitle,omitempty"`
	ImageURL    string            `json:"image_url,omitempty"`
	Description string            `json:"description,omitempty"`
	Price       *PriceDisplay     `json:"price,omitempty"`
	Rating      *RatingDisplay    `json:"rating,omitempty"`
	Actions     []ActionButton    `json:"actions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type PriceDisplay struct {
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Unit        string  `json:"unit,omitempty"` // 'per_hour', 'per_day', 'flat'
	OriginalAmt *float64 `json:"original_amount,omitempty"`
	Discount    string  `json:"discount,omitempty"`
}

type RatingDisplay struct {
	Score       float64 `json:"score"`
	Count       int     `json:"count"`
	Label       string  `json:"label,omitempty"`
}

// ActionButton for interactive elements
type ActionButton struct {
	Type    string `json:"type"` // 'url', 'postback', 'call', 'book'
	Title   string `json:"title"`
	Payload string `json:"payload"`
	URL     string `json:"url,omitempty"`
	Style   string `json:"style,omitempty"` // 'primary', 'secondary', 'danger'
}

// =============================================================================
// 2.2 NATURAL LANGUAGE UNDERSTANDING (NLU)
// =============================================================================

// Intent represents detected user intent
type Intent struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	Slots      map[string]SlotValue `json:"slots,omitempty"`
}

// Entity represents extracted entities
type Entity struct {
	Type       string      `json:"type"`
	Value      interface{} `json:"value"`
	Text       string      `json:"text"`
	StartPos   int         `json:"start_pos"`
	EndPos     int         `json:"end_pos"`
	Confidence float64     `json:"confidence"`
}

// SlotValue represents a filled conversation slot
type SlotValue struct {
	Value       interface{} `json:"value"`
	Source      string      `json:"source"` // 'user', 'inferred', 'default'
	Confidence  float64     `json:"confidence"`
	Timestamp   time.Time   `json:"timestamp"`
	Confirmed   bool        `json:"confirmed"`
}

// NLUEngine processes natural language input
type NLUEngine struct {
	db               *pgxpool.Pool
	intentClassifier *IntentClassifier
	entityExtractor  *EntityExtractor
	slotFiller       *SlotFiller
	contextManager   *ContextManager
}

// IntentClassifier classifies user intents
type IntentClassifier struct {
	// Model configuration
	modelEndpoint string
	fallbackRules []IntentRule
}

// IntentRule for rule-based fallback
type IntentRule struct {
	IntentName string
	Patterns   []string
	Keywords   []string
	Priority   int
}

// Common intents for event planning
var EventPlanningIntents = []IntentRule{
	{
		IntentName: "create_event",
		Patterns: []string{
			`(?i)(plan|organize|create|start|need help with|want to).*(wedding|birthday|party|event|celebration|funeral|graduation)`,
			`(?i)(getting married|having a party|throwing a celebration)`,
			`(?i)(i'm planning|we're planning|help me plan)`,
		},
		Keywords: []string{"plan", "organize", "wedding", "birthday", "party", "event"},
		Priority: 100,
	},
	{
		IntentName: "find_vendor",
		Patterns: []string{
			`(?i)(find|search|look for|need|recommend|suggest).*(vendor|photographer|caterer|dj|decorator|planner)`,
			`(?i)(who can|looking for someone to|need someone to)`,
		},
		Keywords: []string{"find", "vendor", "recommend", "suggest", "photographer", "caterer"},
		Priority: 90,
	},
	{
		IntentName: "get_quote",
		Patterns: []string{
			`(?i)(how much|what's the price|cost|quote|estimate|budget)`,
			`(?i)(what would it cost|price range|pricing)`,
		},
		Keywords: []string{"price", "cost", "quote", "budget", "estimate"},
		Priority: 85,
	},
	{
		IntentName: "book_service",
		Patterns: []string{
			`(?i)(book|reserve|hire|engage|confirm).*(vendor|service|photographer|caterer)`,
			`(?i)(i want to book|let's book|ready to book)`,
		},
		Keywords: []string{"book", "reserve", "hire", "confirm"},
		Priority: 95,
	},
	{
		IntentName: "compare_options",
		Patterns: []string{
			`(?i)(compare|difference|which is better|vs|versus|or)`,
			`(?i)(what are my options|show me alternatives)`,
		},
		Keywords: []string{"compare", "options", "alternatives", "difference"},
		Priority: 80,
	},
	{
		IntentName: "check_availability",
		Patterns: []string{
			`(?i)(available|availability|free|open).*(date|day|time)`,
			`(?i)(can they do|are they free|when can)`,
		},
		Keywords: []string{"available", "availability", "free", "date"},
		Priority: 85,
	},
	{
		IntentName: "update_preference",
		Patterns: []string{
			`(?i)(change|update|modify|actually|instead|prefer)`,
			`(?i)(i meant|i want to change|let me change)`,
		},
		Keywords: []string{"change", "update", "instead", "prefer", "actually"},
		Priority: 75,
	},
	{
		IntentName: "get_recommendation",
		Patterns: []string{
			`(?i)(what do you (suggest|recommend|think)|what should i|any suggestions)`,
			`(?i)(help me (choose|decide|pick)|which one)`,
		},
		Keywords: []string{"suggest", "recommend", "advice", "help", "decide"},
		Priority: 80,
	},
	{
		IntentName: "view_plan",
		Patterns: []string{
			`(?i)(show|view|see|display).*(plan|timeline|checklist|tasks|progress)`,
			`(?i)(what's the plan|where are we|my plan)`,
		},
		Keywords: []string{"plan", "timeline", "progress", "checklist"},
		Priority: 70,
	},
	{
		IntentName: "ask_question",
		Patterns: []string{
			`(?i)(what is|what are|how do|how does|why|when|where|who)`,
			`(?i)(can you explain|tell me about|i don't understand)`,
		},
		Keywords: []string{"what", "how", "why", "explain"},
		Priority: 50,
	},
	{
		IntentName: "greeting",
		Patterns: []string{
			`(?i)^(hi|hello|hey|good morning|good afternoon|good evening)`,
		},
		Keywords: []string{"hi", "hello", "hey"},
		Priority: 30,
	},
	{
		IntentName: "thanks",
		Patterns: []string{
			`(?i)(thank|thanks|thank you|appreciate|grateful)`,
		},
		Keywords: []string{"thank", "thanks", "appreciate"},
		Priority: 30,
	},
	{
		IntentName: "cancel",
		Patterns: []string{
			`(?i)(cancel|stop|quit|exit|nevermind|forget it)`,
		},
		Keywords: []string{"cancel", "stop", "quit", "nevermind"},
		Priority: 60,
	},
}

func (c *IntentClassifier) ClassifyIntent(ctx context.Context, text string, conversationContext *ConversationContext) (*Intent, error) {
	// First try rule-based classification for common patterns
	for _, rule := range c.fallbackRules {
		for _, pattern := range rule.Patterns {
			matched, _ := regexp.MatchString(pattern, text)
			if matched {
				return &Intent{
					Name:       rule.IntentName,
					Confidence: 0.9,
				}, nil
			}
		}
	}
	
	// Keyword-based fallback
	textLower := strings.ToLower(text)
	for _, rule := range c.fallbackRules {
		matchCount := 0
		for _, keyword := range rule.Keywords {
			if strings.Contains(textLower, keyword) {
				matchCount++
			}
		}
		if matchCount >= 2 || (matchCount == 1 && len(rule.Keywords) <= 2) {
			confidence := float64(matchCount) / float64(len(rule.Keywords)) * 0.8
			return &Intent{
				Name:       rule.IntentName,
				Confidence: confidence,
			}, nil
		}
	}
	
	// Default to general inquiry
	return &Intent{
		Name:       "ask_question",
		Confidence: 0.5,
	}, nil
}

// EntityExtractor extracts entities from text
type EntityExtractor struct {
	patterns map[string]*regexp.Regexp
}

func NewEntityExtractor() *EntityExtractor {
	return &EntityExtractor{
		patterns: map[string]*regexp.Regexp{
			"date": regexp.MustCompile(`(?i)(\d{1,2}[\/\-]\d{1,2}[\/\-]\d{2,4}|` +
				`(january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{1,2}(st|nd|rd|th)?,?\s*\d{0,4}|` +
				`(next|this)\s+(week|month|year|saturday|sunday|monday|tuesday|wednesday|thursday|friday)|` +
				`(tomorrow|today|weekend))`),
			"number": regexp.MustCompile(`(\d+)\s*(people|guests|persons|attendees|pax)`),
			"budget": regexp.MustCompile(`(?i)(₦|ngn|naira)?\s*(\d{1,3}(?:,?\d{3})*(?:\.\d{2})?)\s*(million|m|k|thousand)?`),
			"location": regexp.MustCompile(`(?i)(in|at|around|near)\s+([A-Za-z\s]+?)(?:\s*,|\s*$|\s+(?:on|for|with))`),
			"event_type": regexp.MustCompile(`(?i)(wedding|birthday|party|funeral|graduation|anniversary|baby shower|naming ceremony|corporate event|conference|product launch)`),
			"vendor_type": regexp.MustCompile(`(?i)(photographer|videographer|caterer|decorator|dj|mc|planner|florist|makeup artist|hair stylist|cake baker|venue)`),
			"time": regexp.MustCompile(`(?i)(\d{1,2}:\d{2}\s*(am|pm)?|\d{1,2}\s*(am|pm)|morning|afternoon|evening|night)`),
			"style": regexp.MustCompile(`(?i)(traditional|modern|minimalist|elegant|rustic|vintage|glamorous|simple|luxurious)`),
		},
	}
}

func (e *EntityExtractor) ExtractEntities(text string) []Entity {
	var entities []Entity
	
	for entityType, pattern := range e.patterns {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				value := text[match[0]:match[1]]
				entities = append(entities, Entity{
					Type:       entityType,
					Value:      e.parseEntityValue(entityType, value),
					Text:       value,
					StartPos:   match[0],
					EndPos:     match[1],
					Confidence: 0.85,
				})
			}
		}
	}
	
	return entities
}

func (e *EntityExtractor) parseEntityValue(entityType string, text string) interface{} {
	switch entityType {
	case "number":
		// Extract just the number
		re := regexp.MustCompile(`(\d+)`)
		match := re.FindString(text)
		var num int
		fmt.Sscanf(match, "%d", &num)
		return num
		
	case "budget":
		// Parse budget with multipliers
		text = strings.ToLower(text)
		text = strings.ReplaceAll(text, "₦", "")
		text = strings.ReplaceAll(text, "ngn", "")
		text = strings.ReplaceAll(text, "naira", "")
		text = strings.ReplaceAll(text, ",", "")
		text = strings.TrimSpace(text)
		
		var amount float64
		fmt.Sscanf(text, "%f", &amount)
		
		if strings.Contains(text, "million") || strings.Contains(text, "m") {
			amount *= 1000000
		} else if strings.Contains(text, "thousand") || strings.Contains(text, "k") {
			amount *= 1000
		}
		
		return amount
		
	default:
		return text
	}
}

// SlotFiller manages conversation slots
type SlotFiller struct {
	slotDefinitions map[string]SlotDefinition
}

type SlotDefinition struct {
	Name        string
	EntityTypes []string
	Required    bool
	Prompts     []string
	Validators  []SlotValidator
}

type SlotValidator func(value interface{}) (bool, string)

// Required slots for event creation
var EventCreationSlots = map[string]SlotDefinition{
	"event_type": {
		Name:        "event_type",
		EntityTypes: []string{"event_type"},
		Required:    true,
		Prompts: []string{
			"What type of event are you planning?",
			"What kind of celebration is this?",
		},
	},
	"event_date": {
		Name:        "event_date",
		EntityTypes: []string{"date"},
		Required:    true,
		Prompts: []string{
			"When is your event?",
			"What date do you have in mind?",
		},
	},
	"guest_count": {
		Name:        "guest_count",
		EntityTypes: []string{"number"},
		Required:    true,
		Prompts: []string{
			"How many guests are you expecting?",
			"Approximately how many people will attend?",
		},
	},
	"location": {
		Name:        "location",
		EntityTypes: []string{"location"},
		Required:    true,
		Prompts: []string{
			"Where will your event take place?",
			"Which city or area are you considering?",
		},
	},
	"budget": {
		Name:        "budget",
		EntityTypes: []string{"budget"},
		Required:    false,
		Prompts: []string{
			"Do you have a budget in mind?",
			"What's your approximate budget for this event?",
		},
	},
}

func (sf *SlotFiller) FillSlots(entities []Entity, currentSlots map[string]SlotValue, intent string) map[string]SlotValue {
	if currentSlots == nil {
		currentSlots = make(map[string]SlotValue)
	}
	
	// Get slot definitions based on intent
	var relevantSlots map[string]SlotDefinition
	switch intent {
	case "create_event":
		relevantSlots = EventCreationSlots
	default:
		relevantSlots = EventCreationSlots // Default to event slots
	}
	
	// Fill slots from entities
	for _, entity := range entities {
		for slotName, slotDef := range relevantSlots {
			for _, entityType := range slotDef.EntityTypes {
				if entity.Type == entityType {
					// Only fill if not already filled or new value has higher confidence
					existing, exists := currentSlots[slotName]
					if !exists || entity.Confidence > existing.Confidence {
						currentSlots[slotName] = SlotValue{
							Value:      entity.Value,
							Source:     "user",
							Confidence: entity.Confidence,
							Timestamp:  time.Now(),
							Confirmed:  false,
						}
					}
				}
			}
		}
	}
	
	return currentSlots
}

func (sf *SlotFiller) GetMissingRequiredSlots(currentSlots map[string]SlotValue, intent string) []SlotDefinition {
	var missing []SlotDefinition
	
	var relevantSlots map[string]SlotDefinition
	switch intent {
	case "create_event":
		relevantSlots = EventCreationSlots
	default:
		return missing
	}
	
	for name, slotDef := range relevantSlots {
		if slotDef.Required {
			if _, exists := currentSlots[name]; !exists {
				missing = append(missing, slotDef)
			}
		}
	}
	
	return missing
}

// =============================================================================
// 2.3 DIALOG MANAGEMENT
// =============================================================================

// DialogManager orchestrates conversation flow
type DialogManager struct {
	nlu            *NLUEngine
	responseGen    *ResponseGenerator
	actionExecutor *ActionExecutor
	memoryManager  *MemoryManager
	db             *pgxpool.Pool
	cache          *redis.Client
}

// ConversationContext provides context for dialog decisions
type ConversationContext struct {
	UserID          uuid.UUID
	ConversationID  uuid.UUID
	EventID         *uuid.UUID
	CurrentState    ConversationState
	Intent          *Intent
	Slots           map[string]SlotValue
	TurnCount       int
	LastMessages    []Message
	UserProfile     *UserProfile
}

type UserProfile struct {
	PreferredName   string
	PastEvents      []PastEvent
	Preferences     map[string]interface{}
	CommunicationStyle string
}

type PastEvent struct {
	EventType    string
	EventDate    time.Time
	Vendors      []uuid.UUID
	Satisfaction int
}

// ProcessMessage is the main entry point for handling user messages
func (dm *DialogManager) ProcessMessage(ctx context.Context, conv *Conversation, userMessage string) (*Message, error) {
	startTime := time.Now()
	
	// 1. Add user message to conversation
	userMsg := Message{
		ID:        uuid.New(),
		Role:      RoleUser,
		Content:   userMessage,
		Timestamp: time.Now(),
	}
	
	// 2. Build conversation context
	convContext := dm.buildContext(conv)
	
	// 3. Run NLU pipeline
	intent, err := dm.nlu.intentClassifier.ClassifyIntent(ctx, userMessage, convContext)
	if err != nil {
		return nil, fmt.Errorf("intent classification failed: %w", err)
	}
	userMsg.Intent = intent
	
	entities := dm.nlu.entityExtractor.ExtractEntities(userMessage)
	userMsg.Entities = entities
	
	// 4. Fill slots with extracted entities
	conv.SlotValues = dm.nlu.slotFiller.FillSlots(entities, conv.SlotValues, intent.Name)
	
	// 5. Update conversation state
	conv.CurrentIntent = *intent
	conv.Messages = append(conv.Messages, userMsg)
	conv.TurnCount++
	conv.LastMessageAt = time.Now()
	
	// 6. Determine response strategy
	responseStrategy := dm.determineResponseStrategy(conv, intent)
	
	// 7. Execute any required actions
	actionResults, err := dm.actionExecutor.ExecuteActions(ctx, responseStrategy.Actions, conv)
	if err != nil {
		// Log but don't fail
	}
	
	// 8. Generate response
	response, err := dm.responseGen.GenerateResponse(ctx, conv, responseStrategy, actionResults)
	if err != nil {
		return nil, fmt.Errorf("response generation failed: %w", err)
	}
	
	response.ID = uuid.New()
	response.Role = RoleAssistant
	response.Timestamp = time.Now()
	response.ProcessingTime = time.Since(startTime).Milliseconds()
	
	// 9. Update conversation state based on response
	conv.ConversationState = responseStrategy.NextState
	conv.Messages = append(conv.Messages, *response)
	
	// 10. Persist conversation
	dm.saveConversation(ctx, conv)
	
	return response, nil
}

func (dm *DialogManager) buildContext(conv *Conversation) *ConversationContext {
	ctx := &ConversationContext{
		UserID:         conv.UserID,
		ConversationID: conv.ID,
		EventID:        conv.EventID,
		CurrentState:   conv.ConversationState,
		Slots:          conv.SlotValues,
		TurnCount:      conv.TurnCount,
	}
	
	// Get last N messages for context
	if len(conv.Messages) > 10 {
		ctx.LastMessages = conv.Messages[len(conv.Messages)-10:]
	} else {
		ctx.LastMessages = conv.Messages
	}
	
	return ctx
}

// ResponseStrategy defines how to respond
type ResponseStrategy struct {
	Type           ResponseType
	Template       string
	NextState      ConversationState
	Actions        []ActionDefinition
	DataNeeded     []string
	QuickReplies   []QuickReply
	ShouldConfirm  bool
	ConfirmSlots   []string
}

type ResponseType string
const (
	ResponseText        ResponseType = "text"
	ResponseCards       ResponseType = "cards"
	ResponseConfirm     ResponseType = "confirm"
	ResponseQuestion    ResponseType = "question"
	ResponseComparison  ResponseType = "comparison"
	ResponseSummary     ResponseType = "summary"
	ResponseHandoff     ResponseType = "handoff"
)

type ActionDefinition struct {
	Type       string
	Parameters map[string]interface{}
}

func (dm *DialogManager) determineResponseStrategy(conv *Conversation, intent *Intent) *ResponseStrategy {
	strategy := &ResponseStrategy{
		Type:      ResponseText,
		NextState: conv.ConversationState,
	}
	
	switch intent.Name {
	case "greeting":
		return dm.handleGreeting(conv)
		
	case "create_event":
		return dm.handleCreateEvent(conv)
		
	case "find_vendor":
		return dm.handleFindVendor(conv)
		
	case "get_quote":
		return dm.handleGetQuote(conv)
		
	case "book_service":
		return dm.handleBookService(conv)
		
	case "compare_options":
		return dm.handleCompareOptions(conv)
		
	case "check_availability":
		return dm.handleCheckAvailability(conv)
		
	case "get_recommendation":
		return dm.handleGetRecommendation(conv)
		
	case "view_plan":
		return dm.handleViewPlan(conv)
		
	case "update_preference":
		return dm.handleUpdatePreference(conv)
		
	case "cancel":
		return dm.handleCancel(conv)
		
	case "thanks":
		return dm.handleThanks(conv)
		
	default:
		return dm.handleGeneralQuestion(conv, intent)
	}
	
	return strategy
}

func (dm *DialogManager) handleGreeting(conv *Conversation) *ResponseStrategy {
	// Check if this is a new conversation or returning user
	if conv.TurnCount == 1 {
		return &ResponseStrategy{
			Type:      ResponseText,
			Template:  "greeting_new",
			NextState: StateWelcome,
			QuickReplies: []QuickReply{
				{Title: "Plan a wedding", Payload: "create_event:wedding"},
				{Title: "Plan a birthday", Payload: "create_event:birthday"},
				{Title: "Find a vendor", Payload: "find_vendor"},
				{Title: "Get recommendations", Payload: "get_recommendation"},
			},
		}
	}
	
	return &ResponseStrategy{
		Type:      ResponseText,
		Template:  "greeting_returning",
		NextState: conv.ConversationState,
	}
}

func (dm *DialogManager) handleCreateEvent(conv *Conversation) *ResponseStrategy {
	// Check which slots are missing
	missingSlots := dm.nlu.slotFiller.GetMissingRequiredSlots(conv.SlotValues, "create_event")
	
	if len(missingSlots) > 0 {
		// Ask for the first missing slot
		slot := missingSlots[0]
		return &ResponseStrategy{
			Type:      ResponseQuestion,
			Template:  fmt.Sprintf("ask_%s", slot.Name),
			NextState: StateGatheringInfo,
			DataNeeded: []string{slot.Name},
		}
	}
	
	// All required slots filled - confirm before creating
	return &ResponseStrategy{
		Type:         ResponseConfirm,
		Template:     "confirm_event_details",
		NextState:    StateConfirming,
		ShouldConfirm: true,
		ConfirmSlots: []string{"event_type", "event_date", "guest_count", "location"},
		Actions: []ActionDefinition{
			{Type: "prepare_event_summary"},
		},
		QuickReplies: []QuickReply{
			{Title: "Yes, looks good!", Payload: "confirm:yes"},
			{Title: "Make changes", Payload: "confirm:edit"},
			{Title: "Start over", Payload: "confirm:restart"},
		},
	}
}

func (dm *DialogManager) handleFindVendor(conv *Conversation) *ResponseStrategy {
	// Check if we know what type of vendor
	vendorType, hasVendor := conv.SlotValues["vendor_type"]
	eventType, hasEvent := conv.SlotValues["event_type"]
	location, hasLocation := conv.SlotValues["location"]
	
	if !hasVendor {
		return &ResponseStrategy{
			Type:      ResponseQuestion,
			Template:  "ask_vendor_type",
			NextState: StateGatheringInfo,
			QuickReplies: []QuickReply{
				{Title: "Photographer", Payload: "vendor_type:photographer"},
				{Title: "Caterer", Payload: "vendor_type:caterer"},
				{Title: "Decorator", Payload: "vendor_type:decorator"},
				{Title: "DJ/Entertainment", Payload: "vendor_type:dj"},
				{Title: "Event Planner", Payload: "vendor_type:planner"},
			},
		}
	}
	
	if !hasLocation && !hasEvent {
		return &ResponseStrategy{
			Type:      ResponseQuestion,
			Template:  "ask_location_for_vendor",
			NextState: StateGatheringInfo,
		}
	}
	
	// We have enough info - search for vendors
	return &ResponseStrategy{
		Type:      ResponseCards,
		Template:  "vendor_results",
		NextState: StateRecommending,
		Actions: []ActionDefinition{
			{
				Type: "search_vendors",
				Parameters: map[string]interface{}{
					"vendor_type": vendorType.Value,
					"event_type":  eventType.Value,
					"location":    location.Value,
				},
			},
		},
	}
}

func (dm *DialogManager) handleGetQuote(conv *Conversation) *ResponseStrategy {
	// Check if we have a specific vendor in context
	if vendorID, ok := conv.ShortTermMemory["selected_vendor_id"].(uuid.UUID); ok {
		return &ResponseStrategy{
			Type:      ResponseText,
			Template:  "quote_for_vendor",
			NextState: StateRecommending,
			Actions: []ActionDefinition{
				{
					Type: "get_vendor_quote",
					Parameters: map[string]interface{}{
						"vendor_id": vendorID,
						"slots":     conv.SlotValues,
					},
				},
			},
		}
	}
	
	// No specific vendor - give general pricing info
	return &ResponseStrategy{
		Type:      ResponseText,
		Template:  "general_pricing",
		NextState: conv.ConversationState,
		Actions: []ActionDefinition{
			{
				Type: "get_pricing_estimates",
				Parameters: map[string]interface{}{
					"slots": conv.SlotValues,
				},
			},
		},
	}
}

func (dm *DialogManager) handleBookService(conv *Conversation) *ResponseStrategy {
	// Check if we have a vendor selected
	vendorID, hasVendor := conv.ShortTermMemory["selected_vendor_id"].(uuid.UUID)
	serviceID, hasService := conv.ShortTermMemory["selected_service_id"].(uuid.UUID)
	
	if !hasVendor || !hasService {
		return &ResponseStrategy{
			Type:      ResponseText,
			Template:  "no_vendor_selected",
			NextState: StateRecommending,
		}
	}
	
	// Check if we have required booking info
	eventDate, hasDate := conv.SlotValues["event_date"]
	
	if !hasDate {
		return &ResponseStrategy{
			Type:      ResponseQuestion,
			Template:  "ask_event_date_for_booking",
			NextState: StateBooking,
		}
	}
	
	// Ready to book - confirm first
	return &ResponseStrategy{
		Type:         ResponseConfirm,
		Template:     "confirm_booking",
		NextState:    StateBooking,
		ShouldConfirm: true,
		Actions: []ActionDefinition{
			{
				Type: "prepare_booking",
				Parameters: map[string]interface{}{
					"vendor_id":  vendorID,
					"service_id": serviceID,
					"event_date": eventDate.Value,
				},
			},
		},
		QuickReplies: []QuickReply{
			{Title: "Confirm Booking", Payload: "booking:confirm"},
			{Title: "Change Date", Payload: "booking:change_date"},
			{Title: "Cancel", Payload: "booking:cancel"},
		},
	}
}

func (dm *DialogManager) handleCompareOptions(conv *Conversation) *ResponseStrategy {
	// Get vendors from memory
	vendors, ok := conv.ShortTermMemory["vendor_results"].([]VendorResult)
	if !ok || len(vendors) < 2 {
		return &ResponseStrategy{
			Type:      ResponseText,
			Template:  "no_vendors_to_compare",
			NextState: StateRecommending,
		}
	}
	
	return &ResponseStrategy{
		Type:      ResponseComparison,
		Template:  "vendor_comparison",
		NextState: StateComparing,
		Actions: []ActionDefinition{
			{
				Type: "generate_comparison",
				Parameters: map[string]interface{}{
					"vendors": vendors[:min(4, len(vendors))], // Compare top 4
				},
			},
		},
	}
}

func (dm *DialogManager) handleCheckAvailability(conv *Conversation) *ResponseStrategy {
	vendorID, hasVendor := conv.ShortTermMemory["selected_vendor_id"].(uuid.UUID)
	eventDate, hasDate := conv.SlotValues["event_date"]
	
	if !hasVendor {
		return &ResponseStrategy{
			Type:      ResponseText,
			Template:  "which_vendor_availability",
			NextState: conv.ConversationState,
		}
	}
	
	if !hasDate {
		return &ResponseStrategy{
			Type:      ResponseQuestion,
			Template:  "ask_date_for_availability",
			NextState: conv.ConversationState,
		}
	}
	
	return &ResponseStrategy{
		Type:      ResponseText,
		Template:  "availability_result",
		NextState: conv.ConversationState,
		Actions: []ActionDefinition{
			{
				Type: "check_availability",
				Parameters: map[string]interface{}{
					"vendor_id": vendorID,
					"date":      eventDate.Value,
				},
			},
		},
	}
}

func (dm *DialogManager) handleGetRecommendation(conv *Conversation) *ResponseStrategy {
	// Check what context we have
	eventType, hasEvent := conv.SlotValues["event_type"]
	
	if !hasEvent {
		return &ResponseStrategy{
			Type:      ResponseQuestion,
			Template:  "what_event_for_recommendation",
			NextState: StateGatheringInfo,
			QuickReplies: []QuickReply{
				{Title: "Wedding", Payload: "event_type:wedding"},
				{Title: "Birthday Party", Payload: "event_type:birthday"},
				{Title: "Corporate Event", Payload: "event_type:corporate"},
				{Title: "Other", Payload: "event_type:other"},
			},
		}
	}
	
	return &ResponseStrategy{
		Type:      ResponseCards,
		Template:  "recommendations",
		NextState: StateRecommending,
		Actions: []ActionDefinition{
			{
				Type: "get_personalized_recommendations",
				Parameters: map[string]interface{}{
					"event_type": eventType.Value,
					"slots":      conv.SlotValues,
				},
			},
		},
	}
}

func (dm *DialogManager) handleViewPlan(conv *Conversation) *ResponseStrategy {
	if conv.EventID == nil {
		return &ResponseStrategy{
			Type:      ResponseText,
			Template:  "no_event_yet",
			NextState: conv.ConversationState,
		}
	}
	
	return &ResponseStrategy{
		Type:      ResponseSummary,
		Template:  "event_plan_summary",
		NextState: conv.ConversationState,
		Actions: []ActionDefinition{
			{
				Type: "load_event_plan",
				Parameters: map[string]interface{}{
					"event_id": conv.EventID,
				},
			},
		},
	}
}

func (dm *DialogManager) handleUpdatePreference(conv *Conversation) *ResponseStrategy {
	return &ResponseStrategy{
		Type:      ResponseText,
		Template:  "what_to_change",
		NextState: StateGatheringInfo,
		QuickReplies: []QuickReply{
			{Title: "Date", Payload: "change:event_date"},
			{Title: "Location", Payload: "change:location"},
			{Title: "Budget", Payload: "change:budget"},
			{Title: "Guest Count", Payload: "change:guest_count"},
		},
	}
}

func (dm *DialogManager) handleCancel(conv *Conversation) *ResponseStrategy {
	return &ResponseStrategy{
		Type:      ResponseText,
		Template:  "confirm_cancel",
		NextState: StateCompleted,
		QuickReplies: []QuickReply{
			{Title: "Yes, cancel", Payload: "cancel:confirm"},
			{Title: "No, continue", Payload: "cancel:resume"},
		},
	}
}

func (dm *DialogManager) handleThanks(conv *Conversation) *ResponseStrategy {
	return &ResponseStrategy{
		Type:      ResponseText,
		Template:  "you_are_welcome",
		NextState: conv.ConversationState,
		QuickReplies: []QuickReply{
			{Title: "Continue planning", Payload: "continue"},
			{Title: "That's all for now", Payload: "end"},
		},
	}
}

func (dm *DialogManager) handleGeneralQuestion(conv *Conversation, intent *Intent) *ResponseStrategy {
	return &ResponseStrategy{
		Type:      ResponseText,
		Template:  "general_answer",
		NextState: conv.ConversationState,
		Actions: []ActionDefinition{
			{
				Type: "search_knowledge_base",
				Parameters: map[string]interface{}{
					"query":   conv.Messages[len(conv.Messages)-1].Content,
					"context": conv.SlotValues,
				},
			},
		},
	}
}

func (dm *DialogManager) saveConversation(ctx context.Context, conv *Conversation) error {
	messagesJSON, _ := json.Marshal(conv.Messages)
	slotsJSON, _ := json.Marshal(conv.SlotValues)
	memoryJSON, _ := json.Marshal(conv.ShortTermMemory)
	
	query := `
		INSERT INTO conversations (
			id, user_id, event_id, session_type,
			current_intent, conversation_state, slot_values,
			messages, turn_count, short_term_memory,
			language, channel, started_at, last_message_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			current_intent = $5,
			conversation_state = $6,
			slot_values = $7,
			messages = $8,
			turn_count = $9,
			short_term_memory = $10,
			last_message_at = $14
	`
	
	intentJSON, _ := json.Marshal(conv.CurrentIntent)
	
	_, err := dm.db.Exec(ctx, query,
		conv.ID, conv.UserID, conv.EventID, conv.SessionType,
		intentJSON, conv.ConversationState, slotsJSON,
		messagesJSON, conv.TurnCount, memoryJSON,
		conv.Language, conv.Channel, conv.StartedAt, conv.LastMessageAt,
	)
	
	return err
}

// =============================================================================
// 2.4 RESPONSE GENERATION
// =============================================================================

// ResponseGenerator creates natural language responses
type ResponseGenerator struct {
	templates map[string]ResponseTemplate
	db        *pgxpool.Pool
}

type ResponseTemplate struct {
	Name       string
	Variations []string
	Variables  []string
}

// Response templates for EventGPT
var ResponseTemplates = map[string]ResponseTemplate{
	"greeting_new": {
		Name: "greeting_new",
		Variations: []string{
			"Hello! 👋 I'm EventGPT, your AI event planning assistant. I can help you plan weddings, birthdays, corporate events, and more. What are you celebrating?",
			"Hi there! 🎉 Welcome to EventGPT. I'm here to make your event planning smooth and stress-free. What event can I help you plan today?",
			"Hey! I'm EventGPT, ready to help you create an amazing event. Whether it's a wedding, birthday, or any celebration - I've got you covered. What would you like to plan?",
		},
	},
	"greeting_returning": {
		Name: "greeting_returning",
		Variations: []string{
			"Welcome back, {user_name}! How can I help you today?",
			"Hi again! Ready to continue with your {event_type} planning?",
		},
	},
	"ask_event_type": {
		Name: "ask_event_type",
		Variations: []string{
			"What type of event are you planning? Is it a wedding, birthday, corporate event, or something else?",
			"Great! Let's get started. What kind of celebration are we planning?",
		},
	},
	"ask_event_date": {
		Name: "ask_event_date",
		Variations: []string{
			"When is your {event_type}? You can give me an exact date or just a general timeframe like 'next December' or 'summer 2025'.",
			"What date do you have in mind for your {event_type}?",
		},
	},
	"ask_guest_count": {
		Name: "ask_guest_count",
		Variations: []string{
			"How many guests are you expecting at your {event_type}?",
			"Approximately how many people will be attending?",
		},
	},
	"ask_location": {
		Name: "ask_location",
		Variations: []string{
			"Where will your {event_type} take place? Let me know the city or area, and I can find the best vendors nearby.",
			"What location are you considering for your event?",
		},
	},
	"ask_budget": {
		Name: "ask_budget",
		Variations: []string{
			"Do you have a budget in mind? This helps me recommend vendors that fit your range. (You can skip this if you prefer)",
			"What's your approximate budget for this event? Don't worry, you can always adjust this later.",
		},
	},
	"confirm_event_details": {
		Name: "confirm_event_details",
		Variations: []string{
			"Perfect! Here's what I have:\n\n📋 **Event:** {event_type}\n📅 **Date:** {event_date}\n👥 **Guests:** {guest_count}\n📍 **Location:** {location}\n💰 **Budget:** {budget}\n\nDoes this look correct?",
		},
	},
	"ask_vendor_type": {
		Name: "ask_vendor_type",
		Variations: []string{
			"What type of vendor are you looking for?",
			"Which service do you need? I can help you find photographers, caterers, decorators, and more.",
		},
	},
	"vendor_results": {
		Name: "vendor_results",
		Variations: []string{
			"I found {vendor_count} great {vendor_type}s in {location}. Here are my top recommendations:",
			"Here are the best {vendor_type}s I found for your {event_type}:",
		},
	},
	"no_vendors_found": {
		Name: "no_vendors_found",
		Variations: []string{
			"I couldn't find any {vendor_type}s matching your criteria. Would you like me to expand the search?",
			"No exact matches found. Should I show you similar options or adjust the search?",
		},
	},
	"availability_result": {
		Name: "availability_result",
		Variations: []string{
			"{vendor_name} is {availability_status} on {date}. {additional_info}",
		},
	},
	"confirm_booking": {
		Name: "confirm_booking",
		Variations: []string{
			"Ready to book {service_name} with {vendor_name} for {date}?\n\n💰 Total: {price}\n\nShall I proceed with the booking?",
		},
	},
	"booking_confirmed": {
		Name: "booking_confirmed",
		Variations: []string{
			"🎉 Excellent! Your booking with {vendor_name} is confirmed!\n\n**Booking Details:**\n📅 Date: {date}\n💰 Amount: {price}\n📧 Confirmation sent to your email\n\nWhat else can I help you with?",
		},
	},
	"you_are_welcome": {
		Name: "you_are_welcome",
		Variations: []string{
			"You're welcome! Is there anything else I can help you with?",
			"Happy to help! Let me know if you need anything else. 😊",
		},
	},
	"no_event_yet": {
		Name: "no_event_yet",
		Variations: []string{
			"You haven't started planning an event yet. Would you like to create one?",
		},
	},
	"general_pricing": {
		Name: "general_pricing",
		Variations: []string{
			"Here are typical price ranges for {event_type} services in {location}:\n\n{pricing_breakdown}\n\nWould you like specific quotes from vendors?",
		},
	},
}

func NewResponseGenerator(db *pgxpool.Pool) *ResponseGenerator {
	return &ResponseGenerator{
		templates: ResponseTemplates,
		db:        db,
	}
}

func (rg *ResponseGenerator) GenerateResponse(ctx context.Context, conv *Conversation, strategy *ResponseStrategy, actionResults map[string]interface{}) (*Message, error) {
	response := &Message{
		Role: RoleAssistant,
	}
	
	// Get template
	template, ok := rg.templates[strategy.Template]
	if !ok {
		template = ResponseTemplate{
			Variations: []string{"I understand. Let me help you with that."},
		}
	}
	
	// Select a variation (could use more sophisticated selection)
	variation := template.Variations[conv.TurnCount%len(template.Variations)]
	
	// Fill in variables
	responseText := rg.fillVariables(variation, conv.SlotValues, actionResults)
	response.Content = responseText
	
	// Add quick replies if specified
	if len(strategy.QuickReplies) > 0 {
		response.QuickReplies = strategy.QuickReplies
	}
	
	// Add cards if this is a card response
	if strategy.Type == ResponseCards {
		if vendors, ok := actionResults["vendors"].([]VendorResult); ok {
			response.Cards = rg.vendorsToCards(vendors)
		}
	}
	
	// Add comparison if needed
	if strategy.Type == ResponseComparison {
		if comparison, ok := actionResults["comparison"].(*VendorComparison); ok {
			response.Cards = rg.comparisonToCards(comparison)
		}
	}
	
	return response, nil
}

func (rg *ResponseGenerator) fillVariables(template string, slots map[string]SlotValue, actionResults map[string]interface{}) string {
	result := template
	
	// Fill from slots
	for name, slot := range slots {
		placeholder := fmt.Sprintf("{%s}", name)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", slot.Value))
	}
	
	// Fill from action results
	for key, value := range actionResults {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	
	return result
}

func (rg *ResponseGenerator) vendorsToCards(vendors []VendorResult) []Card {
	var cards []Card
	
	for _, v := range vendors {
		card := Card{
			Type:        "vendor",
			Title:       v.VendorName,
			Subtitle:    v.ServiceName,
			ImageURL:    v.ImageURL,
			Description: v.ShortDescription,
			Price: &PriceDisplay{
				Amount:   v.Price,
				Currency: "NGN",
			},
			Rating: &RatingDisplay{
				Score: v.Rating,
				Count: v.ReviewCount,
			},
			Actions: []ActionButton{
				{Type: "postback", Title: "View Profile", Payload: fmt.Sprintf("view_vendor:%s", v.VendorID)},
				{Type: "postback", Title: "Get Quote", Payload: fmt.Sprintf("quote_vendor:%s", v.VendorID), Style: "primary"},
				{Type: "postback", Title: "Book Now", Payload: fmt.Sprintf("book_vendor:%s", v.VendorID), Style: "primary"},
			},
			Metadata: map[string]interface{}{
				"vendor_id":  v.VendorID,
				"service_id": v.ServiceID,
			},
		}
		cards = append(cards, card)
	}
	
	return cards
}

func (rg *ResponseGenerator) comparisonToCards(comparison *VendorComparison) []Card {
	// Create a comparison card
	card := Card{
		Type:        "comparison",
		Title:       "Vendor Comparison",
		Description: comparison.Summary,
		Metadata: map[string]interface{}{
			"vendors":      comparison.Vendors,
			"criteria":     comparison.Criteria,
			"winner":       comparison.Recommendation,
		},
	}
	
	return []Card{card}
}

// =============================================================================
// 2.5 ACTION EXECUTOR
// =============================================================================

// ActionExecutor executes actions during conversation
type ActionExecutor struct {
	db              *pgxpool.Pool
	cache           *redis.Client
	vendorService   *VendorService
	bookingService  *BookingService
	pricingService  *PricingService
}

type VendorResult struct {
	VendorID         uuid.UUID
	VendorName       string
	ServiceID        uuid.UUID
	ServiceName      string
	ImageURL         string
	ShortDescription string
	Price            float64
	Rating           float64
	ReviewCount      int
	MatchScore       float64
}

type VendorComparison struct {
	Vendors        []VendorResult
	Criteria       []string
	Summary        string
	Recommendation *VendorResult
}

func (ae *ActionExecutor) ExecuteActions(ctx context.Context, actions []ActionDefinition, conv *Conversation) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	
	for _, action := range actions {
		switch action.Type {
		case "search_vendors":
			vendors, err := ae.searchVendors(ctx, action.Parameters)
			if err != nil {
				continue
			}
			results["vendors"] = vendors
			results["vendor_count"] = len(vendors)
			// Store in conversation memory
			conv.ShortTermMemory["vendor_results"] = vendors
			
		case "get_vendor_quote":
			quote, err := ae.getVendorQuote(ctx, action.Parameters)
			if err != nil {
				continue
			}
			results["quote"] = quote
			
		case "check_availability":
			available, msg, err := ae.checkAvailability(ctx, action.Parameters)
			if err != nil {
				continue
			}
			results["availability_status"] = available
			results["additional_info"] = msg
			
		case "prepare_booking":
			booking, err := ae.prepareBooking(ctx, action.Parameters)
			if err != nil {
				continue
			}
			results["booking"] = booking
			results["price"] = booking.TotalAmount
			
		case "generate_comparison":
			comparison, err := ae.generateComparison(ctx, action.Parameters)
			if err != nil {
				continue
			}
			results["comparison"] = comparison
			
		case "get_pricing_estimates":
			estimates, err := ae.getPricingEstimates(ctx, action.Parameters)
			if err != nil {
				continue
			}
			results["pricing_breakdown"] = estimates
			
		case "load_event_plan":
			plan, err := ae.loadEventPlan(ctx, action.Parameters)
			if err != nil {
				continue
			}
			results["plan"] = plan
			
		case "get_personalized_recommendations":
			recs, err := ae.getPersonalizedRecommendations(ctx, action.Parameters)
			if err != nil {
				continue
			}
			results["vendors"] = recs
			results["vendor_count"] = len(recs)
		}
	}
	
	return results, nil
}

func (ae *ActionExecutor) searchVendors(ctx context.Context, params map[string]interface{}) ([]VendorResult, error) {
	vendorType := params["vendor_type"].(string)
	location := params["location"]
	
	query := `
		SELECT 
			v.id as vendor_id,
			v.business_name,
			s.id as service_id,
			s.name as service_name,
			v.logo_url,
			s.short_description,
			s.base_price,
			v.rating_average,
			v.rating_count
		FROM services s
		JOIN vendors v ON v.id = s.vendor_id
		JOIN service_categories sc ON sc.id = s.category_id
		WHERE LOWER(sc.name) LIKE $1
		  AND v.is_active = TRUE
		  AND s.is_available = TRUE
		ORDER BY v.rating_average DESC, v.rating_count DESC
		LIMIT 10
	`
	
	rows, err := ae.db.Query(ctx, query, "%"+vendorType+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var vendors []VendorResult
	for rows.Next() {
		var v VendorResult
		if err := rows.Scan(&v.VendorID, &v.VendorName, &v.ServiceID, &v.ServiceName,
			&v.ImageURL, &v.ShortDescription, &v.Price, &v.Rating, &v.ReviewCount); err != nil {
			continue
		}
		vendors = append(vendors, v)
	}
	
	_ = location // Would use for geo filtering
	
	return vendors, nil
}

func (ae *ActionExecutor) getVendorQuote(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	vendorID := params["vendor_id"].(uuid.UUID)
	slots := params["slots"].(map[string]SlotValue)
	
	// Get vendor's base price
	var basePrice float64
	ae.db.QueryRow(ctx, `
		SELECT s.base_price FROM services s
		JOIN vendors v ON v.id = s.vendor_id
		WHERE v.id = $1 LIMIT 1
	`, vendorID).Scan(&basePrice)
	
	// Adjust for event parameters
	guestCount := 100
	if gc, ok := slots["guest_count"]; ok {
		guestCount = gc.Value.(int)
	}
	
	// Simple quote calculation
	adjustedPrice := basePrice * (1 + float64(guestCount-50)/100*0.5)
	
	return map[string]interface{}{
		"base_price":     basePrice,
		"adjusted_price": adjustedPrice,
		"currency":       "NGN",
		"valid_until":    time.Now().AddDate(0, 0, 7),
	}, nil
}

func (ae *ActionExecutor) checkAvailability(ctx context.Context, params map[string]interface{}) (string, string, error) {
	vendorID := params["vendor_id"].(uuid.UUID)
	date := params["date"]
	
	// Check booking calendar
	var bookingCount int
	ae.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM bookings
		WHERE vendor_id = $1 AND scheduled_date = $2 AND status NOT IN ('cancelled')
	`, vendorID, date).Scan(&bookingCount)
	
	// Get vendor's max concurrent bookings
	var maxBookings int
	ae.db.QueryRow(ctx, `SELECT max_concurrent_bookings FROM vendors WHERE id = $1`, vendorID).Scan(&maxBookings)
	
	if bookingCount >= maxBookings {
		return "unavailable", "They're fully booked on this date. Would you like to see alternative dates?", nil
	} else if bookingCount >= maxBookings-1 {
		return "limited", "Only 1 slot remaining! I'd recommend booking soon.", nil
	}
	
	return "available", "Great news! They have availability.", nil
}

type BookingDraft struct {
	VendorID    uuid.UUID
	ServiceID   uuid.UUID
	Date        time.Time
	TotalAmount float64
}

func (ae *ActionExecutor) prepareBooking(ctx context.Context, params map[string]interface{}) (*BookingDraft, error) {
	vendorID := params["vendor_id"].(uuid.UUID)
	serviceID := params["service_id"].(uuid.UUID)
	eventDate := params["event_date"]
	
	// Get service price
	var price float64
	ae.db.QueryRow(ctx, `SELECT base_price FROM services WHERE id = $1`, serviceID).Scan(&price)
	
	// Parse date
	var parsedDate time.Time
	switch v := eventDate.(type) {
	case time.Time:
		parsedDate = v
	case string:
		parsedDate, _ = time.Parse("2006-01-02", v)
	}
	
	return &BookingDraft{
		VendorID:    vendorID,
		ServiceID:   serviceID,
		Date:        parsedDate,
		TotalAmount: price,
	}, nil
}

func (ae *ActionExecutor) generateComparison(ctx context.Context, params map[string]interface{}) (*VendorComparison, error) {
	vendors := params["vendors"].([]VendorResult)
	
	comparison := &VendorComparison{
		Vendors:  vendors,
		Criteria: []string{"Price", "Rating", "Experience", "Reviews"},
	}
	
	// Find best overall
	var best *VendorResult
	bestScore := 0.0
	
	for i := range vendors {
		v := &vendors[i]
		// Simple scoring: normalize rating and invert price
		score := v.Rating/5.0*0.5 + float64(v.ReviewCount)/100*0.3 + (1-v.Price/1000000)*0.2
		if score > bestScore {
			bestScore = score
			best = v
		}
	}
	
	comparison.Recommendation = best
	comparison.Summary = fmt.Sprintf("Based on ratings, reviews, and pricing, I recommend %s as the best overall choice.", best.VendorName)
	
	return comparison, nil
}

func (ae *ActionExecutor) getPricingEstimates(ctx context.Context, params map[string]interface{}) (string, error) {
	slots := params["slots"].(map[string]SlotValue)
	
	eventType := "event"
	if et, ok := slots["event_type"]; ok {
		eventType = et.Value.(string)
	}
	
	// Build pricing breakdown
	breakdown := fmt.Sprintf(`
📸 Photography: ₦150,000 - ₦500,000
🍽️ Catering: ₦3,000 - ₦8,000 per guest
🎵 DJ/Entertainment: ₦100,000 - ₦300,000
🌸 Decoration: ₦200,000 - ₦1,000,000
📍 Venue: ₦300,000 - ₦2,000,000

*Prices vary based on %s size and requirements`, eventType)
	
	return breakdown, nil
}

func (ae *ActionExecutor) loadEventPlan(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	eventID := params["event_id"].(*uuid.UUID)
	
	// Load event details
	// This would integrate with LifeOS
	return map[string]interface{}{
		"event_id": eventID,
		"status":   "planning",
		"progress": 25,
	}, nil
}

func (ae *ActionExecutor) getPersonalizedRecommendations(ctx context.Context, params map[string]interface{}) ([]VendorResult, error) {
	// Get top-rated vendors for the event type
	return ae.searchVendors(ctx, map[string]interface{}{
		"vendor_type": "photographer", // Default to common service
		"limit":       5,
	})
}

// =============================================================================
// 2.6 MEMORY MANAGER
// =============================================================================

// MemoryManager handles conversation memory
type MemoryManager struct {
	cache *redis.Client
	db    *pgxpool.Pool
}

// ContextManager manages conversation context
type ContextManager struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

// VendorService placeholder
type VendorService struct{}

// BookingService placeholder
type BookingService struct{}

// PricingService placeholder
type PricingService struct{}

/*
================================================================================
SECTION 3: API SPECIFICATION
================================================================================
*/

// EventGPTAPI provides the API for EventGPT
type EventGPTAPI struct {
	dialogManager *DialogManager
	db            *pgxpool.Pool
	cache         *redis.Client
}

// ChatRequest for sending a message
type ChatRequest struct {
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
	Message        string     `json:"message"`
	Channel        Channel    `json:"channel"`
	Attachments    []Attachment `json:"attachments,omitempty"`
}

// ChatResponse from EventGPT
type ChatResponse struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	Message        Message   `json:"message"`
	EventID        *uuid.UUID `json:"event_id,omitempty"`
	SessionType    SessionType `json:"session_type"`
}

// Chat handles a chat message
func (api *EventGPTAPI) Chat(ctx context.Context, userID uuid.UUID, req ChatRequest) (*ChatResponse, error) {
	// Get or create conversation
	var conv *Conversation
	var err error
	
	if req.ConversationID != nil {
		conv, err = api.loadConversation(ctx, *req.ConversationID)
		if err != nil {
			return nil, err
		}
	} else {
		conv = api.createConversation(userID, req.Channel)
	}
	
	// Process message
	response, err := api.dialogManager.ProcessMessage(ctx, conv, req.Message)
	if err != nil {
		return nil, err
	}
	
	return &ChatResponse{
		ConversationID: conv.ID,
		Message:        *response,
		EventID:        conv.EventID,
		SessionType:    conv.SessionType,
	}, nil
}

func (api *EventGPTAPI) createConversation(userID uuid.UUID, channel Channel) *Conversation {
	return &Conversation{
		ID:                uuid.New(),
		UserID:            userID,
		SessionType:       SessionGeneralInquiry,
		ConversationState: StateWelcome,
		SlotValues:        make(map[string]SlotValue),
		Messages:          []Message{},
		TurnCount:         0,
		ShortTermMemory:   make(map[string]interface{}),
		Language:          "en",
		Channel:           channel,
		StartedAt:         time.Now(),
		LastMessageAt:     time.Now(),
	}
}

func (api *EventGPTAPI) loadConversation(ctx context.Context, convID uuid.UUID) (*Conversation, error) {
	query := `
		SELECT id, user_id, event_id, session_type,
		       current_intent, conversation_state, slot_values,
		       messages, turn_count, short_term_memory,
		       language, channel, started_at, last_message_at
		FROM conversations
		WHERE id = $1
	`
	
	var conv Conversation
	var intentJSON, slotsJSON, messagesJSON, memoryJSON []byte
	
	err := api.db.QueryRow(ctx, query, convID).Scan(
		&conv.ID, &conv.UserID, &conv.EventID, &conv.SessionType,
		&intentJSON, &conv.ConversationState, &slotsJSON,
		&messagesJSON, &conv.TurnCount, &memoryJSON,
		&conv.Language, &conv.Channel, &conv.StartedAt, &conv.LastMessageAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(intentJSON, &conv.CurrentIntent)
	json.Unmarshal(slotsJSON, &conv.SlotValues)
	json.Unmarshal(messagesJSON, &conv.Messages)
	json.Unmarshal(memoryJSON, &conv.ShortTermMemory)
	
	return &conv, nil
}

// LoadConversation is a public wrapper for loadConversation
func (api *EventGPTAPI) LoadConversation(ctx context.Context, convID uuid.UUID) (*Conversation, error) {
	return api.loadConversation(ctx, convID)
}

/*
================================================================================
SECTION 4: BUSINESS MODEL
================================================================================

MONETIZATION:

1. FREE TIER
   - Unlimited conversations
   - Basic vendor search
   - Event planning assistance
   - Limited to 3 vendor comparisons/day

2. PREMIUM ($9.99/month)
   - Unlimited comparisons
   - Priority vendor matching
   - Price negotiation assistance
   - Vendor availability alerts
   - Chat export and sharing

3. PRO ($29.99/month)
   - Everything in Premium
   - Dedicated event concierge
   - Multi-event management
   - Team collaboration
   - API access

4. VENDOR REVENUE
   - Lead generation fees
   - Featured placement in conversations
   - Instant booking commission

================================================================================
*/

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
