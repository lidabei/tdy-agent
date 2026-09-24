package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tdy-manager/backend/middleware"
	"github.com/tdy-manager/backend/models"
	"github.com/tdy-manager/backend/services"
	"gorm.io/gorm"
)

type RoleHandler struct {
	DB *gorm.DB
}

type upsertRoleReq struct {
	Code          string `json:"code"`
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	PermissionIDs []uint `json:"permission_ids"`
}

func (h *RoleHandler) ListPermissions(c *gin.Context) {
	var list []models.Permission
	if err := h.DB.Order("sort_order asc, id asc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *RoleHandler) ListRoles(c *gin.Context) {
	var list []models.AdminRole
	if err := h.DB.Preload("Permissions").Order("id asc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req upsertRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不完整"})
		return
	}
	if req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写角色编码"})
		return
	}
	role := models.AdminRole{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		IsSystem:    false,
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		return replaceRolePerms(tx, &role, req.PermissionIDs)
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建失败，编码可能已存在"})
		return
	}
	_ = h.DB.Preload("Permissions").First(&role, role.ID)
	services.WriteAudit(h.DB, c, "role.create", "admin_role", role.ID, role.Name)
	c.JSON(http.StatusOK, role)
}

func (h *RoleHandler) UpdateRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req upsertRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不完整"})
		return
	}
	var role models.AdminRole
	if err := h.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}
	role.Name = req.Name
	role.Description = req.Description
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&role).Error; err != nil {
			return err
		}
		return replaceRolePerms(tx, &role, req.PermissionIDs)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.DB.Preload("Permissions").First(&role, role.ID)
	services.WriteAudit(h.DB, c, "role.update", "admin_role", role.ID, role.Name)
	c.JSON(http.StatusOK, role)
}

func (h *RoleHandler) DeleteRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var role models.AdminRole
	if err := h.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}
	if role.IsSystem {
		c.JSON(http.StatusBadRequest, gin.H{"error": "系统角色不可删除"})
		return
	}
	var cnt int64
	h.DB.Model(&models.User{}).Where("admin_role_id = ?", role.ID).Count(&cnt)
	if cnt > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仍有管理员使用该角色"})
		return
	}
	if err := h.DB.Select("Permissions").Delete(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.WriteAudit(h.DB, c, "role.delete", "admin_role", role.ID, role.Name)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func replaceRolePerms(tx *gorm.DB, role *models.AdminRole, ids []uint) error {
	perms := make([]models.Permission, 0, len(ids))
	for _, id := range ids {
		perms = append(perms, models.Permission{ID: id})
	}
	return tx.Model(role).Association("Permissions").Replace(perms)
}

func (h *RoleHandler) SetUserAdminRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		AdminRoleID uint `json:"admin_role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择角色"})
		return
	}
	var user models.User
	if err := h.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if user.Role != models.RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅管理员可分配管理角色"})
		return
	}
	var role models.AdminRole
	if err := h.DB.First(&role, req.AdminRoleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}
	// 防止把自己从超管改掉导致无人可管：允许，但至少保留一个超管
	if user.ID == middleware.CurrentUserID(c) && role.Code != models.AdminRoleSuper {
		var super models.AdminRole
		if err := h.DB.Where("code = ?", models.AdminRoleSuper).First(&super).Error; err == nil {
			var other int64
			h.DB.Model(&models.User{}).
				Where("role = ? AND admin_role_id = ? AND id <> ? AND enabled = true", models.RoleAdmin, super.ID, user.ID).
				Count(&other)
			if other == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "至少保留一名启用的超级管理员"})
				return
			}
		}
	}
	rid := role.ID
	user.AdminRoleID = &rid
	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.WriteAudit(h.DB, c, "admin.set_role", "user", user.ID, role.Code)
	_ = h.DB.Preload("AdminRole").First(&user, user.ID)
	c.JSON(http.StatusOK, user)
}
