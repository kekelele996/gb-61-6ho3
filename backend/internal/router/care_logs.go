package router

import (
	"github.com/gin-gonic/gin"

	"github.com/gbplantwiki/gbplantwiki/internal/config"
	"github.com/gbplantwiki/gbplantwiki/internal/handler"
	"github.com/gbplantwiki/gbplantwiki/internal/middleware"
)

func registerCareLogRoutes(v1 *gin.RouterGroup, cfg *config.Config, h *handler.CareLogHandler, limiter *middleware.RateLimiter) {
	logs := v1.Group("/care-logs", middleware.AuthRequired(cfg))
	logs.GET("", h.List)
	logs.POST("", limiter.Limit(), h.Save)
	logs.DELETE("/:id", h.Delete)
}
