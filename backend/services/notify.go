package services

import (
	"github.com/tdy-manager/backend/models"
	"gorm.io/gorm"
)

func Notify(db *gorm.DB, userID uint, title, content, link string) {
	_ = db.Create(&models.Notification{
		UserID:  userID,
		Title:   title,
		Content: content,
		Link:    link,
	}).Error
}

func NotifyAdmins(db *gorm.DB, title, content, link string) {
	var admins []models.User
	db.Where("role = ? AND enabled = ?", models.RoleAdmin, true).Find(&admins)
	for _, a := range admins {
		Notify(db, a.ID, title, content, link)
	}
}

func Audit(db *gorm.DB, actorID uint, actorName, action, targetType string, targetID uint, detail, ip string) {
	_ = db.Create(&models.AuditLog{
		ActorID:    actorID,
		ActorName:  actorName,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
		IP:         ip,
	}).Error
}

func WriteAudit(db *gorm.DB, c interface {
	ClientIP() string
	Get(string) (interface{}, bool)
}, action, targetType string, targetID uint, detail string) {
	uid, _ := c.Get("user_id")
	actorID, _ := uid.(uint)
	name, _ := c.Get("username")
	actorName, _ := name.(string)
	Audit(db, actorID, actorName, action, targetType, targetID, detail, c.ClientIP())
}

func MarkOrderOverdue(orders []models.Order) {
	now := Now()
	for i := range orders {
		o := &orders[i]
		if o.Deadline == nil {
			continue
		}
		if o.Status == models.OrderApproved || o.Status == models.OrderCancelled {
			continue
		}
		o.Overdue = o.Deadline.Before(now)
	}
}
