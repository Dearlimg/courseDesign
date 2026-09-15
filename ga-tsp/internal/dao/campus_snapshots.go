package dao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"simple_tuan/internal/models"
	"simple_tuan/pkg/campus"
)

type campusSnapshot struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	OwnerID   int64  `gorm:"index:idx_campus_owner_kind"`
	Kind      string `gorm:"type:varchar(16);index:idx_campus_owner_kind"`
	Payload   []byte `gorm:"type:longblob"`
	CreatedAt time.Time
}

func (campusSnapshot) TableName() string { return "tuan_campus_snapshots" }

type campusMap struct {
	Version string `gorm:"type:varchar(64);primaryKey"`
	Payload []byte `gorm:"type:longblob"`
}

func (campusMap) TableName() string { return "tuan_campus_maps" }

type CampusSnapshots struct{ db *gorm.DB }

func (u *MySQLUsers) CampusStore(ctx context.Context) (*CampusSnapshots, error) {
	if err := u.db.WithContext(ctx).AutoMigrate(&campusSnapshot{}, &campusMap{}); err != nil {
		return nil, fmt.Errorf("校园方案表初始化失败")
	}
	m := campus.Default()
	body, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	row := campusMap{Version: m.Version, Payload: body}
	if err := u.db.WithContext(ctx).Where("version = ?", m.Version).FirstOrCreate(&row).Error; err != nil {
		return nil, err
	}
	return &CampusSnapshots{db: u.db}, nil
}

func (s *CampusSnapshots) Save(ctx context.Context, owner int64, kind string, body json.RawMessage) (models.Snapshot, error) {
	if owner <= 0 || (kind != "batch" && kind != "plan") {
		return models.Snapshot{}, fmt.Errorf("非法快照归属或类型")
	}
	if len(body) > 1<<20 || !json.Valid(body) {
		return models.Snapshot{}, fmt.Errorf("非法快照内容")
	}
	row := campusSnapshot{OwnerID: owner, Kind: kind, Payload: body}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return models.Snapshot{}, err
	}
	return models.Snapshot{ID: row.ID, Kind: row.Kind, CreatedAt: row.CreatedAt, Payload: body}, nil
}

func (s *CampusSnapshots) List(ctx context.Context, owner int64, kind string) ([]models.Snapshot, error) {
	rows := []campusSnapshot{}
	err := s.db.WithContext(ctx).Select("id", "kind", "created_at").Where("owner_id = ? AND kind = ?", owner, kind).
		Order("id DESC").Limit(50).Find(&rows).Error
	result := make([]models.Snapshot, 0, len(rows))
	for _, row := range rows {
		result = append(result, models.Snapshot{ID: row.ID, Kind: row.Kind, CreatedAt: row.CreatedAt})
	}
	return result, err
}

func (s *CampusSnapshots) Get(ctx context.Context, owner int64, id uint64) (models.Snapshot, error) {
	var row campusSnapshot
	err := s.db.WithContext(ctx).Where("owner_id = ? AND id = ?", owner, id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Snapshot{}, ErrNotFound
	}
	return models.Snapshot{ID: row.ID, Kind: row.Kind, CreatedAt: row.CreatedAt, Payload: row.Payload}, err
}
