package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/middleware"
	"github.com/gbplantwiki/gbplantwiki/internal/service"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// CareLogHandler exposes per-plant care log endpoints nested under gardens.
type CareLogHandler struct {
	svc    *service.CareLogService
	logger *slog.Logger
}

// NewCareLogHandler creates a CareLogHandler.
func NewCareLogHandler(svc *service.CareLogService, logger *slog.Logger) *CareLogHandler {
	return &CareLogHandler{svc: svc, logger: logger}
}

// List handles GET /gardens/:id/care-logs?type=.
func (h *CareLogHandler) List(c *gin.Context) {
	gardenID, ok := parseGardenID(c)
	if !ok {
		return
	}
	data, err := h.svc.List(middleware.GetUserID(c), gardenID, c.Query("type"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(data))
}

// Save handles POST /gardens/:id/care-logs; same plant + day + type updates
// the existing log (200) instead of creating a second row (201).
func (h *CareLogHandler) Save(c *gin.Context) {
	gardenID, ok := parseGardenID(c)
	if !ok {
		return
	}
	var req dto.CareLogSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	m, updated, err := h.svc.Save(middleware.GetUserID(c), gardenID, req)
	if err != nil {
		c.Error(err)
		return
	}
	if updated {
		c.JSON(http.StatusOK, dto.OK(service.ToCareLogResponse(m)))
		return
	}
	c.JSON(http.StatusCreated, dto.OK(service.ToCareLogResponse(m)))
}

// Update handles PUT /care-logs/:id.
func (h *CareLogHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid care log id"))
		return
	}
	var req dto.CareLogUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	m, err := h.svc.Update(middleware.GetUserID(c), uint(id), req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(service.ToCareLogResponse(m)))
}

// Delete handles DELETE /care-logs/:id.
func (h *CareLogHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid care log id"))
		return
	}
	if err := h.svc.Delete(middleware.GetUserID(c), uint(id)); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"deleted": true}))
}

// parseGardenID reads the :id route param used by nested /gardens/:id routes.
func parseGardenID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid garden id"))
		return 0, false
	}
	return uint(id), true
}
