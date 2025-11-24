package domain

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobStatusActive    JobStatus = "ACTIVE"
	JobStatusPaused    JobStatus = "PAUSED"
	JobStatusCompleted JobStatus = "COMPLETED"
)

type ExecutionStatus string

const (
	ExecStatusPending  ExecutionStatus = "PENDING"
	ExecStatusSuccess  ExecutionStatus = "SUCCESS"
	ExecStatusFailed   ExecutionStatus = "FAILED"
	ExecStatusRetrying ExecutionStatus = "RETRYING"
)

type Job struct {
	ID             uuid.UUID         `json:"id" db:"id"`
	TenantID       uuid.UUID         `json:"tenant_id" db:"tenant_id"`
	Title          string            `json:"title" db:"title"`
	CronExpression string            `json:"cron_expression,omitempty" db:"cron_expression"`
	IsRecurring    bool              `json:"is_recurring" db:"is_recurring"`
	EndpointURL    string            `json:"endpoint_url" db:"endpoint_url"`
	HTTPMethod     string            `json:"http_method" db:"http_method"`
	Headers        map[string]string `json:"headers" db:"headers"`
	Payload        map[string]any    `json:"payload" db:"payload"`
	Status         JobStatus         `json:"status" db:"status"`
	NextRunAt      time.Time         `json:"next_run_at" db:"next_run_at"`
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" db:"updated_at"`
}

type Execution struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	JobID        uuid.UUID       `json:"job_id" db:"job_id"`
	TenantID     uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	Status       ExecutionStatus `json:"status" db:"status"`
	StartedAt    time.Time       `json:"started_at" db:"started_at"`
	FinishedAt   *time.Time      `json:"finished_at" db:"finished_at"`
	ResponseCode int             `json:"response_code" db:"response_status"`
	RetryCount   int             `json:"retry_count" db:"retry_count"`
}
