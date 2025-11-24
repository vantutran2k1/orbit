package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vantutran2k1/orbit/internal/core/domain"
)

type JobRepository interface {
	Create(ctx context.Context, job *domain.Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	ListDueJobs(ctx context.Context) ([]domain.Job, error)
	UpdateNextRun(ctx context.Context, id uuid.UUID, nextRun time.Time) error
	SaveExecution(ctx context.Context, exec *domain.Execution) error
	UpdateJobSchedule(ctx context.Context, jobID uuid.UUID, nextRun time.Time, status domain.JobStatus) error
	UpdateJobStatusAfterRun(ctx context.Context, jobID uuid.UUID, nextRun time.Time, failures int) error
}
