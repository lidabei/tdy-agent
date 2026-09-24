package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tdy-manager/backend/middleware"
	"github.com/tdy-manager/backend/models"
	"github.com/tdy-manager/backend/services"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserHandler struct {
	DB             *gorm.DB
	PasswordMinLen int
}

type createUserReq struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Role        string `json:"role"` // user | admin，默认 user
	AdminRoleID *uint  `json:"admin_role_id"`
}

type resetPasswordReq struct {
	Password string `json:"password" binding:"required"`
}

type setEnabledReq struct {
	Enabled bool `json:"enabled"`
}

func (h *UserHandler) List(c *gin.Context) {
	page, size := parsePage(c)
	q := h.DB.Model(&models.User{}).Preload("AdminRole").Order("id asc")
	if role := c.Query("role"); role != "" {
		q = q.Where("role = ?", role)
	}
	if c.Query("all") == "1" {
		var users []models.User
		if err := q.Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, users)
		return
	}
	var total int64
	q.Count(&total)
	var users []models.User
	if err := q.Offset((page - 1) * size).Limit(size).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PageResult{List: users, Total: total, Page: page, Size: size})
}

func (h *UserHandler) Create(c *gin.Context) {
	var req createUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不完整"})
		return
	}
	if err := validatePassword(req.Password, h.PasswordMinLen); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := req.Role
	if role == "" {
		role = models.RoleUser
	}
	if role != models.RoleUser && role != models.RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色无效"})
		return
	}
	if role == models.RoleAdmin {
		if !services.UserHasPerm(h.DB, middleware.CurrentUserID(c), models.PermAdminManage) {
			c.JSON(http.StatusForbidden, gin.H{"error": "无创建管理员权限"})
			return
		}
		if req.AdminRoleID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请为管理员选择角色"})
			return
		}
		var ar models.AdminRole
		if err := h.DB.First(&ar, *req.AdminRoleID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "管理角色不存在"})
			return
		}
	} else if !services.UserHasPerm(h.DB, middleware.CurrentUserID(c), models.PermUserCreate) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无创建用户权限"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	user := models.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		DisplayName:  req.DisplayName,
		Role:         role,
		AdminRoleID:  req.AdminRoleID,
		Enabled:      true,
	}
	if role == models.RoleUser {
		user.AdminRoleID = nil
	}
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if role == models.RoleUser {
			return tx.Create(&models.Wallet{UserID: user.ID, Balance: 0}).Error
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建失败，用户名可能已存在"})
		return
	}
	services.WriteAudit(h.DB, c, "user.create", "user", user.ID, user.Username+"/"+user.Role)
	_ = h.DB.Preload("AdminRole").First(&user, user.ID)
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) SetEnabled(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req setEnabledReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	var user models.User
	if err := h.DB.Preload("AdminRole").First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if user.ID == middleware.CurrentUserID(c) && !req.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能禁用自己"})
		return
	}
	if user.Role == models.RoleAdmin {
		if !services.UserHasPerm(h.DB, middleware.CurrentUserID(c), models.PermAdminManage) {
			c.JSON(http.StatusForbidden, gin.H{"error": "无管理管理员权限"})
			return
		}
		if !req.Enabled && user.AdminRole != nil && user.AdminRole.Code == models.AdminRoleSuper {
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
	}
	user.Enabled = req.Enabled
	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.WriteAudit(h.DB, c, "user.set_enabled", "user", user.ID, strconv.FormatBool(req.Enabled))
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) ResetPassword(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req resetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := validatePassword(req.Password, h.PasswordMinLen); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user models.User
	if err := h.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if user.Role == models.RoleAdmin && !services.UserHasPerm(h.DB, middleware.CurrentUserID(c), models.PermAdminManage) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无管理管理员权限"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	user.PasswordHash = string(hash)
	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.WriteAudit(h.DB, c, "user.reset_password", "user", user.ID, user.Username)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
