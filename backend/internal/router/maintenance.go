package router

import (
	"github.com/gin-gonic/gin"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/middleware"
)

// registerMaintenance 土壤养护单路由。
func (r *Router) registerMaintenance(g *gin.RouterGroup) {
	// 按地块查看养护历史（登录即可，service 内校验认养关系）
	plotHistory := g.Group("/plots")
	plotHistory.Use(middleware.Auth(r.cfg, r.logger))
	{
		plotHistory.GET("/:id/maintenances", r.maintenanceHandler.HistoryByPlot)
	}

	orders := g.Group("/maintenance-orders")
	orders.Use(middleware.Auth(r.cfg, r.logger))
	{
		orders.GET("", r.maintenanceHandler.List)
		orders.GET("/:id", r.maintenanceHandler.Get)
	}

	// 登记 / 开始 / 完成 / 取消均为管理员操作
	admin := orders.Group("")
	admin.Use(middleware.RequireRoles(string(constants.RoleAdmin)))
	{
		admin.POST("", r.maintenanceHandler.Create)
		admin.POST("/:id/start", r.maintenanceHandler.Start)
		admin.POST("/:id/complete", r.maintenanceHandler.Complete)
		admin.POST("/:id/cancel", r.maintenanceHandler.Cancel)
	}
}
