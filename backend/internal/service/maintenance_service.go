package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/dto"
	"github.com/communitygarden/server/internal/model"
	"github.com/communitygarden/server/internal/repository"
	"github.com/communitygarden/server/internal/util"
)

// MaintenanceStatusTransitions 土壤养护单状态机（服务层 + 前端按钮显隐 + 日志模板 + formatters 多处定义）。
var MaintenanceStatusTransitions = map[constants.MaintenanceStatus][]constants.MaintenanceStatus{
	constants.MaintenancePending:    {constants.MaintenanceProcessing, constants.MaintenanceCancelled},
	constants.MaintenanceProcessing: {constants.MaintenanceCompleted, constants.MaintenanceCancelled},
	constants.MaintenanceCompleted:  {},
	constants.MaintenanceCancelled:  {},
}

// MaintenanceService 土壤养护单服务。
type MaintenanceService struct {
	maintRepo repository.MaintenanceRepository
	plotRepo  repository.PlotRepository
	plotSvc   *PlotService
	db        *gorm.DB
	logger    *slog.Logger
}

// NewMaintenanceService 构造土壤养护单服务。
func NewMaintenanceService(maintRepo repository.MaintenanceRepository, plotRepo repository.PlotRepository, plotSvc *PlotService, db *gorm.DB, logger *slog.Logger) *MaintenanceService {
	return &MaintenanceService{maintRepo: maintRepo, plotRepo: plotRepo, plotSvc: plotSvc, db: db, logger: logger}
}

// Create 管理员登记养护单（事务：锁定地块、仅已认养地块可登记、同一地块只能有一张未完成养护单、地块置为养护中）。
func (s *MaintenanceService) Create(req *dto.CreateMaintenanceRequest, operatorID uint, operatorName string) (*model.MaintenanceOrder, error) {
	sampledAt, err := dto.ParseDate(req.SampledAt)
	if err != nil || sampledAt == nil {
		return nil, util.NewAppError(constants.CodeValidationFailed, 400, "sampled_at 字段格式必须为 yyyy-MM-dd")
	}
	today := truncateDate(time.Now())
	if sampledAt.After(today) {
		return nil, util.NewAppError(constants.CodeValidationFailed, 400, "sampled_at 采样日期不能晚于今天")
	}

	var created *model.MaintenanceOrder
	err = s.db.Transaction(func(tx *gorm.DB) error {
		plot, err := s.plotRepo.FindByIDForUpdate(tx, req.PlotID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("地块实体 id=%d 不存在", req.PlotID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if plot.AdopterID == nil {
			return util.NewAppError(constants.CodeForbidden, 403, fmt.Sprintf("地块 %s 尚未认养，无法登记土壤养护单", plot.Code))
		}
		active, err := s.maintRepo.FindActiveByPlotForUpdate(tx, req.PlotID)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if active != nil {
			return util.NewAppError(constants.CodeMaintenanceActive, 409, fmt.Sprintf("地块 %s 已存在未完成的土壤养护单 id=%d（状态 %s），同一地块只能有一张未完成养护单", plot.Code, active.ID, util.MaintenanceStatusText(active.Status)))
		}
		order := &model.MaintenanceOrder{
			PlotID:         req.PlotID,
			OperatorID:     operatorID,
			SampledAt:      sampledAt,
			PHValue:        req.PHValue,
			FertilityIssue: req.FertilityIssue,
			Suggestion:     req.Suggestion,
			Status:         string(constants.MaintenancePending),
		}
		if err := s.maintRepo.CreateWithTx(tx, order); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if err := s.plotSvc.MarkMaintaining(tx, req.PlotID); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		created = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogMaintenanceCreated, "order_id", created.ID, "plot_id", created.PlotID, "operator", operatorName, "ph", created.PHValue, "issue", created.FertilityIssue)
	return s.GetByID(created.ID)
}

// Start 开始处理养护单（pending -> processing，地块仍处于养护中）。
func (s *MaintenanceService) Start(id uint, operatorName string) (*model.MaintenanceOrder, error) {
	order, err := s.loadAndCheck(id)
	if err != nil {
		return nil, err
	}
	if err := s.transition(order, constants.MaintenanceProcessing); err != nil {
		return nil, err
	}
	if err := s.maintRepo.Update(order); err != nil {
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	s.logger.Info(constants.LogMaintenanceStarted, "order_id", order.ID, "plot_id", order.PlotID, "operator", operatorName)
	return s.GetByID(order.ID)
}

// Complete 完成养护单（processing -> completed，填写实际措施与完成日期，地块随即恢复可种植）。
func (s *MaintenanceService) Complete(id uint, req *dto.CompleteMaintenanceRequest, operatorName string) (*model.MaintenanceOrder, error) {
	completedAt, err := dto.ParseDate(req.CompletedAt)
	if err != nil || completedAt == nil {
		return nil, util.NewAppError(constants.CodeValidationFailed, 400, "completed_at 字段格式必须为 yyyy-MM-dd")
	}
	today := truncateDate(time.Now())
	if completedAt.After(today) {
		return nil, util.NewAppError(constants.CodeValidationFailed, 400, "completed_at 完成日期不能晚于今天")
	}
	var out *model.MaintenanceOrder
	err = s.db.Transaction(func(tx *gorm.DB) error {
		order, err := s.maintRepo.FindByIDForUpdate(tx, id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("土壤养护单实体 id=%d 不存在", id))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if err := s.transition(order, constants.MaintenanceCompleted); err != nil {
			return err
		}
		if completedAt.Before(truncateDatePtr(order.SampledAt)) {
			return util.NewAppError(constants.CodeValidationFailed, 400, "completed_at 完成日期不能早于采样日期")
		}
		order.ActualMeasures = req.ActualMeasures
		order.CompletedAt = completedAt
		if err := s.maintRepo.UpdateWithTx(tx, order); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if err := s.plotSvc.RestoreAdopted(tx, order.PlotID); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		out = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogMaintenanceCompleted, "order_id", out.ID, "plot_id", out.PlotID, "operator", operatorName, "completed_at", req.CompletedAt)
	return s.GetByID(out.ID)
}

// Cancel 取消养护单（未完成 -> cancelled，必须留下取消原因，地块恢复可种植）。
func (s *MaintenanceService) Cancel(id uint, req *dto.CancelMaintenanceRequest, operatorName string) (*model.MaintenanceOrder, error) {
	var out *model.MaintenanceOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		order, err := s.maintRepo.FindByIDForUpdate(tx, id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("土壤养护单实体 id=%d 不存在", id))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if err := s.transition(order, constants.MaintenanceCancelled); err != nil {
			return err
		}
		now := time.Now()
		order.CancelReason = req.CancelReason
		order.CancelledAt = &now
		if err := s.maintRepo.UpdateWithTx(tx, order); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if err := s.plotSvc.RestoreAdopted(tx, order.PlotID); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		out = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogMaintenanceCancelled, "order_id", out.ID, "plot_id", out.PlotID, "operator", operatorName, "reason", out.CancelReason)
	return s.GetByID(out.ID)
}

// List 分页查询养护单（可按地块、状态过滤）。
func (s *MaintenanceService) List(pq util.PageQuery, plotID uint, status string) ([]model.MaintenanceOrder, int64, error) {
	orders, total, err := s.maintRepo.List(pq, plotID, status)
	if err != nil {
		return nil, 0, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	return orders, total, nil
}

// ListForUser 养护单分页（管理员可看全部；认养人仅看本人认养地块的养护单进度，需指定地块）。
func (s *MaintenanceService) ListForUser(pq util.PageQuery, plotID uint, status string, userID uint, role string) ([]model.MaintenanceOrder, int64, error) {
	if role == string(constants.RoleAdmin) {
		return s.List(pq, plotID, status)
	}
	if plotID == 0 {
		return []model.MaintenanceOrder{}, 0, nil
	}
	plot, err := s.plotRepo.FindByID(plotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, 0, util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("地块实体 id=%d 不存在", plotID))
		}
		return nil, 0, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	if plot.AdopterID == nil || *plot.AdopterID != userID {
		return nil, 0, util.NewAppError(constants.CodeForbidden, 403, fmt.Sprintf("角色 %s 无权查看地块 %s 的土壤养护单", util.RoleText(role), plot.Code))
	}
	return s.List(pq, plotID, status)
}

// GetForUser 养护单详情（认养人仅可查看本人认养地块的养护单）。
func (s *MaintenanceService) GetForUser(id, userID uint, role string) (*model.MaintenanceOrder, error) {
	order, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if role == string(constants.RoleAdmin) {
		return order, nil
	}
	if order.Plot == nil || order.Plot.AdopterID == nil || *order.Plot.AdopterID != userID {
		return nil, util.NewAppError(constants.CodeForbidden, 403, fmt.Sprintf("角色 %s 无权查看该土壤养护单 id=%d", util.RoleText(role), id))
	}
	return order, nil
}

// HistoryByPlotForUser 按地块展示养护历史（认养人仅限本人地块，管理员不限）。
func (s *MaintenanceService) HistoryByPlotForUser(plotID, userID uint, role string) ([]model.MaintenanceOrder, error) {
	plot, err := s.plotRepo.FindByID(plotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("地块实体 id=%d 不存在", plotID))
		}
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	if role != string(constants.RoleAdmin) && (plot.AdopterID == nil || *plot.AdopterID != userID) {
		return nil, util.NewAppError(constants.CodeForbidden, 403, fmt.Sprintf("角色 %s 无权查看地块 %s 的养护历史", util.RoleText(role), plot.Code))
	}
	return s.ListByPlot(plotID)
}

// ListByPlot 按地块查询养护历史（地块历史接口与创建计划校验复用同一仓储方法族）。
func (s *MaintenanceService) ListByPlot(plotID uint) ([]model.MaintenanceOrder, error) {
	orders, err := s.maintRepo.ListByPlot(plotID)
	if err != nil {
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	return orders, nil
}

// GetByID 查询养护单详情。
func (s *MaintenanceService) GetByID(id uint) (*model.MaintenanceOrder, error) {
	order, err := s.maintRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("土壤养护单实体 id=%d 不存在", id))
		}
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	return order, nil
}

// CountByStatus 养护单状态统计（仪表盘复用）。
func (s *MaintenanceService) CountByStatus() (map[string]int64, error) {
	return s.maintRepo.CountByStatus()
}

// loadAndCheck 查询养护单并校验存在性（Start 使用）。
func (s *MaintenanceService) loadAndCheck(id uint) (*model.MaintenanceOrder, error) {
	order, err := s.maintRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("土壤养护单实体 id=%d 不存在", id))
		}
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	return order, nil
}

// transition 养护单状态机校验。
func (s *MaintenanceService) transition(order *model.MaintenanceOrder, next constants.MaintenanceStatus) error {
	cur := constants.MaintenanceStatus(order.Status)
	allowed := MaintenanceStatusTransitions[cur]
	if !containsMaintenanceStatus(allowed, next) {
		return util.NewAppError(constants.CodeMaintenanceState, 409, fmt.Sprintf("养护单状态不允许从 %s 流转到 %s", util.MaintenanceStatusText(string(cur)), util.MaintenanceStatusText(string(next))))
	}
	order.Status = string(next)
	return nil
}

func truncateDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func truncateDatePtr(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return truncateDate(*t)
}

func containsMaintenanceStatus(list []constants.MaintenanceStatus, v constants.MaintenanceStatus) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
