package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantutran2k1/orbit/internal/core/domain"
	"github.com/vantutran2k1/orbit/internal/core/ports"
)

type JobRepository struct {
	db *pgxpool.Pool
}

func NewJobRepository(db *pgxpool.Pool) ports.JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(ctx context.Context, job *domain.Job) error {
	query := `
		INSERT INTO jobs (
			tenant_id, title, cron_expression, is_recurring, 
			endpoint_url, http_method, headers, payload, 
			status, next_run_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		job.TenantID, job.Title, job.CronExpression, job.IsRecurring,
		job.EndpointURL, job.HTTPMethod, job.Headers, job.Payload,
		job.Status, job.NextRunAt, job.CreatedAt, job.UpdatedAt,
	).Scan(&job.ID)
	return err
}

func (r *JobRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	query := `
		SELECT id, tenant_id, title, cron_expression, is_recurring,
		       endpoint_url, http_method, headers, payload,
		       status, next_run_at, created_at, updated_at
		FROM jobs
		WHERE id = $1
	`

	var j domain.Job
	err := r.db.QueryRow(ctx, query, id).Scan(
		&j.ID, &j.TenantID, &j.Title, &j.CronExpression, &j.IsRecurring,
		&j.EndpointURL, &j.HTTPMethod, &j.Headers, &j.Payload,
		&j.Status, &j.NextRunAt, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &j, nil
}

func (r *JobRepository) ListDueJobs(ctx context.Context) ([]domain.Job, error) {
	query := `
		UPDATE jobs
		SET next_run_at = NOW() + INTERVAL '1 minute' -- Lease time
		WHERE id IN (
			SELECT id FROM jobs
			WHERE status = 'ACTIVE' AND next_run_at <= NOW()
			ORDER BY next_run_at ASC
			LIMIT 50
			FOR UPDATE SKIP LOCKED
		)
		RETURNING 
			id, tenant_id, title,
			COALESCE(cron_expression, ''),
			is_recurring, endpoint_url, 
			http_method, headers, payload, status, next_run_at, failure_count, max_retries
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var j domain.Job
		if err := rows.Scan(
			&j.ID, &j.TenantID, &j.Title, &j.CronExpression, &j.IsRecurring, &j.EndpointURL,
			&j.HTTPMethod, &j.Headers, &j.Payload, &j.Status, &j.NextRunAt, &j.FailureCount, &j.MaxRetries,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}

	return jobs, nil
}

func (r *JobRepository) UpdateNextRun(ctx context.Context, id uuid.UUID, nextRun time.Time) error {
	query := `
		UPDATE jobs
		SET next_run_at = $1,
		    updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, nextRun, id)
	return err
}

func (r *JobRepository) SaveExecution(ctx context.Context, exec *domain.Execution) error {
	query := `
		INSERT INTO executions (
			id, job_id, tenant_id, status, started_at, finished_at, 
			response_status, response_body, error_message, retry_count, workflow_run_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	if exec.ID == uuid.Nil {
		exec.ID = uuid.New()
	}

	_, err := r.db.Exec(ctx, query,
		exec.ID, exec.JobID, exec.TenantID, exec.Status, exec.StartedAt, exec.FinishedAt,
		exec.ResponseCode, exec.ResponseBody, exec.ErrorMessage, exec.RetryCount, exec.WorkflowRunID,
	)
	return err
}

func (r *JobRepository) UpdateJobSchedule(ctx context.Context, jobID uuid.UUID, nextRun time.Time, status domain.JobStatus) error {
	query := `
		UPDATE jobs 
		SET next_run_at = $1, 
		    status = $2,
		    updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, nextRun, status, jobID)
	return err
}

func (r *JobRepository) UpdateJobStatusAfterRun(ctx context.Context, jobID uuid.UUID, nextRun time.Time, failures int, nextRunID *uuid.UUID) error {
	query := `
        UPDATE jobs 
        SET next_run_at = $1, 
            failure_count = $2,
			next_workflow_run_id = $3,
            updated_at = NOW()
        WHERE id = $4
    `
	_, err := r.db.Exec(ctx, query, nextRun, failures, nextRunID, jobID)
	return err
}
