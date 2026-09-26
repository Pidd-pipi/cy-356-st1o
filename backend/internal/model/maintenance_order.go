package model

import "time"

// MaintenanceOrder 土壤养护单实体（状态机：pending -> processing -> completed，pending/processing -> cancelled）。
// 同一地块只允许存在一张未完成（pending/processing）的养护单；养护期间地块置为 maintaining，禁止创建种植计划。
type MaintenanceOrder struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	PlotID         uint       `gorm:"index;not null" json:"plot_id"`
	Plot           *Plot      `gorm:"foreignKey:PlotID" json:"plot"`
	OperatorID     uint       `gorm:"index;not null" json:"operator_id"`
	Operator       *User      `gorm:"foreignKey:OperatorID" json:"operator"`
	SampledAt      *time.Time `json:"sampled_at"`
	PHValue        float64    `gorm:"not null" json:"ph_value"`
	FertilityIssue string     `gorm:"size:32;not null" json:"fertility_issue"`
	Suggestion     string     `gorm:"size:512;not null" json:"suggestion"`
	Status         string     `gorm:"size:32;not null;default:pending;index" json:"status"`
	ActualMeasures string     `gorm:"size:512" json:"actual_measures"`
	CompletedAt    *time.Time `json:"completed_at"`
	CancelReason   string     `gorm:"size:512" json:"cancel_reason"`
	CancelledAt    *time.Time `json:"cancelled_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
