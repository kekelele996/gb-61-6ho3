package model

import "time"

// CareLog is a single dated care activity (watering, fertilizing, medication,
// pruning or observation) recorded for one plant instance in the owner's
// garden. At most one entry may exist per (user, garden, date, type);
// re-submitting the same combination updates the existing entry.
type CareLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_carelog_user_garden_date_type,priority:1;not null" json:"user_id"`
	GardenID  uint      `gorm:"uniqueIndex:uk_carelog_user_garden_date_type,priority:2;not null" json:"garden_id"`
	LogDate   time.Time `gorm:"type:date;uniqueIndex:uk_carelog_user_garden_date_type,priority:3;not null" json:"log_date"`
	LogType   string    `gorm:"size:16;uniqueIndex:uk_carelog_user_garden_date_type,priority:4;not null" json:"log_type"`
	Note      string    `gorm:"size:512" json:"note"`
	ImageURL  string    `gorm:"size:255" json:"image_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
