package router

import (
	"github.com/gin-gonic/gin"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/middleware"
)

// registerSoilCare 土壤养护单路由。
func (r *Router) registerSoilCare(g *gin.RouterGroup) {
	care := g.Group("/soil-care-orders")
	care.Use(middleware.Auth(r.cfg, r.logger))
	{
		// 管理员与认养人均可查看：认养人看进度（仅本人地块），管理员看全部历史
		care.GET("", r.careHandler.List)
		care.GET("/:id", r.careHandler.Get)

		// 管理员写操作：登记 / 开始 / 完成 / 取消
		admin := care.Group("")
		admin.Use(middleware.RequireRoles(string(constants.RoleAdmin)))
		{
			admin.POST("", r.careHandler.Create)
			admin.POST("/:id/start", r.careHandler.Start)
			admin.POST("/:id/complete", r.careHandler.Complete)
			admin.POST("/:id/cancel", r.careHandler.Cancel)
		}
	}

	// 可登记养护单的已认养地块（管理员，复用 PlotService）
	plots := g.Group("/care/plots")
	plots.Use(middleware.Auth(r.cfg, r.logger), middleware.RequireRoles(string(constants.RoleAdmin)))
	{
		plots.GET("/adopted", r.careHandler.AdoptedPlots)
	}
}
