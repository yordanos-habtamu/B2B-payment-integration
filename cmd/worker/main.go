package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yordanos-habtamu/b2b-payments/internal/config"
	"github.com/yordanos-habtamu/b2b-payments/internal/service"
	"github.com/yordanos-habtamu/b2b-payments/internal/worker"
)


func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Redis client
	redisClient, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	rdb := redis.NewClient(redisClient)

	// Test Redis connection
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize services
	paymentService := service.NewPaymentService()

	// Initialize worker
	paymentWorker := worker.NewPaymentWorker(rdb, paymentService)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start delayed job processor
	go func() {
		ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := paymentWorker.ProcessDelayedJobs(ctx); err != nil {
					log.Printf("Error processing delayed jobs: %v", err)
				}
			}
		}
	}()

	// Start main worker
	go func() {
		if err := paymentWorker.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("Worker stopped with error: %v", err)
		}
	}()

	// Start stats reporter
	go func() {
		ticker := time.NewTicker(60 * time.Second) // Report every minute
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats, err := paymentWorker.GetQueueStats(ctx)
				if err != nil {
					log.Printf("Error getting queue stats: %v", err)
				} else {
					log.Printf("Queue stats: %+v", stats)
				}
			}
		}
	}()

	log.Println("Payment worker started successfully")

	// Wait for shutdown signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down worker...")
	cancel()

	// Give some time for graceful shutdown
	time.Sleep(5 * time.Second)
	log.Println("Worker shutdown complete")
}