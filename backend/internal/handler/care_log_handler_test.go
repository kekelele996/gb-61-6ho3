package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/gbplantwiki/gbplantwiki/internal/config"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/middleware"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"github.com/gbplantwiki/gbplantwiki/internal/service"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

type careLogEnv struct {
	router *gin.Engine
	db     *gorm.DB
	cfg    *config.Config
}

func newCareLogEnv(t *testing.T) *careLogEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.UserGarden{}, &model.CareLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := config.Load()
	careLogRepo := repository.NewCareLogRepository(db)
	gardenRepo := repository.NewUserGardenRepository(db)
	svc := service.NewCareLogService(careLogRepo, gardenRepo, logger)
	h := NewCareLogHandler(svc, logger)

	r := gin.New()
	r.Use(middleware.ErrorHandler(logger))
	auth := r.Group("/api/v1", middleware.AuthRequired(cfg))
	{
		auth.GET("/gardens/:id/care-logs", h.List)
		auth.POST("/gardens/:id/care-logs", h.Save)
		auth.PUT("/care-logs/:id", h.Update)
		auth.DELETE("/care-logs/:id", h.Delete)
	}

	t.Cleanup(func() {
		db.Exec("DELETE FROM care_logs")
		db.Exec("DELETE FROM user_gardens")
	})
	return &careLogEnv{router: r, db: db, cfg: cfg}
}

func (e *careLogEnv) token(t *testing.T, userID uint) string {
	t.Helper()
	tok, err := util.GenerateToken(userID, "tester", "user", e.cfg.JWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return tok
}

func (e *careLogEnv) do(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func seedGarden(t *testing.T, db *gorm.DB, id, userID uint) {
	t.Helper()
	if err := db.Create(&model.UserGarden{ID: id, UserID: userID, PlantSpeciesID: 1}).Error; err != nil {
		t.Fatalf("seed garden: %v", err)
	}
}

func TestCareLogHTTPEndToEnd(t *testing.T) {
	env := newCareLogEnv(t)
	seedGarden(t, env.db, 1, 2) // user 2 owns garden 1
	seedGarden(t, env.db, 2, 3) // garden 2 belongs to someone else

	today := time.Now().Format("2006-01-02")
	oldDay := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	token2 := env.token(t, 2)
	token3 := env.token(t, 3)

	// Unauthenticated request is rejected.
	if w := env.do(t, "GET", "/api/v1/gardens/1/care-logs", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("no token: code=%d want 401", w.Code)
	}

	// Another user's plant: access refused.
	if w := env.do(t, "GET", "/api/v1/gardens/2/care-logs", token2, nil); w.Code != http.StatusNotFound {
		t.Fatalf("foreign garden list: code=%d want 404 body=%s", w.Code, w.Body.String())
	}

	// Create first log -> 201.
	w := env.do(t, "POST", "/api/v1/gardens/1/care-logs", token2, map[string]any{
		"log_date": today, "log_type": "watering", "note": "第一次浇水", "images": []string{"a.jpg"},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: code=%d body=%s", w.Code, w.Body.String())
	}
	var created struct {
		Data dto.CareLogResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if created.Data.ID == 0 || len(created.Data.Images) != 1 {
		t.Fatalf("unexpected created payload: %s", w.Body.String())
	}

	// Same plant + day + type -> update (200), not a second row.
	w = env.do(t, "POST", "/api/v1/gardens/1/care-logs", token2, map[string]any{
		"log_date": today, "log_type": "watering", "note": "更新后的浇水备注",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert: code=%d body=%s", w.Code, w.Body.String())
	}
	var count int64
	env.db.Model(&model.CareLog{}).Where("garden_id = ? AND log_type = ?", 1, "watering").Count(&count)
	if count != 1 {
		t.Errorf("same day+type must stay one row, got %d", count)
	}

	// A different type on the same day is allowed -> 201.
	if w := env.do(t, "POST", "/api/v1/gardens/1/care-logs", token2, map[string]any{
		"log_date": today, "log_type": "pruning", "note": "修剪残花",
	}); w.Code != http.StatusCreated {
		t.Fatalf("other type create: code=%d body=%s", w.Code, w.Body.String())
	}

	// An older watering log feeds the 7-day counts.
	if w := env.do(t, "POST", "/api/v1/gardens/1/care-logs", token2, map[string]any{
		"log_date": oldDay, "log_type": "watering",
	}); w.Code != http.StatusCreated {
		t.Fatalf("old log create: code=%d body=%s", w.Code, w.Body.String())
	}

	// Future date rejected.
	if w := env.do(t, "POST", "/api/v1/gardens/1/care-logs", token2, map[string]any{
		"log_date": time.Now().AddDate(0, 0, 1).Format("2006-01-02"), "log_type": "watering",
	}); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("future date: code=%d want 422 body=%s", w.Code, w.Body.String())
	}

	// List: newest first + weekly counts.
	w = env.do(t, "GET", "/api/v1/gardens/1/care-logs", token2, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: code=%d body=%s", w.Code, w.Body.String())
	}
	var listResp struct {
		Data dto.CareLogListData `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if len(listResp.Data.List) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(listResp.Data.List))
	}
	if listResp.Data.List[0].LogDate.Before(listResp.Data.List[2].LogDate) {
		t.Error("logs must be ordered newest first")
	}
	stat := map[string]int64{}
	for _, c := range listResp.Data.WeeklyStat {
		stat[c.LogType] = c.Count
	}
	if stat["watering"] != 2 || stat["pruning"] != 1 {
		t.Errorf("weekly stat wrong: watering=%d pruning=%d", stat["watering"], stat["pruning"])
	}

	// Type filter.
	w = env.do(t, "GET", "/api/v1/gardens/1/care-logs?type=pruning", token2, nil)
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if len(listResp.Data.List) != 1 || listResp.Data.List[0].LogType != "pruning" {
		t.Fatalf("type filter returned %d logs", len(listResp.Data.List))
	}

	// Non-owner update / delete of user 2's logs are forbidden.
	w = env.do(t, "PUT", "/api/v1/care-logs/"+strconv.FormatUint(uint64(created.Data.ID), 10), token3, map[string]any{
		"note": "恶意修改",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-owner update: code=%d want 403", w.Code)
	}
	w = env.do(t, "DELETE", "/api/v1/care-logs/"+strconv.FormatUint(uint64(created.Data.ID), 10), token3, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-owner delete: code=%d want 403", w.Code)
	}

	// Owner edit via PUT moves a note, then owner delete succeeds.
	if w := env.do(t, "PUT", "/api/v1/care-logs/"+strconv.FormatUint(uint64(created.Data.ID), 10), token2, map[string]any{
		"note": "主人自己改",
	}); w.Code != http.StatusOK {
		t.Fatalf("owner update: code=%d body=%s", w.Code, w.Body.String())
	}
	if w := env.do(t, "DELETE", "/api/v1/care-logs/"+strconv.FormatUint(uint64(created.Data.ID), 10), token2, nil); w.Code != http.StatusOK {
		t.Fatalf("owner delete: code=%d body=%s", w.Code, w.Body.String())
	}
}
