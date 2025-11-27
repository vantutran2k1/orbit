package domain

import (
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	KeyHash   string    `json:"-" db:"key_hash"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
