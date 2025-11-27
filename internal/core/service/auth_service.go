package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthService struct {
	db *pgxpool.Pool
}

func NewAuthService(db *pgxpool.Pool) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) CreateKey(ctx context.Context, tenantID uuid.UUID, name string) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	rawKey := "orbit_sk_" + base64.RawURLEncoding.EncodeToString(bytes)

	hash := sha256.Sum256([]byte(rawKey))
	hashString := hex.EncodeToString(hash[:])

	query := `INSERT INTO api_keys (tenant_id, key_hash, name) VALUES ($1, $2, $3)`
	_, err := s.db.Exec(ctx, query, tenantID, hashString, name)

	return rawKey, err
}

func (s *AuthService) ValidateKey(ctx context.Context, rawKey string) (uuid.UUID, error) {
	hash := sha256.Sum256([]byte(rawKey))
	hashString := hex.EncodeToString(hash[:])

	var tenantID uuid.UUID
	query := `SELECT tenant_id FROM api_keys WHERE key_hash = $1`

	if err := s.db.QueryRow(ctx, query, hashString).Scan(&tenantID); err != nil {
		return uuid.Nil, errors.New("invalid api key")
	}

	return tenantID, nil
}
