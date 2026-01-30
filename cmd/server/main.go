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
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/api/bookings"
	"github.com/BillyRonksGlobal/vendorplatform/api/vendors"
	"github.com/BillyRonksGlobal/vendorplatform/internal/booking"
	apiauth "github.com/BillyRonksGlobal/vendorplatform/api/auth"
	"github.com/BillyRonksGlobal/vendorplatform/api/vendors"
	"github.com/BillyRonksGlobal/vendorplatform/internal/auth"
	"github.com/BillyRonksGlobal/vendorplatform/internal/vendor"
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
	config *Config
	db     *pgxpool.Pool
	cache  *redis.Client
	logger *zap.Logger
	router *gin.Engine
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

	// Create application
	app := &App{
		config: config,
		db:     db,
		cache:  cache,
		logger: logger,
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
	vendorService := vendor.NewService(app.db, app.cache)
	bookingService := booking.NewService(app.db, app.cache)

	// Initialize handlers
	authHandler := apiauth.NewHandler(authService, app.logger)
	vendorHandler := vendors.NewHandler(vendorService, app.logger)
	bookingHandler := bookings.NewHandler(bookingService, app.logger)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication (public)
		authHandler.RegisterRoutes(v1)

		// Vendor Management
		vendorHandler.RegisterRoutes(v1)

		// Booking Management
		bookingHandler.RegisterRoutes(v1)

		// LifeOS - Life Event Orchestration
		lifeos := v1.Group("/lifeos")
		{
			lifeos.POST("/events", app.createLifeEvent)
			lifeos.GET("/events/:id", app.getLifeEvent)
			lifeos.GET("/events/:id/plan", app.getEventPlan)
			lifeos.POST("/events/:id/confirm", app.confirmDetectedEvent)
			lifeos.GET("/detected", app.getDetectedEvents)
		}

		// EventGPT - Conversational AI Planner
		eventgpt := v1.Group("/eventgpt")
		{
			eventgpt.POST("/conversations", app.startConversation)
			eventgpt.POST("/conversations/:id/messages", app.sendMessage)
			eventgpt.GET("/conversations/:id", app.getConversation)
			eventgpt.DELETE("/conversations/:id", app.endConversation)
		}

		// VendorNet - B2B Partnership Network
		vendornet := v1.Group("/vendornet")
		{
			vendornet.GET("/partners/matches", app.getPartnerMatches)
			vendornet.POST("/partnerships", app.createPartnership)
			vendornet.GET("/partnerships/:id", app.getPartnership)
			vendornet.POST("/referrals", app.createReferral)
			vendornet.PUT("/referrals/:id/status", app.updateReferralStatus)
			vendornet.GET("/analytics", app.getNetworkAnalytics)
		}

		// HomeRescue - Emergency Services
		homerescue := v1.Group("/homerescue")
		{
			homerescue.POST("/emergencies", app.createEmergency)
			homerescue.GET("/emergencies/:id", app.getEmergencyStatus)
			homerescue.GET("/emergencies/:id/tracking", app.getEmergencyTracking)
			homerescue.POST("/technicians/location", app.updateTechLocation)
			homerescue.PUT("/emergencies/:id/accept", app.acceptEmergency)
			homerescue.PUT("/emergencies/:id/complete", app.completeEmergency)
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

// Placeholder handlers (to be implemented with actual logic)
func (app *App) createLifeEvent(c *gin.Context)       { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) getLifeEvent(c *gin.Context)          { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) getEventPlan(c *gin.Context)          { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) confirmDetectedEvent(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) getDetectedEvents(c *gin.Context)     { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }

func (app *App) startConversation(c *gin.Context)     { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) sendMessage(c *gin.Context)           { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) getConversation(c *gin.Context)       { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) endConversation(c *gin.Context)       { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }

func (app *App) getPartnerMatches(c *gin.Context)     { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) createPartnership(c *gin.Context)     { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) getPartnership(c *gin.Context)        { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) createReferral(c *gin.Context)        { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) updateReferralStatus(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) getNetworkAnalytics(c *gin.Context)   { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }

func (app *App) createEmergency(c *gin.Context)       { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) getEmergencyStatus(c *gin.Context)    { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) getEmergencyTracking(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) updateTechLocation(c *gin.Context)    { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) acceptEmergency(c *gin.Context)       { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) completeEmergency(c *gin.Context)     { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }

func (app *App) getServiceRecommendations(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) getVendorRecommendations(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) getBundleRecommendations(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
