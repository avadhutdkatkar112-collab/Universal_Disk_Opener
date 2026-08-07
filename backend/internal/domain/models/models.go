// Package models contains shared domain models used across the application.
package models

import "time"

// RecentFile represents a recently opened VHD file.
type RecentFile struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	FilePath     string    `json:"filePath" gorm:"uniqueIndex;not null"`
	FileName     string    `json:"fileName" gorm:"not null"`
	FileSize     int64     `json:" fileSize"`
	DiskType     string    `json:"diskType"`
	DiskSize     int64     `json:"diskSize"`
	CreatedAt    time.Time `json:"createdAt"`
	LastOpenedAt time.Time `json:"lastOpenedAt"`
	OpenedCount  int       `json:"openedCount" gorm:"default:1"`
}

// TableName overrides the table name for GORM.
func (RecentFile) TableName() string {
	return "recent_files"
}
