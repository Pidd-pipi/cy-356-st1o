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

// CareStatusTransitions 土壤养护单状态机（与前端按钮显隐 / formatters / 错误码多处对应）。
var CareStatusTransitions = map[constants.CareStatus][]constants.CareStatus{
	constants.CareStatusPending:    {constants.CareStatusInProgress, constants.CareStatusCancelled},
	constants.CareStatusInProgress: {constants.CareStatusCompleted, constants.CareStatusCancelled},
	constants.CareStatusCompleted:  {},
	constants.CareStatusCancelled:  {},
}

// SoilCareService 土壤养护单服务（事务 + 地块行锁，保证同一地块仅一张未完成养护单）。
type SoilCareService struct {
	careRepo repository.SoilCareOrderRepository
	plotRepo repository.PlotRepository
	plotSvc  *PlotService
	db       *gorm.DB
	logger   *slog.Logger
}

// NewSoilCareService 构造土壤养护单服务。
func NewSoilCareService(careRepo repository.SoilCareOrderRepository, plotRepo repository.PlotRepository, plotSvc *PlotService, db *gorm.DB, logger *slog.Logger) *SoilCareService {
	return &SoilCareService{careRepo: careRepo, plotRepo: plotRepo, plotSvc: plotSvc, db: db, logger: logger}
}

// Create 管理员登记养护单（事务：锁定地块 → 校验已认养/无未完成单 → 创建养护单 → 地块进入 caring）。
func (s *SoilCareService) Create(req *dto.CreateCareOrderRequest, adminID uint, adminName string) (*model.SoilCareOrder, error) {
	sampledDate, err := dto.ParseCareDate(req.SampledDate)
	if err != nil {
		return nil, util.NewAppError(constants.CodeValidationFailed, 400, "sampled_date 字段格式必须为 yyyy-MM-dd")
	}

	var created *model.SoilCareOrder
	err = s.db.Transaction(func(tx *gorm.DB) error {
		plot, err := s.plotRepo.FindByIDForUpdate(tx, req.PlotID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("地块实体 id=%d 不存在", req.PlotID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		// 同一地块仅允许一张未完成养护单（行锁内校验，避免并发重复登记）。
		if open, err := s.careRepo.FindOpenByPlotForUpdate(tx, plot.ID); err == nil && open != nil {
			return util.NewAppError(constants.CodeCareOpenOrderExists, 409, fmt.Sprintf("地块 %s 已存在未完成养护单 id=%d（状态 %s），不能重复登记", plot.Code, open.ID, util.CareStatusText(open.Status)))
		} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if plot.Status != string(constants.PlotStatusAdopted) || plot.AdopterID == nil {
			return util.NewAppError(constants.CodePlotNotAdopted, 409, fmt.Sprintf("地块 %s 当前状态为 %s，仅已认养地块可登记养护单", plot.Code, util.PlotStatusText(plot.Status)))
		}

		order := &model.SoilCareOrder{
			PlotID:          plot.ID,
			AdminID:         adminID,
			Status:          string(constants.CareStatusPending),
			SampledDate:     sampledDate,
			PHValue:         req.PHValue,
			FertilityIssue:  req.FertilityIssue,
			TreatmentAdvice: req.TreatmentAdvice,
		}
		if err := s.careRepo.CreateWithTx(tx, order); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if err := s.plotSvc.MarkCaring(tx, plot.ID); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		created = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogCareOrderCreated, "care_id", created.ID, "plot_id", created.PlotID, "admin", adminName, "ph", created.PHValue)
	return s.GetByID(created.ID)
}

// Start 开始处理（pending -> in_progress）。
func (s *SoilCareService) Start(id uint, adminName string) (*model.SoilCareOrder, error) {
	err := s.transition(id, constants.CareStatusInProgress, nil)
	if err != nil {
		return nil, err
	}
	got, _ := s.GetByID(id)
	s.logger.Info(constants.LogCareOrderStarted, "care_id", id, "plot_id", got.PlotID, "admin", adminName)
	return got, nil
}

// Complete 完成养护（in_progress -> completed）：填写实际措施、完成日期，地块恢复可种植。
func (s *SoilCareService) Complete(id uint, req *dto.CompleteCareOrderRequest, adminName string) (*model.SoilCareOrder, error) {
	completedDate, err := dto.ParseCareDate(req.CompletedDate)
	if err != nil {
		return nil, util.NewAppError(constants.CodeValidationFailed, 400, "completed_date 字段格式必须为 yyyy-MM-dd")
	}
	err = s.transition(id, constants.CareStatusCompleted, func(tx *gorm.DB, o *model.SoilCareOrder) error {
		o.ActualMeasures = req.ActualMeasures
		o.CompletedDate = &completedDate
		return nil
	})
	if err != nil {
		return nil, err
	}
	got, _ := s.GetByID(id)
	s.logger.Info(constants.LogCareOrderCompleted, "care_id", id, "plot_id", got.PlotID, "admin", adminName, "completed_date", completedDate.Format("2006-01-02"))
	return got, nil
}

// Cancel 取消养护（pending/in_progress -> cancelled）：必填取消原因，地块恢复可种植。
func (s *SoilCareService) Cancel(id uint, req *dto.CancelCareOrderRequest, adminName string) (*model.SoilCareOrder, error) {
	if req.CancelReason == "" {
		return nil, util.NewAppError(constants.CodeCareCancelReason, 400, "养护单取消原因 cancel_reason 不能为空")
	}
	err := s.transition(id, constants.CareStatusCancelled, func(tx *gorm.DB, o *model.SoilCareOrder) error {
		o.CancelReason = req.CancelReason
		now := time.Now()
		o.CompletedDate = &now
		return nil
	})
	if err != nil {
		return nil, err
	}
	got, _ := s.GetByID(id)
	s.logger.Info(constants.LogCareOrderCancelled, "care_id", id, "plot_id", got.PlotID, "admin", adminName, "reason", req.CancelReason)
	return got, nil
}

// transition 通用状态流转：校验状态机 → 更新养护单 → 终态时在同事务恢复地块。
func (s *SoilCareService) transition(id uint, target constants.CareStatus, mutate func(tx *gorm.DB, o *model.SoilCareOrder) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		order, err := s.careRepo.FindByIDForUpdate(tx, id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("土壤养护单实体 id=%d 不存在", id))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		cur := constants.CareStatus(order.Status)
		if !containsCareStatus(CareStatusTransitions[cur], target) {
			return util.NewAppError(constants.CodeCareStateNotAllowed, 409, fmt.Sprintf("养护单状态不允许从 %s 流转到 %s", util.CareStatusText(string(cur)), util.CareStatusText(string(target))))
		}
		if mutate != nil {
			if err := mutate(tx, order); err != nil {
				return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
			}
		}
		order.Status = string(target)
		if err := s.careRepo.UpdateWithTx(tx, order); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		// 完成或取消均释放地块，使其恢复为已认养可种植状态。
		if target == constants.CareStatusCompleted || target == constants.CareStatusCancelled {
			if err := s.plotSvc.RestoreAdopted(tx, order.PlotID); err != nil {
				return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
			}
		}
		return nil
	})
}

// GetByID 查询养护单详情。
func (s *SoilCareService) GetByID(id uint) (*model.SoilCareOrder, error) {
	order, err := s.careRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("土壤养护单实体 id=%d 不存在", id))
		}
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	return order, nil
}

// List 分页查询养护单（按地块/状态过滤）；adopterID>0 时限定为该认养人名下地块（认养人进度视角）。
func (s *SoilCareService) List(pq util.PageQuery, plotID uint, status, operatorRole string, operatorID uint) ([]model.SoilCareOrder, int64, error) {
	var adopterID uint
	if operatorRole != string(constants.RoleAdmin) {
		adopterID = operatorID
	}
	orders, total, err := s.careRepo.List(pq, plotID, status, adopterID)
	if err != nil {
		return nil, 0, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	return orders, total, nil
}

// CountByStatus 养护单状态统计（仪表盘复用）。
func (s *SoilCareService) CountByStatus() (map[string]int64, error) {
	return s.careRepo.CountByStatus()
}

func containsCareStatus(list []constants.CareStatus, v constants.CareStatus) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
