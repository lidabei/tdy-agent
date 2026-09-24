package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tdy-manager/backend/middleware"
	"github.com/tdy-manager/backend/models"
	"gorm.io/gorm"
)

type NotifyHandler struct {
	DB *gorm.DB
}

func (h *NotifyHandler) List(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	page, size := parsePage(c)
	q := h.DB.Model(&models.Notification{}).Where("user_id = ?", uid)
	var total int64
	q.Count(&total)
	var list []models.Notification
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var unread int64
	h.DB.Model(&models.Notification{}).Where("user_id = ? AND read = ?", uid, false).Count(&unread)
	c.JSON(http.StatusOK, gin.H{"list": list, "total": total, "page": page, "size": size, "unread": unread})
}

func (h *NotifyHandler) UnreadCount(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var unread int64
	h.DB.Model(&models.Notification{}).Where("user_id = ? AND read = ?", uid, false).Count(&unread)
	c.JSON(http.StatusOK, gin.H{"unread": unread})
}

func (h *NotifyHandler) MarkRead(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Model(&models.Notification{}).Where("id = ? AND user_id = ?", id, uid).Update("read", true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *NotifyHandler) MarkAllRead(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	h.DB.Model(&models.Notification{}).Where("user_id = ? AND read = ?", uid, false).Update("read", true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
