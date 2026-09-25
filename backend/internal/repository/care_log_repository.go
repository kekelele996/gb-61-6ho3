package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

// CareLogView is a care log joined with its garden item and plant species,
// used for list rendering.
type CareLogView struct {
	model.CareLog
	Nickname       string `gorm:"column:nickname"`
	Location       string `gorm:"column:location"`
	PlantSpeciesID uint   `gorm:"column:plant_species_id"`
	PlantName      string `gorm:"column:plant_name"`
}

// CareLogTypeCount holds a per-type aggregate count.
type CareLogTypeCount struct {
	LogType string
	Count   int64
}

// CareLogRepository handles persistence of plant care logs.
type CareLogRepository struct {
	db *gorm.DB
}

// NewCareLogRepository creates a CareLogRepository.
func NewCareLogRepository(db *gorm.DB) *CareLogRepository {
	return &CareLogRepository{db: db}
}

// Create inserts a care log.
func (r *CareLogRepository) Create(l *model.CareLog) error {
	if err := r.db.Create(l).Error; err != nil {
		if isDuplicate(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

// FindByKey locates the unique log for a user/garden/date/type combination.
func (r *CareLogRepository) FindByKey(userID, gardenID uint, logDate time.Time, logType string) (*model.CareLog, error) {
	var l model.CareLog
	if err := r.db.Where(
		"user_id = ? AND garden_id = ? AND log_date = ? AND log_type = ?",
		userID, gardenID, logDate, logType,
	).First(&l).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}

// FindByID locates a care log by primary key.
func (r *CareLogRepository) FindByID(id uint) (*model.CareLog, error) {
	var l model.CareLog
	if err := r.db.First(&l, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}

// Update persists changes on a care log.
func (r *CareLogRepository) Update(l *model.CareLog) error {
	return r.db.Save(l).Error
}

// Delete removes a care log by id.
func (r *CareLogRepository) Delete(id uint) error {
	res := r.db.Delete(&model.CareLog{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// List returns care logs of a user, optionally filtered by garden and type,
// ordered by date descending (most recent first), then type and id.
func (r *CareLogRepository) List(userID, gardenID uint, logType string) ([]CareLogView, error) {
	var items []CareLogView
	q := r.db.Table("care_logs AS cl").
		Select("cl.*, g.nickname, g.location, g.plant_species_id, p.name AS plant_name").
		Joins("LEFT JOIN user_gardens AS g ON g.id = cl.garden_id").
		Joins("LEFT JOIN plant_species AS p ON p.id = g.plant_species_id").
		Where("cl.user_id = ?", userID)
	if gardenID != 0 {
		q = q.Where("cl.garden_id = ?", gardenID)
	}
	if logType != "" {
		q = q.Where("cl.log_type = ?", logType)
	}
	if err := q.Order("cl.log_date DESC, cl.log_type ASC, cl.id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CountByTypeSince counts logs per type since the given date (inclusive),
// optionally restricted to one garden item.
func (r *CareLogRepository) CountByTypeSince(userID, gardenID uint, since time.Time) ([]CareLogTypeCount, error) {
	var counts []CareLogTypeCount
	q := r.db.Model(&model.CareLog{}).
		Select("log_type, COUNT(*) AS count").
		Where("user_id = ? AND log_date >= ?", userID, since)
	if gardenID != 0 {
		q = q.Where("garden_id = ?", gardenID)
	}
	if err := q.Group("log_type").Scan(&counts).Error; err != nil {
		return nil, err
	}
	return counts, nil
}
