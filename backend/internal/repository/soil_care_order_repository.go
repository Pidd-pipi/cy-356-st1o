package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/communitygarden/server/internal/model"
	"github.com/communitygarden/server/internal/util"
)

// SoilCareOrderRepository 土壤养护单仓储接口。
type SoilCareOrderRepository interface {
	CreateWithTx(tx *gorm.DB, o *model.SoilCareOrder) error
	UpdateWithTx(tx *gorm.DB, o *model.SoilCareOrder) error
	FindByID(id uint) (*model.SoilCareOrder, error)
	FindByIDForUpdate(tx *gorm.DB, id uint) (*model.SoilCareOrder, error)
	FindOpenByPlotForUpdate(tx *gorm.DB, plotID uint) (*model.SoilCareOrder, error)
	List(pq util.PageQuery, plotID uint, status string, adopterID uint) ([]model.SoilCareOrder, int64, error)
	CountByStatus() (map[string]int64, error)
	CountOpen() (int64, error)
}

type soilCareOrderRepository struct {
	db *gorm.DB
}

// NewSoilCareOrderRepository 构造土壤养护单仓储。
func NewSoilCareOrderRepository(db *gorm.DB) SoilCareOrderRepository {
	return &soilCareOrderRepository{db: db}
}

// CreateWithTx 在指定事务内创建养护单。
func (r *soilCareOrderRepository) CreateWithTx(tx *gorm.DB, o *model.SoilCareOrder) error {
	return tx.Create(o).Error
}

// UpdateWithTx 在指定事务内更新养护单。
func (r *soilCareOrderRepository) UpdateWithTx(tx *gorm.DB, o *model.SoilCareOrder) error {
	return tx.Save(o).Error
}

func (r *soilCareOrderRepository) FindByID(id uint) (*model.SoilCareOrder, error) {
	var o model.SoilCareOrder
	if err := r.db.Preload("Plot").Preload("Admin").First(&o, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

// FindByIDForUpdate 事务内对养护单加 SELECT ... FOR UPDATE 行锁。
func (r *soilCareOrderRepository) FindByIDForUpdate(tx *gorm.DB, id uint) (*model.SoilCareOrder, error) {
	var o model.SoilCareOrder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

// FindOpenByPlotForUpdate 查询地块的未完成养护单（事务内加行锁），无则返回 ErrNotFound。
func (r *soilCareOrderRepository) FindOpenByPlotForUpdate(tx *gorm.DB, plotID uint) (*model.SoilCareOrder, error) {
	var o model.SoilCareOrder
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("plot_id = ? AND status IN ?", plotID, []string{"pending", "in_progress"}).
		First(&o).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

// List 分页查询养护单历史；plotID 过滤地块，status 过滤状态，
// adopterID>0 时仅返回该认养人名下地块的养护单（认养人视角）。
func (r *soilCareOrderRepository) List(pq util.PageQuery, plotID uint, status string, adopterID uint) ([]model.SoilCareOrder, int64, error) {
	var orders []model.SoilCareOrder
	var total int64
	q := r.db.Model(&model.SoilCareOrder{}).Preload("Plot").Preload("Admin")
	if plotID > 0 {
		q = q.Where("plot_id = ?", plotID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if adopterID > 0 {
		q = q.Joins("JOIN plots ON plots.id = soil_care_orders.plot_id").
			Where("plots.adopter_id = ?", adopterID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := util.Paginate(q.Order("soil_care_orders.id DESC"), pq).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func (r *soilCareOrderRepository) CountByStatus() (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	if err := r.db.Model(&model.SoilCareOrder{}).Select("status, count(*) as count").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, v := range rows {
		out[v.Status] = v.Count
	}
	return out, nil
}

// CountOpen 未完成养护单数量（仪表盘/统计复用）。
func (r *soilCareOrderRepository) CountOpen() (int64, error) {
	var total int64
	err := r.db.Model(&model.SoilCareOrder{}).
		Where("status IN ?", []string{"pending", "in_progress"}).
		Count(&total).Error
	return total, err
}
