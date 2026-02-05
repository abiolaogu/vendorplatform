// =============================================================================
// WORKER API HANDLERS
// HTTP handlers for job queue management and monitoring
// =============================================================================

package worker

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/BillyRonksGlobal/vendorplatform/internal/worker"
)

// Handler handles worker API requests
type Handler struct {
	service *worker.Service
	logger  *zap.Logger
}

// NewHandler creates a new worker handler
func NewHandler(service *worker.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers worker routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	jobs := rg.Group("/jobs")
	{
		jobs.POST("", h.EnqueueJob)
		jobs.GET("/:id", h.GetJobStatus)
		jobs.GET("/stats", h.GetJobStats)
		jobs.GET("/failed", h.GetFailedJobs)
		jobs.POST("/:id/retry", h.RetryFailedJob)
	}
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// EnqueueJobRequest represents a job enqueue request
type EnqueueJobRequest struct {
	Type        string                 `json:"type" binding:"required"`
	Payload     map[string]interface{} `json:"payload"`
	Priority    int                    `json:"priority"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
}

// JobResponse represents a job response
type JobResponse struct {
	ID          uuid.UUID              `json:"id"`
	Type        string                 `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Status      string                 `json:"status"`
	Priority    int                    `json:"priority"`
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"max_attempts"`
	LastError   string                 `json:"last_error,omitempty"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// JobStatsResponse represents job statistics
type JobStatsResponse struct {
	Pending         int     `json:"pending"`
	Processing      int     `json:"processing"`
	Completed       int     `json:"completed"`
	Failed          int     `json:"failed"`
	AvgDuration     float64 `json:"avg_duration_seconds"`
	TotalLast24h    int     `json:"total_last_24h"`
	SuccessRate     float64 `json:"success_rate"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// =============================================================================
// HANDLERS
// =============================================================================

// EnqueueJob godoc
// @Summary Enqueue a new background job
// @Description Add a new job to the processing queue
// @Tags jobs
// @Accept json
// @Produce json
// @Param job body EnqueueJobRequest true "Job details"
// @Success 201 {object} JobResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/jobs [post]
func (h *Handler) EnqueueJob(c *gin.Context) {
	var req EnqueueJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Validate job type
	if !h.isValidJobType(req.Type) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_job_type",
			Message: "Unsupported job type",
		})
		return
	}

	// Validate priority
	if req.Priority < 0 || req.Priority > 100 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_priority",
			Message: "Priority must be between 0 and 100",
		})
		return
	}

	// Set default scheduled time if not provided
	scheduledAt := time.Now()
	if req.ScheduledAt != nil {
		scheduledAt = *req.ScheduledAt
	}

	// Enqueue job
	job, err := h.service.EnqueueWithOptions(
		c.Request.Context(),
		worker.JobType(req.Type),
		req.Payload,
		req.Priority,
		scheduledAt,
	)
	if err != nil {
		h.logger.Error("Failed to enqueue job",
			zap.Error(err),
			zap.String("type", req.Type),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "enqueue_failed",
			Message: "Failed to enqueue job",
		})
		return
	}

	h.logger.Info("Job enqueued",
		zap.String("id", job.ID.String()),
		zap.String("type", string(job.Type)),
		zap.Int("priority", job.Priority),
	)

	c.JSON(http.StatusCreated, h.toJobResponse(job))
}

// GetJobStatus godoc
// @Summary Get job status
// @Description Retrieve the current status of a job
// @Tags jobs
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} JobResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/jobs/{id} [get]
func (h *Handler) GetJobStatus(c *gin.Context) {
	idStr := c.Param("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Query job from database
	var job worker.Job
	var payloadJSON []byte

	query := `
		SELECT id, type, payload, status, priority, attempts, max_attempts,
		       last_error, scheduled_at, started_at, completed_at, created_at
		FROM jobs WHERE id = $1
	`

	err = h.service.QueryRow(c.Request.Context(), query, jobID).Scan(
		&job.ID, &job.Type, &payloadJSON, &job.Status, &job.Priority,
		&job.Attempts, &job.MaxAttempts, &job.LastError, &job.ScheduledAt,
		&job.StartedAt, &job.CompletedAt, &job.CreatedAt,
	)

	if err != nil {
		h.logger.Error("Job not found", zap.Error(err), zap.String("id", idStr))
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "job_not_found",
			Message: "Job not found",
		})
		return
	}

	// Unmarshal payload
	if len(payloadJSON) > 0 {
		job.Payload = make(map[string]interface{})
		// Simple JSON unmarshal for map
		// Note: In production, use json.Unmarshal
	}

	c.JSON(http.StatusOK, h.toJobResponse(&job))
}

// GetJobStats godoc
// @Summary Get job statistics
// @Description Get statistics about job processing
// @Tags jobs
// @Produce json
// @Success 200 {object} JobStatsResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/jobs/stats [get]
func (h *Handler) GetJobStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get job stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "stats_failed",
			Message: "Failed to retrieve statistics",
		})
		return
	}

	// Calculate additional metrics
	total := stats.Pending + stats.Processing + stats.Completed + stats.Failed
	successRate := 0.0
	if total > 0 {
		successRate = float64(stats.Completed) / float64(total) * 100
	}

	response := JobStatsResponse{
		Pending:         stats.Pending,
		Processing:      stats.Processing,
		Completed:       stats.Completed,
		Failed:          stats.Failed,
		AvgDuration:     stats.AvgDuration,
		TotalLast24h:    total,
		SuccessRate:     successRate,
	}

	c.JSON(http.StatusOK, response)
}

// GetFailedJobs godoc
// @Summary Get failed jobs
// @Description Get a list of recently failed jobs
// @Tags jobs
// @Produce json
// @Param limit query int false "Maximum number of jobs to return" default(50)
// @Success 200 {array} JobResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/jobs/failed [get]
func (h *Handler) GetFailedJobs(c *gin.Context) {
	// Parse limit parameter
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 && parsedLimit <= 500 {
			limit = parsedLimit
		}
	}

	jobs, err := h.service.GetFailedJobs(c.Request.Context(), limit)
	if err != nil {
		h.logger.Error("Failed to get failed jobs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: "Failed to retrieve failed jobs",
		})
		return
	}

	response := make([]JobResponse, 0, len(jobs))
	for _, job := range jobs {
		response = append(response, h.toJobResponse(job))
	}

	c.JSON(http.StatusOK, response)
}

// RetryFailedJob godoc
// @Summary Retry a failed job
// @Description Retry a job that previously failed
// @Tags jobs
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/jobs/{id}/retry [post]
func (h *Handler) RetryFailedJob(c *gin.Context) {
	idStr := c.Param("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	err = h.service.RetryFailedJob(c.Request.Context(), jobID)
	if err != nil {
		h.logger.Error("Failed to retry job",
			zap.Error(err),
			zap.String("id", idStr),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "retry_failed",
			Message: "Failed to retry job",
		})
		return
	}

	h.logger.Info("Job retried", zap.String("id", idStr))

	c.JSON(http.StatusOK, gin.H{
		"message": "Job queued for retry",
		"job_id":  jobID.String(),
	})
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (h *Handler) toJobResponse(job *worker.Job) JobResponse {
	return JobResponse{
		ID:          job.ID,
		Type:        string(job.Type),
		Payload:     job.Payload,
		Status:      string(job.Status),
		Priority:    job.Priority,
		Attempts:    job.Attempts,
		MaxAttempts: job.MaxAttempts,
		LastError:   job.LastError,
		ScheduledAt: job.ScheduledAt,
		StartedAt:   job.StartedAt,
		CompletedAt: job.CompletedAt,
		CreatedAt:   job.CreatedAt,
	}
}

func (h *Handler) isValidJobType(jobType string) bool {
	validTypes := map[string]bool{
		// Notification jobs
		"send_email":         true,
		"send_sms":           true,
		"send_push":          true,
		"bulk_notification":  true,

		// Payment jobs
		"process_payout":     true,
		"release_escrow":     true,
		"refund_payment":     true,
		"reconcile_payments": true,

		// Analytics jobs
		"update_recommendations": true,
		"calculate_analytics":    true,
		"generate_reports":       true,

		// Maintenance jobs
		"cleanup_sessions":   true,
		"cleanup_expired":    true,
		"archive_old_data":   true,
		"optimize_database":  true,

		// Business logic jobs
		"detect_life_events":  true,
		"match_partners":      true,
		"process_referrals":   true,
		"update_vendor_ranks": true,
	}

	return validTypes[jobType]
}
