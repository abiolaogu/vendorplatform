// Package eventgpt provides HTTP handlers for EventGPT
package eventgpt

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/eventgpt"
)

// Handler handles EventGPT HTTP requests
type Handler struct {
	service *eventgpt.Service
	logger  *zap.Logger
}

// NewHandler creates a new EventGPT handler
func NewHandler(service *eventgpt.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers EventGPT routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	eventgpt := router.Group("/eventgpt")
	{
		eventgpt.POST("/conversations", h.StartConversation)
		eventgpt.POST("/conversations/:id/messages", h.SendMessage)
		eventgpt.GET("/conversations/:id", h.GetConversation)
		eventgpt.DELETE("/conversations/:id", h.EndConversation)
		eventgpt.GET("/conversations", h.ListConversations)
	}
}

// StartConversation handles POST /api/v1/eventgpt/conversations
func (h *Handler) StartConversation(c *gin.Context) {
	var req struct {
		Channel string `json:"channel" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Get user ID from context (would come from auth middleware)
	// For now, use a test user ID
	userID := uuid.New() // In production, get from auth context

	var channel Channel
	switch req.Channel {
	case "web":
		channel = ChannelWeb
	case "mobile":
		channel = ChannelMobile
	case "whatsapp":
		channel = ChannelWhatsApp
	default:
		channel = ChannelWeb
	}

	conv, err := h.service.StartConversation(c.Request.Context(), userID, channel)
	if err != nil {
		h.logger.Error("Failed to start conversation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create conversation",
		})
		return
	}

	// Send welcome message
	welcomeMsg := Message{
		ID:      uuid.New(),
		Role:    RoleAssistant,
		Content: "Hello! 👋 I'm EventGPT, your AI event planning assistant. I can help you plan weddings, birthdays, corporate events, and more. What are you celebrating?",
		QuickReplies: []QuickReply{
			{Title: "Plan a wedding", Payload: "create_event:wedding"},
			{Title: "Plan a birthday", Payload: "create_event:birthday"},
			{Title: "Find a vendor", Payload: "find_vendor"},
			{Title: "Get recommendations", Payload: "get_recommendation"},
		},
	}

	conv.Messages = append(conv.Messages, welcomeMsg)

	c.JSON(http.StatusCreated, gin.H{
		"conversation_id": conv.ID,
		"message":         welcomeMsg,
		"session_type":    conv.SessionType,
		"state":           conv.ConversationState,
	})
}

// SendMessage handles POST /api/v1/eventgpt/conversations/:id/messages
func (h *Handler) SendMessage(c *gin.Context) {
	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid conversation ID",
		})
		return
	}

	var req struct {
		Message     string       `json:"message" binding:"required"`
		Attachments []Attachment `json:"attachments"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Get user ID from context
	userID := uuid.New() // In production, get from auth context

	chatReq := ChatRequest{
		ConversationID: &conversationID,
		Message:        req.Message,
		Channel:        ChannelWeb,
		Attachments:    req.Attachments,
	}

	response, err := h.service.SendMessage(c.Request.Context(), userID, chatReq)
	if err != nil {
		h.logger.Error("Failed to send message",
			zap.Error(err),
			zap.String("conversation_id", conversationID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process message",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation_id": response.ConversationID,
		"message":         response.Message,
		"event_id":        response.EventID,
		"session_type":    response.SessionType,
	})
}

// GetConversation handles GET /api/v1/eventgpt/conversations/:id
func (h *Handler) GetConversation(c *gin.Context) {
	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid conversation ID",
		})
		return
	}

	conv, err := h.service.GetConversation(c.Request.Context(), conversationID)
	if err != nil {
		h.logger.Error("Failed to get conversation",
			zap.Error(err),
			zap.String("conversation_id", conversationID.String()),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Conversation not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                 conv.ID,
		"user_id":            conv.UserID,
		"event_id":           conv.EventID,
		"session_type":       conv.SessionType,
		"conversation_state": conv.ConversationState,
		"messages":           conv.Messages,
		"turn_count":         conv.TurnCount,
		"started_at":         conv.StartedAt,
		"last_message_at":    conv.LastMessageAt,
		"slot_values":        conv.SlotValues,
	})
}

// EndConversation handles DELETE /api/v1/eventgpt/conversations/:id
func (h *Handler) EndConversation(c *gin.Context) {
	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid conversation ID",
		})
		return
	}

	if err := h.service.EndConversation(c.Request.Context(), conversationID); err != nil {
		h.logger.Error("Failed to end conversation",
			zap.Error(err),
			zap.String("conversation_id", conversationID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to end conversation",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Conversation ended successfully",
	})
}

// ListConversations handles GET /api/v1/eventgpt/conversations
func (h *Handler) ListConversations(c *gin.Context) {
	// Get user ID from context
	userID := uuid.New() // In production, get from auth context

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := uuid.Parse(limitStr); err == nil {
			_ = l // Would parse as int
		}
	}

	conversations, err := h.service.GetUserConversations(c.Request.Context(), userID, limit)
	if err != nil {
		h.logger.Error("Failed to list conversations",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve conversations",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": conversations,
		"count":         len(conversations),
	})
}
