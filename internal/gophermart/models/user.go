package models

import (
	"time"

	"gorm.io/gorm"
)

type UserID uint

type User struct {
	ID           UserID         `json:"id" gorm:"primaryKey"`
	Login        string         `json:"login" gorm:"uniqueIndex;not null"`
	PasswordHash string         `json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"-"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}
