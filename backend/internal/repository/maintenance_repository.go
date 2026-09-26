package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/communitygarden/server/internal/model"
	"github.com/communitygarden/server/internal/util"
)

// MaintenanceRepository 土壤养护单仓储接口。
type MaintenanceRepository interface {
	CreateWithTx(tx *gorm.DB, m *model.MaintenanceOrder) error
	Update(m *model.MaintenanceOrder) error
	UpdateWithTx(tx *gorm.DB, m *model.MaintenanceOrder) error
	FindByID(id uint) (*model.MaintenanceOrder, error)
	FindByIDForUpdate(tx *gorm.DB, id uint) (*model.MaintenanceOrder, error)
	FindActiveByPlotForUpdate(tx *gorm.DB, plotID uint) (*model.MaintenanceOrder, error)
	List(pq util.PageQuery, plotID uint, status string) ([]model.MaintenanceOrder, int64, error)
	ListByPlot(plotID uint) ([]model.MaintenanceOrder, error)
	CountByStatus() (map[string]int64, error)
}

type maintenanceRepository struct {
	db *gorm.DB
}

// NewMaintenanceRepository 构造土壤养护单仓储。
func NewMaintenanceRepository(db *gorm.DB) MaintenanceRepository {
	return &maintenanceRepository{db: db}
}

// CreateWithTx 在指定事务内创建养护单。
func (r *maintenanceRepository) CreateWithTx(tx *gorm.DB, m *model.MaintenanceOrder) error {
	return tx.Create(m).Error
}

func (r *maintenanceRepository) Update(m *model.MaintenanceOrder) error {
	return r.db.Save(m).Error
}

// UpdateWithTx 在指定事务内更新养护单。
func (r *maintenanceRepository) UpdateWithTx(tx *gorm.DB, m *model.MaintenanceOrder) error {
	return tx.Save(m).Error
}

func (r *maintenanceRepository) FindByID(id uint) (*model.MaintenanceOrder, error) {
	var m model.MaintenanceOrder
	if err := r.db.Preload("Plot").Preload("Plot.Adopter").Preload("Operator").First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// FindByIDForUpdate 事务内行锁查询养护单。
func (r *maintenanceRepository) FindByIDForUpdate(tx *gorm.DB, id uint) (*model.MaintenanceOrder, error) {
	var m model.MaintenanceOrder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// FindActiveByPlotForUpdate 查询某地块未完成（pending/processing）的养护单（事务行锁）。
func (r *maintenanceRepository) FindActiveByPlotForUpdate(tx *gorm.DB, plotID uint) (*model.MaintenanceOrder, error) {
	var m model.MaintenanceOrder
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("plot_id = ? AND status IN ?", plotID, []string{"pending", "processing"}).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// List 分页查询养护单（可按地块、状态过滤）。
func (r *maintenanceRepository) List(pq util.PageQuery, plotID uint, status string) ([]model.MaintenanceOrder, int64, error) {
	var orders []model.MaintenanceOrder
	var total int64
	q := r.db.Model(&model.MaintenanceOrder{}).Preload("Plot").Preload("Plot.Adopter").Preload("Operator")
	if plotID > 0 {
		q = q.Where("plot_id = ?", plotID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := util.Paginate(q.Order("id DESC"), pq).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// ListByPlot 按地块查询全部养护历史（养护历史页复用，不分页）。
func (r *maintenanceRepository) ListByPlot(plotID uint) ([]model.MaintenanceOrder, error) {
	var orders []model.MaintenanceOrder
	err := r.db.Preload("Plot").Preload("Operator").
		Where("plot_id = ?", plotID).
		Order("id DESC").
		Find(&orders).Error
	return orders, err
}

func (r *maintenanceRepository) CountByStatus() (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	if err := r.db.Model(&model.MaintenanceOrder{}).Select("status, count(*) as count").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, v := range rows {
		out[v.Status] = v.Count
	}
	return out, nil
}
