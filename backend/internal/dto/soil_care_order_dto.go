package dto

import (
	"time"

	"github.com/communitygarden/server/internal/model"
)

// CreateCareOrderRequest 管理员登记土壤养护单。
type CreateCareOrderRequest struct {
	PlotID          uint    `json:"plot_id" binding:"required,gt=0"`
	SampledDate     string  `json:"sampled_date" binding:"required"`
	PHValue         float64 `json:"ph_value" binding:"required,gte=0,lte=14"`
	FertilityIssue  string  `json:"fertility_issue" binding:"required,max=512"`
	TreatmentAdvice string  `json:"treatment_advice" binding:"required,max=512"`
}

// StartCareOrderRequest 开始处理养护单（pending -> in_progress）。
type StartCareOrderRequest struct{}

// CompleteCareOrderRequest 完成养护单（管理员填写实际措施与完成日期）。
type CompleteCareOrderRequest struct {
	ActualMeasures string `json:"actual_measures" binding:"required,max=512"`
	CompletedDate  string `json:"completed_date" binding:"omitempty"`
}

// CancelCareOrderRequest 取消养护单（必须填写取消原因）。
type CancelCareOrderRequest struct {
	CancelReason string `json:"cancel_reason" binding:"required,min=1,max=512"`
}

// CareOrderOutDTO 土壤养护单输出（认养人可查看进度）。
type CareOrderOutDTO struct {
	ID              uint    `json:"id"`
	PlotID          uint    `json:"plot_id"`
	PlotCode        string  `json:"plot_code"`
	PlotName        string  `json:"plot_name"`
	AdminID         uint    `json:"admin_id"`
	AdminName       string  `json:"admin_name"`
	Status          string  `json:"status"`
	SampledDate     string  `json:"sampled_date"`
	PHValue         float64 `json:"ph_value"`
	FertilityIssue  string  `json:"fertility_issue"`
	TreatmentAdvice string  `json:"treatment_advice"`
	ActualMeasures  string  `json:"actual_measures"`
	CompletedDate   *string `json:"completed_date"`
	CancelReason    string  `json:"cancel_reason"`
	CreatedAt       string  `json:"created_at"`
}

// ToCareOrderOutDTO 模型转 DTO。
func ToCareOrderOutDTO(o *model.SoilCareOrder) *CareOrderOutDTO {
	out := &CareOrderOutDTO{
		ID:              o.ID,
		PlotID:          o.PlotID,
		AdminID:         o.AdminID,
		Status:          o.Status,
		SampledDate:     o.SampledDate.Format("2006-01-02"),
		PHValue:         o.PHValue,
		FertilityIssue:  o.FertilityIssue,
		TreatmentAdvice: o.TreatmentAdvice,
		ActualMeasures:  o.ActualMeasures,
		CancelReason:    o.CancelReason,
		CreatedAt:       o.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if o.Plot != nil {
		out.PlotCode = o.Plot.Code
		out.PlotName = o.Plot.Name
	}
	if o.Admin != nil {
		out.AdminName = o.Admin.Nickname
		if out.AdminName == "" {
			out.AdminName = o.Admin.Username
		}
	}
	if o.CompletedDate != nil {
		s := o.CompletedDate.Format("2006-01-02")
		out.CompletedDate = &s
	}
	return out
}

// ParseCareDate 解析 yyyy-MM-dd 日期；空串返回当前时间。
func ParseCareDate(s string) (time.Time, error) {
	t, err := ParseDate(s)
	if err != nil {
		return time.Time{}, err
	}
	if t == nil {
		return time.Now(), nil
	}
	return *t, nil
}
