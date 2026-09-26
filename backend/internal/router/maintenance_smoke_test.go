package router_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/communitygarden/server/internal/config"
	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/handler"
	"github.com/communitygarden/server/internal/model"
	"github.com/communitygarden/server/internal/repository"
	"github.com/communitygarden/server/internal/router"
	"github.com/communitygarden/server/internal/service"
	"github.com/communitygarden/server/internal/util"
)

var smokeDBCounter uint64

type smokeAudit struct{}

func (smokeAudit) Write(userID uint, username, role, action, resourceType, resourceID, detail, ip, requestID string) error {
	return nil
}

// TestMaintenanceAPI_Smoke 通过真实 Gin 路由验证养护单完整 HTTP 流程（SQLite 内存库）。
func TestMaintenanceAPI_Smoke(t *testing.T) {
	n := atomic.AddUint64(&smokeDBCounter, 1)
	dsn := fmt.Sprintf("file:smoke%d?mode=memory&cache=shared", n)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Plot{}, &model.PlantingPlan{}, &model.HarvestRecord{},
		&model.DiaryEntry{}, &model.DiaryComment{}, &model.CommunityPost{}, &model.CommunityComment{},
		&model.MaintenanceOrder{}, &model.AuditLog{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	admin := &model.User{Username: "admin", Password: "x", Nickname: "管理员", Role: "admin", Status: "active"}
	farmer := &model.User{Username: "farmer", Password: "x", Nickname: "老农", Role: "farmer", Status: "active"}
	db.Create(admin)
	db.Create(farmer)
	plot := &model.Plot{Name: "T-01", Code: "T-01", Area: 10, SoilType: "loam", Sunlight: "full", Latitude: 31, Longitude: 121, Status: "adopted", AdopterID: &farmer.ID}
	db.Create(plot)

	userRepo := repository.NewUserRepository(db)
	plotRepo := repository.NewPlotRepository(db)
	planRepo := repository.NewPlantingPlanRepository(db)
	harvestRepo := repository.NewHarvestRecordRepository(db)
	diaryRepo := repository.NewDiaryRepository(db)
	postRepo := repository.NewCommunityRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	maintRepo := repository.NewMaintenanceRepository(db)

	lg := slog.New(slog.NewTextHandler(discardW{}, nil))
	plotSvc := service.NewPlotService(plotRepo, db, lg)
	authSvc := service.NewAuthService(userRepo, lg, "secret", 24)
	userSvc := service.NewUserService(userRepo, lg)
	auditSvc := service.NewAuditService(auditRepo, lg)
	planSvc := service.NewPlantingPlanService(planRepo, plotRepo, plotSvc, db, lg)
	harvestSvc := service.NewHarvestRecordService(harvestRepo, planRepo, db, lg)
	diarySvc := service.NewDiaryService(diaryRepo, planRepo, lg)
	communitySvc := service.NewCommunityService(postRepo, lg)
	statsSvc := service.NewStatsService(userRepo, plotRepo, planRepo, harvestRepo, diaryRepo, postRepo, maintRepo, lg)
	maintSvc := service.NewMaintenanceService(maintRepo, plotRepo, plotSvc, db, lg)

	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "secret", JWTExpireHours: 24}
	r := router.New(
		cfg, lg, nil,
		handler.NewAuthHandler(authSvc),
		handler.NewUserHandler(userSvc, auditSvc),
		handler.NewPlotHandler(plotSvc, auditSvc),
		handler.NewPlantingPlanHandler(planSvc, harvestSvc),
		handler.NewHarvestHandler(harvestSvc, auditSvc),
		handler.NewDiaryHandler(diarySvc),
		handler.NewCommunityHandler(communitySvc),
		handler.NewAuditHandler(auditSvc),
		handler.NewStatsHandler(statsSvc),
		handler.NewMaintenanceHandler(maintSvc, smokeAudit{}),
		auditSvc, nil,
	).Build()

	adminToken, _ := util.GenerateToken("secret", 24, admin.ID, "admin", "admin")
	farmerToken, _ := util.GenerateToken("secret", 24, farmer.ID, "farmer", "farmer")
	today := time.Now().Format("2006-01-02")

	doReq := func(method, path, token string, body interface{}) (int, map[string]interface{}) {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		return w.Code, resp
	}

	// 1. 认养人不能登记养护单（RBAC 403）
	code, resp := doReq(http.MethodPost, "/api/v1/maintenance-orders", farmerToken, map[string]interface{}{
		"plot_id": plot.ID, "sampled_at": today, "ph_value": 5.2,
		"fertility_issue": "acidic", "suggestion": "调酸",
	})
	if code != http.StatusForbidden {
		t.Fatalf("farmer create status=%d resp=%v", code, resp)
	}

	// 2. 管理员登记
	code, resp = doReq(http.MethodPost, "/api/v1/maintenance-orders", adminToken, map[string]interface{}{
		"plot_id": plot.ID, "sampled_at": today, "ph_value": 5.2,
		"fertility_issue": "acidic", "suggestion": "撒生石灰调酸，增施有机肥",
	})
	if code != http.StatusOK {
		t.Fatalf("admin create status=%d resp=%v", code, resp)
	}
	data := resp["data"].(map[string]interface{})
	orderID := int(data["id"].(float64))
	if data["status"] != "pending" {
		t.Fatalf("status=%v", data["status"])
	}

	// 3. 重复登记冲突 409 + CodeMaintenanceActive 2010
	code, resp = doReq(http.MethodPost, "/api/v1/maintenance-orders", adminToken, map[string]interface{}{
		"plot_id": plot.ID, "sampled_at": today, "ph_value": 5.2,
		"fertility_issue": "acidic", "suggestion": "重复",
	})
	if code != http.StatusConflict || int(resp["code"].(float64)) != constants.CodeMaintenanceActive {
		t.Fatalf("duplicate status=%d resp=%v", code, resp)
	}

	// 4. 养护期间认养人创建种植计划被拦截 409 + 2012
	code, resp = doReq(http.MethodPost, "/api/v1/planting-plans", farmerToken, map[string]interface{}{
		"plot_id": plot.ID, "crop_name": "菠菜", "crop_type": "vegetable", "season": "spring",
	})
	if code != http.StatusConflict || int(resp["code"].(float64)) != constants.CodePlotUnderMaintenance {
		t.Fatalf("plan during maintenance status=%d resp=%v", code, resp)
	}

	// 5. 认养人可查看进度与地块历史
	code, _ = doReq(http.MethodGet, fmt.Sprintf("/api/v1/maintenance-orders/%d", orderID), farmerToken, nil)
	if code != http.StatusOK {
		t.Fatalf("farmer get order status=%d", code)
	}
	code, resp = doReq(http.MethodGet, fmt.Sprintf("/api/v1/plots/%d/maintenances", plot.ID), farmerToken, nil)
	if code != http.StatusOK {
		t.Fatalf("history status=%d", code)
	}
	if list, ok := resp["data"].([]interface{}); !ok || len(list) != 1 {
		t.Fatalf("history data=%v", resp["data"])
	}

	// 6. pending 直接完成应 409
	code, _ = doReq(http.MethodPost, fmt.Sprintf("/api/v1/maintenance-orders/%d/complete", orderID), adminToken, map[string]interface{}{
		"actual_measures": "x", "completed_at": today,
	})
	if code != http.StatusConflict {
		t.Fatalf("complete pending status=%d", code)
	}

	// 7. 取消必须填原因（参数校验 400）
	code, _ = doReq(http.MethodPost, fmt.Sprintf("/api/v1/maintenance-orders/%d/cancel", orderID), adminToken, map[string]interface{}{
		"cancel_reason": "",
	})
	if code != http.StatusBadRequest {
		t.Fatalf("cancel empty reason status=%d", code)
	}

	// 8. 开始处理 -> 完成，地块恢复 adopted
	if code, _ = doReq(http.MethodPost, fmt.Sprintf("/api/v1/maintenance-orders/%d/start", orderID), adminToken, nil); code != http.StatusOK {
		t.Fatalf("start status=%d", code)
	}
	code, resp = doReq(http.MethodPost, fmt.Sprintf("/api/v1/maintenance-orders/%d/complete", orderID), adminToken, map[string]interface{}{
		"actual_measures": "撒生石灰 2kg、增施羊粪深翻，复测 pH 6.5", "completed_at": today,
	})
	if code != http.StatusOK {
		t.Fatalf("complete status=%d resp=%v", code, resp)
	}

	// 9. 地块详情状态恢复为 adopted，且此时可创建种植计划（季节正确即可通过到创建成功）
	code, resp = doReq(http.MethodGet, fmt.Sprintf("/api/v1/plots/%d", plot.ID), "", nil)
	if code != http.StatusOK || resp["data"].(map[string]interface{})["status"] != "adopted" {
		t.Fatalf("plot after complete status=%d resp=%v", code, resp)
	}
	code, resp = doReq(http.MethodPost, "/api/v1/planting-plans", farmerToken, map[string]interface{}{
		"plot_id": plot.ID, "crop_name": "菠菜", "crop_type": "vegetable", "season": "spring",
	})
	if code != http.StatusOK {
		t.Fatalf("plan after maintenance status=%d resp=%v", code, resp)
	}

	// 10. 第二张养护单取消流程：取消留原因，历史变为 2 张
	code, resp = doReq(http.MethodPost, "/api/v1/maintenance-orders", adminToken, map[string]interface{}{
		"plot_id": plot.ID, "sampled_at": today, "ph_value": 6.9,
		"fertility_issue": "healthy", "suggestion": "常规轮作观察",
	})
	if code != http.StatusOK {
		t.Fatalf("create second status=%d resp=%v", code, resp)
	}
	orderID2 := int(resp["data"].(map[string]interface{})["id"].(float64))
	code, resp = doReq(http.MethodPost, fmt.Sprintf("/api/v1/maintenance-orders/%d/cancel", orderID2), adminToken, map[string]interface{}{
		"cancel_reason": "连续降雨无法施工，延期至下季度",
	})
	if code != http.StatusOK {
		t.Fatalf("cancel status=%d resp=%v", code, resp)
	}
	if resp["data"].(map[string]interface{})["cancel_reason"] != "连续降雨无法施工，延期至下季度" {
		t.Fatalf("cancel reason missing: %v", resp["data"])
	}
	code, resp = doReq(http.MethodGet, fmt.Sprintf("/api/v1/plots/%d/maintenances", plot.ID), farmerToken, nil)
	if list, _ := resp["data"].([]interface{}); len(list) != 2 {
		t.Fatalf("history len after cancel = %v", resp["data"])
	}

	// 11. 已取消不能再次流转
	if code, _ = doReq(http.MethodPost, fmt.Sprintf("/api/v1/maintenance-orders/%d/start", orderID2), adminToken, nil); code != http.StatusConflict {
		t.Fatalf("restart cancelled status=%d", code)
	}

	// 12. 管理员列表与仪表盘统计
	code, resp = doReq(http.MethodGet, "/api/v1/maintenance-orders?page=1&page_size=10", adminToken, nil)
	if code != http.StatusOK {
		t.Fatalf("admin list status=%d", code)
	}
	if list, _ := resp["data"].(map[string]interface{})["list"].([]interface{}); len(list) != 2 {
		t.Fatalf("admin list=%v", resp["data"])
	}
	code, resp = doReq(http.MethodGet, "/api/v1/dashboard/stats", adminToken, nil)
	if code != http.StatusOK {
		t.Fatalf("dashboard status=%d", code)
	}
	maint := resp["data"].(map[string]interface{})["maintenance_by_status"].(map[string]interface{})
	if maint["completed"].(float64) != 1 || maint["cancelled"].(float64) != 1 {
		t.Fatalf("maintenance stats=%v", maint)
	}
}

type discardW struct{}

func (discardW) Write(p []byte) (int, error) { return len(p), nil }
