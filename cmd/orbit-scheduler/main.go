package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantutran2k1/orbit/internal/adapter/storage/postgres"
	"github.com/vantutran2k1/orbit/internal/adapter/storage/redis"
	"github.com/vantutran2k1/orbit/internal/core/service"
	"github.com/vantutran2k1/orbit/internal/platform/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	dbPool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		log.Fatal("unable to connect to database:", err)
	}
	defer dbPool.Close()

	redis.Init()

	jobRepo := postgres.NewJobRepository(dbPool)
	executor := service.NewExecutorService(jobRepo)

	wp := worker.NewPool(50)
	wp.Start(ctx)

	log.Println("orbit scheduler system started, waiting for jobs...")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-ticker.C:
				jobs, err := jobRepo.ListDueJobs(ctx)
				if err != nil {
					log.Printf("error fetching jobs: %v", err)
					continue
				}

				if len(jobs) > 0 {
					log.Printf("found %d jobs due for execution", len(jobs))
				}

				for _, job := range jobs {
					currentJob := job
					wp.Submit(func() {
						executor.ExecuteJob(ctx, currentJob)
					})
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	<-stop
	log.Println("shutting down gracefully...")

	cancel()
	wp.Stop()

	log.Println("orbit scheduler stopped")
}
