package migrate

import (
	"log"

	"github.com/tdy-manager/backend/models"
	"github.com/tdy-manager/backend/services"
	"gorm.io/gorm"
)

func Run(db *gorm.DB) {
	_ = db.AutoMigrate(&models.SchemaMeta{})

	var meta models.SchemaMeta
	err := db.Order("id asc").First(&meta).Error
	needMigrate := err != nil || meta.Version != models.SchemaVersion

	if !needMigrate {
		log.Printf("schema version %s up-to-date, skip AutoMigrate", models.SchemaVersion)
	} else {
		log.Printf("migrating schema -> %s", models.SchemaVersion)
		if err := db.AutoMigrate(
			&models.Permission{},
			&models.AdminRole{},
			&models.User{},
			&models.Wallet{},
			&models.WalletTx{},
			&models.Order{},
			&models.OrderAttachment{},
			&models.Withdrawal{},
			&models.Notification{},
			&models.AuditLog{},
		); err != nil {
			log.Fatal("migrate failed:", err)
		}
		db.Exec(`UPDATE wallet_txs SET idempotent_key = 'legacy:' || id::text WHERE idempotent_key IS NULL OR idempotent_key = ''`)

		if err != nil {
			db.Create(&models.SchemaMeta{Version: models.SchemaVersion})
		} else {
			meta.Version = models.SchemaVersion
			db.Save(&meta)
		}
	}

	// 权限与系统角色每次启动幂等同步
	if err := services.EnsureRBAC(db); err != nil {
		log.Fatal("ensure rbac failed:", err)
	}
}
