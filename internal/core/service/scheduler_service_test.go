package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vantutran2k1/orbit/internal/core/domain"
	"github.com/vantutran2k1/orbit/internal/core/ports"
)

func TestScheduleJob_CalculatesNextRun(t *testing.T) {
	mockRepo := new(ports.JobRepositoryMock)
	service := NewSchedulerService(mockRepo)

	job := &domain.Job{
		Title:          "Test Job",
		CronExpression: "0 10 * * *",
		EndpointURL:    "url",
		IsRecurring:    true,
	}

	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := service.ScheduleJob(context.Background(), job)

	assert.NoError(t, err)
	assert.True(t, job.NextRunAt.After(time.Now()))
}
