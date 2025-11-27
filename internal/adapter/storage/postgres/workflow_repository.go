package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantutran2k1/orbit/internal/core/domain"
	"github.com/vantutran2k1/orbit/internal/core/ports"
)

type WorkflowRepository struct {
	db *pgxpool.Pool
}

func NewWorkflowRepository(db *pgxpool.Pool) ports.WorkflowRepository {
	return &WorkflowRepository{db: db}
}

func (r *WorkflowRepository) Create(ctx context.Context, wf *domain.Workflow) error {
	query := `INSERT INTO workflows (id, tenant_id, name) VALUES ($1, $2, $3)`
	if wf.ID == uuid.Nil {
		wf.ID = uuid.New()
	}
	_, err := r.db.Exec(ctx, query, wf.ID, wf.TenantID, wf.Name)
	return err
}

func (r *WorkflowRepository) AddDependency(ctx context.Context, upstreamID, downstreamID, tenantID uuid.UUID) error {
	query := `INSERT INTO job_dependencies (upstream_job_id, downstream_job_id, tenant_id) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(ctx, query, upstreamID, downstreamID, tenantID)
	return err
}

func (r *WorkflowRepository) GetDownstreamJobs(ctx context.Context, jobID uuid.UUID) ([]domain.Job, error) {
	query := `
        SELECT j.id, j.tenant_id, j.title, j.endpoint_url, j.http_method, j.headers, j.payload, j.workflow_id
        FROM job_dependencies d
        JOIN jobs j ON d.downstream_job_id = j.id
        WHERE d.upstream_job_id = $1
    `
	rows, err := r.db.Query(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var j domain.Job
		if err := rows.Scan(&j.ID, &j.TenantID, &j.Title, &j.EndpointURL, &j.HTTPMethod, &j.Headers, &j.Payload, &j.WorkflowID); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}

	return jobs, nil
}

func (r *WorkflowRepository) CreateWorkflowRun(ctx context.Context, wfID uuid.UUID) (uuid.UUID, error) {
	id := uuid.New()
	query := `INSERT INTO workflow_runs (id, workflow_id, status) VALUES ($1, $2, 'RUNNING')`
	_, err := r.db.Exec(ctx, query, id, wfID)
	return id, err
}

func (r *WorkflowRepository) ScheduleDownstream(ctx context.Context, jobID uuid.UUID, runID uuid.UUID) error {
	query := `
		UPDATE jobs 
		SET next_run_at = NOW(),
		    next_workflow_run_id = $1,
		    status = 'ACTIVE',
		    updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, runID, jobID)
	return err
}
