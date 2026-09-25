package router

import (
	"github.com/gin-gonic/gin"

	"github.com/gbplantwiki/gbplantwiki/internal/config"
	"github.com/gbplantwiki/gbplantwiki/internal/handler"
	"github.com/gbplantwiki/gbplantwiki/internal/middleware"
)

func registerGardenRoutes(v1 *gin.RouterGroup, cfg *config.Config, h *handler.UserGardenHandler, careLogHandler *handler.CareLogHandler, limiter *middleware.RateLimiter) {
	gardens := v1.Group("/gardens", middleware.AuthRequired(cfg))
	gardens.GET("", h.List)
	gardens.POST("", limiter.Limit(), h.Add)
	gardens.PUT("/:id/reminder", h.BindReminder)
	gardens.DELETE("/:id", h.Remove)

	// Per-plant care logs nested under a garden item; ownership of the plant
	// is enforced in the service via the :id path param.
	gardens.GET("/:id/care-logs", careLogHandler.List)
	gardens.POST("/:id/care-logs", limiter.Limit(), careLogHandler.Save)

	careLogs := v1.Group("/care-logs", middleware.AuthRequired(cfg))
	careLogs.PUT("/:id", careLogHandler.Update)
	careLogs.DELETE("/:id", careLogHandler.Delete)
}
