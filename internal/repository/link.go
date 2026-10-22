package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"sniplink/internal/model"
)

type LinkRepo struct {
	db *gorm.DB
}

func NewLinkRepo(db *gorm.DB) *LinkRepo { return &LinkRepo{db: db} }

func (r *LinkRepo) Create(ctx context.Context, link *model.Link) error {
	return r.db.WithContext(ctx).Create(link).Error
}

func (r *LinkRepo) FindByCode(ctx context.Context, code string) (*model.Link, error) {
	var l model.Link
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *LinkRepo) IncrClick(ctx context.Context, code string, delta int64) error {
	return r.db.WithContext(ctx).
		Model(&model.Link{}).
		Where("code = ?", code).
		UpdateColumn("click_count", gorm.Expr("click_count + ?", delta)).Error
}

func (r *LinkRepo) DeleteByCode(ctx context.Context, code string) error {
	return r.db.WithContext(ctx).Where("code = ?", code).Delete(&model.Link{}).Error
}

// DeleteExpired 供定时任务清理过期链接(本项目未接调度器,可自行用 cron 调用)
func (r *LinkRepo) DeleteExpired(ctx context.Context, limit int) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Limit(limit).
		Delete(&model.Link{})
	return res.RowsAffected, res.Error
}
