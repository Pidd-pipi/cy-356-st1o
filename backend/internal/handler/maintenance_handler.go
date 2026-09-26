package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/dto"
	"github.com/communitygarden/server/internal/middleware"
	"github.com/communitygarden/server/internal/service"
	"github.com/communitygarden/server/internal/util"
)

// MaintenanceHandler 土壤养护单接口。
type MaintenanceHandler struct {
	maintService *service.MaintenanceService
	audit        middleware.AuditWriter
}

// NewMaintenanceHandler 构造土壤养护单接口。
func NewMaintenanceHandler(maintService *service.MaintenanceService, audit middleware.AuditWriter) *MaintenanceHandler {
	return &MaintenanceHandler{maintService: maintService, audit: audit}
}

// Create 管理员登记养护单。
func (h *MaintenanceHandler) Create(c *gin.Context) {
	var req dto.CreateMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidationFailed, constants.ErrorText[constants.CodeValidationFailed]+": "+err.Error())
		return
	}
	claims, _ := util.GetClaims(c)
	order, err := h.maintService.Create(&req, claims.UserID, claims.Username)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "CREATE_MAINTENANCE", "maintenance", strconv.FormatUint(uint64(order.ID), 10),
		"登记土壤养护单，地块 id="+strconv.FormatUint(uint64(req.PlotID), 10), c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToMaintenanceOutDTO(order))
}

// Start 管理员开始处理养护单。
func (h *MaintenanceHandler) Start(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	claims, _ := util.GetClaims(c)
	order, err := h.maintService.Start(uint(id), claims.Username)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	util.OK(c, dto.ToMaintenanceOutDTO(order))
}

// Complete 管理员完成养护单。
func (h *MaintenanceHandler) Complete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	var req dto.CompleteMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidationFailed, constants.ErrorText[constants.CodeValidationFailed]+": "+err.Error())
		return
	}
	claims, _ := util.GetClaims(c)
	order, err := h.maintService.Complete(uint(id), &req, claims.Username)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "COMPLETE_MAINTENANCE", "maintenance", strconv.FormatUint(uint64(id), 10),
		"完成土壤养护单："+req.ActualMeasures, c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToMaintenanceOutDTO(order))
}

// Cancel 管理员取消养护单。
func (h *MaintenanceHandler) Cancel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	var req dto.CancelMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidationFailed, constants.ErrorText[constants.CodeValidationFailed]+": "+err.Error())
		return
	}
	claims, _ := util.GetClaims(c)
	order, err := h.maintService.Cancel(uint(id), &req, claims.Username)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "CANCEL_MAINTENANCE", "maintenance", strconv.FormatUint(uint64(id), 10),
		"取消土壤养护单，原因："+req.CancelReason, c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToMaintenanceOutDTO(order))
}

// List 养护单分页列表（管理员可看全部；认养人仅看自己地块的养护单）。
func (h *MaintenanceHandler) List(c *gin.Context) {
	pq := util.ParsePageQuery(c)
	claims, _ := util.GetClaims(c)
	plotID, _ := strconv.ParseUint(c.Query("plot_id"), 10, 64)
	status := c.Query("status")
	orders, total, err := h.maintService.ListForUser(pq, uint(plotID), status, claims.UserID, claims.Role)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	list := make([]*dto.MaintenanceOutDTO, 0, len(orders))
	for i := range orders {
		list = append(list, dto.ToMaintenanceOutDTO(&orders[i]))
	}
	util.OK(c, util.PageResult{List: list, Total: total, Page: pq.Page, PageSize: pq.PageSize})
}

// Get 养护单详情（认养人仅可查看自己地块的养护单进度）。
func (h *MaintenanceHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	claims, _ := util.GetClaims(c)
	order, err := h.maintService.GetForUser(uint(id), claims.UserID, claims.Role)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	util.OK(c, dto.ToMaintenanceOutDTO(order))
}

// HistoryByPlot 按地块展示养护历史。
func (h *MaintenanceHandler) HistoryByPlot(c *gin.Context) {
	plotID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	claims, _ := util.GetClaims(c)
	orders, err := h.maintService.HistoryByPlotForUser(uint(plotID), claims.UserID, claims.Role)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	list := make([]*dto.MaintenanceOutDTO, 0, len(orders))
	for i := range orders {
		list = append(list, dto.ToMaintenanceOutDTO(&orders[i]))
	}
	util.OK(c, list)
}
