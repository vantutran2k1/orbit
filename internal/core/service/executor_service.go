package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/vantutran2k1/orbit/internal/adapter/storage/redis"
	"github.com/vantutran2k1/orbit/internal/core/domain"
	"github.com/vantutran2k1/orbit/internal/core/ports"
)

const (
	MaxBodySize = 10 * 1024
)

type ExecutorService struct {
	httpClient *http.Client
	cronParser cron.Parser
	jobRepo    ports.JobRepository
	wfRepo     ports.WorkflowRepository
}

func NewExecutorService(jobRepo ports.JobRepository, wfRepo ports.WorkflowRepository) *ExecutorService {
	return &ExecutorService{
		jobRepo: jobRepo,
		wfRepo:  wfRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cronParser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

func (e *ExecutorService) ExecuteJob(ctx context.Context, job domain.Job) {
	var runID uuid.UUID

	if job.WorkflowID != nil {
		if job.NextWorkflowRunID != nil {
			runID = *job.NextWorkflowRunID
		} else {
			newID, err := e.wfRepo.CreateWorkflowRun(ctx, *job.WorkflowID)
			if err != nil {
				log.Printf("failed to create workflow run: %v", err)
				return
			}
			runID = newID
		}
	}

	execution := &domain.Execution{
		ID:            uuid.New(),
		JobID:         job.ID,
		TenantID:      job.TenantID,
		StartedAt:     time.Now(),
		RetryCount:    0,
		WorkflowRunID: nil,
	}

	if runID != uuid.Nil {
		execution.WorkflowRunID = &runID
	}

	e.performRequest(ctx, job, execution)

	finishedAt := time.Now()
	execution.FinishedAt = &finishedAt
	if err := e.jobRepo.SaveExecution(ctx, execution); err != nil {
		fmt.Printf("failed to save execution log for job %s: %v\n", job.ID, err)
	}

	e.handleRescheduling(ctx, job, execution, runID)
}

func (e *ExecutorService) performRequest(ctx context.Context, job domain.Job, exec *domain.Execution) {
	if strings.HasPrefix(job.EndpointURL, "tunnel:") {
		e.performTunnelRequest(ctx, job, exec)
		return
	}

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

func (e *ExecutorService) performTunnelRequest(ctx context.Context, job domain.Job, exec *domain.Execution) {
	msg := map[string]any{
		"job_id":  job.ID.String(),
		"payload": job.Payload,
	}
	data, _ := json.Marshal(msg)

	channel := "orbit:tunnel:" + job.TenantID.String()

	if err := redis.RDB.Publish(ctx, channel, data).Err(); err != nil {
		exec.Status = domain.ExecStatusFailed
		exec.ErrorMessage = "redis publish error: " + err.Error()
		return
	}

	// TODO: wait for an ack from the agent
	exec.Status = domain.ExecStatusSuccess
	exec.ResponseBody = "forwarded via redis pub/sub"
	exec.ResponseCode = 200
}

func (e *ExecutorService) handleRescheduling(ctx context.Context, job domain.Job, exec *domain.Execution, currentRunID uuid.UUID) {
	var nextRun time.Time
	var failures int
	var nextRunID *uuid.UUID

	if exec.Status == domain.ExecStatusSuccess {
		failures = 0

		if currentRunID != uuid.Nil {
			e.triggerDownstream(ctx, job, currentRunID)
		}

		if job.IsRecurring {
			schedule, _ := e.cronParser.Parse(job.CronExpression)
			nextRun = schedule.Next(time.Now())
			nextRunID = nil
		} else {
			nextRun = time.Time{}
			nextRunID = nil
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

			if currentRunID != uuid.Nil {
				nextRunID = &currentRunID
			}

			fmt.Printf("job %s failed, retrying in %v (attempt %d/%d)\n", job.Title, backoff, failures, maxRetries)

		} else {
			fmt.Printf("job %s failed max times, abandoning this run\n", job.Title)
			failures = 0

			if job.IsRecurring {
				schedule, _ := e.cronParser.Parse(job.CronExpression)
				nextRun = schedule.Next(time.Now())

				nextRunID = nil
			} else {
				nextRun = time.Time{}
				nextRunID = nil
			}
		}
	}

	if err := e.jobRepo.UpdateJobStatusAfterRun(ctx, job.ID, nextRun, failures, nextRunID); err != nil {
		fmt.Printf("CRITICAL: Failed to update job status %s: %v\n", job.ID, err)
	}
}

func (e *ExecutorService) triggerDownstream(ctx context.Context, upstreamJob domain.Job, runID uuid.UUID) {
	if runID == uuid.Nil {
		log.Println("empty run id")
	}

	children, err := e.wfRepo.GetDownstreamJobs(ctx, upstreamJob.ID)
	if err != nil {
		log.Printf("failed to fetch downstream jobs for parent %s: %v", upstreamJob.ID, err)
		return
	}

	if len(children) == 0 {
		return
	}

	log.Printf("triggering %d downstream jobs for %s", len(children), upstreamJob.Title)

	for _, child := range children {
		err := e.wfRepo.ScheduleDownstream(ctx, child.ID, runID)
		if err != nil {
			log.Printf("failed to trigger child job '%s' (%s): %v", child.Title, child.ID, err)
		} else {
			log.Printf("scheduled child '%s'", child.Title)
		}
	}
}
