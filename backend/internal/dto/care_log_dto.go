package dto

import "time"

// CareLogSaveRequest creates or updates (upsert) a care log for a plant in the
// user's garden. One garden item + log_date + log_type maps to a single row,
// so a repeated submission updates the existing record instead of inserting.
// LogDate uses the "YYYY-MM-DD" form sent by the date picker.
type CareLogSaveRequest struct {
	LogDate string   `json:"log_date" binding:"required"`
	LogType string   `json:"log_type" binding:"required,max=32"`
	Note    string   `json:"note" binding:"omitempty,max=2000"`
	Images  []string `json:"images"`
}

// CareLogUpdateRequest edits an existing log identified by its own id. Date
// and type are changeable; moving onto another (date, type) slot that already
// has a log is rejected as a conflict by the service.
type CareLogUpdateRequest struct {
	LogDate *string  `json:"log_date"`
	LogType string   `json:"log_type" binding:"omitempty,max=32"`
	Note    *string  `json:"note"`
	Images  []string `json:"images"`
}

// CareLogTypeCount is one entry of the last-7-days activity summary.
type CareLogTypeCount struct {
	LogType string `json:"log_type"`
	Count   int64  `json:"count"`
}

// CareLogListData is the payload of GET /gardens/:id/care-logs: the filtered
// logs (newest first) plus per-type counts within the last 7 days.
type CareLogListData struct {
	List       []CareLogResponse  `json:"list"`
	WeeklyStat []CareLogTypeCount `json:"weekly_stat"`
}

// CareLogResponse is a care log with parsed image list.
type CareLogResponse struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	GardenID  uint      `json:"garden_id"`
	LogDate   time.Time `json:"log_date"`
	LogType   string    `json:"log_type"`
	Note      string    `json:"note"`
	Images    []string  `json:"images"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
