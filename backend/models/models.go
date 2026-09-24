package models

import "time"

const SchemaVersion = "4"

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// 管理端角色编码
const (
	AdminRoleSuper   = "super_admin"
	AdminRoleOps     = "ops"
	AdminRoleFinance = "finance"
	AdminRoleAuditor = "auditor"
)

// 权限点
const (
	PermDashboardView   = "dashboard.view"
	PermOrderView       = "order.view"
	PermOrderCreate     = "order.create"
	PermOrderApprove    = "order.approve"
	PermOrderReassign   = "order.reassign"
	PermOrderCancel     = "order.cancel"
	PermUserView        = "user.view"
	PermUserCreate      = "user.create"
	PermUserManage      = "user.manage"
	PermWalletView      = "wallet.view"
	PermWalletAdjust    = "wallet.adjust"
	PermWithdrawView    = "withdraw.view"
	PermWithdrawReview  = "withdraw.review"
	PermReconcileView   = "reconcile.view"
	PermReconcileExport = "reconcile.export"
	PermAuditView       = "audit.view"
	PermRoleManage      = "role.manage"
	PermAdminManage     = "admin.manage"
)

const (
	OrderPending   = "pending"
	OrderAccepted  = "accepted"
	OrderSubmitted = "submitted"
	OrderApproved  = "approved"
	OrderRejected  = "rejected"
	OrderCancelled = "cancelled"
)

const (
	TxCredit = "credit"
	TxDebit  = "debit"
)

const (
	WithdrawPending  = "pending"
	WithdrawApproved = "approved"
	WithdrawRejected = "rejected"
)

type SchemaMeta struct {
	ID      uint   `gorm:"primaryKey"`
	Version string `gorm:"size:32;not null"`
}

type Permission struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Code      string `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name      string `gorm:"size:64;not null" json:"name"`
	Module    string `gorm:"size:32;not null;index" json:"module"`
	SortOrder int    `gorm:"not null;default:0" json:"sort_order"`
}

type AdminRole struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	Code        string       `gorm:"size:32;uniqueIndex;not null" json:"code"`
	Name        string       `gorm:"size:64;not null" json:"name"`
	Description string       `gorm:"size:255" json:"description"`
	IsSystem    bool         `gorm:"not null;default:false" json:"is_system"`
	Permissions []Permission `gorm:"many2many:admin_role_permissions;" json:"permissions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func (AdminRole) TableName() string { return "admin_roles" }

type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	DisplayName  string     `gorm:"size:64;not null" json:"display_name"`
	Role         string     `gorm:"size:16;not null;index" json:"role"` // admin | user（门户）
	AdminRoleID  *uint      `gorm:"index" json:"admin_role_id,omitempty"`
	Enabled      bool       `gorm:"not null;default:true;index" json:"enabled"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	AdminRole    *AdminRole `gorm:"foreignKey:AdminRoleID" json:"admin_role,omitempty"`
}

type Wallet struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	Balance   int64     `gorm:"not null;default:0" json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type WalletTx struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index;not null" json:"user_id"`
	OrderID       *uint     `gorm:"index" json:"order_id,omitempty"`
	WithdrawalID  *uint     `gorm:"index" json:"withdrawal_id,omitempty"`
	Type          string    `gorm:"size:16;not null" json:"type"`
	Amount        int64     `gorm:"not null" json:"amount"`
	Balance       int64     `gorm:"not null" json:"balance"`
	Remark        string    `gorm:"size:255" json:"remark"`
	IdempotentKey string    `gorm:"size:64;uniqueIndex" json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	Order         *Order    `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}

func (WalletTx) TableName() string { return "wallet_txs" }

type Order struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Title        string     `gorm:"size:128;not null" json:"title"`
	Description  string     `gorm:"type:text" json:"description"`
	Reward       int64      `gorm:"not null" json:"reward"`
	Status       string     `gorm:"size:16;not null;index:idx_orders_assignee_status,priority:2;index:idx_orders_status_created,priority:1" json:"status"`
	AssigneeID   uint       `gorm:"index:idx_orders_assignee_status,priority:1;not null" json:"assignee_id"`
	CreatedByID  uint       `gorm:"index;not null" json:"created_by_id"`
	SubmitNote   string     `gorm:"type:text" json:"submit_note"`
	RejectReason string     `gorm:"size:255" json:"reject_reason"`
	Deadline     *time.Time `gorm:"index" json:"deadline,omitempty"`
	AcceptedAt   *time.Time `json:"accepted_at,omitempty"`
	SubmittedAt  *time.Time `json:"submitted_at,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt    time.Time  `gorm:"index:idx_orders_status_created,priority:2" json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	Assignee    *User            `gorm:"foreignKey:AssigneeID" json:"assignee,omitempty"`
	CreatedBy   *User            `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
	Attachments []OrderAttachment `gorm:"foreignKey:OrderID" json:"attachments,omitempty"`
	Overdue     bool             `gorm:"-" json:"overdue"`
}

type OrderAttachment struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	OrderID      uint      `gorm:"index;not null" json:"order_id"`
	UploaderID   uint      `gorm:"index;not null" json:"uploader_id"`
	FileName     string    `gorm:"size:255;not null" json:"file_name"`
	StoredName   string    `gorm:"size:255;not null" json:"stored_name"`
	Size         int64     `json:"size"`
	ContentType  string    `gorm:"size:128" json:"content_type"`
	CreatedAt    time.Time `json:"created_at"`
}

type Withdrawal struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"index;not null" json:"user_id"`
	Amount      int64      `gorm:"not null" json:"amount"`
	Status      string     `gorm:"size:16;not null;index" json:"status"`
	AccountInfo string     `gorm:"size:255;not null" json:"account_info"`
	UserRemark  string     `gorm:"size:255" json:"user_remark"`
	AdminRemark string     `gorm:"size:255" json:"admin_remark"`
	ReviewedBy  *uint      `gorm:"index" json:"reviewed_by,omitempty"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	User     *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Reviewer *User `gorm:"foreignKey:ReviewedBy" json:"reviewer,omitempty"`
}

type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index:idx_notif_user_read,priority:1;not null" json:"user_id"`
	Title     string    `gorm:"size:128;not null" json:"title"`
	Content   string    `gorm:"size:512" json:"content"`
	Link      string    `gorm:"size:255" json:"link"`
	Read      bool      `gorm:"not null;default:false;index:idx_notif_user_read,priority:2" json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ActorID    uint      `gorm:"index;not null" json:"actor_id"`
	ActorName  string    `gorm:"size:64" json:"actor_name"`
	Action     string    `gorm:"size:64;index;not null" json:"action"`
	TargetType string    `gorm:"size:32;index" json:"target_type"`
	TargetID   uint      `gorm:"index" json:"target_id"`
	Detail     string    `gorm:"size:512" json:"detail"`
	IP         string    `gorm:"size:64" json:"ip"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}
