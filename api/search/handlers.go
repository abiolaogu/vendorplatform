// =============================================================================
// SEARCH API HANDLERS
// HTTP handlers for search, autocomplete, and indexing operations
// =============================================================================

package search

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/search"
)

// Handler handles search HTTP requests
type Handler struct {
	service *search.Service
	logger  *zap.Logger
}

// NewHandler creates a new search handler
func NewHandler(service *search.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers all search routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	searchGroup := rg.Group("/search")
	{
		searchGroup.POST("", h.Search)
		searchGroup.GET("/suggest", h.Suggest)
		searchGroup.POST("/reindex", h.Reindex) // Admin only
	}
}

// =============================================================================
// HANDLERS
// =============================================================================

// Search handles POST /api/v1/search
// @Summary Perform full-text search
// @Description Search for vendors, services, and categories with filters
// @Tags Search
// @Accept json
// @Produce json
// @Param request body search.SearchRequest true "Search request"
// @Success 200 {object} search.SearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/search [post]
func (h *Handler) Search(c *gin.Context) {
	var req search.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid search request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Validate request
	if err := h.validateSearchRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	// Execute search
	resp, err := h.service.Search(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Search failed", zap.Error(err), zap.String("query", req.Query))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Search failed",
			Message: "An error occurred while searching. Please try again.",
		})
		return
	}

	h.logger.Info("Search executed",
		zap.String("query", req.Query),
		zap.String("type", string(req.Type)),
		zap.Int64("total_results", resp.Total),
		zap.Int64("took_ms", resp.TookMs),
	)

	c.JSON(http.StatusOK, resp)
}

// Suggest handles GET /api/v1/search/suggest
// @Summary Get autocomplete suggestions
// @Description Get autocomplete suggestions for a search query prefix
// @Tags Search
// @Accept json
// @Produce json
// @Param q query string true "Search prefix"
// @Param limit query int false "Max suggestions (default: 10)"
// @Success 200 {object} SuggestResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/search/suggest [get]
func (h *Handler) Suggest(c *gin.Context) {
	prefix := c.Query("q")
	if prefix == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Missing query parameter",
			Message: "Query parameter 'q' is required",
		})
		return
	}

	// Validate prefix length
	if len(prefix) < 2 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Query too short",
			Message: "Query must be at least 2 characters",
		})
		return
	}

	// Parse limit
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	// Enforce max limit
	if limit > 50 {
		limit = 50
	}

	// Get suggestions
	suggestions, err := h.service.Suggest(c.Request.Context(), prefix, limit)
	if err != nil {
		h.logger.Error("Suggestion failed", zap.Error(err), zap.String("prefix", prefix))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Suggestion failed",
			Message: "An error occurred while fetching suggestions. Please try again.",
		})
		return
	}

	h.logger.Debug("Suggestions fetched",
		zap.String("prefix", prefix),
		zap.Int("count", len(suggestions)),
	)

	c.JSON(http.StatusOK, SuggestResponse{
		Query:       prefix,
		Suggestions: suggestions,
	})
}

// Reindex handles POST /api/v1/search/reindex
// @Summary Reindex documents
// @Description Reindex all vendors and services from database to Elasticsearch (admin only)
// @Tags Search
// @Accept json
// @Produce json
// @Param request body ReindexRequest true "Reindex request"
// @Success 200 {object} ReindexResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/search/reindex [post]
func (h *Handler) Reindex(c *gin.Context) {
	// TODO: Add admin authentication middleware check
	// For now, this is open but should be restricted in production

	var req ReindexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Default to all types if not specified
	if len(req.Types) == 0 {
		req.Types = []string{"vendor", "service"}
	}

	h.logger.Info("Starting reindex", zap.Strings("types", req.Types))

	result := ReindexResponse{
		Success: true,
		Results: make(map[string]ReindexResult),
	}

	// Reindex based on requested types
	for _, docType := range req.Types {
		switch docType {
		case "vendor", "vendors":
			if err := h.service.ReindexVendors(c.Request.Context()); err != nil {
				h.logger.Error("Vendor reindex failed", zap.Error(err))
				result.Success = false
				result.Results["vendor"] = ReindexResult{
					Success: false,
					Error:   err.Error(),
				}
			} else {
				h.logger.Info("Vendor reindex completed")
				result.Results["vendor"] = ReindexResult{
					Success: true,
					Message: "Vendors reindexed successfully",
				}
			}

		case "service", "services":
			if err := h.service.ReindexServices(c.Request.Context()); err != nil {
				h.logger.Error("Service reindex failed", zap.Error(err))
				result.Success = false
				result.Results["service"] = ReindexResult{
					Success: false,
					Error:   err.Error(),
				}
			} else {
				h.logger.Info("Service reindex completed")
				result.Results["service"] = ReindexResult{
					Success: true,
					Message: "Services reindexed successfully",
				}
			}

		default:
			h.logger.Warn("Unknown reindex type", zap.String("type", docType))
			result.Results[docType] = ReindexResult{
				Success: false,
				Error:   "Unknown document type: " + docType,
			}
		}
	}

	statusCode := http.StatusOK
	if !result.Success {
		statusCode = http.StatusInternalServerError
	}

	c.JSON(statusCode, result)
}

// =============================================================================
// VALIDATION
// =============================================================================

func (h *Handler) validateSearchRequest(req *search.SearchRequest) error {
	// Page size validation
	if req.PageSize < 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// Page validation
	if req.Page < 1 {
		req.Page = 1
	}

	// Radius validation
	if req.RadiusKM < 0 {
		req.RadiusKM = 0
	}
	if req.RadiusKM > 1000 {
		req.RadiusKM = 1000 // Max 1000km radius
	}

	// Location validation
	if req.Location != nil {
		if req.Location.Lat < -90 || req.Location.Lat > 90 {
			return &ValidationError{Field: "location.lat", Message: "Latitude must be between -90 and 90"}
		}
		if req.Location.Lon < -180 || req.Location.Lon > 180 {
			return &ValidationError{Field: "location.lon", Message: "Longitude must be between -180 and 180"}
		}
	}

	return nil
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// SuggestResponse for autocomplete suggestions
type SuggestResponse struct {
	Query       string   `json:"query"`
	Suggestions []string `json:"suggestions"`
}

// ReindexRequest for reindexing documents
type ReindexRequest struct {
	Types []string `json:"types" example:"vendor,service"` // Types to reindex: 'vendor', 'service'
}

// ReindexResponse for reindex operation results
type ReindexResponse struct {
	Success bool                      `json:"success"`
	Results map[string]ReindexResult  `json:"results"`
}

// ReindexResult for individual reindex operation
type ReindexResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ErrorResponse for error responses
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ValidationError for validation errors
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
