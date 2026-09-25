package service

import (
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"

	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// anyTime matches any time.Time argument in sqlmock expectations.
type anyTime struct{}

func (anyTime) Match(driver.Value) bool { return true }

func newCareLogService(db *gorm.DB) *CareLogService {
	return NewCareLogService(
		repository.NewCareLogRepository(db),
		repository.NewUserGardenRepository(db),
		newTestLogger(),
	)
}

func expectOwnGarden(mock sqlmock.Sqlmock, userID, gardenID uint) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `user_gardens` WHERE user_id = ? AND id = ? ORDER BY `user_gardens`.`id` LIMIT ?")).
		WithArgs(userID, gardenID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "plant_species_id"}).
			AddRow(gardenID, userID, 4))
}

// Saving a log for a plant the user does not own must be refused.
func TestCareLogSaveForeignGardenRejected(t *testing.T) {
	db, mock := newServiceDB(t)
	svc := newCareLogService(db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `user_gardens` WHERE user_id = ? AND id = ? ORDER BY `user_gardens`.`id` LIMIT ?")).
		WithArgs(uint(2), uint(99), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}))

	_, _, err := svc.Save(2, 99, dto.CareLogSaveRequest{
		LogDate: time.Now().Format("2006-01-02"), LogType: "watering",
	})
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != 404 {
		t.Fatalf("expected 404 AppError for foreign garden, got %v", err)
	}
}

// Future dates must never be saved.
func TestCareLogSaveFutureDateRejected(t *testing.T) {
	db, mock := newServiceDB(t)
	svc := newCareLogService(db)
	expectOwnGarden(mock, 2, 1)

	_, _, err := svc.Save(2, 1, dto.CareLogSaveRequest{
		LogDate: time.Now().AddDate(0, 0, 1).Format("2006-01-02"), LogType: "watering",
	})
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != 422 {
		t.Fatalf("expected 422 AppError for future date, got %v", err)
	}
}

// Unknown log types are rejected.
func TestCareLogSaveInvalidTypeRejected(t *testing.T) {
	db, mock := newServiceDB(t)
	svc := newCareLogService(db)
	expectOwnGarden(mock, 2, 1)

	_, _, err := svc.Save(2, 1, dto.CareLogSaveRequest{
		LogDate: time.Now().Format("2006-01-02"), LogType: "singing",
	})
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != 422 {
		t.Fatalf("expected 422 AppError for invalid type, got %v", err)
	}
}

// Malformed date strings are rejected instead of being stored as zero date.
func TestCareLogSaveMalformedDateRejected(t *testing.T) {
	db, mock := newServiceDB(t)
	svc := newCareLogService(db)
	expectOwnGarden(mock, 2, 1)

	_, _, err := svc.Save(2, 1, dto.CareLogSaveRequest{
		LogDate: "2026/09/25", LogType: "watering",
	})
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != 422 {
		t.Fatalf("expected 422 AppError for malformed date, got %v", err)
	}
}

// First submission of a (plant, day, type) inserts a new row.
func TestCareLogSaveCreatesWhenAbsent(t *testing.T) {
	db, mock := newServiceDB(t)
	svc := newCareLogService(db)
	expectOwnGarden(mock, 2, 1)

	// FindByKey: no existing log.
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `care_logs` WHERE user_id = ? AND garden_id = ? AND log_date = ? AND log_type = ? ORDER BY `care_logs`.`id` LIMIT ?")).
		WithArgs(uint(2), uint(1), anyTime{}, "watering", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "garden_id", "log_date", "log_type"}))

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `care_logs`")).
		WillReturnResult(sqlmock.NewResult(10, 1))
	mock.ExpectCommit()

	log, updated, err := svc.Save(2, 1, dto.CareLogSaveRequest{
		LogDate: time.Now().Format("2006-01-02"), LogType: "watering", Note: "浇透",
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if updated {
		t.Error("expected create (updated=false)")
	}
	if log.ID != 10 || log.Note != "浇透" {
		t.Errorf("unexpected log: %+v", log)
	}
}

// Same plant + day + type updates the existing record instead of inserting.
func TestCareLogSaveUpsertsSameDaySameType(t *testing.T) {
	db, mock := newServiceDB(t)
	svc := newCareLogService(db)
	expectOwnGarden(mock, 2, 1)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `care_logs` WHERE user_id = ? AND garden_id = ? AND log_date = ? AND log_type = ? ORDER BY `care_logs`.`id` LIMIT ?")).
		WithArgs(uint(2), uint(1), anyTime{}, "watering", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "garden_id", "log_date", "log_type", "note", "images"}).
			AddRow(10, 2, 1, time.Now(), "watering", "旧备注", "[]"))

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `care_logs` SET")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	log, updated, err := svc.Save(2, 1, dto.CareLogSaveRequest{
		LogDate: time.Now().Format("2006-01-02"), LogType: "watering", Note: "新备注",
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !updated {
		t.Error("expected update (updated=true)")
	}
	if log.ID != 10 || log.Note != "新备注" {
		t.Errorf("existing log should be updated in place: %+v", log)
	}
}

// Updating or deleting another user's log is forbidden.
func TestCareLogUpdateForbiddenForNonOwner(t *testing.T) {
	db, mock := newServiceDB(t)
	svc := newCareLogService(db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `care_logs` WHERE `care_logs`.`id` = ? ORDER BY `care_logs`.`id` LIMIT ?")).
		WithArgs(uint(10), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "garden_id", "log_date", "log_type"}).
			AddRow(10, 1, 1, time.Now(), "watering"))

	note := "hijack"
	_, err := svc.Update(2, 10, dto.CareLogUpdateRequest{Note: &note})
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != 403 {
		t.Fatalf("expected 403 AppError for non-owner update, got %v", err)
	}
}
