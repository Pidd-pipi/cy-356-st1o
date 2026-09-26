package model

import "time"

// SoilCareOrder 土壤养护单实体（状态机：pending -> in_progress -> completed/cancelled）。
// 同一地块同一时刻只允许存在一张未完成（pending/in_progress）的养护单；
// 该约束由 service 层在事务内对地块加 SELECT ... FOR UPDATE 行锁后校验。
// 养护期间地块进入 caring 状态，禁止创建新的种植计划。
type SoilCareOrder struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	PlotID          uint       `gorm:"index;not null" json:"plot_id"`
	Plot            *Plot      `gorm:"foreignKey:PlotID" json:"plot"`
	AdminID         uint       `gorm:"index;not null" json:"admin_id"`
	Admin           *User      `gorm:"foreignKey:AdminID" json:"admin"`
	Status          string     `gorm:"size:32;not null;default:pending;index" json:"status"`
	SampledDate     time.Time  `gorm:"not null" json:"sampled_date"`
	PHValue         float64    `gorm:"not null" json:"ph_value"`
	FertilityIssue  string     `gorm:"size:512;not null" json:"fertility_issue"`
	TreatmentAdvice string     `gorm:"size:512;not null" json:"treatment_advice"`
	ActualMeasures  string     `gorm:"size:512" json:"actual_measures"`
	CompletedDate   *time.Time `json:"completed_date"`
	CancelReason    string     `gorm:"size:512" json:"cancel_reason"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
