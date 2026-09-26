package service

import (
	"testing"
	"time"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/dto"
	"github.com/communitygarden/server/internal/repository"
	"github.com/communitygarden/server/internal/util"
)

func TestMaintenanceStatusTransitions_Table(t *testing.T) {
	tests := []struct {
		from constants.MaintenanceStatus
		to   constants.MaintenanceStatus
		ok   bool
	}{
		{constants.MaintenancePending, constants.MaintenanceProcessing, true},
		{constants.MaintenancePending, constants.MaintenanceCancelled, true},
		{constants.MaintenanceProcessing, constants.MaintenanceCompleted, true},
		{constants.MaintenanceProcessing, constants.MaintenanceCancelled, true},
		{constants.MaintenancePending, constants.MaintenanceCompleted, false},
		{constants.MaintenanceCompleted, constants.MaintenanceProcessing, false},
		{constants.MaintenanceCancelled, constants.MaintenanceProcessing, false},
	}
	for _, tt := range tests {
		allowed := MaintenanceStatusTransitions[tt.from]
		found := false
		for _, s := range allowed {
			if s == tt.to {
				found = true
			}
		}
		if found != tt.ok {
			t.Errorf("transition %s->%s ok=%v, want %v", tt.from, tt.to, found, tt.ok)
		}
	}
}

func TestMaintenanceService_FullLifecycle(t *testing.T) {
	db := newTestServiceDB(t)
	maintRepo := repository.NewMaintenanceRepository(db)
	plotRepo := repository.NewPlotRepository(db)
	planRepo := repository.NewPlantingPlanRepository(db)
	plotSvc, _ := newPlotService(t, db)
	maintSvc := NewMaintenanceService(maintRepo, plotRepo, plotSvc, db, testLogger())
	planSvc := NewPlantingPlanService(planRepo, plotRepo, plotSvc, db, testLogger())

	admin := newTestUser(t, db, "admin", "admin")
	farmer := newTestUser(t, db, "farmer-m", "farmer")
	plot := newTestPlot(t, db, "P-MAINT", "available", nil)
	if _, err := plotSvc.Adopt(plot.ID, farmer.ID, "farmer", "farmer-m"); err != nil {
		t.Fatalf("adopt: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	createReq := &dto.CreateMaintenanceRequest{
		PlotID:         plot.ID,
		SampledAt:      today,
		PHValue:        5.2,
		FertilityIssue: string(constants.FertilityAcidic),
		Suggestion:     "撒生石灰调酸，增施有机肥",
	}

	// 登记养护单：地块变为养护中
	order, err := maintSvc.Create(createReq, admin.ID, "admin")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if order.Status != string(constants.MaintenancePending) {
		t.Fatalf("status=%s, want pending", order.Status)
	}
	plotLocked, err := plotSvc.GetByID(plot.ID)
	if err != nil || plotLocked.Status != string(constants.PlotStatusMaintaining) {
		t.Fatalf("plot status=%s err=%v, want maintaining", plotLocked.Status, err)
	}

	// 同一地块不能重复登记
	if _, err := maintSvc.Create(createReq, admin.ID, "admin"); err == nil {
		t.Fatalf("expected duplicate active maintenance error")
	}

	// 养护期间不能创建种植计划
	planReq := &dto.CreatePlanRequest{PlotID: plot.ID, CropName: "菠菜", CropType: "vegetable", Season: "spring"}
	if _, err := planSvc.Create(planReq, farmer.ID); err == nil {
		t.Fatalf("expected plan blocked during maintenance")
	} else if ae, ok := err.(*util.AppError); !ok || ae.Code != constants.CodePlotUnderMaintenance {
		t.Fatalf("unexpected err: %v", err)
	}

	// pending 不能直接完成
	if _, err := maintSvc.Complete(order.ID, &dto.CompleteMaintenanceRequest{ActualMeasures: "x", CompletedAt: today}, "admin"); err == nil {
		t.Fatalf("expected invalid transition pending->completed")
	}

	// pending -> processing -> completed，地块恢复 adopted
	if _, err := maintSvc.Start(order.ID, "admin"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	completed, err := maintSvc.Complete(order.ID, &dto.CompleteMaintenanceRequest{
		ActualMeasures: "撒生石灰 2kg，增施羊粪并深翻",
		CompletedAt:    today,
	}, "admin")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if completed.Status != string(constants.MaintenanceCompleted) || completed.ActualMeasures == "" || completed.CompletedAt == nil {
		t.Fatalf("completed order invalid: %+v", completed)
	}
	restored, err := plotSvc.GetByID(plot.ID)
	if err != nil || restored.Status != string(constants.PlotStatusAdopted) {
		t.Fatalf("plot restored status=%s err=%v", restored.Status, err)
	}

	// 完成后可再次登记（历史保留）
	order2, err := maintSvc.Create(createReq, admin.ID, "admin")
	if err != nil {
		t.Fatalf("Create second order: %v", err)
	}

	// 取消必须留原因：空原因被 binding 拦截（service 层直接构造也应被状态机外的校验保护）
	if _, err := maintSvc.Cancel(order2.ID, &dto.CancelMaintenanceRequest{CancelReason: "连续雨天无法施工，延期至下季度"}, "admin"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	restored2, _ := plotSvc.GetByID(plot.ID)
	if restored2.Status != string(constants.PlotStatusAdopted) {
		t.Fatalf("plot after cancel status=%s", restored2.Status)
	}

	// 已取消不可再流转
	if _, err := maintSvc.Start(order2.ID, "admin"); err == nil {
		t.Fatalf("expected restart cancelled error")
	}

	// 按地块历史应包含 2 张
	history, err := maintSvc.ListByPlot(plot.ID)
	if err != nil || len(history) != 2 {
		t.Fatalf("history len=%d err=%v, want 2", len(history), err)
	}
}

func TestMaintenanceService_PermissionAndValidation(t *testing.T) {
	db := newTestServiceDB(t)
	maintRepo := repository.NewMaintenanceRepository(db)
	plotRepo := repository.NewPlotRepository(db)
	plotSvc, _ := newPlotService(t, db)
	maintSvc := NewMaintenanceService(maintRepo, plotRepo, plotSvc, db, testLogger())

	admin := newTestUser(t, db, "admin2", "admin")
	owner := newTestUser(t, db, "owner", "farmer")
	other := newTestUser(t, db, "other", "citizen")
	plot := newTestPlot(t, db, "P-MAINT2", "available", nil)
	if _, err := plotSvc.Adopt(plot.ID, owner.ID, "farmer", "owner"); err != nil {
		t.Fatalf("adopt: %v", err)
	}

	// 未来采样日期不允许
	badFuture := &dto.CreateMaintenanceRequest{
		PlotID: plot.ID, SampledAt: time.Now().AddDate(0, 0, 3).Format("2006-01-02"),
		PHValue: 7.0, FertilityIssue: "healthy", Suggestion: "常规轮作",
	}
	if _, err := maintSvc.Create(badFuture, admin.ID, "admin2"); err == nil {
		t.Fatalf("expected future sampled_at error")
	}

	// 错误日期格式
	badFmt := &dto.CreateMaintenanceRequest{
		PlotID: plot.ID, SampledAt: "2026/09/01",
		PHValue: 7.0, FertilityIssue: "healthy", Suggestion: "常规轮作",
	}
	if _, err := maintSvc.Create(badFmt, admin.ID, "admin2"); err == nil {
		t.Fatalf("expected bad date format error")
	}

	req := &dto.CreateMaintenanceRequest{
		PlotID: plot.ID, SampledAt: time.Now().Format("2006-01-02"),
		PHValue: 6.8, FertilityIssue: "nutrient_low", Suggestion: "追施复合肥",
	}
	order, err := maintSvc.Create(req, admin.ID, "admin2")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// 非认养人不能查看
	if _, err := maintSvc.GetForUser(order.ID, other.ID, "citizen"); err == nil {
		t.Fatalf("expected forbidden for non-owner")
	}
	// 认养人本人可查看进度
	if _, err := maintSvc.GetForUser(order.ID, owner.ID, "farmer"); err != nil {
		t.Fatalf("owner GetForUser: %v", err)
	}
	// 非认养人不能查看地块历史
	if _, err := maintSvc.HistoryByPlotForUser(plot.ID, other.ID, "citizen"); err == nil {
		t.Fatalf("expected forbidden history for non-owner")
	}
	// 未认养地块不能登记养护单
	freePlot := newTestPlot(t, db, "P-FREE", "available", nil)
	freeReq := &dto.CreateMaintenanceRequest{
		PlotID: freePlot.ID, SampledAt: time.Now().Format("2006-01-02"),
		PHValue: 7.0, FertilityIssue: "healthy", Suggestion: "无需处理",
	}
	if _, err := maintSvc.Create(freeReq, admin.ID, "admin2"); err == nil {
		t.Fatalf("expected error creating maintenance on unadopted plot")
	}
}
