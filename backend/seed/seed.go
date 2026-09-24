package seed

import (
	"log"

	"github.com/tdy-manager/backend/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Run(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}

	hash := func(pw string) string {
		b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal(err)
		}
		return string(b)
	}

	var super models.AdminRole
	if err := db.Where("code = ?", models.AdminRoleSuper).First(&super).Error; err != nil {
		log.Fatal("seed: super_admin role missing:", err)
	}
	superID := super.ID

	admin := models.User{
		Username:     "admin",
		PasswordHash: hash("admin123"),
		DisplayName:  "系统管理员",
		Role:         models.RoleAdmin,
		AdminRoleID:  &superID,
		Enabled:      true,
	}
	u1 := models.User{
		Username:     "zhangsan",
		PasswordHash: hash("pass1234"),
		DisplayName:  "张三",
		Role:         models.RoleUser,
		Enabled:      true,
	}
	u2 := models.User{
		Username:     "lisi",
		PasswordHash: hash("pass1234"),
		DisplayName:  "李四",
		Role:         models.RoleUser,
		Enabled:      true,
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, u := range []*models.User{&admin, &u1, &u2} {
			if err := tx.Create(u).Error; err != nil {
				return err
			}
			if u.Role == models.RoleUser {
				if err := tx.Create(&models.Wallet{UserID: u.ID, Balance: 0}).Error; err != nil {
					return err
				}
			}
		}
		order := models.Order{
			Title:       "示例任务：整理本周数据报表",
			Description: "请按模板整理 Excel，完成后在系统提交说明。",
			Reward:      5000, // 50.00 元
			Status:      models.OrderPending,
			AssigneeID:  u1.ID,
			CreatedByID: admin.ID,
		}
		return tx.Create(&order).Error
	}); err != nil {
		log.Fatal("seed failed:", err)
	}
	log.Println("seed ok: admin/admin123 (super_admin), zhangsan/pass1234, lisi/pass1234")
}
