package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/vantutran2k1/orbit/internal/core/domain"
)

var _ JobRepository = (*JobRepositoryMock)(nil)

type JobRepositoryMock struct {
	mock.Mock
}

func (m *JobRepositoryMock) Create(ctx context.Context, job *domain.Job) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *JobRepositoryMock) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.Job), args.Error(1)
}

func (m *JobRepositoryMock) ListDueJobs(ctx context.Context) ([]domain.Job, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Job), args.Error(1)
}

func (m *JobRepositoryMock) UpdateNextRun(ctx context.Context, id uuid.UUID, nextRun time.Time) error {
	args := m.Called(ctx, id, nextRun)
	return args.Error(0)
}
