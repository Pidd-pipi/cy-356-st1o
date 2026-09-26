package repository

import (
	"errors"
	"testing"

	"github.com/communitygarden/server/internal/model"
	"github.com/communitygarden/server/internal/util"
)

func TestMaintenanceRepository_ActiveAndHistory(t *testing.T) {
	db := newTestDB(t)
	repo := NewMaintenanceRepository(db)
	plotRepo := NewPlotRepository(db)
	admin := seedUser(t, db, "admin", "admin")
	adopter := seedUser(t, db, "adopter", "citizen")
	uid := adopter.ID
	plot := &model.Plot{Name: "P-M", Code: "P-M", Area: 10, SoilType: "loam", Sunlight: "full", Latitude: 31.0, Longitude: 121.0, Status: "adopted", AdopterID: &uid}
	if err := plotRepo.Create(plot); err != nil {
		t.Fatalf("create plot: %v", err)
	}

	createOrder := func(status string) *model.MaintenanceOrder {
		o := &model.MaintenanceOrder{
			PlotID: plot.ID, OperatorID: admin.ID, PHValue: 6.0,
			FertilityIssue: "nutrient_low", Suggestion: "增施有机肥", Status: status,
		}
		if err := db.Create(o).Error; err != nil {
			t.Fatalf("create order: %v", err)
		}
		return o
	}

	// 无未完成养护单时返回 ErrNotFound
	if _, err := repo.FindActiveByPlotForUpdate(db, plot.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	active := createOrder("processing")
	createOrder("completed")
	createOrder("cancelled")

	got, err := repo.FindActiveByPlotForUpdate(db, plot.ID)
	if err != nil || got.ID != active.ID {
		t.Fatalf("FindActiveByPlotForUpdate id=%d err=%v", func() uint {
			if got != nil {
				return got.ID
			}
			return 0
		}(), err)
	}

	list, total, err := repo.List(util.PageQuery{Page: 1, PageSize: 10}, plot.ID, "")
	if err != nil || total != 3 || len(list) != 3 {
		t.Fatalf("List total=%d len=%d err=%v", total, len(list), err)
	}
	pendingOnly, total, err := repo.List(util.PageQuery{Page: 1, PageSize: 10}, plot.ID, "processing")
	if err != nil || total != 1 || len(pendingOnly) != 1 {
		t.Fatalf("List filter total=%d len=%d err=%v", total, len(pendingOnly), err)
	}
	history, err := repo.ListByPlot(plot.ID)
	if err != nil || len(history) != 3 {
		t.Fatalf("ListByPlot len=%d err=%v", len(history), err)
	}
	counts, err := repo.CountByStatus()
	if err != nil {
		t.Fatalf("CountByStatus: %v", err)
	}
	if counts["processing"] != 1 || counts["completed"] != 1 || counts["cancelled"] != 1 {
		t.Errorf("counts = %v", counts)
	}
}
