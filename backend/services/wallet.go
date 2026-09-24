package services

import (
	"fmt"
	"time"

	"github.com/tdy-manager/backend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreditWallet 验收通过后给用户入账（事务内调用，幂等键防双入账）
func CreditWallet(tx *gorm.DB, userID uint, orderID uint, amount int64, remark string) error {
	if amount <= 0 {
		return fmt.Errorf("入账金额必须大于 0")
	}
	key := fmt.Sprintf("order:%d:credit", orderID)
	var exists int64
	if err := tx.Model(&models.WalletTx{}).Where("idempotent_key = ?", key).Count(&exists).Error; err != nil {
		return err
	}
	if exists > 0 {
		return fmt.Errorf("该工单已入账，请勿重复操作")
	}

	var wallet models.Wallet
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return err
	}
	wallet.Balance += amount
	if err := tx.Save(&wallet).Error; err != nil {
		return err
	}
	return tx.Create(&models.WalletTx{
		UserID:        userID,
		OrderID:       &orderID,
		Type:          models.TxCredit,
		Amount:        amount,
		Balance:       wallet.Balance,
		Remark:        remark,
		IdempotentKey: key,
	}).Error
}

// DebitWallet 扣减余额（提现通过等）
func DebitWallet(tx *gorm.DB, userID uint, amount int64, remark, idempotentKey string, withdrawalID *uint) error {
	if amount <= 0 {
		return fmt.Errorf("扣减金额必须大于 0")
	}
	if idempotentKey != "" {
		var exists int64
		if err := tx.Model(&models.WalletTx{}).Where("idempotent_key = ?", idempotentKey).Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			return fmt.Errorf("该扣款已处理，请勿重复操作")
		}
	}

	var wallet models.Wallet
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return err
	}
	if wallet.Balance < amount {
		return fmt.Errorf("余额不足")
	}
	wallet.Balance -= amount
	if err := tx.Save(&wallet).Error; err != nil {
		return err
	}
	return tx.Create(&models.WalletTx{
		UserID:        userID,
		WithdrawalID:  withdrawalID,
		Type:          models.TxDebit,
		Amount:        amount,
		Balance:       wallet.Balance,
		Remark:        remark,
		IdempotentKey: idempotentKey,
	}).Error
}

// AdjustWallet 管理员手动调账
func AdjustWallet(tx *gorm.DB, userID uint, amount int64, remark string) error {
	if amount == 0 {
		return fmt.Errorf("调整金额不能为 0")
	}
	var wallet models.Wallet
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return err
	}
	newBalance := wallet.Balance + amount
	if newBalance < 0 {
		return fmt.Errorf("余额不足")
	}
	wallet.Balance = newBalance
	if err := tx.Save(&wallet).Error; err != nil {
		return err
	}
	txType := models.TxCredit
	abs := amount
	if amount < 0 {
		txType = models.TxDebit
		abs = -amount
	}
	return tx.Create(&models.WalletTx{
		UserID:        userID,
		Type:          txType,
		Amount:        abs,
		Balance:       wallet.Balance,
		Remark:        remark,
		IdempotentKey: fmt.Sprintf("adjust:%d:%d", userID, time.Now().UnixNano()),
	}).Error
}
