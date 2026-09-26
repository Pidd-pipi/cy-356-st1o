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

// SoilCareHandler 土壤养护单接口。
type SoilCareHandler struct {
	careService *service.SoilCareService
	plotService *service.PlotService
	audit       middleware.AuditWriter
}

// NewSoilCareHandler 构造土壤养护单接口。
func NewSoilCareHandler(careService *service.SoilCareService, plotService *service.PlotService, audit middleware.AuditWriter) *SoilCareHandler {
	return &SoilCareHandler{careService: careService, plotService: plotService, audit: audit}
}

// Create 管理员登记养护单。
func (h *SoilCareHandler) Create(c *gin.Context) {
	var req dto.CreateCareOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidationFailed, constants.ErrorText[constants.CodeValidationFailed]+": "+err.Error())
		return
	}
	claims, _ := util.GetClaims(c)
	order, err := h.careService.Create(&req, claims.UserID, claims.Username)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "CREATE_SOIL_CARE", "soil-care-order", strconv.FormatUint(uint64(order.ID), 10),
		"登记土壤养护单，地块 id="+strconv.FormatUint(uint64(req.PlotID), 10), c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToCareOrderOutDTO(order))
}

// Start 开始处理养护单。
func (h *SoilCareHandler) Start(c *gin.Context) {
	id, ok := parseCareID(c)
	if !ok {
		return
	}
	claims, _ := util.GetClaims(c)
	order, err := h.careService.Start(id, claims.Username)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "START_SOIL_CARE", "soil-care-order", strconv.FormatUint(uint64(id), 10),
		"开始处理土壤养护单", c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToCareOrderOutDTO(order))
}

// Complete 完成养护单（填写实际措施与完成日期）。
func (h *SoilCareHandler) Complete(c *gin.Context) {
	id, ok := parseCareID(c)
	if !ok {
		return
	}
	var req dto.CompleteCareOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidationFailed, constants.ErrorText[constants.CodeValidationFailed]+": "+err.Error())
		return
	}
	claims, _ := util.GetClaims(c)
	order, err := h.careService.Complete(id, &req, claims.Username)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "COMPLETE_SOIL_CARE", "soil-care-order", strconv.FormatUint(uint64(id), 10),
		"完成土壤养护："+req.ActualMeasures, c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToCareOrderOutDTO(order))
}

// Cancel 取消养护单（必填原因）。
func (h *SoilCareHandler) Cancel(c *gin.Context) {
	id, ok := parseCareID(c)
	if !ok {
		return
	}
	var req dto.CancelCareOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidationFailed, constants.ErrorText[constants.CodeValidationFailed]+": "+err.Error())
		return
	}
	claims, _ := util.GetClaims(c)
	order, err := h.careService.Cancel(id, &req, claims.Username)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "CANCEL_SOIL_CARE", "soil-care-order", strconv.FormatUint(uint64(id), 10),
		"取消土壤养护，原因："+req.CancelReason, c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToCareOrderOutDTO(order))
}

// List 养护单历史列表（可按地块、状态过滤；认养人仅见本人地块）。
func (h *SoilCareHandler) List(c *gin.Context) {
	pq := util.ParsePageQuery(c)
	claims, _ := util.GetClaims(c)
	plotID, _ := strconv.ParseUint(c.Query("plot_id"), 10, 64)
	status := c.Query("status")
	orders, total, err := h.careService.List(pq, uint(plotID), status, claims.Role, claims.UserID)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	list := make([]*dto.CareOrderOutDTO, 0, len(orders))
	for i := range orders {
		list = append(list, dto.ToCareOrderOutDTO(&orders[i]))
	}
	util.OK(c, util.PageResult{List: list, Total: total, Page: pq.Page, PageSize: pq.PageSize})
}

// Get 养护单详情。
func (h *SoilCareHandler) Get(c *gin.Context) {
	id, ok := parseCareID(c)
	if !ok {
		return
	}
	claims, _ := util.GetClaims(c)
	order, err := h.careService.GetByID(id)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	// 认养人只能查看自己地块的养护单（FindByID 已预加载 Plot）。
	if claims.Role != string(constants.RoleAdmin) {
		if order.Plot == nil || order.Plot.AdopterID == nil || *order.Plot.AdopterID != claims.UserID {
			util.Fail(c, http.StatusForbidden, constants.CodeForbidden, "无权查看非本人认养地块的养护单")
			return
		}
	}
	util.OK(c, dto.ToCareOrderOutDTO(order))
}

// AdoptedPlots 可登记养护单的已认养地块下拉数据（复用 PlotService）。
func (h *SoilCareHandler) AdoptedPlots(c *gin.Context) {
	plots, err := h.plotService.ListAdoptedPlots()
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	list := make([]*dto.PlotOutDTO, 0, len(plots))
	for i := range plots {
		list = append(list, dto.ToPlotOutDTO(&plots[i]))
	}
	util.OK(c, list)
}

func parseCareID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return 0, false
	}
	return uint(id), true
}
