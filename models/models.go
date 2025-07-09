package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex"`
}

type Bucket struct {
	gorm.Model
	BucketID  string `gorm:"uniqueIndex"`
	UserID    uint
	Partition string
}

type BucketFile struct {
	gorm.Model
	FileName string
	BucketID string
}

import "fmt"

func (b BucketFile) TableName() string {
    return fmt.Sprintf("bucket_files_%s", services.CalculatePartition(b.BucketID))
}
