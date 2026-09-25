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

// CareLogHandler exposes plant care log endpoints.
type CareLogHandler struct {
	svc    *service.CareLogService
	logger *slog.Logger
}

// NewCareLogHandler creates a CareLogHandler.
func NewCareLogHandler(svc *service.CareLogService, logger *slog.Logger) *CareLogHandler {
	return &CareLogHandler{svc: svc, logger: logger}
}

// List handles GET /care-logs?garden_id=&type=.
func (h *CareLogHandler) List(c *gin.Context) {
	gardenID, _ := strconv.ParseUint(c.Query("garden_id"), 10, 64)
	data, err := h.svc.List(middleware.GetUserID(c), uint(gardenID), c.Query("type"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(data))
}

// Save handles POST /care-logs.
func (h *CareLogHandler) Save(c *gin.Context) {
	var req dto.CareLogSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	item, updated, err := h.svc.Save(middleware.GetUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"updated": updated, "log": item}))
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
