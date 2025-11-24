package http

import (
	"errors"

	"github.com/vantutran2k1/orbit/internal/core/domain"
)

type CreateJobRequest struct {
	Title          string                 `json:"title"`
	CronExpression string                 `json:"cron_expression"`
	EndpointURL    string                 `json:"endpoint_url"`
	HTTPMethod     string                 `json:"http_method"`
	Headers        map[string]string      `json:"headers"`
	Payload        map[string]interface{} `json:"payload"`
}

type JobResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	NextRunAt string `json:"next_run_at,omitempty"`
}

func (r *CreateJobRequest) Validate() error {
	if r.Title == "" {
		return errors.New("title is required")
	}
	if r.EndpointURL == "" {
		return errors.New("endpoint_url is required")
	}
	if r.CronExpression == "" {
		return errors.New("cron_expression is required")
	}
	return nil
}

func (r *CreateJobRequest) ToDomain(tenantID string) *domain.Job {
	method := r.HTTPMethod
	if method == "" {
		method = "POST"
	}

	return &domain.Job{
		Title:          r.Title,
		CronExpression: r.CronExpression,
		IsRecurring:    true,
		EndpointURL:    r.EndpointURL,
		HTTPMethod:     method,
		Headers:        r.Headers,
		Payload:        r.Payload,
	}
}
