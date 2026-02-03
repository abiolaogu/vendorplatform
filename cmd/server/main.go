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
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	apiauth "github.com/BillyRonksGlobal/vendorplatform/api/auth"
	"github.com/BillyRonksGlobal/vendorplatform/api/bookings"
	eventgptAPI "github.com/BillyRonksGlobal/vendorplatform/api/eventgpt"
	"github.com/BillyRonksGlobal/vendorplatform/api/payments"
	"github.com/BillyRonksGlobal/vendorplatform/api/reviews"
	"github.com/BillyRonksGlobal/vendorplatform/api/vendors"
	homerescueAPI "github.com/BillyRonksGlobal/vendorplatform/api/homerescue"
	lifeosAPI "github.com/BillyRonksGlobal/vendorplatform/api/lifeos"
	"github.com/BillyRonksGlobal/vendorplatform/internal/auth"
	"github.com/BillyRonksGlobal/vendorplatform/internal/booking"
	"github.com/BillyRonksGlobal/vendorplatform/internal/eventgpt"
	"github.com/BillyRonksGlobal/vendorplatform/internal/homerescue"
	"github.com/BillyRonksGlobal/vendorplatform/internal/lifeos"
	"github.com/BillyRonksGlobal/vendorplatform/internal/payment"
	"github.com/BillyRonksGlobal/vendorplatform/internal/review"
	"github.com/BillyRonksGlobal/vendorplatform/internal/service"
	"github.com/BillyRonksGlobal/vendorplatform/internal/vendor"
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
	eventgptService := eventgpt.NewService(app.db, app.cache)
	bookingService := booking.NewService(app.db, app.cache)
	reviewService := review.NewService(app.db, app.cache)

	paymentConfig := &payment.Config{
		PaystackSecretKey:    getEnv("PAYSTACK_SECRET_KEY", ""),
		PaystackPublicKey:    getEnv("PAYSTACK_PUBLIC_KEY", ""),
		FlutterwaveSecretKey: getEnv("FLUTTERWAVE_SECRET_KEY", ""),
		FlutterwavePublicKey: getEnv("FLUTTERWAVE_PUBLIC_KEY", ""),
		DefaultCurrency:      getEnv("DEFAULT_CURRENCY", "NGN"),
		PlatformFeePercent:   5.0, // 5% platform fee
		EscrowExpiryDays:     30,
	}
	paymentService := payment.NewService(app.db, app.cache, paymentConfig)

	// Initialize handlers
	authHandler := apiauth.NewHandler(authService, app.logger)
	paymentHandler := payments.NewHandler(paymentService, app.logger)
	vendorHandler := vendors.NewHandler(vendorService, serviceManager, app.logger)
	homerescueHandler := homerescueAPI.NewHandler(homerescueService, app.logger)
	lifeosHandler := lifeosAPI.NewHandler(lifeosService, app.logger)
	eventgptHandler := eventgptAPI.NewHandler(eventgptService, app.logger)
	bookingHandler := bookings.NewHandler(bookingService, app.logger)
	reviewHandler := reviews.NewHandler(reviewService, app.logger)

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

		// EventGPT - Conversational AI Planner
		eventgptHandler.RegisterRoutes(v1)

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

// Placeholder handlers (to be implemented with actual logic)
func (app *App) getPartnerMatches(c *gin.Context)     { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) createPartnership(c *gin.Context)     { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) getPartnership(c *gin.Context)        { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) createReferral(c *gin.Context)        { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) updateReferralStatus(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func (app *App) getNetworkAnalytics(c *gin.Context)   { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }

// HomeRescue handlers are now implemented in api/homerescue/handlers.go

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
