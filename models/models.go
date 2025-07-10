package models

import (
	"time"
)

type User struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Username  string `gorm:"uniqueIndex;size:50"`
	Status    int    `gorm:"default:1"`
}

type Bucket struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	BucketID  string `gorm:"uniqueIndex;size:32"`
	UserID    uint   `gorm:"index"`
	Partition string `gorm:"size:20"`
	Storage   int64  `gorm:"default:0"`
}

type BucketFile struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	FileName  string `gorm:"size:255;index"`
	BucketID  string `gorm:"size:32;index"`
	FileSize  int64  `gorm:"default:0"`
	FileHash  string `gorm:"size:64;index"`
}
