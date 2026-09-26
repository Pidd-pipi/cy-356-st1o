package service

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/dto"
	"github.com/communitygarden/server/internal/repository"
)

func newCareOrderService(t *testing.T, db *gorm.DB) (*SoilCareService, *PlantingPlanService) {
	t.Helper()
	plotRepo := repository.NewPlotRepository(db)
	careRepo := repository.NewSoilCareOrderRepository(db)
	planRepo := repository.NewPlantingPlanRepository(db)
	plotSvc := NewPlotService(plotRepo, db, testLogger())
	careSvc := NewSoilCareService(careRepo, plotRepo, plotSvc, db, testLogger())
	planSvc := NewPlantingPlanService(planRepo, plotRepo, plotSvc, db, testLogger())
	return careSvc, planSvc
}

func sampleCreateCareReq(plotID uint) *dto.CreateCareOrderRequest {
	return &dto.CreateCareOrderRequest{
		PlotID:          plotID,
		SampledDate:     time.Now().Format("2006-01-02"),
		PHValue:         5.8,
		FertilityIssue:  "土壤板结、有机质偏低",
		TreatmentAdvice: "深翻并增施腐熟堆肥",
	}
}

func TestSoilCareService_CreateLifecycle(t *testing.T) {
	db := newTestServiceDB(t)
	careSvc, _ := newCareOrderService(t, db)
	admin := newTestUser(t, db, "admin", "admin")
	ownerID := newTestUser(t, db, "owner", "citizen").ID
	plot := newTestPlot(t, db, "P-CARE", string(constants.PlotStatusAdopted), &ownerID)

	// 登记成功，地块进入 caring
	order, err := careSvc.Create(sampleCreateCareReq(plot.ID), admin.ID, "admin")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if order.Status != string(constants.CareStatusPending) {
		t.Fatalf("new order status=%s, want pending", order.Status)
	}
	gotPlot, _ := careSvc.plotSvc.GetByID(plot.ID)
	if gotPlot.Status != string(constants.PlotStatusCaring) {
		t.Fatalf("plot status=%s, want caring", gotPlot.Status)
	}

	// 同一地块不能重复登记未完成养护单
	if _, err := careSvc.Create(sampleCreateCareReq(plot.ID), admin.ID, "admin"); err == nil {
		t.Fatalf("expected conflict for duplicate open care order")
	}

	// 开始处理
	started, err := careSvc.Start(order.ID, "admin")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if started.Status != string(constants.CareStatusInProgress) {
		t.Fatalf("started status=%s, want in_progress", started.Status)
	}

	// 完成养护，地块恢复 adopted
	completed, err := careSvc.Complete(order.ID, &dto.CompleteCareOrderRequest{
		ActualMeasures: "深翻 20cm，施入堆肥 5kg",
		CompletedDate:  time.Now().Format("2006-01-02"),
	}, "admin")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if completed.Status != string(constants.CareStatusCompleted) || completed.ActualMeasures == "" || completed.CompletedDate == nil {
		t.Fatalf("completed order invalid: %+v", completed)
	}
	gotPlot, _ = careSvc.plotSvc.GetByID(plot.ID)
	if gotPlot.Status != string(constants.PlotStatusAdopted) {
		t.Fatalf("plot status=%s after complete, want adopted", gotPlot.Status)
	}

	// 已完成后可重新登记新单
	if _, err := careSvc.Create(sampleCreateCareReq(plot.ID), admin.ID, "admin"); err != nil {
		t.Fatalf("re-create after complete should be allowed: %v", err)
	}
}

func TestSoilCareService_CancelRequiresReasonAndRestoresPlot(t *testing.T) {
	db := newTestServiceDB(t)
	careSvc, _ := newCareOrderService(t, db)
	admin := newTestUser(t, db, "admin2", "admin")
	ownerID := newTestUser(t, db, "owner2", "citizen").ID
	plot := newTestPlot(t, db, "P-CANCEL", string(constants.PlotStatusAdopted), &ownerID)

	order, err := careSvc.Create(sampleCreateCareReq(plot.ID), admin.ID, "admin")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// 取消必须填写原因
	if _, err := careSvc.Cancel(order.ID, &dto.CancelCareOrderRequest{CancelReason: ""}, "admin"); err == nil {
		t.Fatalf("expected error when cancel reason empty")
	}
	cancelled, err := careSvc.Cancel(order.ID, &dto.CancelCareOrderRequest{CancelReason: "连续降雨，采样异常，择期重新检测"}, "admin")
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if cancelled.Status != string(constants.CareStatusCancelled) || cancelled.CancelReason == "" {
		t.Fatalf("cancelled order invalid: %+v", cancelled)
	}
	gotPlot, _ := careSvc.plotSvc.GetByID(plot.ID)
	if gotPlot.Status != string(constants.PlotStatusAdopted) {
		t.Fatalf("plot status=%s after cancel, want adopted", gotPlot.Status)
	}
}

func TestSoilCareService_StateTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    constants.CareStatus
		to      constants.CareStatus
		allowed bool
	}{
		{"pending to in_progress", constants.CareStatusPending, constants.CareStatusInProgress, true},
		{"pending to cancelled", constants.CareStatusPending, constants.CareStatusCancelled, true},
		{"pending to completed", constants.CareStatusPending, constants.CareStatusCompleted, false},
		{"in_progress to completed", constants.CareStatusInProgress, constants.CareStatusCompleted, true},
		{"in_progress to cancelled", constants.CareStatusInProgress, constants.CareStatusCancelled, true},
		{"in_progress to pending", constants.CareStatusInProgress, constants.CareStatusPending, false},
		{"completed to in_progress", constants.CareStatusCompleted, constants.CareStatusInProgress, false},
		{"cancelled to pending", constants.CareStatusCancelled, constants.CareStatusPending, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := containsCareStatus(CareStatusTransitions[tt.from], tt.to)
			if allowed != tt.allowed {
				t.Fatalf("transition %s->%s allowed=%v, want %v", tt.from, tt.to, allowed, tt.allowed)
			}
		})
	}
}

func TestSoilCareService_CreateRejectsNonAdoptedPlot(t *testing.T) {
	db := newTestServiceDB(t)
	careSvc, _ := newCareOrderService(t, db)
	admin := newTestUser(t, db, "admin3", "admin")
	plot := newTestPlot(t, db, "P-FREE", string(constants.PlotStatusAvailable), nil)

	if _, err := careSvc.Create(sampleCreateCareReq(plot.ID), admin.ID, "admin"); err == nil {
		t.Fatalf("expected error creating care order on available plot")
	}
}

func TestSoilCareService_BlocksPlanCreationWhileCaring(t *testing.T) {
	db := newTestServiceDB(t)
	careSvc, planSvc := newCareOrderService(t, db)
	admin := newTestUser(t, db, "admin4", "admin")
	owner := newTestUser(t, db, "owner4", "citizen")
	ownerID := owner.ID
	plot := newTestPlot(t, db, "P-BLOCK", string(constants.PlotStatusAdopted), &ownerID)

	if _, err := careSvc.Create(sampleCreateCareReq(plot.ID), admin.ID, "admin"); err != nil {
		t.Fatalf("Create care: %v", err)
	}
	planReq := &dto.CreatePlanRequest{
		PlotID:   plot.ID,
		CropName: "番茄",
		CropType: string(constants.CropVegetable),
		Season:   string(constants.SeasonSummer),
	}
	if _, err := planSvc.Create(planReq, owner.ID); err == nil {
		t.Fatalf("expected plan creation blocked while plot under caring")
	}
}
