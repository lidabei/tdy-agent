package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdy-manager/backend/middleware"
	"github.com/tdy-manager/backend/models"
	"github.com/tdy-manager/backend/services"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WithdrawHandler struct {
	DB              *gorm.DB
	MaxFen          int64
	DailyMaxFen     int64
}

type withdrawReq struct {
	AmountYuan  float64 `json:"amount_yuan" binding:"required,gt=0"`
	AccountInfo string  `json:"account_info" binding:"required"`
	UserRemark  string  `json:"user_remark"`
}

type withdrawReviewReq struct {
	AdminRemark string `json:"admin_remark"`
}

type batchIDsReq struct {
	IDs         []uint `json:"ids" binding:"required,min=1"`
	AdminRemark string `json:"admin_remark"`
}

func (h *WithdrawHandler) Apply(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var req withdrawReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写提现金额和收款信息"})
		return
	}
	amount := yuanToFen(req.AmountYuan)
	if amount < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "提现金额至少 0.01 元"})
		return
	}
	if h.MaxFen > 0 && amount > h.MaxFen {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("单笔提现不能超过 ¥%.2f", float64(h.MaxFen)/100)})
		return
	}

	var w models.Withdrawal
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var wallet models.Wallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", uid).First(&wallet).Error; err != nil {
			return err
		}
		var pendingSum int64
		if err := tx.Model(&models.Withdrawal{}).
			Select("COALESCE(SUM(amount),0)").
			Where("user_id = ? AND status = ?", uid, models.WithdrawPending).
			Scan(&pendingSum).Error; err != nil {
			return err
		}
		if wallet.Balance < pendingSum+amount {
			return fmt.Errorf("可用余额不足（已占用待审提现 ¥%.2f）", float64(pendingSum)/100)
		}

		dayStart := time.Now().Truncate(24 * time.Hour)
		var daySum int64
		tx.Model(&models.Withdrawal{}).Select("COALESCE(SUM(amount),0)").
			Where("user_id = ? AND created_at >= ? AND status IN ?", uid, dayStart,
				[]string{models.WithdrawPending, models.WithdrawApproved}).Scan(&daySum)
		if h.DailyMaxFen > 0 && daySum+amount > h.DailyMaxFen {
			return fmt.Errorf("今日提现额度不足（上限 ¥%.2f）", float64(h.DailyMaxFen)/100)
		}

		w = models.Withdrawal{
			UserID: uid, Amount: amount, Status: models.WithdrawPending,
			AccountInfo: req.AccountInfo, UserRemark: req.UserRemark,
		}
		return tx.Create(&w).Error
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.NotifyAdmins(h.DB, "新提现待审核",
		fmt.Sprintf("用户 #%d 申请提现 ¥%.2f", uid, float64(amount)/100),
		"/admin/withdrawals")
	c.JSON(http.StatusOK, w)
}

func (h *WithdrawHandler) MyList(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	page, size := parsePage(c)
	q := h.DB.Model(&models.Withdrawal{}).Where("user_id = ?", uid)
	var total int64
	q.Count(&total)
	var list []models.Withdrawal
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PageResult{List: list, Total: total, Page: page, Size: size})
}

func (h *WithdrawHandler) AdminList(c *gin.Context) {
	page, size := parsePage(c)
	q := h.DB.Model(&models.Withdrawal{}).Preload("User")
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	var total int64
	q.Count(&total)
	var list []models.Withdrawal
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PageResult{List: list, Total: total, Page: page, Size: size})
}

func (h *WithdrawHandler) approveOne(tx *gorm.DB, id uint, adminID uint, remark string) error {
	var w models.Withdrawal
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&w, id).Error; err != nil {
		return err
	}
	if w.Status != models.WithdrawPending {
		return fmt.Errorf("提现 #%d 状态不可审核", id)
	}
	key := fmt.Sprintf("withdraw:%d:debit", w.ID)
	if err := services.DebitWallet(tx, w.UserID, w.Amount, fmt.Sprintf("提现扣款 #%d", id), key, &w.ID); err != nil {
		return err
	}
	now := time.Now()
	w.Status = models.WithdrawApproved
	w.AdminRemark = remark
	w.ReviewedBy = &adminID
	w.ReviewedAt = &now
	if err := tx.Save(&w).Error; err != nil {
		return err
	}
	services.Notify(tx, w.UserID, "提现已通过", fmt.Sprintf("¥%.2f 已扣款，请查收打款", float64(w.Amount)/100), "/app/wallet")
	return nil
}

func (h *WithdrawHandler) Approve(c *gin.Context) {
	id := c.Param("id")
	var req withdrawReviewReq
	_ = c.ShouldBindJSON(&req)
	adminID := middleware.CurrentUserID(c)
	var wid uint
	fmt.Sscanf(id, "%d", &wid)
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		return h.approveOne(tx, wid, adminID, req.AdminRemark)
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(h.DB, adminID, actorName(c), "withdraw.approve", "withdrawal", wid, req.AdminRemark, c.ClientIP())
	var w models.Withdrawal
	h.DB.Preload("User").First(&w, id)
	c.JSON(http.StatusOK, w)
}

func (h *WithdrawHandler) BatchApprove(c *gin.Context) {
	var req batchIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择提现单"})
		return
	}
	adminID := middleware.CurrentUserID(c)
	ok, fail := 0, 0
	for _, id := range req.IDs {
		err := h.DB.Transaction(func(tx *gorm.DB) error {
			return h.approveOne(tx, id, adminID, req.AdminRemark)
		})
		if err != nil {
			fail++
		} else {
			ok++
		}
	}
	services.Audit(h.DB, adminID, actorName(c), "withdraw.batch_approve", "withdrawal", 0,
		fmt.Sprintf("ok=%d fail=%d", ok, fail), c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"ok": ok, "fail": fail})
}

func (h *WithdrawHandler) Reject(c *gin.Context) {
	id := c.Param("id")
	var req withdrawReviewReq
	if err := c.ShouldBindJSON(&req); err != nil || req.AdminRemark == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写拒绝原因"})
		return
	}
	adminID := middleware.CurrentUserID(c)
	var w models.Withdrawal
	if err := h.DB.First(&w, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "提现单不存在"})
		return
	}
	if w.Status != models.WithdrawPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅待审核提现可拒绝"})
		return
	}
	now := time.Now()
	w.Status = models.WithdrawRejected
	w.AdminRemark = req.AdminRemark
	w.ReviewedBy = &adminID
	w.ReviewedAt = &now
	if err := h.DB.Save(&w).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.Notify(h.DB, w.UserID, "提现已拒绝", req.AdminRemark, "/app/wallet")
	services.Audit(h.DB, adminID, actorName(c), "withdraw.reject", "withdrawal", w.ID, req.AdminRemark, c.ClientIP())
	c.JSON(http.StatusOK, w)
}
