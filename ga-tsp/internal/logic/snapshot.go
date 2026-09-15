package logic

import (
	"context"
	"encoding/json"
	"simple_tuan/internal/models"
)

// CampusStore persists immutable inputs/results, scoped to the authenticated user.
type CampusStore interface {
	Save(context.Context, int64, string, json.RawMessage) (models.Snapshot, error)
	List(context.Context, int64, string) ([]models.Snapshot, error)
	Get(context.Context, int64, uint64) (models.Snapshot, error)
}
