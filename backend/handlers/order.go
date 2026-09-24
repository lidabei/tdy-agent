package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdy-manager/backend/middleware"
	"github.com/tdy-manager/backend/models"
	"github.com/tdy-manager/backend/services"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrderHandler struct {
	DB *gorm.DB
}

type createOrderReq struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	RewardYuan  float64 `json:"reward_yuan" binding:"required,gt=0"`
	AssigneeID  uint    `json:"assignee_id" binding:"required"`
	Deadline    string  `json:"deadline"` // RFC3339 or YYYY-MM-DD
}

type batchCreateReq struct {
	Title        string  `json:"title" binding:"required"`
	Description  string  `json:"description"`
	RewardYuan   float64 `json:"reward_yuan" binding:"required,gt=0"`
	AssigneeIDs  []uint  `json:"assignee_ids" binding:"required,min=1"`
	Deadline     string  `json:"deadline"`
}

type reassignReq struct {
	AssigneeID uint `json:"assignee_id" binding:"required"`
}

type submitReq struct {
	Note string `json:"note"`
}

type rejectReq struct {
	Reason string `json:"reason" binding:"required"`
}

func parseDeadline(s string) *time.Time {
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			if layout == "2006-01-02" {
				t = t.Add(23*time.Hour + 59*time.Minute)
			}
			return &t
		}
	}
	return nil
}

func actorName(c *gin.Context) string {
	v, _ := c.Get("username")
	s, _ := v.(string)
	return s
}

func (h *OrderHandler) AdminList(c *gin.Context) {
	page, size := parsePage(c)
	q := h.DB.Model(&models.Order{}).Preload("Assignee").Preload("CreatedBy").Preload("Attachments")
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	if c.Query("overdue") == "1" {
		q = q.Where("deadline IS NOT NULL AND deadline < ? AND status NOT IN ?", time.Now(), []string{models.OrderApproved, models.OrderCancelled})
	}
	var total int64
	q.Count(&total)
	var orders []models.Order
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.MarkOrderOverdue(orders)
	c.JSON(http.StatusOK, PageResult{List: orders, Total: total, Page: page, Size: size})
}

func (h *OrderHandler) Create(c *gin.Context) {
	var req createOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：标题、金额、指定用户必填"})
		return
	}
	var assignee models.User
	if err := h.DB.Where("id = ? AND role = ? AND enabled = ?", req.AssigneeID, models.RoleUser, true).
		First(&assignee).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "指定用户不存在、已禁用或不是普通用户"})
		return
	}
	order := models.Order{
		Title:       req.Title,
		Description: req.Description,
		Reward:      yuanToFen(req.RewardYuan),
		Status:      models.OrderPending,
		AssigneeID:  req.AssigneeID,
		CreatedByID: middleware.CurrentUserID(c),
		Deadline:    parseDeadline(req.Deadline),
	}
	if err := h.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.Notify(h.DB, order.AssigneeID, "新工单待接单", order.Title, "/app/orders")
	services.Audit(h.DB, middleware.CurrentUserID(c), actorName(c), "order.create", "order", order.ID, order.Title, c.ClientIP())
	h.DB.Preload("Assignee").Preload("CreatedBy").First(&order, order.ID)
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) BatchCreate(c *gin.Context) {
	var req batchCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写标题、金额和至少一个用户"})
		return
	}
	deadline := parseDeadline(req.Deadline)
	created := make([]models.Order, 0, len(req.AssigneeIDs))
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		for _, aid := range req.AssigneeIDs {
			var u models.User
			if err := tx.Where("id = ? AND role = ? AND enabled = ?", aid, models.RoleUser, true).First(&u).Error; err != nil {
				return fmt.Errorf("用户 %d 无效", aid)
			}
			o := models.Order{
				Title: req.Title, Description: req.Description,
				Reward: yuanToFen(req.RewardYuan), Status: models.OrderPending,
				AssigneeID: aid, CreatedByID: middleware.CurrentUserID(c), Deadline: deadline,
			}
			if err := tx.Create(&o).Error; err != nil {
				return err
			}
			created = append(created, o)
			services.Notify(tx, aid, "新工单待接单", o.Title, "/app/orders")
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(h.DB, middleware.CurrentUserID(c), actorName(c), "order.batch_create", "order", 0,
		fmt.Sprintf("%s x%d", req.Title, len(created)), c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"count": len(created), "orders": created})
}

func (h *OrderHandler) Reassign(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req reassignReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请指定新用户"})
		return
	}
	var order models.Order
	if err := h.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	if order.Status == models.OrderApproved || order.Status == models.OrderCancelled || order.Status == models.OrderSubmitted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前状态不可改派"})
		return
	}
	var u models.User
	if err := h.DB.Where("id = ? AND role = ? AND enabled = ?", req.AssigneeID, models.RoleUser, true).First(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "目标用户无效"})
		return
	}
	old := order.AssigneeID
	order.AssigneeID = req.AssigneeID
	order.Status = models.OrderPending
	order.AcceptedAt = nil
	if err := h.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.Notify(h.DB, req.AssigneeID, "工单已改派给你", order.Title, "/app/orders")
	if old != req.AssigneeID {
		services.Notify(h.DB, old, "工单已改派给他人", order.Title, "/app/orders")
	}
	services.Audit(h.DB, middleware.CurrentUserID(c), actorName(c), "order.reassign", "order", order.ID,
		fmt.Sprintf("%d -> %d", old, req.AssigneeID), c.ClientIP())
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) Cancel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var order models.Order
	if err := h.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	if order.Status != models.OrderPending && order.Status != models.OrderAccepted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前状态不可取消"})
		return
	}
	order.Status = models.OrderCancelled
	if err := h.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.Notify(h.DB, order.AssigneeID, "工单已取消", order.Title, "/app/orders")
	services.Audit(h.DB, middleware.CurrentUserID(c), actorName(c), "order.cancel", "order", order.ID, order.Title, c.ClientIP())
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) Approve(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var o models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, id).Error; err != nil {
			return err
		}
		if o.Status != models.OrderSubmitted {
			return errors.New("仅已提交工单可验收")
		}
		now := time.Now()
		o.Status = models.OrderApproved
		o.ReviewedAt = &now
		if err := tx.Save(&o).Error; err != nil {
			return err
		}
		if err := services.CreditWallet(tx, o.AssigneeID, o.ID, o.Reward, "工单验收入账: "+o.Title); err != nil {
			return err
		}
		services.Notify(tx, o.AssigneeID, "工单已验收通过", fmt.Sprintf("%s，入账 ¥%.2f", o.Title, float64(o.Reward)/100), "/app/wallet")
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(h.DB, middleware.CurrentUserID(c), actorName(c), "order.approve", "order", uint(id), "", c.ClientIP())
	var order models.Order
	h.DB.Preload("Assignee").First(&order, id)
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) Reject(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req rejectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写拒绝原因"})
		return
	}
	var order models.Order
	if err := h.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	if order.Status != models.OrderSubmitted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅已提交工单可拒绝"})
		return
	}
	now := time.Now()
	order.Status = models.OrderRejected
	order.RejectReason = req.Reason
	order.ReviewedAt = &now
	if err := h.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.Notify(h.DB, order.AssigneeID, "工单已退回", order.Title+"："+req.Reason, "/app/orders")
	services.Audit(h.DB, middleware.CurrentUserID(c), actorName(c), "order.reject", "order", order.ID, req.Reason, c.ClientIP())
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) MyList(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	page, size := parsePage(c)
	q := h.DB.Model(&models.Order{}).Preload("CreatedBy").Preload("Attachments").Where("assignee_id = ?", uid)
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	var total int64
	q.Count(&total)
	var orders []models.Order
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.MarkOrderOverdue(orders)
	c.JSON(http.StatusOK, PageResult{List: orders, Total: total, Page: page, Size: size})
}

func (h *OrderHandler) Accept(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.CurrentUserID(c)
	var order models.Order
	if err := h.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	if order.AssigneeID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "不是你的工单"})
		return
	}
	if order.Status != models.OrderPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前状态不可接单"})
		return
	}
	now := time.Now()
	order.Status = models.OrderAccepted
	order.AcceptedAt = &now
	if err := h.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) Submit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.CurrentUserID(c)
	var req submitReq
	_ = c.ShouldBindJSON(&req)
	var order models.Order
	if err := h.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工单不存在"})
		return
	}
	if order.AssigneeID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "不是你的工单"})
		return
	}
	if order.Status != models.OrderAccepted && order.Status != models.OrderRejected {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前状态不可提交"})
		return
	}
	now := time.Now()
	order.Status = models.OrderSubmitted
	order.SubmitNote = req.Note
	order.SubmittedAt = &now
	order.RejectReason = ""
	if err := h.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.Notify(h.DB, order.CreatedByID, "工单待验收", order.Title, "/admin/orders")
	c.JSON(http.StatusOK, order)
}
