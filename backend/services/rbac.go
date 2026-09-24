package services

import (
	"github.com/tdy-manager/backend/models"
	"gorm.io/gorm"
)

type PermDef struct {
	Code      string
	Name      string
	Module    string
	SortOrder int
}

func AllPermissionDefs() []PermDef {
	return []PermDef{
		{models.PermDashboardView, "查看仪表盘", "dashboard", 10},
		{models.PermOrderView, "查看工单", "order", 20},
		{models.PermOrderCreate, "创建/派单", "order", 21},
		{models.PermOrderApprove, "验收/驳回", "order", 22},
		{models.PermOrderReassign, "改派", "order", 23},
		{models.PermOrderCancel, "取消工单", "order", 24},
		{models.PermUserView, "查看用户", "user", 30},
		{models.PermUserCreate, "创建用户", "user", 31},
		{models.PermUserManage, "启用/禁用/重置密码", "user", 32},
		{models.PermWalletView, "查看钱包", "wallet", 40},
		{models.PermWalletAdjust, "调账", "wallet", 41},
		{models.PermWithdrawView, "查看提现", "withdraw", 50},
		{models.PermWithdrawReview, "审核提现", "withdraw", 51},
		{models.PermReconcileView, "查看对账", "reconcile", 60},
		{models.PermReconcileExport, "导出对账", "reconcile", 61},
		{models.PermAuditView, "查看审计", "audit", 70},
		{models.PermRoleManage, "管理角色权限", "role", 80},
		{models.PermAdminManage, "管理管理员", "role", 81},
	}
}

func EnsureRBAC(db *gorm.DB) error {
	defs := AllPermissionDefs()
	codeToID := make(map[string]uint, len(defs))
	for _, d := range defs {
		var p models.Permission
		err := db.Where("code = ?", d.Code).First(&p).Error
		if err == gorm.ErrRecordNotFound {
			p = models.Permission{Code: d.Code, Name: d.Name, Module: d.Module, SortOrder: d.SortOrder}
			if err := db.Create(&p).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			p.Name = d.Name
			p.Module = d.Module
			p.SortOrder = d.SortOrder
			if err := db.Save(&p).Error; err != nil {
				return err
			}
		}
		codeToID[d.Code] = p.ID
	}

	allCodes := make([]string, 0, len(defs))
	for _, d := range defs {
		allCodes = append(allCodes, d.Code)
	}

	type roleDef struct {
		Code, Name, Desc string
		Perms            []string
	}
	roles := []roleDef{
		{models.AdminRoleSuper, "超级管理员", "全部权限", allCodes},
		{models.AdminRoleOps, "运营", "工单与用户", []string{
			models.PermDashboardView,
			models.PermOrderView, models.PermOrderCreate, models.PermOrderApprove, models.PermOrderReassign, models.PermOrderCancel,
			models.PermUserView, models.PermUserCreate, models.PermUserManage,
		}},
		{models.AdminRoleFinance, "财务", "钱包提现对账", []string{
			models.PermDashboardView,
			models.PermWalletView, models.PermWalletAdjust,
			models.PermWithdrawView, models.PermWithdrawReview,
			models.PermReconcileView, models.PermReconcileExport,
			models.PermAuditView,
		}},
		{models.AdminRoleAuditor, "审计", "只读审计", []string{
			models.PermDashboardView,
			models.PermOrderView, models.PermUserView, models.PermWalletView,
			models.PermWithdrawView, models.PermReconcileView, models.PermAuditView,
		}},
	}

	for _, rd := range roles {
		var role models.AdminRole
		err := db.Where("code = ?", rd.Code).First(&role).Error
		created := false
		if err == gorm.ErrRecordNotFound {
			role = models.AdminRole{Code: rd.Code, Name: rd.Name, Description: rd.Desc, IsSystem: true}
			if err := db.Create(&role).Error; err != nil {
				return err
			}
			created = true
		} else if err != nil {
			return err
		} else {
			role.Name = rd.Name
			role.Description = rd.Desc
			role.IsSystem = true
			if err := db.Save(&role).Error; err != nil {
				return err
			}
		}
		// 仅新建系统角色时写入默认权限，避免覆盖管理员后续自定义
		if created {
			perms := make([]models.Permission, 0, len(rd.Perms))
			for _, code := range rd.Perms {
				if id, ok := codeToID[code]; ok {
					perms = append(perms, models.Permission{ID: id})
				}
			}
			if err := db.Model(&role).Association("Permissions").Replace(perms); err != nil {
				return err
			}
		}
	}

	// 已有管理员未绑角色 → 绑超级管理员
	var super models.AdminRole
	if err := db.Where("code = ?", models.AdminRoleSuper).First(&super).Error; err != nil {
		return err
	}
	return db.Model(&models.User{}).
		Where("role = ? AND admin_role_id IS NULL", models.RoleAdmin).
		Update("admin_role_id", super.ID).Error
}

func LoadUserPermissions(db *gorm.DB, userID uint) ([]string, *models.AdminRole, error) {
	var user models.User
	if err := db.Preload("AdminRole.Permissions").First(&user, userID).Error; err != nil {
		return nil, nil, err
	}
	if user.Role != models.RoleAdmin {
		return nil, nil, nil
	}
	if user.AdminRole == nil {
		// 兼容：管理员无角色视为超管
		all := AllPermissionDefs()
		codes := make([]string, len(all))
		for i, d := range all {
			codes[i] = d.Code
		}
		return codes, nil, nil
	}
	if user.AdminRole.Code == models.AdminRoleSuper {
		all := AllPermissionDefs()
		codes := make([]string, len(all))
		for i, d := range all {
			codes[i] = d.Code
		}
		return codes, user.AdminRole, nil
	}
	codes := make([]string, 0, len(user.AdminRole.Permissions))
	for _, p := range user.AdminRole.Permissions {
		codes = append(codes, p.Code)
	}
	return codes, user.AdminRole, nil
}

func UserHasPerm(db *gorm.DB, userID uint, code string) bool {
	perms, _, err := LoadUserPermissions(db, userID)
	if err != nil {
		return false
	}
	for _, p := range perms {
		if p == code {
			return true
		}
	}
	return false
}
