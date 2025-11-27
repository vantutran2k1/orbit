package domain

import (
	"time"

	"github.com/google/uuid"
)

type Workflow struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Jobs      []Job     `json:"jobs,omitempty"`
}

type Dependency struct {
	UpstreamJobID   uuid.UUID `db:"upstream_job_id"`
	DownstreamJobID uuid.UUID `db:"downstream_job_id"`
}
