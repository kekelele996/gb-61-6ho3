package dto

// CareLogSaveRequest creates or updates a care log entry. The same
// (garden_id, log_date, log_type) combination updates the existing entry
// instead of creating a new one.
type CareLogSaveRequest struct {
	GardenID uint   `json:"garden_id" binding:"required"`
	LogDate  string `json:"log_date" binding:"required"`
	LogType  string `json:"log_type" binding:"required"`
	Note     string `json:"note" binding:"omitempty,max=512"`
	ImageURL string `json:"image_url" binding:"omitempty,max=255"`
}

// CareLogItem is a care log joined with garden/plant display fields.
type CareLogItem struct {
	ID             uint   `json:"id"`
	UserID         uint   `json:"user_id"`
	GardenID       uint   `json:"garden_id"`
	PlantSpeciesID uint   `json:"plant_species_id"`
	PlantName      string `json:"plant_name"`
	Nickname       string `json:"nickname"`
	Location       string `json:"location"`
	LogDate        string `json:"log_date"`
	LogType        string `json:"log_type"`
	Note           string `json:"note"`
	ImageURL       string `json:"image_url"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// CareLogListResponse bundles the log list with per-type counts for the
// last 7 days.
type CareLogListResponse struct {
	List        []CareLogItem    `json:"list"`
	Recent7Days map[string]int64 `json:"recent_7_days"`
}
