package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"unique;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	FullName     string    `gorm:"not null" json:"full_name"`
	Phone        string    `json:"phone"`
	AvatarURL    string    `json:"avatar_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ExpenseGroup struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	InviteCode  string    `gorm:"unique;not null" json:"invite_code"`
	CreatedBy   uint      `gorm:"not null" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GroupMember struct {
	ID       uint         `gorm:"primaryKey" json:"id"`
	GroupID  uint         `gorm:"not null;index" json:"group_id"`
	UserID   uint         `gorm:"not null;index" json:"user_id"`
	JoinedAt time.Time    `json:"joined_at"`
	Group    ExpenseGroup `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	User     User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type UserRole struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GroupID   uint      `gorm:"not null;index" json:"group_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Role      string    `gorm:"not null" json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type ExpenseRequest struct {
	ID              uint                `gorm:"primaryKey" json:"id"`
	GroupID         uint                `gorm:"not null;index" json:"group_id"`
	RequesterID     uint                `gorm:"not null;index" json:"requester_id"`
	Title           string              `gorm:"not null" json:"title"`
	Category        string              `gorm:"not null" json:"category"`
	Amount          float64             `gorm:"not null" json:"amount"`
	Description     string              `json:"description"`
	Status          string              `gorm:"default:'pending'" json:"status"` // pending, approved, rejected
	ApprovedBy      *uint               `json:"approved_by"`
	ApprovedAt      *time.Time          `json:"approved_at"`
	RejectionReason string              `json:"rejection_reason"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	TargetUserID    *uint               `json:"target_user_id"` // Specific approver (optional)
	TargetUser      *User               `gorm:"foreignKey:TargetUserID" json:"target_user,omitempty"`
	Requester       User                `gorm:"foreignKey:RequesterID" json:"requester,omitempty"`
	Attachments     []ExpenseAttachment `gorm:"foreignKey:ExpenseID" json:"attachments,omitempty"`
	ApprovalSlips   []ApprovalSlip      `gorm:"foreignKey:ExpenseID" json:"approval_slips,omitempty"`
}

type ExpenseAttachment struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ExpenseID  uint      `gorm:"not null;index" json:"expense_id"`
	FileName   string    `gorm:"not null" json:"file_name"`
	FilePath   string    `gorm:"not null" json:"file_path"`
	FileSize   int64     `json:"file_size"`
	FileType   string    `json:"file_type"`
	UploadedBy uint      `gorm:"not null" json:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type ApprovalSlip struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ExpenseID  uint      `gorm:"not null;index" json:"expense_id"`
	FileName   string    `gorm:"not null" json:"file_name"`
	FilePath   string    `gorm:"not null" json:"file_path"`
	FileSize   int64     `json:"file_size"`
	FileType   string    `json:"file_type"`
	Notes      string    `json:"notes"`
	IsVerified bool      `json:"is_verified"`
	SlipOKData string    `gorm:"type:text" json:"slipok_data"` // Storing JSON as text for simplicity
	UploadedBy uint      `gorm:"not null" json:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at"`
}

func Migrate(db *gorm.DB) {
	db.AutoMigrate(
		&User{},
		&ExpenseGroup{},
		&GroupMember{},
		&UserRole{},
		&ExpenseRequest{},
		&ExpenseAttachment{},
		&ApprovalSlip{},
	)
}
