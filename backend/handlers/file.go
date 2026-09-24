package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdy-manager/backend/middleware"
	"github.com/tdy-manager/backend/models"
	"gorm.io/gorm"
)

type FileHandler struct {
	DB        *gorm.DB
	UploadDir string
}

func (h *FileHandler) UploadOrderFile(c *gin.Context) {
	orderID, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.CurrentUserID(c)
	role, _ := c.Get("role")

	var order models.Order
	if err := h.DB.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	if role != models.RoleAdmin && order.AssigneeID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权上传"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择文件"})
		return
	}
	if file.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件不能超过 20MB"})
		return
	}

	_ = os.MkdirAll(h.UploadDir, 0o755)
	ext := filepath.Ext(file.Filename)
	stored := fmt.Sprintf("%d_%d_%d%s", orderID, uid, time.Now().UnixNano(), ext)
	dst := filepath.Join(h.UploadDir, stored)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	att := models.OrderAttachment{
		OrderID:     uint(orderID),
		UploaderID:  uid,
		FileName:    file.Filename,
		StoredName:  stored,
		Size:        file.Size,
		ContentType: file.Header.Get("Content-Type"),
	}
	if err := h.DB.Create(&att).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, att)
}

func (h *FileHandler) Download(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var att models.OrderAttachment
	if err := h.DB.First(&att, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	path := filepath.Join(h.UploadDir, att.StoredName)
	if _, err := os.Stat(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件已丢失"})
		return
	}
	c.FileAttachment(path, att.FileName)
}

func (h *FileHandler) ListByOrder(c *gin.Context) {
	orderID, _ := strconv.Atoi(c.Param("id"))
	var list []models.OrderAttachment
	h.DB.Where("order_id = ?", orderID).Order("id asc").Find(&list)
	c.JSON(http.StatusOK, list)
}

func safeName(name string) string {
	name = strings.ReplaceAll(name, "..", "")
	return filepath.Base(name)
}
