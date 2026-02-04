// EventGPT API Handlers
// Copyright (c) 2024 BillyRonks Global Limited. All rights reserved.

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
	eventgptGroup := router.Group("/eventgpt")
	{
		eventgptGroup.POST("/conversations", h.StartConversation)
		eventgptGroup.POST("/conversations/:id/messages", h.SendMessage)
		eventgptGroup.GET("/conversations/:id", h.GetConversation)
		eventgptGroup.DELETE("/conversations/:id", h.EndConversation)
	}
}

// StartConversation creates a new conversation
// POST /api/v1/eventgpt/conversations
func (h *Handler) StartConversation(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	conversation, err := h.service.StartConversation(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to start conversation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start conversation"})
		return
	}

	// Return conversation with initial message
	response := gin.H{
		"conversation_id": conversation.ID.String(),
		"state":          conversation.State,
		"message": gin.H{
			"role":      conversation.Messages[0].Role,
			"content":   conversation.Messages[0].Content,
			"timestamp": conversation.Messages[0].Timestamp,
		},
	}

	// Add quick replies if available
	if metadata := conversation.Messages[0].Metadata; metadata != nil {
		if quickReplies, ok := metadata["quick_replies"]; ok {
			response["message"].(gin.H)["quick_replies"] = quickReplies
		}
	}

	c.JSON(http.StatusCreated, response)
}

// SendMessage processes a user message
// POST /api/v1/eventgpt/conversations/:id/messages
func (h *Handler) SendMessage(c *gin.Context) {
	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var req struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
		return
	}

	// Process message through service
	responseMsg, err := h.service.ProcessMessage(c.Request.Context(), conversationID, req.Message)
	if err != nil {
		h.logger.Error("Failed to process message",
			zap.Error(err),
			zap.String("conversation_id", conversationID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process message"})
		return
	}

	// Get updated conversation
	conversation, err := h.service.GetConversation(c.Request.Context(), conversationID)
	if err != nil {
		h.logger.Warn("Failed to get conversation", zap.Error(err))
	}

	// Build response
	response := gin.H{
		"conversation_id": conversationID.String(),
		"message": gin.H{
			"id":        responseMsg.ID.String(),
			"role":      responseMsg.Role,
			"content":   responseMsg.Content,
			"timestamp": responseMsg.Timestamp,
		},
	}

	// Add metadata
	if responseMsg.Metadata != nil {
		if quickReplies, ok := responseMsg.Metadata["quick_replies"]; ok {
			response["message"].(gin.H)["quick_replies"] = quickReplies
		}
	}

	// Add conversation state if available
	if conversation != nil {
		response["state"] = conversation.State
		response["turn_count"] = conversation.TurnCount

		// Add extracted slots
		if len(conversation.Slots) > 0 {
			response["slots"] = conversation.Slots
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetConversation retrieves conversation history
// GET /api/v1/eventgpt/conversations/:id
func (h *Handler) GetConversation(c *gin.Context) {
	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	conversation, err := h.service.GetConversation(c.Request.Context(), conversationID)
	if err != nil {
		h.logger.Error("Failed to get conversation",
			zap.Error(err),
			zap.String("conversation_id", conversationID.String()),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	// Format messages
	messages := make([]gin.H, len(conversation.Messages))
	for i, msg := range conversation.Messages {
		messages[i] = gin.H{
			"id":        msg.ID.String(),
			"role":      msg.Role,
			"content":   msg.Content,
			"timestamp": msg.Timestamp,
		}

		if msg.Intent != "" {
			messages[i]["intent"] = msg.Intent
		}

		if len(msg.Slots) > 0 {
			messages[i]["slots"] = msg.Slots
		}

		if msg.Metadata != nil {
			messages[i]["metadata"] = msg.Metadata
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation_id": conversation.ID.String(),
		"user_id":        conversation.UserID.String(),
		"state":          conversation.State,
		"messages":       messages,
		"slots":          conversation.Slots,
		"turn_count":     conversation.TurnCount,
		"started_at":     conversation.StartedAt,
		"last_message_at": conversation.LastMessageAt,
		"ended_at":       conversation.EndedAt,
	})
}

// EndConversation marks a conversation as ended
// DELETE /api/v1/eventgpt/conversations/:id
func (h *Handler) EndConversation(c *gin.Context) {
	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	err = h.service.EndConversation(c.Request.Context(), conversationID)
	if err != nil {
		h.logger.Error("Failed to end conversation",
			zap.Error(err),
			zap.String("conversation_id", conversationID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to end conversation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Conversation ended successfully",
		"conversation_id": conversationID.String(),
	})
}
