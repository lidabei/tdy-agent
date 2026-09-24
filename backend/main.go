package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/tdy-manager/backend/config"
	"github.com/tdy-manager/backend/handlers"
	"github.com/tdy-manager/backend/middleware"
	"github.com/tdy-manager/backend/migrate"
	"github.com/tdy-manager/backend/models"
	"github.com/tdy-manager/backend/seed"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.Load()
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	gormLog := logger.Info
	if cfg.Mode == "release" {
		gormLog = logger.Warn
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{Logger: logger.Default.LogMode(gormLog)})
	if err != nil {
		log.Fatalf("连接 PostgreSQL 失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	migrate.Run(db)
	seed.Run(db)
	_ = os.MkdirAll(cfg.UploadDir, 0o755)

	authH := &handlers.AuthHandler{DB: db, Secret: cfg.JWTSecret, PasswordMinLen: cfg.PasswordMinLen}
	userH := &handlers.UserHandler{DB: db, PasswordMinLen: cfg.PasswordMinLen}
	orderH := &handlers.OrderHandler{DB: db}
	walletH := &handlers.WalletHandler{DB: db}
	withdrawH := &handlers.WithdrawHandler{DB: db, MaxFen: cfg.WithdrawMaxFen, DailyMaxFen: cfg.WithdrawDailyMaxFen}
	healthH := &handlers.HealthHandler{DB: db}
	dashH := &handlers.DashboardHandler{DB: db}
	notifyH := &handlers.NotifyHandler{DB: db}
	auditH := &handlers.AuditHandler{DB: db}
	fileH := &handlers.FileHandler{DB: db, UploadDir: cfg.UploadDir}
	roleH := &handlers.RoleHandler{DB: db}

	r := gin.Default()
	r.MaxMultipartMemory = 32 << 20
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := r.Group("/api")
	{
		api.GET("/health", healthH.Check)
		api.POST("/auth/login", middleware.LoginRateLimit(), authH.Login)

		auth := api.Group("")
		auth.Use(middleware.Auth(cfg.JWTSecret), middleware.RequireEnabled(db))
		{
			auth.GET("/auth/me", authH.Me)
			auth.POST("/auth/change-password", authH.ChangePassword)

			auth.GET("/orders/mine", orderH.MyList)
			auth.POST("/orders/:id/accept", orderH.Accept)
			auth.POST("/orders/:id/submit", orderH.Submit)
			auth.POST("/orders/:id/attachments", fileH.UploadOrderFile)
			auth.GET("/orders/:id/attachments", fileH.ListByOrder)
			auth.GET("/attachments/:id/download", fileH.Download)

			auth.GET("/wallet", walletH.MyWallet)
			auth.GET("/wallet/transactions", walletH.MyTransactions)
			auth.POST("/withdrawals", withdrawH.Apply)
			auth.GET("/withdrawals/mine", withdrawH.MyList)

			auth.GET("/notifications", notifyH.List)
			auth.GET("/notifications/unread-count", notifyH.UnreadCount)
			auth.POST("/notifications/:id/read", notifyH.MarkRead)
			auth.POST("/notifications/read-all", notifyH.MarkAllRead)

			admin := auth.Group("/admin")
			admin.Use(middleware.RequireAdmin())
			{
				admin.GET("/dashboard", middleware.RequirePerm(db, models.PermDashboardView), dashH.Admin)

				admin.GET("/users", middleware.RequirePerm(db, models.PermUserView, models.PermAdminManage), userH.List)
				admin.POST("/users", middleware.RequirePerm(db, models.PermUserCreate, models.PermAdminManage), userH.Create)
				admin.POST("/users/:id/enabled", middleware.RequirePerm(db, models.PermUserManage, models.PermAdminManage), userH.SetEnabled)
				admin.POST("/users/:id/reset-password", middleware.RequirePerm(db, models.PermUserManage, models.PermAdminManage), userH.ResetPassword)
				admin.POST("/users/:id/admin-role", middleware.RequirePerm(db, models.PermAdminManage), roleH.SetUserAdminRole)

				admin.GET("/orders", middleware.RequirePerm(db, models.PermOrderView), orderH.AdminList)
				admin.POST("/orders", middleware.RequirePerm(db, models.PermOrderCreate), orderH.Create)
				admin.POST("/orders/batch", middleware.RequirePerm(db, models.PermOrderCreate), orderH.BatchCreate)
				admin.POST("/orders/:id/cancel", middleware.RequirePerm(db, models.PermOrderCancel), orderH.Cancel)
				admin.POST("/orders/:id/approve", middleware.RequirePerm(db, models.PermOrderApprove), orderH.Approve)
				admin.POST("/orders/:id/reject", middleware.RequirePerm(db, models.PermOrderApprove), orderH.Reject)
				admin.POST("/orders/:id/reassign", middleware.RequirePerm(db, models.PermOrderReassign), orderH.Reassign)

				admin.GET("/wallets", middleware.RequirePerm(db, models.PermWalletView), walletH.AdminList)
				admin.GET("/wallets/transactions", middleware.RequirePerm(db, models.PermWalletView), walletH.AdminTransactions)
				admin.POST("/wallets/adjust", middleware.RequirePerm(db, models.PermWalletAdjust), walletH.Adjust)
				admin.GET("/reconcile", middleware.RequirePerm(db, models.PermReconcileView), walletH.Reconcile)
				admin.GET("/reconcile/export", middleware.RequirePerm(db, models.PermReconcileExport), walletH.ExportReconcile)

				admin.GET("/withdrawals", middleware.RequirePerm(db, models.PermWithdrawView), withdrawH.AdminList)
				admin.POST("/withdrawals/:id/approve", middleware.RequirePerm(db, models.PermWithdrawReview), withdrawH.Approve)
				admin.POST("/withdrawals/:id/reject", middleware.RequirePerm(db, models.PermWithdrawReview), withdrawH.Reject)
				admin.POST("/withdrawals/batch-approve", middleware.RequirePerm(db, models.PermWithdrawReview), withdrawH.BatchApprove)

				admin.GET("/audit-logs", middleware.RequirePerm(db, models.PermAuditView), auditH.List)

				admin.GET("/permissions", middleware.RequirePerm(db, models.PermRoleManage), roleH.ListPermissions)
				admin.GET("/roles", middleware.RequirePerm(db, models.PermRoleManage, models.PermAdminManage), roleH.ListRoles)
				admin.POST("/roles", middleware.RequirePerm(db, models.PermRoleManage), roleH.CreateRole)
				admin.PUT("/roles/:id", middleware.RequirePerm(db, models.PermRoleManage), roleH.UpdateRole)
				admin.DELETE("/roles/:id", middleware.RequirePerm(db, models.PermRoleManage), roleH.DeleteRole)
			}
		}
	}

	serveFrontend(r, cfg.StaticDir)

	log.Printf("TDY Manager listening on :%s (mode=%s)", cfg.Port, cfg.Mode)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

func serveFrontend(r *gin.Engine, staticDir string) {
	abs, err := filepath.Abs(staticDir)
	if err != nil || abs == "" {
		return
	}
	index := filepath.Join(abs, "index.html")
	if _, err := os.Stat(index); err != nil {
		log.Printf("static dir not found (%s), skip SPA hosting", abs)
		return
	}
	r.Static("/assets", filepath.Join(abs, "assets"))
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.File(index)
	})
	log.Printf("serving frontend from %s", abs)
}
