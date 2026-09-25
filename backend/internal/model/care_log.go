package model

import "time"

// CareLog is a single care activity recorded for one plant in the user's
// garden. At most one log of the same type may exist per garden item per day:
// re-submitting that combination updates the existing row (upsert), which is
// enforced both in the service and by the uk_carelog_garden_date_type index.
type CareLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index:idx_carelog_user;not null" json:"user_id"`
	GardenID  uint      `gorm:"index:uk_carelog_garden_date_type,unique,priority:1;not null" json:"garden_id"`
	LogDate   time.Time `gorm:"type:date;index:uk_carelog_garden_date_type,unique,priority:2;not null" json:"log_date"`
	LogType   string    `gorm:"size:32;index:uk_carelog_garden_date_type,unique,priority:3;not null" json:"log_type"`
	Note      string    `gorm:"type:text" json:"note"`
	Images    string    `gorm:"type:json" json:"images"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
