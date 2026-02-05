// =============================================================================
// WORKER SERVICE
// Background job processing with job queue, scheduling, and retries
// =============================================================================

package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

// =============================================================================
// TYPES
// =============================================================================

// Job represents a background job
type Job struct {
	ID          uuid.UUID              `json:"id"`
	Type        JobType                `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Status      JobStatus              `json:"status"`
	Priority    int                    `json:"priority"` // Higher = more urgent
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"max_attempts"`
	LastError   string                 `json:"last_error,omitempty"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

type JobType string
const (
	// Notification jobs
	JobSendEmail          JobType = "send_email"
	JobSendSMS            JobType = "send_sms"
	JobSendPush           JobType = "send_push"
	JobBulkNotification   JobType = "bulk_notification"
	
	// Payment jobs
	JobProcessPayout      JobType = "process_payout"
	JobReleaseEscrow      JobType = "release_escrow"
	JobRefundPayment      JobType = "refund_payment"
	JobReconcilePayments  JobType = "reconcile_payments"
	
	// Analytics jobs
	JobUpdateRecommendations JobType = "update_recommendations"
	JobCalculateAnalytics    JobType = "calculate_analytics"
	JobGenerateReports       JobType = "generate_reports"
	
	// Maintenance jobs
	JobCleanupSessions    JobType = "cleanup_sessions"
	JobCleanupExpired     JobType = "cleanup_expired"
	JobArchiveOldData     JobType = "archive_old_data"
	JobOptimizeDatabase   JobType = "optimize_database"
	
	// Business logic jobs
	JobDetectLifeEvents   JobType = "detect_life_events"
	JobMatchPartners      JobType = "match_partners"
	JobProcessReferrals   JobType = "process_referrals"
	JobUpdateVendorRanks  JobType = "update_vendor_ranks"
)

type JobStatus string
const (
	JobPending    JobStatus = "pending"
	JobProcessing JobStatus = "processing"
	JobCompleted  JobStatus = "completed"
	JobFailed     JobStatus = "failed"
	JobRetrying   JobStatus = "retrying"
)

// JobHandler processes a specific job type
type JobHandler func(ctx context.Context, job *Job) error

// =============================================================================
// SERVICE
// =============================================================================

// Config for worker service
type Config struct {
	NumWorkers      int
	MaxRetries      int
	RetryBackoff    time.Duration
	PollInterval    time.Duration
	JobTimeout      time.Duration
	ShutdownTimeout time.Duration
}

// DefaultConfig returns default worker configuration
func DefaultConfig() *Config {
	return &Config{
		NumWorkers:      5,
		MaxRetries:      3,
		RetryBackoff:    time.Minute,
		PollInterval:    time.Second,
		JobTimeout:      5 * time.Minute,
		ShutdownTimeout: 30 * time.Second,
	}
}

// Service handles background jobs
type Service struct {
	db       *pgxpool.Pool
	cache    *redis.Client
	config   *Config
	handlers map[JobType]JobHandler
	cron     *cron.Cron
	
	wg       sync.WaitGroup
	quit     chan struct{}
	mu       sync.RWMutex
}

// NewService creates a new worker service
func NewService(db *pgxpool.Pool, cache *redis.Client, config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}
	
	return &Service{
		db:       db,
		cache:    cache,
		config:   config,
		handlers: make(map[JobType]JobHandler),
		cron:     cron.New(cron.WithSeconds()),
		quit:     make(chan struct{}),
	}
}

// RegisterHandler registers a handler for a job type
func (s *Service) RegisterHandler(jobType JobType, handler JobHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[jobType] = handler
}

// =============================================================================
// JOB QUEUE OPERATIONS
// =============================================================================

// Enqueue adds a job to the queue
func (s *Service) Enqueue(ctx context.Context, jobType JobType, payload map[string]interface{}) (*Job, error) {
	return s.EnqueueWithOptions(ctx, jobType, payload, 0, time.Now())
}

// EnqueueWithOptions adds a job with custom options
func (s *Service) EnqueueWithOptions(ctx context.Context, jobType JobType, payload map[string]interface{}, priority int, scheduledAt time.Time) (*Job, error) {
	job := &Job{
		ID:          uuid.New(),
		Type:        jobType,
		Payload:     payload,
		Status:      JobPending,
		Priority:    priority,
		Attempts:    0,
		MaxAttempts: s.config.MaxRetries,
		ScheduledAt: scheduledAt,
		CreatedAt:   time.Now(),
	}
	
	payloadJSON, _ := json.Marshal(payload)
	
	query := `
		INSERT INTO jobs (id, type, payload, status, priority, attempts, max_attempts, scheduled_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := s.db.Exec(ctx, query,
		job.ID, job.Type, payloadJSON, job.Status, job.Priority,
		job.Attempts, job.MaxAttempts, job.ScheduledAt, job.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Also push to Redis for faster polling
	s.cache.LPush(ctx, "jobs:queue", job.ID.String())
	
	return job, nil
}

// EnqueueBatch adds multiple jobs at once
func (s *Service) EnqueueBatch(ctx context.Context, jobs []*Job) error {
	batch := &pgxpool.Batch{}
	
	for _, job := range jobs {
		payloadJSON, _ := json.Marshal(job.Payload)
		batch.Queue(`
			INSERT INTO jobs (id, type, payload, status, priority, attempts, max_attempts, scheduled_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, job.ID, job.Type, payloadJSON, job.Status, job.Priority,
			job.Attempts, job.MaxAttempts, job.ScheduledAt, job.CreatedAt)
	}
	
	br := s.db.SendBatch(ctx, batch)
	defer br.Close()
	
	for range jobs {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	
	return nil
}

// =============================================================================
// WORKER LOOP
// =============================================================================

// Start begins processing jobs
func (s *Service) Start(ctx context.Context) error {
	log.Printf("Starting worker service with %d workers", s.config.NumWorkers)
	
	// Start cron scheduler
	s.cron.Start()
	
	// Start workers
	for i := 0; i < s.config.NumWorkers; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
	}
	
	// Start job scheduler (moves scheduled jobs to queue)
	s.wg.Add(1)
	go s.scheduler(ctx)
	
	return nil
}

// Stop gracefully stops the worker service
func (s *Service) Stop() {
	log.Println("Stopping worker service...")
	
	close(s.quit)
	s.cron.Stop()
	
	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("Worker service stopped gracefully")
	case <-time.After(s.config.ShutdownTimeout):
		log.Println("Worker service shutdown timed out")
	}
}

func (s *Service) worker(ctx context.Context, id int) {
	defer s.wg.Done()
	
	log.Printf("Worker %d started", id)
	
	for {
		select {
		case <-s.quit:
			log.Printf("Worker %d stopping", id)
			return
		default:
			// Try to get a job
			job, err := s.fetchJob(ctx)
			if err != nil || job == nil {
				// No job available, wait and try again
				time.Sleep(s.config.PollInterval)
				continue
			}
			
			// Process job
			s.processJob(ctx, job)
		}
	}
}

func (s *Service) fetchJob(ctx context.Context) (*Job, error) {
	// Try Redis first
	jobID, err := s.cache.RPop(ctx, "jobs:queue").Result()
	if err == nil {
		// Fetch from database
		return s.getJobByID(ctx, jobID)
	}
	
	// Fallback to database polling
	var job Job
	var payloadJSON []byte
	
	query := `
		UPDATE jobs 
		SET status = 'processing', started_at = NOW()
		WHERE id = (
			SELECT id FROM jobs 
			WHERE status = 'pending' 
			  AND scheduled_at <= NOW()
			ORDER BY priority DESC, scheduled_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, type, payload, status, priority, attempts, max_attempts, scheduled_at, created_at
	`
	
	err = s.db.QueryRow(ctx, query).Scan(
		&job.ID, &job.Type, &payloadJSON, &job.Status, &job.Priority,
		&job.Attempts, &job.MaxAttempts, &job.ScheduledAt, &job.CreatedAt,
	)
	
	if err != nil {
		return nil, nil // No jobs available
	}
	
	json.Unmarshal(payloadJSON, &job.Payload)
	
	return &job, nil
}

func (s *Service) getJobByID(ctx context.Context, id string) (*Job, error) {
	var job Job
	var payloadJSON []byte
	
	query := `
		UPDATE jobs 
		SET status = 'processing', started_at = NOW()
		WHERE id = $1 AND status = 'pending'
		RETURNING id, type, payload, status, priority, attempts, max_attempts, scheduled_at, created_at
	`
	
	err := s.db.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.Type, &payloadJSON, &job.Status, &job.Priority,
		&job.Attempts, &job.MaxAttempts, &job.ScheduledAt, &job.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(payloadJSON, &job.Payload)
	
	return &job, nil
}

func (s *Service) processJob(ctx context.Context, job *Job) {
	log.Printf("Processing job %s of type %s", job.ID, job.Type)
	
	// Get handler
	s.mu.RLock()
	handler, ok := s.handlers[job.Type]
	s.mu.RUnlock()
	
	if !ok {
		log.Printf("No handler registered for job type: %s", job.Type)
		s.failJob(ctx, job, "no handler registered")
		return
	}
	
	// Create context with timeout
	jobCtx, cancel := context.WithTimeout(ctx, s.config.JobTimeout)
	defer cancel()
	
	// Execute handler
	err := handler(jobCtx, job)
	
	if err != nil {
		log.Printf("Job %s failed: %v", job.ID, err)
		
		job.Attempts++
		if job.Attempts >= job.MaxAttempts {
			s.failJob(ctx, job, err.Error())
		} else {
			s.retryJob(ctx, job, err.Error())
		}
		return
	}
	
	// Mark as completed
	s.completeJob(ctx, job)
}

func (s *Service) completeJob(ctx context.Context, job *Job) {
	_, err := s.db.Exec(ctx, `
		UPDATE jobs SET status = 'completed', completed_at = NOW()
		WHERE id = $1
	`, job.ID)
	
	if err != nil {
		log.Printf("Failed to mark job %s as completed: %v", job.ID, err)
	}
}

func (s *Service) failJob(ctx context.Context, job *Job, errMsg string) {
	_, err := s.db.Exec(ctx, `
		UPDATE jobs SET status = 'failed', last_error = $2, completed_at = NOW()
		WHERE id = $1
	`, job.ID, errMsg)
	
	if err != nil {
		log.Printf("Failed to mark job %s as failed: %v", job.ID, err)
	}
}

func (s *Service) retryJob(ctx context.Context, job *Job, errMsg string) {
	nextAttempt := time.Now().Add(s.config.RetryBackoff * time.Duration(job.Attempts))
	
	_, err := s.db.Exec(ctx, `
		UPDATE jobs SET status = 'pending', last_error = $2, attempts = $3, scheduled_at = $4
		WHERE id = $1
	`, job.ID, errMsg, job.Attempts, nextAttempt)
	
	if err != nil {
		log.Printf("Failed to retry job %s: %v", job.ID, err)
	}
}

// =============================================================================
// SCHEDULER
// =============================================================================

func (s *Service) scheduler(ctx context.Context) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.quit:
			return
		case <-ticker.C:
			// Move scheduled jobs to queue
			s.moveScheduledJobs(ctx)
		}
	}
}

func (s *Service) moveScheduledJobs(ctx context.Context) {
	rows, err := s.db.Query(ctx, `
		SELECT id FROM jobs 
		WHERE status = 'pending' AND scheduled_at <= NOW()
		LIMIT 100
	`)
	if err != nil {
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var id uuid.UUID
		if rows.Scan(&id) == nil {
			s.cache.LPush(ctx, "jobs:queue", id.String())
		}
	}
}

// =============================================================================
// CRON JOBS
// =============================================================================

// ScheduleCron schedules a recurring job
func (s *Service) ScheduleCron(schedule string, jobType JobType, payload map[string]interface{}) error {
	_, err := s.cron.AddFunc(schedule, func() {
		s.Enqueue(context.Background(), jobType, payload)
	})
	return err
}

// RegisterDefaultCronJobs registers standard maintenance jobs
func (s *Service) RegisterDefaultCronJobs() {
	// Cleanup expired sessions every hour
	s.ScheduleCron("0 0 * * * *", JobCleanupSessions, nil)
	
	// Update recommendations every 6 hours
	s.ScheduleCron("0 0 */6 * * *", JobUpdateRecommendations, nil)
	
	// Calculate analytics daily at 2 AM
	s.ScheduleCron("0 0 2 * * *", JobCalculateAnalytics, nil)
	
	// Cleanup expired data weekly on Sunday at 3 AM
	s.ScheduleCron("0 0 3 * * 0", JobCleanupExpired, nil)
	
	// Archive old data monthly on the 1st at 4 AM
	s.ScheduleCron("0 0 4 1 * *", JobArchiveOldData, nil)
	
	// Reconcile payments daily at 1 AM
	s.ScheduleCron("0 0 1 * * *", JobReconcilePayments, nil)
	
	// Update vendor rankings weekly
	s.ScheduleCron("0 0 5 * * 1", JobUpdateVendorRanks, nil)
	
	// Detect life events every 4 hours
	s.ScheduleCron("0 0 */4 * * *", JobDetectLifeEvents, nil)
	
	// Process pending referrals every 30 minutes
	s.ScheduleCron("0 */30 * * * *", JobProcessReferrals, nil)
}

// =============================================================================
// DEFAULT HANDLERS
// =============================================================================

// RegisterDefaultHandlers registers standard job handlers
func (s *Service) RegisterDefaultHandlers() {
	// Cleanup sessions handler
	s.RegisterHandler(JobCleanupSessions, func(ctx context.Context, job *Job) error {
		_, err := s.db.Exec(ctx, "DELETE FROM sessions WHERE expires_at < NOW()")
		return err
	})
	
	// Cleanup expired data handler
	s.RegisterHandler(JobCleanupExpired, func(ctx context.Context, job *Job) error {
		// Delete expired verification tokens from Redis
		// Delete old notifications
		_, err := s.db.Exec(ctx, `
			DELETE FROM notifications 
			WHERE created_at < NOW() - INTERVAL '90 days' AND status = 'read'
		`)
		return err
	})
	
	// Database optimization handler
	s.RegisterHandler(JobOptimizeDatabase, func(ctx context.Context, job *Job) error {
		tables := []string{"users", "vendors", "bookings", "transactions", "notifications"}
		for _, table := range tables {
			s.db.Exec(ctx, fmt.Sprintf("VACUUM ANALYZE %s", table))
		}
		return nil
	})
}

// =============================================================================
// JOB MONITORING
// =============================================================================

// JobStats represents job statistics
type JobStats struct {
	Pending    int       `json:"pending"`
	Processing int       `json:"processing"`
	Completed  int       `json:"completed"`
	Failed     int       `json:"failed"`
	AvgDuration float64  `json:"avg_duration_seconds"`
}

// GetStats returns job statistics
func (s *Service) GetStats(ctx context.Context) (*JobStats, error) {
	var stats JobStats
	
	err := s.db.QueryRow(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'processing') as processing,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) FILTER (WHERE completed_at IS NOT NULL), 0) as avg_duration
		FROM jobs WHERE created_at > NOW() - INTERVAL '24 hours'
	`).Scan(&stats.Pending, &stats.Processing, &stats.Completed, &stats.Failed, &stats.AvgDuration)
	
	return &stats, err
}

// GetFailedJobs returns recent failed jobs
func (s *Service) GetFailedJobs(ctx context.Context, limit int) ([]*Job, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, type, payload, status, priority, attempts, max_attempts, 
		       last_error, scheduled_at, started_at, completed_at, created_at
		FROM jobs WHERE status = 'failed'
		ORDER BY completed_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var jobs []*Job
	for rows.Next() {
		var job Job
		var payloadJSON []byte
		
		err := rows.Scan(
			&job.ID, &job.Type, &payloadJSON, &job.Status, &job.Priority,
			&job.Attempts, &job.MaxAttempts, &job.LastError, &job.ScheduledAt,
			&job.StartedAt, &job.CompletedAt, &job.CreatedAt,
		)
		if err != nil {
			continue
		}
		
		json.Unmarshal(payloadJSON, &job.Payload)
		jobs = append(jobs, &job)
	}
	
	return jobs, nil
}

// RetryFailedJob retries a failed job
func (s *Service) RetryFailedJob(ctx context.Context, jobID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE jobs SET status = 'pending', attempts = 0, scheduled_at = NOW()
		WHERE id = $1 AND status = 'failed'
	`, jobID)

	if err == nil {
		s.cache.LPush(ctx, "jobs:queue", jobID.String())
	}

	return err
}

// QueryRow exposes database QueryRow for handler use
func (s *Service) QueryRow(ctx context.Context, query string, args ...interface{}) pgxRow {
	return s.db.QueryRow(ctx, query, args...)
}

// pgxRow interface for database row scanning
type pgxRow interface {
	Scan(dest ...interface{}) error
}
