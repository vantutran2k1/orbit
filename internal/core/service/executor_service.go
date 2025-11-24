package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/vantutran2k1/orbit/internal/core/domain"
	"github.com/vantutran2k1/orbit/internal/core/ports"
)

const (
	MaxBodySize = 10 * 1024
)

type ExecutorService struct {
	httpClient *http.Client
	repo       ports.JobRepository
	cronParser cron.Parser
}

func NewExecutorService(repo ports.JobRepository) *ExecutorService {
	return &ExecutorService{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cronParser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

func (e *ExecutorService) ExecuteJob(ctx context.Context, job domain.Job) {
	start := time.Now()
	execution := &domain.Execution{
		ID:         uuid.New(),
		JobID:      job.ID,
		TenantID:   job.TenantID,
		StartedAt:  start,
		RetryCount: 0,
	}

	e.performRequest(ctx, job, execution)

	finishedAt := time.Now()
	execution.FinishedAt = &finishedAt

	if err := e.repo.SaveExecution(ctx, execution); err != nil {
		fmt.Printf("failed to save execution log for job %s: %v\n", job.ID, err)
	}

	e.handleRescheduling(ctx, job, execution)
}

func (e *ExecutorService) performRequest(ctx context.Context, job domain.Job, exec *domain.Execution) {
	var bodyReader io.Reader
	if job.Payload != nil {
		jsonBytes, _ := json.Marshal(job.Payload)
		bodyReader = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, job.HTTPMethod, job.EndpointURL, bodyReader)
	if err != nil {
		exec.Status = domain.ExecStatusFailed
		exec.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		return
	}

	for k, v := range job.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("User-Agent", "Orbit-Scheduler/1.0")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		exec.Status = domain.ExecStatusFailed
		exec.ErrorMessage = fmt.Sprintf("network error: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, MaxBodySize))

	exec.ResponseCode = resp.StatusCode
	exec.ResponseBody = string(respBody)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		exec.Status = domain.ExecStatusSuccess
	} else {
		exec.Status = domain.ExecStatusFailed
		exec.ErrorMessage = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
}

func (e *ExecutorService) handleRescheduling(ctx context.Context, job domain.Job, exec *domain.Execution) {
	var nextRun time.Time
	var failures int

	if exec.Status == domain.ExecStatusSuccess {
		failures = 0

		if job.IsRecurring {
			schedule, _ := e.cronParser.Parse(job.CronExpression)
			nextRun = schedule.Next(time.Now())
		} else {
			e.repo.UpdateJobSchedule(ctx, job.ID, time.Time{}, domain.JobStatusCompleted)
			return
		}
	} else {
		failures = job.FailureCount + 1

		maxRetries := job.MaxRetries
		if maxRetries == 0 {
			maxRetries = 3
		}

		if failures <= maxRetries {
			backoff := CalculateBackoff(failures)
			nextRun = time.Now().Add(backoff)

			fmt.Printf("job %s failed, retrying in %v (attempt %d/%d)\n", job.Title, backoff, failures, maxRetries)
		} else {
			fmt.Printf("job %s failed max times, abandoning this run\n", job.Title)
			failures = 0

			if job.IsRecurring {
				schedule, _ := e.cronParser.Parse(job.CronExpression)
				nextRun = schedule.Next(time.Now())
			} else {
				e.repo.UpdateJobSchedule(ctx, job.ID, time.Time{}, domain.JobStatusCompleted)
				return
			}
		}
	}

	e.repo.UpdateJobStatusAfterRun(ctx, job.ID, nextRun, failures)
}
