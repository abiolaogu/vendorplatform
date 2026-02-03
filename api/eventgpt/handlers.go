// Package eventgpt provides HTTP handlers for conversational AI event planning
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
	eventgptRoutes := router.Group("/eventgpt")
	{
		eventgptRoutes.POST("/conversations", h.CreateConversation)
		eventgptRoutes.POST("/conversations/:id/messages", h.SendMessage)
		eventgptRoutes.GET("/conversations/:id", h.GetConversation)
		eventgptRoutes.DELETE("/conversations/:id", h.EndConversation)
	}
}

// CreateConversation handles POST /api/v1/eventgpt/conversations
// @Summary Start a new conversation with EventGPT
// @Description Creates a new conversation session for event planning assistance
// @Tags EventGPT
// @Accept json
// @Produce json
// @Param body body eventgpt.CreateConversationRequest true "Conversation details"
// @Success 201 {object} eventgpt.Conversation
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/eventgpt/conversations [post]
func (h *Handler) CreateConversation(c *gin.Context) {
	var req eventgpt.CreateConversationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind create conversation request",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// TODO: Get user_id from authenticated session
	// For now, using the user_id from request body
	if req.UserID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	conversation, err := h.service.CreateConversation(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create conversation",
			zap.Error(err),
			zap.String("user_id", req.UserID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	// Reload conversation with messages to include welcome message
	fullConv, err := h.service.GetConversation(c.Request.Context(), conversation.ID, req.UserID)
	if err != nil {
		h.logger.Error("Failed to reload conversation with messages",
			zap.Error(err),
			zap.String("conversation_id", conversation.ID.String()),
		)
		// Return conversation without messages as fallback
		c.JSON(http.StatusCreated, conversation)
		return
	}

	c.JSON(http.StatusCreated, fullConv)
}

// SendMessage handles POST /api/v1/eventgpt/conversations/:id/messages
// @Summary Send a message in a conversation
// @Description Sends a user message and receives AI assistant response
// @Tags EventGPT
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Param body body eventgpt.SendMessageRequest true "Message content"
// @Success 200 {object} eventgpt.MessageResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/eventgpt/conversations/{id}/messages [post]
func (h *Handler) SendMessage(c *gin.Context) {
	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var req eventgpt.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind send message request",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message content is required"})
		return
	}

	// TODO: Get user_id from authenticated session
	// For now, we need to get it from query param or request
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required (query param)"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	response, err := h.service.SendMessage(c.Request.Context(), conversationID, userID, &req)
	if err != nil {
		h.logger.Error("Failed to send message",
			zap.Error(err),
			zap.String("conversation_id", conversationID.String()),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process message"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetConversation handles GET /api/v1/eventgpt/conversations/:id
// @Summary Get conversation history
// @Description Retrieves a conversation with all its messages
// @Tags EventGPT
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 200 {object} eventgpt.Conversation
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/eventgpt/conversations/{id} [get]
func (h *Handler) GetConversation(c *gin.Context) {
	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	// TODO: Get user_id from authenticated session
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required (query param)"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	conversation, err := h.service.GetConversation(c.Request.Context(), conversationID, userID)
	if err != nil {
		h.logger.Error("Failed to get conversation",
			zap.Error(err),
			zap.String("conversation_id", conversationID.String()),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	c.JSON(http.StatusOK, conversation)
}

// EndConversation handles DELETE /api/v1/eventgpt/conversations/:id
// @Summary End a conversation
// @Description Marks a conversation as completed
// @Tags EventGPT
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/eventgpt/conversations/{id} [delete]
func (h *Handler) EndConversation(c *gin.Context) {
	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	// TODO: Get user_id from authenticated session
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required (query param)"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	err = h.service.EndConversation(c.Request.Context(), conversationID, userID)
	if err != nil {
		h.logger.Error("Failed to end conversation",
			zap.Error(err),
			zap.String("conversation_id", conversationID.String()),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to end conversation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Conversation ended successfully",
		"conversation_id": conversationID.String(),
	})
}
