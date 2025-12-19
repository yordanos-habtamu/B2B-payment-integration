package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yordanos-habtamu/b2b-payments/internal/service"
)

type PaymentWorker struct {
	redisClient   *redis.Client
	paymentService service.PaymentService
	queueName     string
	maxRetries    int
	retryDelay    time.Duration
	batchSize     int
	pollInterval  time.Duration
}

type PaymentJob struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	TenantID string                 `json:"tenant_id"`
	Data     map[string]interface{} `json:"data"`
	Retries  int                    `json:"retries"`
	CreatedAt time.Time             `json:"created_at"`
}

type JobResult struct {
	JobID    string `json:"job_id"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
	Duration string `json:"duration"`
}

func NewPaymentWorker(redisClient *redis.Client, paymentService service.PaymentService) *PaymentWorker {
	return &PaymentWorker{
		redisClient:   redisClient,
		paymentService: paymentService,
		queueName:     "payment_jobs",
		maxRetries:    3,
		retryDelay:    5 * time.Second,
		batchSize:     10,
		pollInterval:  1 * time.Second,
	}
}

// Start begins the worker to process payment jobs
func (w *PaymentWorker) Start(ctx context.Context) error {
	log.Printf("Starting payment worker, queue: %s", w.queueName)

	for {
		select {
		case <-ctx.Done():
			log.Println("Payment worker shutting down...")
			return ctx.Err()
		default:
			if err := w.processJobs(ctx); err != nil {
				log.Printf("Error processing jobs: %v", err)
				time.Sleep(w.pollInterval)
			}
		}
	}
}

// processJobs processes a batch of jobs from the queue
func (w *PaymentWorker) processJobs(ctx context.Context) error {
	// Use Redis BLPOP for blocking queue operation
	result, err := w.redisClient.BLPop(ctx, w.pollInterval, w.queueName).Result()
	if err != nil {
		if err == redis.Nil {
			// No jobs available, continue
			return nil
		}
		return fmt.Errorf("failed to pop job from queue: %w", err)
	}

	if len(result) < 2 {
		return fmt.Errorf("invalid redis result format")
	}

	var job PaymentJob
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		log.Printf("Failed to unmarshal job: %v", err)
		return nil
	}

	// Process the job
	if err := w.processJob(ctx, &job); err != nil {
		log.Printf("Failed to process job %s: %v", job.ID, err)
		return err
	}

	return nil
}

// processJob processes a single job
func (w *PaymentWorker) processJob(ctx context.Context, job *PaymentJob) error {
	startTime := time.Now()
	log.Printf("Processing job %s (type: %s, tenant: %s, retries: %d)", 
		job.ID, job.Type, job.TenantID, job.Retries)

	var err error
	var result JobResult

	switch job.Type {
	case "process_payment":
		err = w.processPaymentJob(ctx, job)
	case "cancel_payment":
		err = w.cancelPaymentJob(ctx, job)
	case "retry_failed_payment":
		err = w.retryFailedPaymentJob(ctx, job)
	case "payment_notification":
		err = w.sendPaymentNotificationJob(ctx, job)
	default:
		err = fmt.Errorf("unknown job type: %s", job.Type)
	}

	// Prepare result
	duration := time.Since(startTime)
	result = JobResult{
		JobID:    job.ID,
		Success:  err == nil,
		Duration: duration.String(),
	}

	if err != nil {
		result.Error = err.Error()
		log.Printf("Job %s failed: %v", job.ID, err)
		
		// Retry logic
		if job.Retries < w.maxRetries {
			return w.retryJob(ctx, job)
		} else {
			log.Printf("Job %s exceeded max retries, giving up", job.ID)
		}
	} else {
		log.Printf("Job %s completed successfully in %s", job.ID, duration.String())
	}

	// Publish result (optional, for monitoring)
	w.publishJobResult(ctx, &result)

	return nil
}

// processPaymentJob processes a payment
func (w *PaymentWorker) processPaymentJob(ctx context.Context, job *PaymentJob) error {
	paymentID, ok := job.Data["payment_id"].(string)
	if !ok {
		return fmt.Errorf("payment_id not found in job data")
	}

	return w.paymentService.ProcessPayment(ctx, job.TenantID, paymentID)
}

// cancelPaymentJob cancels a payment
func (w *PaymentWorker) cancelPaymentJob(ctx context.Context, job *PaymentJob) error {
	paymentID, ok := job.Data["payment_id"].(string)
	if !ok {
		return fmt.Errorf("payment_id not found in job data")
	}

	return w.paymentService.CancelPayment(ctx, job.TenantID, paymentID)
}

// retryFailedPaymentJob retries a failed payment
func (w *PaymentWorker) retryFailedPaymentJob(ctx context.Context, job *PaymentJob) error {
	paymentID, ok := job.Data["payment_id"].(string)
	if !ok {
		return fmt.Errorf("payment_id not found in job data")
	}

	// In a real implementation, you might want to reset the payment status first
	return w.paymentService.ProcessPayment(ctx, job.TenantID, paymentID)
}

// sendPaymentNotificationJob sends payment notifications
func (w *PaymentWorker) sendPaymentNotificationJob(ctx context.Context, job *PaymentJob) error {
	paymentID, ok := job.Data["payment_id"].(string)
	if !ok {
		return fmt.Errorf("payment_id not found in job data")
	}

	notificationType, ok := job.Data["notification_type"].(string)
	if !ok {
		return fmt.Errorf("notification_type not found in job data")
	}

	// Get payment details
	payment, err := w.paymentService.GetPayment(ctx, job.TenantID, paymentID)
	if err != nil {
		return fmt.Errorf("failed to get payment for notification: %w", err)
	}

	// Send notification (placeholder implementation)
	log.Printf("Sending %s notification for payment %s to tenant %s", 
		notificationType, paymentID, job.TenantID)
	
	// In a real implementation, this would integrate with email/SMS services
	// webhook calls, or other notification systems

	return nil
}

// retryJob retries a failed job
func (w *PaymentWorker) retryJob(ctx context.Context, job *PaymentJob) error {
	job.Retries++
	job.CreatedAt = time.Now()

	// Add delay before retry
	time.Sleep(w.retryDelay)

	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal retry job: %w", err)
	}

	// Push back to queue with delay using Redis sorted set (for delayed processing)
	delayedQueue := fmt.Sprintf("%s_delayed", w.queueName)
	score := float64(time.Now().Add(w.retryDelay).Unix())
	
	if err := w.redisClient.ZAdd(ctx, delayedQueue, redis.Z{
		Score:  score,
		Member: jobJSON,
	}).Err(); err != nil {
		return fmt.Errorf("failed to queue retry job: %w", err)
	}

	log.Printf("Job %s queued for retry %d/%d", job.ID, job.Retries, w.maxRetries)
	return nil
}

// publishJobResult publishes job processing results
func (w *PaymentWorker) publishJobResult(ctx context.Context, result *JobResult) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to marshal job result: %v", err)
		return
	}

	// Publish to Redis channel for monitoring
	channel := "payment_job_results"
	if err := w.redisClient.Publish(ctx, channel, resultJSON).Err(); err != nil {
		log.Printf("Failed to publish job result: %v", err)
	}
}

// EnqueueJob adds a new job to the queue
func (w *PaymentWorker) EnqueueJob(ctx context.Context, job *PaymentJob) error {
	job.ID = generateJobID()
	job.CreatedAt = time.Now()
	job.Retries = 0

	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	if err := w.redisClient.LPush(ctx, w.queueName, jobJSON).Err(); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	log.Printf("Enqueued job %s (type: %s, tenant: %s)", job.ID, job.Type, job.TenantID)
	return nil
}

// ProcessDelayedJobs processes jobs that are ready for retry
func (w *PaymentWorker) ProcessDelayedJobs(ctx context.Context) error {
	delayedQueue := fmt.Sprintf("%s_delayed", w.queueName)
	now := float64(time.Now().Unix())

	// Get all jobs that are ready to be processed
	jobs, err := w.redisClient.ZRangeByScore(ctx, delayedQueue, &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%f", now),
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to get delayed jobs: %w", err)
	}

	for _, jobJSON := range jobs {
		// Move job back to main queue
		if err := w.redisClient.LPush(ctx, w.queueName, jobJSON).Err(); err != nil {
			log.Printf("Failed to move delayed job to main queue: %v", err)
			continue
		}

		// Remove from delayed queue
		if err := w.redisClient.ZRem(ctx, delayedQueue, jobJSON).Err(); err != nil {
			log.Printf("Failed to remove job from delayed queue: %v", err)
		}
	}

	if len(jobs) > 0 {
		log.Printf("Moved %d delayed jobs back to main queue", len(jobs))
	}

	return nil
}

// GetQueueStats returns statistics about the job queue
func (w *PaymentWorker) GetQueueStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Main queue length
	mainQueueLen, err := w.redisClient.LLen(ctx, w.queueName).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get main queue length: %w", err)
	}
	stats["main_queue_length"] = mainQueueLen

	// Delayed queue length
	delayedQueue := fmt.Sprintf("%s_delayed", w.queueName)
	delayedQueueLen, err := w.redisClient.ZCard(ctx, delayedQueue).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get delayed queue length: %w", err)
	}
	stats["delayed_queue_length"] = delayedQueueLen

	return stats, nil
}

func generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}
