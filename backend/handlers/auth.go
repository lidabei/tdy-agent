package handlers

import (
	"fmt"
	"net/http"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/tdy-manager/backend/middleware"
	"github.com/tdy-manager/backend/models"
	"github.com/tdy-manager/backend/services"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB             *gorm.DB
	Secret         string
	PasswordMinLen int
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type changePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type userSessionDTO struct {
	ID          uint              `json:"id"`
	Username    string            `json:"username"`
	DisplayName string            `json:"display_name"`
	Role        string            `json:"role"`
	AdminRoleID *uint             `json:"admin_role_id,omitempty"`
	AdminRole   *models.AdminRole `json:"admin_role,omitempty"`
	Enabled     bool              `json:"enabled"`
	Permissions []string          `json:"permissions"`
	CreatedAt   time.Time         `json:"created_at"`
}

func toSessionDTO(db *gorm.DB, user models.User) (userSessionDTO, error) {
	perms, role, err := services.LoadUserPermissions(db, user.ID)
	if err != nil {
		return userSessionDTO{}, err
	}
	if perms == nil {
		perms = []string{}
	}
	dto := userSessionDTO{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		AdminRoleID: user.AdminRoleID,
		AdminRole:   role,
		Enabled:     user.Enabled,
		Permissions: perms,
		CreatedAt:   user.CreatedAt,
	}
	if dto.AdminRole != nil {
		// 精简权限列表，避免登录响应过大
		slim := *dto.AdminRole
		slim.Permissions = nil
		dto.AdminRole = &slim
	}
	return dto, nil
}

func validatePassword(pw string, minLen int) error {
	if len(pw) < minLen {
		return fmt.Errorf("密码至少 %d 位", minLen)
	}
	hasLetter, hasDigit := false, false
	for _, r := range pw {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("密码需同时包含字母和数字")
	}
	return nil
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入用户名和密码"})
		return
	}
	var user models.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	if !user.Enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已禁用，请联系管理员"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	token, err := middleware.SignToken(h.Secret, user, 7*24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签发令牌失败"})
		return
	}
	dto, err := toSessionDTO(h.DB, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载权限失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": dto})
}

func (h *AuthHandler) Me(c *gin.Context) {
	var user models.User
	if err := h.DB.First(&user, middleware.CurrentUserID(c)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if !user.Enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已禁用"})
		return
	}
	dto, err := toSessionDTO(h.DB, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载权限失败"})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := validatePassword(req.NewPassword, h.PasswordMinLen); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user models.User
	if err := h.DB.First(&user, middleware.CurrentUserID(c)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "原密码错误"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	user.PasswordHash = string(hash)
	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
