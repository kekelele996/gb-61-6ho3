package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

// CareLogRepository handles persistence of per-plant care logs.
type CareLogRepository struct {
	db *gorm.DB
}

// NewCareLogRepository creates a CareLogRepository.
func NewCareLogRepository(db *gorm.DB) *CareLogRepository {
	return &CareLogRepository{db: db}
}

// FindByKey locates the unique log of a garden item for a given date and type.
func (r *CareLogRepository) FindByKey(userID, gardenID uint, logDate time.Time, logType string) (*model.CareLog, error) {
	var m model.CareLog
	if err := r.db.Where("user_id = ? AND garden_id = ? AND log_date = ? AND log_type = ?",
		userID, gardenID, logDate, logType).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// FindByID locates a log by primary key.
func (r *CareLogRepository) FindByID(id uint) (*model.CareLog, error) {
	var m model.CareLog
	if err := r.db.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// Create inserts a new log, translating the unique-index violation into
// ErrDuplicate so the service can fall back to an update (upsert).
func (r *CareLogRepository) Create(m *model.CareLog) error {
	if err := r.db.Create(m).Error; err != nil {
		if isDuplicate(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

// Update persists a log.
func (r *CareLogRepository) Update(m *model.CareLog) error {
	if err := r.db.Save(m).Error; err != nil {
		if isDuplicate(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

// Delete removes a log by id.
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

// ListByGarden returns logs of one garden item for its owner, newest first,
// optionally filtered by type.
func (r *CareLogRepository) ListByGarden(userID, gardenID uint, logType string) ([]model.CareLog, error) {
	var items []model.CareLog
	q := r.db.Where("user_id = ? AND garden_id = ?", userID, gardenID)
	if logType != "" {
		q = q.Where("log_type = ?", logType)
	}
	if err := q.Order("log_date DESC, id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// TypeCount is an aggregate row of recent log counts grouped by type.
type TypeCount struct {
	LogType string
	Count   int64
}

// CountByTypeSince returns per-type log counts of one garden item since the
// given date (inclusive), used by the last-7-days summary.
func (r *CareLogRepository) CountByTypeSince(userID, gardenID uint, since time.Time) ([]TypeCount, error) {
	var rows []TypeCount
	if err := r.db.Model(&model.CareLog{}).
		Select("log_type, COUNT(*) AS count").
		Where("user_id = ? AND garden_id = ? AND log_date >= ?", userID, gardenID, since).
		Group("log_type").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
