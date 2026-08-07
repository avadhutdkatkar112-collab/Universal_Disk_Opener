package services

import (
	"time"

	"github.com/user/vhd-opener/internal/domain/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// RecentFilesService manages the recent files database.
type RecentFilesService struct {
	db *gorm.DB
}

// NewRecentFilesService creates a new recent files service.
func NewRecentFilesService(dbPath string) (*RecentFilesService, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// Auto migrate
	if err := db.AutoMigrate(&models.RecentFile{}); err != nil {
		return nil, err
	}

	return &RecentFilesService{db: db}, nil
}

// AddOrUpdate adds or updates a recent file entry.
func (s *RecentFilesService) AddOrUpdate(entry *models.RecentFile) error {
	var existing models.RecentFile
	result := s.db.Where("file_path = ?", entry.FilePath).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		entry.LastOpenedAt = time.Now()
		entry.CreatedAt = time.Now()
		entry.OpenedCount = 1
		return s.db.Create(entry).Error
	}

	existing.LastOpenedAt = time.Now()
	existing.OpenedCount++
	return s.db.Save(&existing).Error
}

// GetRecent returns recent files, most recently opened first.
func (s *RecentFilesService) GetRecent(limit int) ([]models.RecentFile, error) {
	var files []models.RecentFile
	result := s.db.Order("last_opened_at DESC").Limit(limit).Find(&files)
	return files, result.Error
}

// Delete removes a recent file entry by ID.
func (s *RecentFilesService) Delete(id int64) error {
	return s.db.Delete(&models.RecentFile{}, id).Error
}

// Clear removes all recent file entries.
func (s *RecentFilesService) Clear() error {
	return s.db.Exec("DELETE FROM recent_files").Error
}

// Close closes the database connection.
func (s *RecentFilesService) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
