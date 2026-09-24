package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tdy-manager/backend/models"
	"gorm.io/gorm"
)

type AuditHandler struct {
	DB *gorm.DB
}

func (h *AuditHandler) List(c *gin.Context) {
	page, size := parsePage(c)
	q := h.DB.Model(&models.AuditLog{})
	if a := c.Query("action"); a != "" {
		q = q.Where("action = ?", a)
	}
	var total int64
	q.Count(&total)
	var list []models.AuditLog
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PageResult{List: list, Total: total, Page: page, Size: size})
}
