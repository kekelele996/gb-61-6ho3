package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

func newSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.UserGarden{}, &model.CareLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Each test starts from a clean schema.
	db.Exec("DELETE FROM care_logs")
	db.Exec("DELETE FROM user_gardens")
	return db
}

// The unique index must turn a second (garden, date, type) insert into
// ErrDuplicate, which the service uses to switch to update (upsert).
func TestCareLogUniqueIndexRejectsDuplicate(t *testing.T) {
	db := newSQLiteDB(t)
	repo := NewCareLogRepository(db)
	day := time.Date(2026, 9, 25, 0, 0, 0, 0, time.Local)

	first := &model.CareLog{UserID: 2, GardenID: 1, LogDate: day, LogType: "watering", Note: "a", Images: "[]"}
	if err := repo.Create(first); err != nil {
		t.Fatalf("first create: %v", err)
	}
	second := &model.CareLog{UserID: 2, GardenID: 1, LogDate: day, LogType: "watering", Note: "b", Images: "[]"}
	if err := repo.Create(second); err == nil || !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Fatalf("expected unique constraint violation, got %v", err)
	}

	// Different type on the same day is fine.
	other := &model.CareLog{UserID: 2, GardenID: 1, LogDate: day, LogType: "pruning", Images: "[]"}
	if err := repo.Create(other); err != nil {
		t.Fatalf("different type create: %v", err)
	}
}

// List must order newest first and respect the type filter; the weekly stat
// counts only dates within the last 7 days (inclusive).
func TestCareLogListOrderFilterAndWeeklyCount(t *testing.T) {
	db := newSQLiteDB(t)
	repo := NewCareLogRepository(db)
	now := time.Now()
	mk := func(daysAgo int, typ string) {
		m := &model.CareLog{
			UserID: 2, GardenID: 1,
			LogDate: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, -daysAgo),
			LogType: typ, Images: "[]",
		}
		if err := db.Create(m).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	mk(0, "watering")
	mk(6, "watering")
	mk(7, "watering") // exactly 7 days ago is outside the inclusive 7-day window
	mk(2, "fertilizing")
	mk(20, "pruning")
	// Another user's garden must never leak in.
	if err := db.Create(&model.CareLog{UserID: 3, GardenID: 2, LogDate: now, LogType: "watering", Images: "[]"}).Error; err != nil {
		t.Fatalf("seed other user: %v", err)
	}

	items, err := repo.ListByGarden(2, 1, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 5 {
		t.Fatalf("expected 5 own logs, got %d", len(items))
	}
	if items[0].LogDate.Before(items[len(items)-1].LogDate) {
		t.Error("list must be ordered newest first")
	}

	filtered, err := repo.ListByGarden(2, 1, "watering")
	if err != nil {
		t.Fatalf("filtered list: %v", err)
	}
	if len(filtered) != 3 {
		t.Fatalf("expected 3 watering logs, got %d", len(filtered))
	}

	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, -6)
	counts, err := repo.CountByTypeSince(2, 1, since)
	if err != nil {
		t.Fatalf("weekly count: %v", err)
	}
	got := map[string]int64{}
	for _, c := range counts {
		got[c.LogType] = c.Count
	}
	if got["watering"] != 2 {
		t.Errorf("watering within 7 days = %d, want 2 (today and 6d ago)", got["watering"])
	}
	if got["fertilizing"] != 1 {
		t.Errorf("fertilizing within 7 days = %d, want 1", got["fertilizing"])
	}
	if _, ok := got["pruning"]; ok {
		t.Errorf("pruning 20 days ago must not appear, got %d", got["pruning"])
	}
}
