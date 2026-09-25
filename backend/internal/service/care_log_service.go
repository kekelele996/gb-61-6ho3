package service

import (
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

// CareLogService implements plant care log recording rules: only plants in
// the caller's own garden are accepted, future dates are rejected, and the
// unique (garden, date, type) combination updates the existing entry.
type CareLogService struct {
	repo       *repository.CareLogRepository
	gardenRepo *repository.UserGardenRepository
	plantRepo  *repository.PlantSpeciesRepository
	logger     *slog.Logger
}

// NewCareLogService creates a CareLogService.
func NewCareLogService(repo *repository.CareLogRepository, gardenRepo *repository.UserGardenRepository, plantRepo *repository.PlantSpeciesRepository, logger *slog.Logger) *CareLogService {
	return &CareLogService{repo: repo, gardenRepo: gardenRepo, plantRepo: plantRepo, logger: logger}
}

// Save creates a care log or updates the existing entry for the same
// garden/date/type combination.
func (s *CareLogService) Save(userID uint, req dto.CareLogSaveRequest) (*dto.CareLogItem, bool, error) {
	logDate, err := time.ParseInLocation("2006-01-02", req.LogDate, time.Local)
	if err != nil {
		return nil, false, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareLog[garden_id=%d] save failed: log_date must be YYYY-MM-DD", req.GardenID))
	}
	if !constants.IsValidCareLogType(req.LogType) {
		return nil, false, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareLog[garden_id=%d type=%s] save failed: invalid log type", req.GardenID, req.LogType))
	}
	today := todayInLocation(time.Local)
	if logDate.After(today) {
		return nil, false, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareLog[garden_id=%d log_date=%s] save failed: future date is not allowed", req.GardenID, req.LogDate))
	}
	garden, err := s.gardenRepo.FindByID(req.GardenID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, false, util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("UserGarden[id=%d] not found", req.GardenID))
		}
		return nil, false, fmt.Errorf("care log garden find: %w", err)
	}
	if garden.UserID != userID {
		return nil, false, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("UserGarden[id=%d] care log save failed: not owner", req.GardenID))
	}

	updated := false
	entry, err := s.repo.FindByKey(userID, req.GardenID, logDate, req.LogType)
	switch {
	case err == nil:
		entry.Note = req.Note
		entry.ImageURL = req.ImageURL
		if err := s.repo.Update(entry); err != nil {
			return nil, false, fmt.Errorf("care log update: %w", err)
		}
		updated = true
		s.logger.Info(fmt.Sprintf(constants.LogCareLogUpdateSuccess, entry.ID), "user_id", userID, "garden_id", req.GardenID)
	case errors.Is(err, repository.ErrNotFound):
		entry = &model.CareLog{
			UserID:   userID,
			GardenID: req.GardenID,
			LogDate:  logDate,
			LogType:  req.LogType,
			Note:     req.Note,
			ImageURL: req.ImageURL,
		}
		if err := s.repo.Create(entry); err != nil {
			if errors.Is(err, repository.ErrDuplicate) {
				// Concurrent request for the same unique key: update it.
				entry, err = s.repo.FindByKey(userID, req.GardenID, logDate, req.LogType)
				if err != nil {
					return nil, false, fmt.Errorf("care log re-find: %w", err)
				}
				entry.Note = req.Note
				entry.ImageURL = req.ImageURL
				if err := s.repo.Update(entry); err != nil {
					return nil, false, fmt.Errorf("care log race update: %w", err)
				}
				updated = true
			} else {
				s.logger.Error(fmt.Sprintf(constants.LogCareLogCreateFailed, req.GardenID, userID), "error", err)
				return nil, false, fmt.Errorf("care log create: %w", err)
			}
		}
		if !updated {
			s.logger.Info(fmt.Sprintf(constants.LogCareLogCreateSuccess, entry.ID), "user_id", userID, "garden_id", req.GardenID, "type", req.LogType)
		}
	default:
		return nil, false, fmt.Errorf("care log find: %w", err)
	}

	plantName := ""
	if plant, perr := s.plantRepo.FindByID(garden.PlantSpeciesID); perr == nil {
		plantName = plant.Name
	}
	item := &dto.CareLogItem{
		ID:             entry.ID,
		UserID:         entry.UserID,
		GardenID:       entry.GardenID,
		PlantSpeciesID: garden.PlantSpeciesID,
		PlantName:      plantName,
		Nickname:       garden.Nickname,
		Location:       garden.Location,
		LogDate:        util.FormatDate(entry.LogDate),
		LogType:        entry.LogType,
		Note:           entry.Note,
		ImageURL:       entry.ImageURL,
		CreatedAt:      util.FormatDateTime(entry.CreatedAt),
		UpdatedAt:      util.FormatDateTime(entry.UpdatedAt),
	}
	return item, updated, nil
}

// List returns the caller's care logs with filters and per-type counts for
// the last 7 days.
func (s *CareLogService) List(userID, gardenID uint, logType string) (*dto.CareLogListResponse, error) {
	if logType != "" && !constants.IsValidCareLogType(logType) {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareLog list failed: invalid log type %s", logType))
	}
	if gardenID != 0 {
		if err := s.ensureGardenOwner(userID, gardenID); err != nil {
			return nil, err
		}
	}
	views, err := s.repo.List(userID, gardenID, logType)
	if err != nil {
		return nil, fmt.Errorf("care log list: %w", err)
	}
	items := make([]dto.CareLogItem, 0, len(views))
	for i := range views {
		items = append(items, careLogViewToItem(&views[i]))
	}

	since := todayInLocation(time.Local).AddDate(0, 0, -6)
	counts, err := s.repo.CountByTypeSince(userID, gardenID, since)
	if err != nil {
		return nil, fmt.Errorf("care log counts: %w", err)
	}
	recent := make(map[string]int64, len(constants.ValidCareLogTypes()))
	for _, t := range constants.ValidCareLogTypes() {
		recent[t] = 0
	}
	for _, c := range counts {
		recent[c.LogType] = c.Count
	}
	return &dto.CareLogListResponse{List: items, Recent7Days: recent}, nil
}

// Delete removes a care log owned by the caller.
func (s *CareLogService) Delete(userID, id uint) error {
	entry, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("CareLog[id=%d] not found", id))
		}
		return fmt.Errorf("care log delete find: %w", err)
	}
	if entry.UserID != userID {
		return util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("CareLog[id=%d] delete failed: user_id=%d not owner", id, userID))
	}
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("care log delete: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogCareLogDeleteSuccess, id), "user_id", userID)
	return nil
}

// ensureGardenOwner verifies that the garden item belongs to the user.
func (s *CareLogService) ensureGardenOwner(userID, gardenID uint) error {
	garden, err := s.gardenRepo.FindByID(gardenID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("UserGarden[id=%d] not found", gardenID))
		}
		return fmt.Errorf("care log garden find: %w", err)
	}
	if garden.UserID != userID {
		return util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("UserGarden[id=%d] care log access failed: not owner", gardenID))
	}
	return nil
}

func careLogViewToItem(v *repository.CareLogView) dto.CareLogItem {
	return dto.CareLogItem{
		ID:             v.ID,
		UserID:         v.UserID,
		GardenID:       v.GardenID,
		PlantSpeciesID: v.PlantSpeciesID,
		PlantName:      v.PlantName,
		Nickname:       v.Nickname,
		Location:       v.Location,
		LogDate:        util.FormatDate(v.LogDate),
		LogType:        v.LogType,
		Note:           v.Note,
		ImageURL:       v.ImageURL,
		CreatedAt:      util.FormatDateTime(v.CreatedAt),
		UpdatedAt:      util.FormatDateTime(v.UpdatedAt),
	}
}

// todayInLocation returns the calendar date at midnight in the given location.
func todayInLocation(loc *time.Location) time.Time {
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
}
