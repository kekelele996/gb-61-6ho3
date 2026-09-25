package service

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
)

func TestCareLogSaveRejectsFutureDate(t *testing.T) {
	db, _ := newServiceDB(t)
	svc := NewCareLogService(repository.NewCareLogRepository(db), repository.NewUserGardenRepository(db), repository.NewPlantSpeciesRepository(db), newTestLogger())
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	req := dto.CareLogSaveRequest{GardenID: 1, LogDate: tomorrow, LogType: constants.CareLogWatering}
	if _, _, err := svc.Save(2, req); err == nil {
		t.Fatal("expected future date to be rejected")
	}
}

func TestCareLogSaveRejectsInvalidType(t *testing.T) {
	db, _ := newServiceDB(t)
	svc := NewCareLogService(repository.NewCareLogRepository(db), repository.NewUserGardenRepository(db), repository.NewPlantSpeciesRepository(db), newTestLogger())
	today := time.Now().Format("2006-01-02")
	req := dto.CareLogSaveRequest{GardenID: 1, LogDate: today, LogType: "repotting"}
	if _, _, err := svc.Save(2, req); err == nil {
		t.Fatal("expected invalid type to be rejected")
	}
}

func TestCareLogSaveRejectsForeignGarden(t *testing.T) {
	db, mock := newServiceDB(t)
	gardenRepo := repository.NewUserGardenRepository(db)
	svc := NewCareLogService(repository.NewCareLogRepository(db), gardenRepo, repository.NewPlantSpeciesRepository(db), newTestLogger())
	today := time.Now().Format("2006-01-02")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `user_gardens` WHERE `user_gardens`.`id` = ? ORDER BY `user_gardens`.`id` LIMIT ?")).
		WithArgs(uint64(1), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "plant_species_id", "nickname"}).
			AddRow(1, 99, 4, "别人的月季"))
	req := dto.CareLogSaveRequest{GardenID: 1, LogDate: today, LogType: constants.CareLogWatering}
	if _, _, err := svc.Save(2, req); err == nil {
		t.Fatal("expected foreign garden to be forbidden")
	}
}

func TestCareLogSaveUpdatesExistingSameDaySameType(t *testing.T) {
	db, mock := newServiceDB(t)
	careLogRepo := repository.NewCareLogRepository(db)
	gardenRepo := repository.NewUserGardenRepository(db)
	svc := NewCareLogService(careLogRepo, gardenRepo, repository.NewPlantSpeciesRepository(db), newTestLogger())

	today := time.Now().Format("2006-01-02")
	parsed, _ := time.ParseInLocation("2006-01-02", today, time.Local)

	mock.MatchExpectationsInOrder(true)
	// garden ownership lookup
	mock.ExpectQuery("SELECT \\* FROM `user_gardens`").
		WithArgs(uint64(7), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "plant_species_id", "nickname", "location"}).
			AddRow(7, 2, 4, "阳台月季", "南向阳台"))
	// unique key lookup finds an existing entry
	mock.ExpectQuery("SELECT \\* FROM `care_logs` WHERE user_id").
		WithArgs(uint64(2), uint64(7), parsed, "watering", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "garden_id", "log_date", "log_type", "note", "image_url"}).
			AddRow(10, 2, 7, parsed, "watering", "旧备注", ""))
	// save updates it
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `care_logs` SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	req := dto.CareLogSaveRequest{GardenID: 7, LogDate: today, LogType: constants.CareLogWatering, Note: "新备注"}
	item, updated, err := svc.Save(2, req)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !updated {
		t.Error("expected existing entry to be updated, not created")
	}
	if item.ID != 10 || item.Note != "新备注" {
		t.Errorf("unexpected item: %+v", item)
	}
}

func TestCareLogDeleteForbiddenForNonOwner(t *testing.T) {
	db, mock := newServiceDB(t)
	svc := NewCareLogService(repository.NewCareLogRepository(db), repository.NewUserGardenRepository(db), repository.NewPlantSpeciesRepository(db), newTestLogger())
	mock.ExpectQuery("SELECT \\* FROM `care_logs` WHERE `care_logs`.`id`").
		WithArgs(uint64(10), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "garden_id", "log_date", "log_type"}).
			AddRow(10, 99, 7, time.Now(), "watering"))
	if err := svc.Delete(2, 10); err == nil {
		t.Fatal("expected delete of foreign log to be forbidden")
	}
}
