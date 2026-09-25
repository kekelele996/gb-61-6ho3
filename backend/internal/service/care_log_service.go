package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// CareLogService implements per-plant care log logic: future-date rejection,
// same plant + day + type upsert, ownership checks and the last-7-days
// per-type summary.
type CareLogService struct {
	repo       *repository.CareLogRepository
	gardenRepo *repository.UserGardenRepository
	logger     *slog.Logger
}

// NewCareLogService creates a CareLogService.
func NewCareLogService(repo *repository.CareLogRepository, gardenRepo *repository.UserGardenRepository, logger *slog.Logger) *CareLogService {
	return &CareLogService{repo: repo, gardenRepo: gardenRepo, logger: logger}
}

// maxCareLogImages caps how many optional pictures one log may carry.
const maxCareLogImages = 9

// Save creates a log for one of the user's garden plants, or updates the
// existing log when the (plant, date, type) combination already exists.
func (s *CareLogService) Save(userID, gardenID uint, req dto.CareLogSaveRequest) (*model.CareLog, bool, error) {
	garden, err := s.gardenRepo.FindByUser(userID, gardenID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, false, util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("UserGarden[id=%d] care log save failed: plant not found in own garden", gardenID))
		}
		return nil, false, fmt.Errorf("care log save garden find: %w", err)
	}

	logDate, err := parseLogDate(req.LogDate)
	if err != nil {
		return nil, false, err
	}
	if !constants.IsValidCareLogType(req.LogType) {
		return nil, false, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareLog[log_type=%s] save failed: invalid log type", req.LogType))
	}
	images, err := normalizeImages(req.Images)
	if err != nil {
		return nil, false, err
	}

	exist, err := s.repo.FindByKey(userID, garden.ID, logDate, req.LogType)
	switch {
	case err == nil:
		// Same plant + day + type already logged: update instead of inserting.
		exist.Note = req.Note
		exist.Images = images
		if err := s.repo.Update(exist); err != nil {
			return nil, false, fmt.Errorf("care log save update: %w", err)
		}
		s.logger.Info(fmt.Sprintf(constants.LogCareLogUpdated, exist.ID, req.LogType), "id", exist.ID, "garden_id", garden.ID)
		return exist, true, nil
	case !errors.Is(err, repository.ErrNotFound):
		return nil, false, fmt.Errorf("care log save find: %w", err)
	}

	m := &model.CareLog{
		UserID:   userID,
		GardenID: garden.ID,
		LogDate:  logDate,
		LogType:  req.LogType,
		Note:     req.Note,
		Images:   images,
	}
	if err := s.repo.Create(m); err != nil {
		// Race with a concurrent insert on the same unique key: reload and update.
		if errors.Is(err, repository.ErrDuplicate) {
			exist, findErr := s.repo.FindByKey(userID, garden.ID, logDate, req.LogType)
			if findErr != nil {
				return nil, false, fmt.Errorf("care log save duplicate find: %w", findErr)
			}
			exist.Note = req.Note
			exist.Images = images
			if err := s.repo.Update(exist); err != nil {
				return nil, false, fmt.Errorf("care log save duplicate update: %w", err)
			}
			s.logger.Info(fmt.Sprintf(constants.LogCareLogUpdated, exist.ID, req.LogType), "id", exist.ID, "garden_id", garden.ID)
			return exist, true, nil
		}
		s.logger.Error(fmt.Sprintf(constants.LogCareLogCreateFailed, req.LogType, garden.ID), "error", err)
		return nil, false, fmt.Errorf("care log save create: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogCareLogCreateSuccess, m.ID, req.LogType, garden.ID), "id", m.ID, "user_id", userID)
	return m, false, nil
}

// List returns a garden plant's logs newest-first with optional type filter
// and the per-type counts within the last 7 days.
func (s *CareLogService) List(userID, gardenID uint, logType string) (dto.CareLogListData, error) {
	if _, err := s.requireOwnGarden(userID, gardenID); err != nil {
		return dto.CareLogListData{}, err
	}
	if logType != "" && !constants.IsValidCareLogType(logType) {
		return dto.CareLogListData{}, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareLog[log_type=%s] list failed: invalid log type", logType))
	}

	items, err := s.repo.ListByGarden(userID, gardenID, logType)
	if err != nil {
		return dto.CareLogListData{}, fmt.Errorf("care log list: %w", err)
	}

	weekStart := startOfDay(time.Now()).AddDate(0, 0, -6)
	counts, err := s.repo.CountByTypeSince(userID, gardenID, weekStart)
	if err != nil {
		return dto.CareLogListData{}, fmt.Errorf("care log weekly stat: %w", err)
	}
	stat := make([]dto.CareLogTypeCount, 0, len(constants.ValidCareLogTypes()))
	countByType := make(map[string]int64, len(counts))
	for _, c := range counts {
		countByType[c.LogType] = c.Count
	}
	for _, t := range constants.ValidCareLogTypes() {
		stat = append(stat, dto.CareLogTypeCount{LogType: t, Count: countByType[t]})
	}

	list := make([]dto.CareLogResponse, 0, len(items))
	for i := range items {
		list = append(list, ToCareLogResponse(&items[i]))
	}
	return dto.CareLogListData{List: list, WeeklyStat: stat}, nil
}

// Update edits an existing log, verifying ownership. Moving the date/type onto
// another occupied slot is rejected as a conflict.
func (s *CareLogService) Update(userID, id uint, req dto.CareLogUpdateRequest) (*model.CareLog, error) {
	m, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("CareLog[id=%d] not found", id))
		}
		return nil, fmt.Errorf("care log update find: %w", err)
	}
	if m.UserID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("CareLog[id=%d] update failed: user_id=%d not owner", id, userID))
	}

	targetDate := m.LogDate
	targetType := m.LogType
	if req.LogDate != nil {
		targetDate, err = parseLogDate(*req.LogDate)
		if err != nil {
			return nil, err
		}
	}
	if req.LogType != "" {
		if !constants.IsValidCareLogType(req.LogType) {
			return nil, util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("CareLog[id=%d] update failed: log_type=%s invalid", id, req.LogType))
		}
		targetType = req.LogType
	}
	if startOfDay(targetDate).Format("2006-01-02") != startOfDay(m.LogDate).Format("2006-01-02") || targetType != m.LogType {
		other, findErr := s.repo.FindByKey(userID, m.GardenID, targetDate, targetType)
		switch {
		case findErr == nil && other.ID != m.ID:
			return nil, util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("CareLog[garden_id=%d date=%s type=%s] update failed: log already exists", m.GardenID, targetDate.Format("2006-01-02"), targetType))
		case findErr != nil && !errors.Is(findErr, repository.ErrNotFound):
			return nil, fmt.Errorf("care log update key find: %w", findErr)
		}
	}
	if req.Note != nil {
		m.Note = *req.Note
	}
	if req.Images != nil {
		images, err := normalizeImages(req.Images)
		if err != nil {
			return nil, err
		}
		m.Images = images
	}
	m.LogDate = targetDate
	m.LogType = targetType
	if err := s.repo.Update(m); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("CareLog[garden_id=%d date=%s type=%s] update failed: log already exists", m.GardenID, targetDate.Format("2006-01-02"), targetType))
		}
		return nil, fmt.Errorf("care log update: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogCareLogUpdated, m.ID, m.LogType), "id", m.ID)
	return m, nil
}

// Delete removes a log owned by the user.
func (s *CareLogService) Delete(userID, id uint) error {
	m, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("CareLog[id=%d] not found", id))
		}
		return fmt.Errorf("care log delete find: %w", err)
	}
	if m.UserID != userID {
		return util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("CareLog[id=%d] delete failed: user_id=%d not owner", id, userID))
	}
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("care log delete: %w", err)
	}
	s.logger.Info("care log deleted", "id", id, "user_id", userID)
	return nil
}

// requireOwnGarden verifies the garden plant belongs to the user.
func (s *CareLogService) requireOwnGarden(userID, gardenID uint) (*model.UserGarden, error) {
	garden, err := s.gardenRepo.FindByUser(userID, gardenID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("UserGarden[id=%d] care log access failed: plant not found in own garden", gardenID))
		}
		return nil, fmt.Errorf("care log garden find: %w", err)
	}
	return garden, nil
}

// parseLogDate parses the "YYYY-MM-DD" form in local time and rejects future
// dates. The returned time is normalized to midnight.
func parseLogDate(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, util.NewAppError(422, constants.CodeValidationError, "CareLog save failed: log_date required")
	}
	day, err := time.ParseInLocation("2006-01-02", raw, time.Local)
	if err != nil {
		return time.Time{}, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareLog[log_date=%s] save failed: expected YYYY-MM-DD", raw))
	}
	if day.After(startOfDay(time.Now())) {
		return time.Time{}, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareLog[log_date=%s] save failed: future date not allowed", raw))
	}
	return day, nil
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// normalizeImages validates and serializes the optional image list to JSON.
func normalizeImages(images []string) (string, error) {
	if len(images) > maxCareLogImages {
		return "", util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareLog save failed: at most %d images allowed", maxCareLogImages))
	}
	for _, u := range images {
		if len(u) > 500 {
			return "", util.NewAppError(422, constants.CodeValidationError, "CareLog save failed: image url too long")
		}
	}
	raw, err := json.Marshal(images)
	if err != nil {
		return "", util.NewAppError(422, constants.CodeValidationError, "CareLog save failed: invalid images")
	}
	return string(raw), nil
}

// ToCareLogResponse converts a model row to the DTO with parsed images.
func ToCareLogResponse(m *model.CareLog) dto.CareLogResponse {
	var images []string
	if m.Images != "" {
		if err := json.Unmarshal([]byte(m.Images), &images); err != nil {
			images = []string{}
		}
	}
	if images == nil {
		images = []string{}
	}
	return dto.CareLogResponse{
		ID:        m.ID,
		UserID:    m.UserID,
		GardenID:  m.GardenID,
		LogDate:   m.LogDate,
		LogType:   m.LogType,
		Note:      m.Note,
		Images:    images,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
