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

	e.handleRescheduling(ctx, job)
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

func (e *ExecutorService) handleRescheduling(ctx context.Context, job domain.Job) {
	var nextRun time.Time
	var newStatus domain.JobStatus

	if job.IsRecurring {
		schedule, err := e.cronParser.Parse(job.CronExpression)
		if err != nil {
			fmt.Printf("error parsing cron for job: %s: %v\n", job.ID, err)
			e.repo.UpdateJobSchedule(ctx, job.ID, time.Time{}, domain.JobStatusPaused)
			return
		}

		nextRun = schedule.Next(time.Now())
		newStatus = domain.JobStatusActive
	} else {
		nextRun = time.Time{}
		newStatus = domain.JobStatusCompleted
	}

	if err := e.repo.UpdateJobSchedule(ctx, job.ID, nextRun, newStatus); err != nil {
		fmt.Printf("failed to reschedule job %s: %v\n", job.ID, err)
	}
}
