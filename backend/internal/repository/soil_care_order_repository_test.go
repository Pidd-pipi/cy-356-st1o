package repository

import (
	"testing"
	"time"

	"github.com/communitygarden/server/internal/model"
	"github.com/communitygarden/server/internal/util"
)

func TestSoilCareOrderRepository_OpenAndList(t *testing.T) {
	db := newTestDB(t)
	plotRepo := NewPlotRepository(db)
	repo := NewSoilCareOrderRepository(db)
	owner := seedUser(t, db, "owner", "citizen")
	other := seedUser(t, db, "other", "citizen")
	admin := seedUser(t, db, "careadmin", "admin")

	ownerID := owner.ID
	otherID := other.ID
	p1 := &model.Plot{Name: "P1", Code: "P-C1", Area: 10, SoilType: "loam", Sunlight: "full", Latitude: 31, Longitude: 121, Status: "caring", AdopterID: &ownerID}
	p2 := &model.Plot{Name: "P2", Code: "P-C2", Area: 10, SoilType: "loam", Sunlight: "full", Latitude: 31, Longitude: 121, Status: "caring", AdopterID: &otherID}
	if err := plotRepo.Create(p1); err != nil {
		t.Fatalf("create p1: %v", err)
	}
	if err := plotRepo.Create(p2); err != nil {
		t.Fatalf("create p2: %v", err)
	}

	now := time.Now()
	mk := func(plotID uint, status string) *model.SoilCareOrder {
		return &model.SoilCareOrder{
			PlotID: plotID, AdminID: admin.ID, Status: status, SampledDate: now,
			PHValue: 6.0, FertilityIssue: "板结", TreatmentAdvice: "深翻",
		}
	}
	pending := mk(p1.ID, "pending")
	completed := mk(p1.ID, "completed")
	otherOpen := mk(p2.ID, "in_progress")
	for _, o := range []*model.SoilCareOrder{pending, completed, otherOpen} {
		if err := db.Create(o).Error; err != nil {
			t.Fatalf("create care order: %v", err)
		}
	}

	// 地块 1 只能查到一张未完成单
	open, err := repo.FindOpenByPlotForUpdate(db, p1.ID)
	if err != nil || open.ID != pending.ID {
		t.Fatalf("FindOpenByPlot got=%+v err=%v", open, err)
	}
	// 全部历史
	_, total, err := repo.List(util.PageQuery{Page: 1, PageSize: 10}, 0, "", 0)
	if err != nil || total != 3 {
		t.Fatalf("list all total=%d err=%v", total, err)
	}
	// 按地块
	_, total, err = repo.List(util.PageQuery{Page: 1, PageSize: 10}, p1.ID, "", 0)
	if err != nil || total != 2 {
		t.Fatalf("list by plot total=%d err=%v", total, err)
	}
	// 认养人视角：owner 只能看到自己地块（p1）的两张
	_, total, err = repo.List(util.PageQuery{Page: 1, PageSize: 10}, 0, "", ownerID)
	if err != nil || total != 2 {
		t.Fatalf("list by adopter total=%d err=%v", total, err)
	}
	// 开放单计数
	cnt, err := repo.CountOpen()
	if err != nil || cnt != 2 {
		t.Fatalf("CountOpen cnt=%d err=%v", cnt, err)
	}
}
