package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        int64          `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"type:varchar(255)" json:"name"`
	Email     string         `gorm:"type:varchar(255);index" json:"email"`
	Username  string         `gorm:"type:varchar(255);uniqueIndex" json:"username"` // Unique index
	Password  string         `gorm:"type:varchar(255)" json:"-"`                    // Not returned to frontend
	CreatedAt time.Time      `json:"created_at"`                                    // Auto timestamp
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // Soft delete support
}
