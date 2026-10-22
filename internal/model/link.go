package model

import "time"

type Link struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement" json:"-"`
	Code       string     `gorm:"size:16;uniqueIndex:uk_code;not null" json:"code"`
	URL        string     `gorm:"type:varchar(2048);not null" json:"url"`
	ClickCount uint64     `gorm:"not null;default:0" json:"click_count"`
	ExpiresAt  *time.Time `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"-"`
}

func (l *Link) IsExpired(now time.Time) bool {
	return l.ExpiresAt != nil && now.After(*l.ExpiresAt)
}
