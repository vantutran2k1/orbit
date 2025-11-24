package service

import (
	"context"
	"errors"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/vantutran2k1/orbit/internal/core/domain"
	"github.com/vantutran2k1/orbit/internal/core/ports"
)

type SchedulerService struct {
	repo       ports.JobRepository
	cronParser cron.Parser
}

func NewSchedulerService(repo ports.JobRepository) *SchedulerService {
	return &SchedulerService{
		repo:       repo,
		cronParser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

func (s *SchedulerService) ScheduleJob(ctx context.Context, job *domain.Job) error {
	if job.Title == "" || job.EndpointURL == "" {
		return errors.New("missing required fields")
	}

	if job.IsRecurring {
		schedule, err := s.cronParser.Parse(job.CronExpression)
		if err != nil {
			return errors.New("invalid cron expression: " + err.Error())
		}
		job.NextRunAt = schedule.Next(time.Now())
	} else {
		if job.NextRunAt.Before(time.Now()) {
			return errors.New("cannot schedule in the past")
		}
	}

	job.Status = domain.JobStatusActive
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	return s.repo.Create(ctx, job)
}
