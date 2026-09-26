package dto

import (
	"github.com/communitygarden/server/internal/model"
)

// CreateMaintenanceRequest 管理员登记土壤养护单。
type CreateMaintenanceRequest struct {
	PlotID         uint    `json:"plot_id" binding:"required,gt=0"`
	SampledAt      string  `json:"sampled_at" binding:"required"`
	PHValue        float64 `json:"ph_value" binding:"required,min=0,max=14"`
	FertilityIssue string  `json:"fertility_issue" binding:"required,oneof=acidic alkaline nutrient_low salinized organic_low drainage_poor healthy"`
	Suggestion     string  `json:"suggestion" binding:"required,max=512"`
}

// CompleteMaintenanceRequest 管理员完成养护单（填写实际措施与完成日期）。
type CompleteMaintenanceRequest struct {
	ActualMeasures string `json:"actual_measures" binding:"required,max=512"`
	CompletedAt    string `json:"completed_at" binding:"required"`
}

// CancelMaintenanceRequest 管理员取消养护单（必须留下原因）。
type CancelMaintenanceRequest struct {
	CancelReason string `json:"cancel_reason" binding:"required,max=512"`
}

// MaintenanceOutDTO 土壤养护单输出。
type MaintenanceOutDTO struct {
	ID             uint    `json:"id"`
	PlotID         uint    `json:"plot_id"`
	PlotCode       string  `json:"plot_code"`
	PlotName       string  `json:"plot_name"`
	OperatorID     uint    `json:"operator_id"`
	OperatorName   string  `json:"operator_name"`
	SampledAt      *string `json:"sampled_at"`
	PHValue        float64 `json:"ph_value"`
	FertilityIssue string  `json:"fertility_issue"`
	Suggestion     string  `json:"suggestion"`
	Status         string  `json:"status"`
	ActualMeasures string  `json:"actual_measures"`
	CompletedAt    *string `json:"completed_at"`
	CancelReason   string  `json:"cancel_reason"`
	CancelledAt    *string `json:"cancelled_at"`
	CreatedAt      string  `json:"created_at"`
}

// ToMaintenanceOutDTO 模型转 DTO。
func ToMaintenanceOutDTO(m *model.MaintenanceOrder) *MaintenanceOutDTO {
	dto := &MaintenanceOutDTO{
		ID:             m.ID,
		PlotID:         m.PlotID,
		OperatorID:     m.OperatorID,
		PHValue:        m.PHValue,
		FertilityIssue: m.FertilityIssue,
		Suggestion:     m.Suggestion,
		Status:         m.Status,
		ActualMeasures: m.ActualMeasures,
		CancelReason:   m.CancelReason,
		CreatedAt:      m.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if m.Plot != nil {
		dto.PlotCode = m.Plot.Code
		dto.PlotName = m.Plot.Name
	}
	if m.Operator != nil {
		dto.OperatorName = m.Operator.Nickname
		if dto.OperatorName == "" {
			dto.OperatorName = m.Operator.Username
		}
	}
	if m.SampledAt != nil {
		s := m.SampledAt.Format("2006-01-02")
		dto.SampledAt = &s
	}
	if m.CompletedAt != nil {
		s := m.CompletedAt.Format("2006-01-02")
		dto.CompletedAt = &s
	}
	if m.CancelledAt != nil {
		s := m.CancelledAt.Format("2006-01-02")
		dto.CancelledAt = &s
	}
	return dto
}
