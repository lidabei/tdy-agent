package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tdy-manager/backend/middleware"
	"github.com/tdy-manager/backend/models"
	"github.com/tdy-manager/backend/services"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type WalletHandler struct {
	DB *gorm.DB
}

type adjustReq struct {
	UserID     uint    `json:"user_id" binding:"required"`
	AmountYuan float64 `json:"amount_yuan" binding:"required"`
	Remark     string  `json:"remark" binding:"required"`
}

func (h *WalletHandler) MyWallet(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var wallet models.Wallet
	if err := h.DB.Where("user_id = ?", uid).First(&wallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "钱包不存在"})
		return
	}
	var pending int64
	h.DB.Model(&models.Withdrawal{}).
		Select("COALESCE(SUM(amount),0)").
		Where("user_id = ? AND status = ?", uid, models.WithdrawPending).
		Scan(&pending)
	c.JSON(http.StatusOK, gin.H{
		"id":              wallet.ID,
		"user_id":         wallet.UserID,
		"balance":         wallet.Balance,
		"pending_withdraw": pending,
		"available":       wallet.Balance - pending,
		"updated_at":      wallet.UpdatedAt,
	})
}

func (h *WalletHandler) MyTransactions(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	page, size := parsePage(c)
	q := h.DB.Model(&models.WalletTx{}).Where("user_id = ?", uid)
	var total int64
	q.Count(&total)
	var txs []models.WalletTx
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&txs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PageResult{List: txs, Total: total, Page: page, Size: size})
}

func (h *WalletHandler) AdminList(c *gin.Context) {
	type row struct {
		UserID      uint   `json:"user_id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Enabled     bool   `json:"enabled"`
		Balance     int64  `json:"balance"`
	}
	page, size := parsePage(c)
	base := h.DB.Table("wallets").
		Select("wallets.user_id, users.username, users.display_name, users.enabled, wallets.balance").
		Joins("JOIN users ON users.id = wallets.user_id").
		Where("users.role = ?", models.RoleUser)
	var total int64
	base.Count(&total)
	var rows []row
	err := base.Order("wallets.user_id asc").Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PageResult{List: rows, Total: total, Page: page, Size: size})
}

func (h *WalletHandler) AdminTransactions(c *gin.Context) {
	page, size := parsePage(c)
	q := h.DB.Model(&models.WalletTx{}).Preload("Order")
	if uid := c.Query("user_id"); uid != "" {
		q = q.Where("user_id = ?", uid)
	}
	var total int64
	q.Count(&total)
	var txs []models.WalletTx
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&txs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PageResult{List: txs, Total: total, Page: page, Size: size})
}

func (h *WalletHandler) Adjust(c *gin.Context) {
	var req adjustReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	amount := yuanToFen(req.AmountYuan)
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		return services.AdjustWallet(tx, req.UserID, amount, "管理员调账: "+req.Remark)
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(h.DB, middleware.CurrentUserID(c), actorName(c), "wallet.adjust", "user", req.UserID, req.Remark, c.ClientIP())
	var wallet models.Wallet
	h.DB.Where("user_id = ?", req.UserID).First(&wallet)
	c.JSON(http.StatusOK, wallet)
}

func (h *WalletHandler) Reconcile(c *gin.Context) {
	type userRec struct {
		UserID         uint   `json:"user_id"`
		Username       string `json:"username"`
		DisplayName    string `json:"display_name"`
		WalletBalance  int64  `json:"wallet_balance"`
		ApprovedReward int64  `json:"approved_reward"`
		CreditTotal    int64  `json:"credit_total"`
		DebitTotal     int64  `json:"debit_total"`
		Diff           int64  `json:"diff"`
		OK             bool   `json:"ok"`
	}

	var users []models.User
	if err := h.DB.Where("role = ?", models.RoleUser).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]userRec, 0, len(users))
	var totalBalance, totalApproved, totalCredit, totalDebit int64

	for _, u := range users {
		var wallet models.Wallet
		_ = h.DB.Where("user_id = ?", u.ID).First(&wallet).Error

		var approved int64
		h.DB.Model(&models.Order{}).
			Select("COALESCE(SUM(reward),0)").
			Where("assignee_id = ? AND status = ?", u.ID, models.OrderApproved).
			Scan(&approved)

		var credit, debit int64
		h.DB.Model(&models.WalletTx{}).
			Select("COALESCE(SUM(amount),0)").
			Where("user_id = ? AND type = ?", u.ID, models.TxCredit).
			Scan(&credit)
		h.DB.Model(&models.WalletTx{}).
			Select("COALESCE(SUM(amount),0)").
			Where("user_id = ? AND type = ?", u.ID, models.TxDebit).
			Scan(&debit)

		diff := wallet.Balance - (credit - debit)
		result = append(result, userRec{
			UserID: u.ID, Username: u.Username, DisplayName: u.DisplayName,
			WalletBalance: wallet.Balance, ApprovedReward: approved,
			CreditTotal: credit, DebitTotal: debit, Diff: diff, OK: diff == 0,
		})
		totalBalance += wallet.Balance
		totalApproved += approved
		totalCredit += credit
		totalDebit += debit
	}

	c.JSON(http.StatusOK, gin.H{
		"generated_at": time.Now(),
		"summary": gin.H{
			"total_balance":  totalBalance,
			"total_approved": totalApproved,
			"total_credit":   totalCredit,
			"total_debit":    totalDebit,
			"ledger_ok":      totalBalance == (totalCredit - totalDebit),
		},
		"users": result,
	})
}

func (h *WalletHandler) ExportReconcile(c *gin.Context) {
	var users []models.User
	h.DB.Where("role = ?", models.RoleUser).Find(&users)

	f := excelize.NewFile()
	sheet := "对账"
	_ = f.SetSheetName("Sheet1", sheet)
	headers := []string{"用户ID", "用户名", "昵称", "余额(元)", "验收奖励(元)", "入账(元)", "扣减(元)", "差额(元)", "是否平衡"}
	for i, hname := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, hname)
	}
	for i, u := range users {
		var wallet models.Wallet
		_ = h.DB.Where("user_id = ?", u.ID).First(&wallet).Error
		var approved, credit, debit int64
		h.DB.Model(&models.Order{}).Select("COALESCE(SUM(reward),0)").
			Where("assignee_id = ? AND status = ?", u.ID, models.OrderApproved).Scan(&approved)
		h.DB.Model(&models.WalletTx{}).Select("COALESCE(SUM(amount),0)").
			Where("user_id = ? AND type = ?", u.ID, models.TxCredit).Scan(&credit)
		h.DB.Model(&models.WalletTx{}).Select("COALESCE(SUM(amount),0)").
			Where("user_id = ? AND type = ?", u.ID, models.TxDebit).Scan(&debit)
		diff := wallet.Balance - (credit - debit)
		ok := "是"
		if diff != 0 {
			ok = "否"
		}
		row := i + 2
		vals := []interface{}{
			u.ID, u.Username, u.DisplayName,
			float64(wallet.Balance) / 100, float64(approved) / 100,
			float64(credit) / 100, float64(debit) / 100, float64(diff) / 100, ok,
		}
		for col, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=reconcile.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}
