package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdy-manager/backend/models"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	DB *gorm.DB
}

func (h *DashboardHandler) Admin(c *gin.Context) {
	var pendingReview, pendingWithdraw, activeUsers, openOrders int64
	h.DB.Model(&models.Order{}).Where("status = ?", models.OrderSubmitted).Count(&pendingReview)
	h.DB.Model(&models.Withdrawal{}).Where("status = ?", models.WithdrawPending).Count(&pendingWithdraw)
	h.DB.Model(&models.User{}).Where("role = ? AND enabled = ?", models.RoleUser, true).Count(&activeUsers)
	h.DB.Model(&models.Order{}).Where("status IN ?", []string{
		models.OrderPending, models.OrderAccepted, models.OrderSubmitted, models.OrderRejected,
	}).Count(&openOrders)

	start := time.Now().AddDate(0, 0, -time.Now().Day()+1).Truncate(24 * time.Hour)
	var monthCredit, monthDebit int64
	h.DB.Model(&models.WalletTx{}).Select("COALESCE(SUM(amount),0)").
		Where("type = ? AND created_at >= ?", models.TxCredit, start).Scan(&monthCredit)
	h.DB.Model(&models.WalletTx{}).Select("COALESCE(SUM(amount),0)").
		Where("type = ? AND created_at >= ?", models.TxDebit, start).Scan(&monthDebit)

	var overdue int64
	h.DB.Model(&models.Order{}).
		Where("deadline IS NOT NULL AND deadline < ? AND status NOT IN ?", time.Now(), []string{models.OrderApproved, models.OrderCancelled}).
		Count(&overdue)

	c.JSON(http.StatusOK, gin.H{
		"pending_review":    pendingReview,
		"pending_withdraw":  pendingWithdraw,
		"active_users":      activeUsers,
		"open_orders":       openOrders,
		"overdue_orders":    overdue,
		"month_credit":      monthCredit,
		"month_debit":       monthDebit,
		"month_start":       start,
	})
}
