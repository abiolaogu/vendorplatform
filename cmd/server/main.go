// VendorPlatform - Contextual Commerce Orchestration
// Copyright (c) 2024 BillyRonks Global Limited. All rights reserved.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	apiauth "github.com/BillyRonksGlobal/vendorplatform/api/auth"
	"github.com/BillyRonksGlobal/vendorplatform/api/bookings"
	"github.com/BillyRonksGlobal/vendorplatform/api/payments"
	"github.com/BillyRonksGlobal/vendorplatform/api/reviews"
	"github.com/BillyRonksGlobal/vendorplatform/api/vendors"
	homerescueAPI "github.com/BillyRonksGlobal/vendorplatform/api/homerescue"
	lifeosAPI "github.com/BillyRonksGlobal/vendorplatform/api/lifeos"
	vendornetAPI "github.com/BillyRonksGlobal/vendorplatform/api/vendornet"
	"github.com/BillyRonksGlobal/vendorplatform/internal/auth"
	"github.com/BillyRonksGlobal/vendorplatform/internal/booking"
	"github.com/BillyRonksGlobal/vendorplatform/internal/homerescue"
	"github.com/BillyRonksGlobal/vendorplatform/internal/lifeos"
	"github.com/BillyRonksGlobal/vendorplatform/internal/payment"
	"github.com/BillyRonksGlobal/vendorplatform/internal/review"
	"github.com/BillyRonksGlobal/vendorplatform/internal/service"
	"github.com/BillyRonksGlobal/vendorplatform/internal/vendor"
	"github.com/BillyRonksGlobal/vendorplatform/internal/vendornet"
	"github.com/BillyRonksGlobal/vendorplatform/recommendation-engine"
)

// Config holds application configuration
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	Environment string
}

// App holds the application dependencies
type App struct {
	config            *Config
	db                *pgxpool.Pool
	cache             *redis.Client
	logger            *zap.Logger
	router            *gin.Engine
	recommendationEngine *recommendation.Engine
}

func main() {
	// Load configuration
	config := loadConfig()

	// Initialize logger
	logger := initLogger(config.Environment)
	defer logger.Sync()

	// Initialize database connection
	db, err := initDatabase(config.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Redis connection
	cache, err := initRedis(config.RedisURL)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer cache.Close()

	// Initialize recommendation engine
	recEngine, err := initRecommendationEngine(db, cache, logger)
	if err != nil {
		logger.Fatal("Failed to initialize recommendation engine", zap.Error(err))
	}

	// Create application
	app := &App{
		config:               config,
		db:                   db,
		cache:                cache,
		logger:               logger,
		recommendationEngine: recEngine,
	}

	// Setup router
	app.setupRouter()

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      app.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting server", zap.String("port", config.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}

func loadConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost:5432/vendorplatform"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		Environment: getEnv("ENV", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initLogger(env string) *zap.Logger {
	var logger *zap.Logger
	var err error

	if env == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger
}

func initDatabase(url string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Connection pool settings
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

func initRedis(url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}

func initRecommendationEngine(db *pgxpool.Pool, cache *redis.Client, logger *zap.Logger) (*recommendation.Engine, error) {
	logger.Info("Initializing recommendation engine...")

	config := recommendation.DefaultConfig()

	engine, err := recommendation.NewEngine(db, cache, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create recommendation engine: %w", err)
	}

	logger.Info("Recommendation engine initialized successfully")
	return engine, nil
}

func (app *App) setupRouter() {
	if app.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(app.loggingMiddleware())
	router.Use(app.corsMiddleware())

	// Health check
	router.GET("/health", app.healthCheck)
	router.GET("/ready", app.readinessCheck)

	// Initialize services
	authConfig := &auth.Config{
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production-please"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BCryptCost:         12,
		MaxSessionsPerUser: 5,
		VerificationExpiry: 24 * time.Hour,
	}
	authService := auth.NewService(app.db, app.cache, authConfig)

	paymentConfig := &payment.Config{
		PaystackSecretKey:    getEnv("PAYSTACK_SECRET_KEY", ""),
		PaystackPublicKey:    getEnv("PAYSTACK_PUBLIC_KEY", ""),
		FlutterwaveSecretKey: getEnv("FLUTTERWAVE_SECRET_KEY", ""),
		FlutterwavePublicKey: getEnv("FLUTTERWAVE_PUBLIC_KEY", ""),
		WebhookSecret:        getEnv("WEBHOOK_SECRET", ""),
		DefaultCurrency:      "NGN",
		PlatformFeePercent:   10.0, // 10% platform fee
		EscrowExpiryDays:     30,   // 30 days escrow expiry
	}
	paymentService := payment.NewService(app.db, app.cache, paymentConfig)

	vendorService := vendor.NewService(app.db, app.cache)
	serviceManager := service.NewServiceManager(app.db, app.cache)
	homerescueService := homerescue.NewService(app.db, app.cache, app.logger)
	lifeosService := lifeos.NewService(app.db, app.cache)
	bookingService := booking.NewService(app.db, app.cache)
	reviewService := review.NewService(app.db, app.cache)
	vendornetService := vendornet.NewService(app.db, app.cache)

	// Initialize handlers
	authHandler := apiauth.NewHandler(authService, app.logger)
	paymentHandler := payments.NewHandler(paymentService, app.logger)
	vendorHandler := vendors.NewHandler(vendorService, serviceManager, app.logger)
	homerescueHandler := homerescueAPI.NewHandler(homerescueService, app.logger)
	lifeosHandler := lifeosAPI.NewHandler(lifeosService, app.logger)
	bookingHandler := bookings.NewHandler(bookingService, app.logger)
	reviewHandler := reviews.NewHandler(reviewService, app.logger)
	vendornetHandler := vendornetAPI.NewHandler(vendornetService, app.logger)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication (public)
		authHandler.RegisterRoutes(v1)

		// Payment Processing
		paymentHandler.RegisterRoutes(v1)

		// Vendor Management
		vendorHandler.RegisterRoutes(v1)

		// HomeRescue - Emergency Services
		homerescueHandler.RegisterRoutes(v1)
		// Booking Management
		bookingHandler.RegisterRoutes(v1)

		// Review & Rating System
		reviewHandler.RegisterRoutes(v1)

		// Payment Processing & Escrow
		paymentHandler.RegisterRoutes(v1)

		// LifeOS - Life Event Orchestration
		lifeosHandler.RegisterRoutes(v1)

		// VendorNet - B2B Partnership Network
		vendornetHandler.RegisterRoutes(v1)

		// EventGPT - Conversational AI Planner
		eventgpt := v1.Group("/eventgpt")
		{
			eventgpt.POST("/conversations", app.startConversation)
			eventgpt.POST("/conversations/:id/messages", app.sendMessage)
			eventgpt.GET("/conversations/:id", app.getConversation)
			eventgpt.DELETE("/conversations/:id", app.endConversation)
		}

		// HomeRescue - Emergency Services
		homerescue := v1.Group("/homerescue")
		{
			homerescue.POST("/emergencies", homerescueHandler.CreateEmergency)
			homerescue.GET("/emergencies/:id", homerescueHandler.GetEmergencyStatus)
			homerescue.GET("/emergencies/:id/tracking", homerescueHandler.GetEmergencyTracking)
			homerescue.POST("/technicians/location", homerescueHandler.UpdateTechLocation)
			homerescue.PUT("/emergencies/:id/accept", homerescueHandler.AcceptEmergency)
			homerescue.PUT("/emergencies/:id/complete", homerescueHandler.CompleteEmergency)
		}

		// Recommendations
		recommendations := v1.Group("/recommendations")
		{
			recommendations.GET("/services", app.getServiceRecommendations)
			recommendations.GET("/vendors", app.getVendorRecommendations)
			recommendations.GET("/bundles", app.getBundleRecommendations)
		}
	}

	app.router = router
}

// Middleware
func (app *App) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		app.logger.Info("Request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		)
	}
}

func (app *App) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Health checks
func (app *App) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "vendorplatform",
		"version": "1.0.0",
	})
}

func (app *App) readinessCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Check database
	if err := app.db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "database connection failed",
		})
		return
	}

	// Check Redis
	if err := app.cache.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "cache connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"checks": gin.H{
			"database": "ok",
			"cache":    "ok",
		},
	})
}

// =============================================================================
// EVENTGPT CONVERSATION HANDLERS
// =============================================================================

// startConversation initializes a new EventGPT conversation session
func (app *App) startConversation(c *gin.Context) {
	// Get user ID from context (would be set by auth middleware)
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		// For testing, allow anonymous conversations
		userIDStr = uuid.New().String()
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Create new conversation
	conversationID := uuid.New()
	now := time.Now()

	// Store in database
	query := `
		INSERT INTO conversations (
			id, user_id, session_type, conversation_state,
			current_intent, slot_values, messages, turn_count,
			short_term_memory, language, channel, started_at, last_message_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	channel := c.DefaultQuery("channel", "web")
	emptyJSON := []byte("{}")
	emptyArray := []byte("[]")

	_, err = app.db.Exec(c.Request.Context(), query,
		conversationID, userID, "general_inquiry", "welcome",
		emptyJSON, emptyJSON, emptyArray, 0,
		emptyJSON, "en", channel, now, now,
	)

	if err != nil {
		app.logger.Error("Failed to create conversation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start conversation"})
		return
	}

	// Return welcome message
	c.JSON(http.StatusCreated, gin.H{
		"conversation_id": conversationID.String(),
		"message": gin.H{
			"role":    "assistant",
			"content": "Hello! 👋 I'm EventGPT, your AI event planning assistant. I can help you plan weddings, birthdays, corporate events, and more. What are you celebrating?",
			"quick_replies": []gin.H{
				{"title": "Plan a wedding", "payload": "create_event:wedding"},
				{"title": "Plan a birthday", "payload": "create_event:birthday"},
				{"title": "Find a vendor", "payload": "find_vendor"},
				{"title": "Get recommendations", "payload": "get_recommendation"},
			},
		},
		"session_type": "general_inquiry",
	})
}

// sendMessage processes a user message in an EventGPT conversation
func (app *App) sendMessage(c *gin.Context) {
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

	// Get conversation from database
	var conv struct {
		UserID    uuid.UUID
		State     string
		TurnCount int
		Messages  []byte
	}

	query := `
		SELECT user_id, conversation_state, turn_count, messages
		FROM conversations
		WHERE id = $1
	`

	err = app.db.QueryRow(c.Request.Context(), query, conversationID).Scan(
		&conv.UserID, &conv.State, &conv.TurnCount, &conv.Messages,
	)

	if err != nil {
		app.logger.Error("Failed to get conversation", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	// Simple intent classification based on keywords
	userMessage := req.Message
	intent := classifyIntent(userMessage)

	// Generate response based on intent
	responseText, quickReplies := generateResponse(intent, conv.State)

	// Update conversation in database
	now := time.Now()
	updateQuery := `
		UPDATE conversations
		SET turn_count = turn_count + 1,
		    last_message_at = $2,
		    conversation_state = $3
		WHERE id = $1
	`

	newState := determineNextState(intent, conv.State)

	_, err = app.db.Exec(c.Request.Context(), updateQuery,
		conversationID, now, newState,
	)

	if err != nil {
		app.logger.Warn("Failed to update conversation", zap.Error(err))
	}

	// Return response
	response := gin.H{
		"conversation_id": conversationID.String(),
		"message": gin.H{
			"role":      "assistant",
			"content":   responseText,
			"timestamp": now,
		},
	}

	if len(quickReplies) > 0 {
		response["message"].(gin.H)["quick_replies"] = quickReplies
	}

	c.JSON(http.StatusOK, response)
}

// getConversation retrieves conversation history
func (app *App) getConversation(c *gin.Context) {
	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	// Get conversation from database
	query := `
		SELECT id, user_id, session_type, conversation_state,
		       turn_count, started_at, last_message_at
		FROM conversations
		WHERE id = $1
	`

	var conv struct {
		ID            uuid.UUID
		UserID        uuid.UUID
		SessionType   string
		State         string
		TurnCount     int
		StartedAt     time.Time
		LastMessageAt time.Time
	}

	err = app.db.QueryRow(c.Request.Context(), query, conversationID).Scan(
		&conv.ID, &conv.UserID, &conv.SessionType, &conv.State,
		&conv.TurnCount, &conv.StartedAt, &conv.LastMessageAt,
	)

	if err != nil {
		app.logger.Error("Failed to get conversation", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":               conv.ID.String(),
		"user_id":          conv.UserID.String(),
		"session_type":     conv.SessionType,
		"conversation_state": conv.State,
		"turn_count":       conv.TurnCount,
		"started_at":       conv.StartedAt,
		"last_message_at":  conv.LastMessageAt,
	})
}

// endConversation closes an EventGPT conversation session
func (app *App) endConversation(c *gin.Context) {
	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	// Update conversation to completed state
	now := time.Now()
	query := `
		UPDATE conversations
		SET conversation_state = 'completed',
		    last_message_at = $2
		WHERE id = $1
	`

	result, err := app.db.Exec(c.Request.Context(), query, conversationID, now)
	if err != nil {
		app.logger.Error("Failed to end conversation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to end conversation"})
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Conversation ended successfully. Thank you for using EventGPT!",
		"conversation_id": conversationID.String(),
	})
}

// Helper functions for EventGPT conversation logic

func classifyIntent(message string) string {
	message = strings.ToLower(message)

	// Simple keyword-based intent classification
	if strings.Contains(message, "wedding") || strings.Contains(message, "married") {
		return "create_event_wedding"
	}
	if strings.Contains(message, "birthday") || strings.Contains(message, "bday") {
		return "create_event_birthday"
	}
	if strings.Contains(message, "find") && (strings.Contains(message, "vendor") ||
		strings.Contains(message, "photographer") || strings.Contains(message, "caterer")) {
		return "find_vendor"
	}
	if strings.Contains(message, "quote") || strings.Contains(message, "price") || strings.Contains(message, "cost") {
		return "get_quote"
	}
	if strings.Contains(message, "book") || strings.Contains(message, "hire") {
		return "book_service"
	}
	if strings.Contains(message, "recommend") || strings.Contains(message, "suggest") {
		return "get_recommendation"
	}
	if strings.Contains(message, "thank") {
		return "thanks"
	}
	if strings.Contains(message, "hi") || strings.Contains(message, "hello") || strings.Contains(message, "hey") {
		return "greeting"
	}

	return "general_question"
}

func generateResponse(intent string, currentState string) (string, []gin.H) {
	switch intent {
	case "create_event_wedding":
		return "Great! Let's plan your wedding. 💍 When is the big day? You can give me an exact date or just a general timeframe.",
			[]gin.H{
				{"title": "Next month", "payload": "date:next_month"},
				{"title": "In 6 months", "payload": "date:6_months"},
				{"title": "Next year", "payload": "date:next_year"},
			}

	case "create_event_birthday":
		return "Awesome! Planning a birthday party. 🎂 How many guests are you expecting?",
			[]gin.H{
				{"title": "10-20 guests", "payload": "guests:15"},
				{"title": "20-50 guests", "payload": "guests:35"},
				{"title": "50+ guests", "payload": "guests:75"},
			}

	case "find_vendor":
		return "I can help you find the perfect vendor! What type of service are you looking for?",
			[]gin.H{
				{"title": "Photographer", "payload": "vendor:photographer"},
				{"title": "Caterer", "payload": "vendor:caterer"},
				{"title": "Decorator", "payload": "vendor:decorator"},
				{"title": "DJ/Entertainment", "payload": "vendor:dj"},
			}

	case "get_quote":
		return "I'd be happy to help you get pricing estimates. What service are you interested in?",
			nil

	case "book_service":
		return "Ready to book! To help you with booking, I'll need a few details. Which vendor or service are you interested in?",
			nil

	case "get_recommendation":
		return "I can recommend the best vendors for your event! What type of event are you planning?",
			[]gin.H{
				{"title": "Wedding", "payload": "event:wedding"},
				{"title": "Birthday Party", "payload": "event:birthday"},
				{"title": "Corporate Event", "payload": "event:corporate"},
				{"title": "Other", "payload": "event:other"},
			}

	case "thanks":
		return "You're welcome! Is there anything else I can help you with? 😊",
			[]gin.H{
				{"title": "Continue planning", "payload": "continue"},
				{"title": "That's all for now", "payload": "end"},
			}

	case "greeting":
		return "Hello! Welcome back! How can I assist you with your event planning today?",
			nil

	default:
		return "I understand. Let me help you with that. Could you provide a bit more detail about what you're looking for?",
			nil
	}
}

func determineNextState(intent string, currentState string) string {
	switch intent {
	case "create_event_wedding", "create_event_birthday":
		return "gathering_info"
	case "find_vendor":
		return "recommending"
	case "book_service":
		return "booking"
	case "get_quote":
		return "recommending"
	case "thanks":
		return "completed"
	default:
		return currentState
	}
}

// HomeRescue handlers are now implemented in api/homerescue/handlers.go
// VendorNet handlers are now implemented in api/vendornet/handlers.go

// getServiceRecommendations returns adjacent service recommendations based on context
func (app *App) getServiceRecommendations(c *gin.Context) {
	// Parse query parameters
	categoryID := c.Query("category_id")
	serviceID := c.Query("service_id")
	eventType := c.Query("event_type")
	userID := c.Query("user_id")
	limitStr := c.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	// Build recommendation request
	req := &recommendation.RecommendationRequest{
		EventType: eventType,
		Limit:     limit,
		RequestedTypes: []recommendation.RecommendationType{
			recommendation.AdjacentService,
			recommendation.EventBasedSuggest,
		},
	}

	// Parse user ID if provided
	if userID != "" {
		if uid, err := uuid.Parse(userID); err == nil {
			req.UserID = uid
		}
	}

	// Parse current entity context
	if serviceID != "" {
		if sid, err := uuid.Parse(serviceID); err == nil {
			req.CurrentEntityID = sid
			req.CurrentEntityType = recommendation.EntityService
		}
	} else if categoryID != "" {
		if cid, err := uuid.Parse(categoryID); err == nil {
			req.CurrentEntityID = cid
			req.CurrentEntityType = recommendation.EntityCategory
		}
	}

	// Parse location if provided
	latStr := c.Query("latitude")
	lonStr := c.Query("longitude")
	if latStr != "" && lonStr != "" {
		if lat, errLat := strconv.ParseFloat(latStr, 64); errLat == nil {
			if lon, errLon := strconv.ParseFloat(lonStr, 64); errLon == nil {
				req.Location = &recommendation.GeoPoint{
					Latitude:  lat,
					Longitude: lon,
				}
			}
		}
	}

	// Get recommendations from engine
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := app.recommendationEngine.GetRecommendations(ctx, req)
	if err != nil {
		app.logger.Error("Failed to get service recommendations",
			zap.Error(err),
			zap.String("service_id", serviceID),
			zap.String("category_id", categoryID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate recommendations",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recommendations": resp.Recommendations,
		"total_candidates": resp.TotalCandidates,
		"processing_time_ms": resp.ProcessingTimeMs,
		"algorithm_version": resp.AlgorithmVersion,
	})
}

// getVendorRecommendations returns similar or complementary vendor recommendations
func (app *App) getVendorRecommendations(c *gin.Context) {
	vendorID := c.Query("vendor_id")
	categoryID := c.Query("category_id")
	userID := c.Query("user_id")
	limitStr := c.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	// Build recommendation request
	req := &recommendation.RecommendationRequest{
		Limit: limit,
		RequestedTypes: []recommendation.RecommendationType{
			recommendation.SimilarVendor,
		},
	}

	// Parse user ID if provided
	if userID != "" {
		if uid, err := uuid.Parse(userID); err == nil {
			req.UserID = uid
		}
	}

	// Parse vendor context
	if vendorID != "" {
		if vid, err := uuid.Parse(vendorID); err == nil {
			req.CurrentEntityID = vid
			req.CurrentEntityType = recommendation.EntityVendor
		}
	} else if categoryID != "" {
		if cid, err := uuid.Parse(categoryID); err == nil {
			req.CurrentEntityID = cid
			req.CurrentEntityType = recommendation.EntityCategory
		}
	}

	// Parse location if provided
	latStr := c.Query("latitude")
	lonStr := c.Query("longitude")
	if latStr != "" && lonStr != "" {
		if lat, errLat := strconv.ParseFloat(latStr, 64); errLat == nil {
			if lon, errLon := strconv.ParseFloat(lonStr, 64); errLon == nil {
				req.Location = &recommendation.GeoPoint{
					Latitude:  lat,
					Longitude: lon,
				}
			}
		}
	}

	// Get recommendations from engine
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := app.recommendationEngine.GetRecommendations(ctx, req)
	if err != nil {
		app.logger.Error("Failed to get vendor recommendations",
			zap.Error(err),
			zap.String("vendor_id", vendorID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate recommendations",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recommendations": resp.Recommendations,
		"total_candidates": resp.TotalCandidates,
		"processing_time_ms": resp.ProcessingTimeMs,
		"algorithm_version": resp.AlgorithmVersion,
	})
}

// getBundleRecommendations returns service bundle recommendations for events
func (app *App) getBundleRecommendations(c *gin.Context) {
	eventType := c.Query("event_type")
	userID := c.Query("user_id")
	projectID := c.Query("project_id")
	budgetStr := c.Query("budget")
	limitStr := c.DefaultQuery("limit", "5")

	if eventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_type parameter is required",
		})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 5
	}

	// Build recommendation request
	req := &recommendation.RecommendationRequest{
		EventType: eventType,
		Limit:     limit,
		RequestedTypes: []recommendation.RecommendationType{
			recommendation.BundleSuggestion,
			recommendation.EventBasedSuggest,
		},
		DiversityFactor: 0.5, // Bundles should have good category diversity
	}

	// Parse user ID if provided
	if userID != "" {
		if uid, err := uuid.Parse(userID); err == nil {
			req.UserID = uid
		}
	}

	// Parse project ID if provided
	if projectID != "" {
		if pid, err := uuid.Parse(projectID); err == nil {
			req.ProjectID = pid
		}
	}

	// Parse budget if provided
	if budgetStr != "" {
		if budget, err := strconv.ParseFloat(budgetStr, 64); err == nil {
			req.Budget = &recommendation.BudgetRange{
				Max:      budget,
				Currency: "NGN", // Default to Nigerian Naira
			}
		}
	}

	// Parse location if provided
	latStr := c.Query("latitude")
	lonStr := c.Query("longitude")
	if latStr != "" && lonStr != "" {
		if lat, errLat := strconv.ParseFloat(latStr, 64); errLat == nil {
			if lon, errLon := strconv.ParseFloat(lonStr, 64); errLon == nil {
				req.Location = &recommendation.GeoPoint{
					Latitude:  lat,
					Longitude: lon,
				}
			}
		}
	}

	// Get recommendations from engine
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := app.recommendationEngine.GetRecommendations(ctx, req)
	if err != nil {
		app.logger.Error("Failed to get bundle recommendations",
			zap.Error(err),
			zap.String("event_type", eventType),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate bundle recommendations",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"event_type": eventType,
		"recommendations": resp.Recommendations,
		"total_candidates": resp.TotalCandidates,
		"processing_time_ms": resp.ProcessingTimeMs,
		"algorithm_version": resp.AlgorithmVersion,
	})
}
