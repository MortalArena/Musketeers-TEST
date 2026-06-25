package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// UpgradeManager مدير الترقية
type UpgradeManager struct {
	versions    map[string]*Version
	upgrades    map[string]*Upgrade
	logger      *zap.Logger
	mu          sync.RWMutex
	storage     UpgradeStorage
	eventBus    EventBus
	config      *UpgradeConfig
}

// UpgradeStorage واجهة تخزين الترقية
type UpgradeStorage interface {
	StoreVersion(version *Version) error
	RetrieveVersion(versionID string) (*Version, error)
	StoreUpgrade(upgrade *Upgrade) error
	RetrieveUpgrade(upgradeID string) (*Upgrade, error)
	ListUpgrades(filter UpgradeFilter) ([]*Upgrade, error)
}

// EventBus واجهة ناقل الأحداث
type EventBus interface {
	Publish(event string, data interface{}) error
	Subscribe(event string, handler func(data interface{})) error
}

// UpgradeConfig تكوين الترقية
type UpgradeConfig struct {
	AutoCheck      bool          `json:"auto_check"`
	CheckInterval  time.Duration `json:"check_interval"`
	AutoDownload   bool          `json:"auto_download"`
	AutoInstall    bool          `json:"auto_install"`
	BackupBefore   bool          `json:"backup_before"`
	Channel        string        `json:"channel"` // stable, beta, alpha
}

// Version معلومات الإصدار
type Version struct {
	ID          string                 `json:"id"`
	Major       int                    `json:"major"`
	Minor       int                    `json:"minor"`
	Patch       int                    `json:"patch"`
	PreRelease string                 `json:"pre_release"`
	Build       string                 `json:"build"`
	ReleasedAt  time.Time              `json:"released_at"`
	IsCurrent   bool                   `json:"is_current"`
	Changelog   string                 `json:"changelog"`
	Checksum    string                 `json:"checksum"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Upgrade ترقية
type Upgrade struct {
	ID              string                 `json:"id"`
	VersionID       string                 `json:"version_id"`
	Type            UpgradeType           `json:"type"`
	Status          UpgradeStatus         `json:"status"`
	DownloadURL     string                 `json:"download_url"`
	DownloadPath    string                 `json:"download_path"`
	Size            int64                  `json:"size"`
	Checksum        string                 `json:"checksum"`
	Progress       float64                `json:"progress"`
	StartedAt       time.Time              `json:"started_at"`
	CompletedAt     time.Time              `json:"completed_at,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// UpgradeType نوع الترقية
type UpgradeType string

const (
	UpgradeTypeMajor      UpgradeType = "major"
	UpgradeTypeMinor      UpgradeType = "minor"
	UpgradeTypePatch      UpgradeType = "patch"
	UpgradeTypeBuild      UpgradeType = "build"
	UpgradeTypeHotfix     UpgradeType = "hotfix"
)

// UpgradeStatus حالة الترقية
type UpgradeStatus string

const (
	UpgradeStatusPending    UpgradeStatus = "pending"
	UpgradeStatusDownloading UpgradeStatus = "downloading"
	UpgradeStatusDownloaded UpgradeStatus = "downloaded"
	UpgradeStatusInstalling UpgradeStatus = "installing"
	UpgradeStatusCompleted  UpgradeStatus = "completed"
	UpgradeStatusFailed     UpgradeStatus = "failed"
	UpgradeStatusRolledBack UpgradeStatus = "rolled_back"
)

// UpgradeFilter فلتر الترقية
type UpgradeFilter struct {
	Type       UpgradeType
	Status     UpgradeStatus
	StartTime  time.Time
	EndTime    time.Time
	Limit      int
	Offset     int
}

// NewUpgradeManager ينشئ مدير ترقية جديد
func NewUpgradeManager(logger *zap.Logger, storage UpgradeStorage, eventBus EventBus, config *UpgradeConfig) *UpgradeManager {
	return &UpgradeManager{
		versions: make(map[string]*Version),
		upgrades: make(map[string]*Upgrade),
		logger:   logger,
		storage:  storage,
		eventBus: eventBus,
		config:   config,
	}
}

// RegisterVersion يسجل إصدار جديد
func (um *UpgradeManager) RegisterVersion(version *Version) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	if _, exists := um.versions[version.ID]; exists {
		return fmt.Errorf("version already registered: %s", version.ID)
	}

	version.ReleasedAt = time.Now()
	um.versions[version.ID] = version

	um.logger.Info("تم تسجيل إصدار جديد",
		zap.String("version_id", version.ID),
		zap.Int("major", version.Major),
		zap.Int("minor", version.Minor),
		zap.Int("patch", version.Patch))

	// تخزين الإصدار
	if um.storage != nil {
		if err := um.storage.StoreVersion(version); err != nil {
			um.logger.Error("فشل تخزين الإصدار",
				zap.String("version_id", version.ID),
				zap.Error(err))
		}
	}

	return nil
}

// GetCurrentVersion يحصل على الإصدار الحالي
func (um *UpgradeManager) GetCurrentVersion() (*Version, error) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	for _, version := range um.versions {
		if version.IsCurrent {
			return version, nil
		}
	}

	return nil, fmt.Errorf("no current version found")
}

// SetCurrentVersion يضبط الإصدار الحالي
func (um *UpgradeManager) SetCurrentVersion(versionID string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	version, exists := um.versions[versionID]
	if !exists {
		return fmt.Errorf("version not found: %s", versionID)
	}

	// إزالة العلامة من جميع الإصدارات
	for _, v := range um.versions {
		v.IsCurrent = false
	}

	// تعيين الإصدار الحالي
	version.IsCurrent = true

	um.logger.Info("تم تعيين الإصدار الحالي",
		zap.String("version_id", versionID))

	return nil
}

// GetLatestVersion يحصل على أحدث إصدار
func (um *UpgradeManager) GetLatestVersion() (*Version, error) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	var latest *Version
	for _, version := range um.versions {
		if latest == nil || um.isNewerVersion(version, latest) {
			latest = version
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no versions found")
	}

	return latest, nil
}

// isNewerVersion يتحقق من أن الإصدار أحدث
func (um *UpgradeManager) isNewerVersion(v1, v2 *Version) bool {
	if v1.Major > v2.Major {
		return true
	}
	if v1.Major < v2.Major {
		return false
	}

	if v1.Minor > v2.Minor {
		return true
	}
	if v1.Minor < v2.Minor {
		return false
	}

	if v1.Patch > v2.Patch {
		return true
	}
	if v1.Patch < v2.Patch {
		return false
	}

	return false
}

// CheckForUpdates يتحقق من وجود تحديثات
func (um *UpgradeManager) CheckForUpdates() ([]*Version, error) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	current, err := um.GetCurrentVersion()
	if err != nil {
		return nil, err
	}

	available := make([]*Version, 0)
	for _, version := range um.versions {
		if um.isNewerVersion(version, current) {
			available = append(available, version)
		}
	}

	um.logger.Info("تم التحقق من التحديثات",
		zap.Int("available_updates", len(available)))

	return available, nil
}

// StartUpgrade يبدأ عملية الترقية
func (um *UpgradeManager) StartUpgrade(versionID string) (*Upgrade, error) {
	um.mu.Lock()
	defer um.mu.Unlock()

	version, exists := um.versions[versionID]
	if !exists {
		return nil, fmt.Errorf("version not found: %s", versionID)
	}

	upgradeID := fmt.Sprintf("upgrade_%d", time.Now().UnixNano())

	upgrade := &Upgrade{
		ID:         upgradeID,
		VersionID:  versionID,
		Type:       um.determineUpgradeType(version),
		Status:     UpgradeStatusPending,
		Progress:   0,
		StartedAt:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	um.upgrades[upgradeID] = upgrade

	um.logger.Info("تم بدء عملية الترقية",
		zap.String("upgrade_id", upgradeID),
		zap.String("version_id", versionID),
		zap.String("type", string(upgrade.Type)))

	// بدء عملية الترقية في الخلفية
	go um.performUpgrade(upgrade, version)

	return upgrade, nil
}

// determineUpgradeType يحدد نوع الترقية
func (um *UpgradeManager) determineUpgradeType(version *Version) UpgradeType {
	current, err := um.GetCurrentVersion()
	if err != nil {
		return UpgradeTypePatch
	}

	if version.Major > current.Major {
		return UpgradeTypeMajor
	}
	if version.Minor > current.Minor {
		return UpgradeTypeMinor
	}
	if version.Patch > current.Patch {
		return UpgradeTypePatch
	}

	return UpgradeTypeBuild
}

// performUpgrade ينفذ عملية الترقية
func (um *UpgradeManager) performUpgrade(upgrade *Upgrade, version *Version) {
	downloadURL := upgrade.DownloadURL
	if downloadURL == "" {
		if url, ok := version.Metadata["download_url"].(string); ok {
			downloadURL = url
		} else if url, ok := version.Metadata["url"].(string); ok {
			downloadURL = url
		}
	}

	if downloadURL == "" {
		um.failUpgrade(upgrade, "no download URL available")
		return
	}

	um.mu.Lock()
	upgrade.Status = UpgradeStatusDownloading
	um.mu.Unlock()

	downloadPath := upgrade.DownloadPath
	if downloadPath == "" {
		ext := filepath.Ext(downloadURL)
		if ext == "" {
			ext = ".bin"
		}
		downloadPath = filepath.Join(os.TempDir(), fmt.Sprintf("upgrade_%s%s", version.ID, ext))
	}

	if err := os.MkdirAll(filepath.Dir(downloadPath), 0755); err != nil {
		um.failUpgrade(upgrade, fmt.Sprintf("failed to create download directory: %v", err))
		return
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(downloadPath), "upgrade-tmp-*")
	if err != nil {
		um.failUpgrade(upgrade, fmt.Sprintf("failed to create temp file: %v", err))
		return
	}
	tmpPath := tmpFile.Name()

	resp, err := http.Get(downloadURL)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		um.failUpgrade(upgrade, fmt.Sprintf("failed to download: %v", err))
		return
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		tmpFile.Close()
		os.Remove(tmpPath)
		um.failUpgrade(upgrade, fmt.Sprintf("download returned status: %s", resp.Status))
		return
	}

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmpFile, hash), resp.Body)
	resp.Body.Close()
	tmpFile.Close()

	if err != nil {
		os.Remove(tmpPath)
		um.failUpgrade(upgrade, fmt.Sprintf("failed to write download: %v", err))
		return
	}

	expectedChecksum := upgrade.Checksum
	if expectedChecksum == "" {
		expectedChecksum = version.Checksum
	}
	if expectedChecksum != "" {
		computed := hex.EncodeToString(hash.Sum(nil))
		if computed != expectedChecksum {
			os.Remove(tmpPath)
			um.failUpgrade(upgrade, fmt.Sprintf("checksum mismatch: expected %s, got %s", expectedChecksum, computed))
			return
		}
		um.logger.Info("checksum verified",
			zap.String("upgrade_id", upgrade.ID))
	}

	if err := os.Rename(tmpPath, downloadPath); err != nil {
		if err := copyFile(tmpPath, downloadPath); err != nil {
			os.Remove(tmpPath)
			um.failUpgrade(upgrade, fmt.Sprintf("failed to move downloaded file: %v", err))
			return
		}
	}
	os.Remove(tmpPath)

	um.mu.Lock()
	upgrade.Status = UpgradeStatusDownloaded
	upgrade.DownloadPath = downloadPath
	upgrade.Size = written
	upgrade.Progress = 50
	um.mu.Unlock()

	um.logger.Info("تم تحميل الترقية",
		zap.String("upgrade_id", upgrade.ID),
		zap.String("download_path", downloadPath),
		zap.Int64("size", written))

	// نشر حدث التحميل
	if um.eventBus != nil {
		um.eventBus.Publish("upgrade.downloaded", map[string]interface{}{
			"upgrade_id": upgrade.ID,
			"version_id": version.ID,
			"size":       written,
		})
	}

	um.mu.Lock()
	upgrade.Status = UpgradeStatusInstalling
	um.mu.Unlock()

	if _, err := os.Stat(downloadPath); err != nil {
		um.failUpgrade(upgrade, fmt.Sprintf("downloaded file not found: %v", err))
		return
	}

	um.mu.Lock()
	upgrade.Status = UpgradeStatusCompleted
	upgrade.Progress = 100
	upgrade.CompletedAt = time.Now()
	um.mu.Unlock()

	if err := um.SetCurrentVersion(version.ID); err != nil {
		um.failUpgrade(upgrade, fmt.Sprintf("failed to set current version: %v", err))
		return
	}

	um.logger.Info("تم إكمال الترقية",
		zap.String("upgrade_id", upgrade.ID),
		zap.String("version_id", version.ID))

	// تخزين الترقية
	if um.storage != nil {
		if err := um.storage.StoreUpgrade(upgrade); err != nil {
			um.logger.Error("فشل تخزين الترقية",
				zap.String("upgrade_id", upgrade.ID),
				zap.Error(err))
		}
	}

	// نشر حدث إكمال الترقية
	if um.eventBus != nil {
		um.eventBus.Publish("upgrade.completed", map[string]interface{}{
			"upgrade_id": upgrade.ID,
			"version_id": version.ID,
		})
	}
}

// failUpgrade يضبط حالة الترقية على فاشلة مع رسالة خطأ
func (um *UpgradeManager) failUpgrade(upgrade *Upgrade, errMsg string) {
	um.mu.Lock()
	defer um.mu.Unlock()
	upgrade.Status = UpgradeStatusFailed
	upgrade.ErrorMessage = errMsg
	upgrade.CompletedAt = time.Now()

	um.logger.Error("فشلت الترقية",
		zap.String("upgrade_id", upgrade.ID),
		zap.String("error", errMsg))

	if um.eventBus != nil {
		um.eventBus.Publish("upgrade.failed", map[string]interface{}{
			"upgrade_id": upgrade.ID,
			"error":      errMsg,
		})
	}

	if um.storage != nil {
		if err := um.storage.StoreUpgrade(upgrade); err != nil {
			um.logger.Error("فشل تخزين حالة الترقية الفاشلة",
				zap.String("upgrade_id", upgrade.ID),
				zap.Error(err))
		}
	}
}

// copyFile ينسخ ملف من src إلى dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// RollbackUpgrade يتراجع عن الترقية
func (um *UpgradeManager) RollbackUpgrade(upgradeID string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	upgrade, exists := um.upgrades[upgradeID]
	if !exists {
		return fmt.Errorf("upgrade not found: %s", upgradeID)
	}

	if upgrade.Status != UpgradeStatusCompleted {
		return fmt.Errorf("upgrade is not completed: %s", upgradeID)
	}

	// الحصول على الإصدار السابق
	current, err := um.GetCurrentVersion()
	if err != nil {
		return err
	}

	// إزالة علامة الإصدار الحالي
	current.IsCurrent = false

	// العثور على الإصدار السابق
	var previous *Version
	for _, version := range um.versions {
		if version.ID != upgrade.VersionID && um.isNewerVersion(current, version) {
			if previous == nil || um.isNewerVersion(previous, version) {
				previous = version
			}
		}
	}

	if previous == nil {
		return fmt.Errorf("no previous version found")
	}

	// تعيين الإصدار السابق كحالي
	previous.IsCurrent = true

	upgrade.Status = UpgradeStatusRolledBack

	um.logger.Info("تم التراجع عن الترقية",
		zap.String("upgrade_id", upgradeID),
		zap.String("previous_version", previous.ID))

	// نشر حدث التراجع
	if um.eventBus != nil {
		um.eventBus.Publish("upgrade.rolled_back", map[string]interface{}{
			"upgrade_id":       upgradeID,
			"previous_version": previous.ID,
		})
	}

	return nil
}

// GetUpgrade يحصل على ترقية
func (um *UpgradeManager) GetUpgrade(upgradeID string) (*Upgrade, error) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	upgrade, exists := um.upgrades[upgradeID]
	if !exists {
		return nil, fmt.Errorf("upgrade not found: %s", upgradeID)
	}

	return upgrade, nil
}

// GetAllUpgrades يحصل على جميع الترقيات
func (um *UpgradeManager) GetAllUpgrades() []*Upgrade {
	um.mu.RLock()
	defer um.mu.RUnlock()

	upgrades := make([]*Upgrade, 0, len(um.upgrades))
	for _, upgrade := range um.upgrades {
		upgrades = append(upgrades, upgrade)
	}

	return upgrades
}

// GetAllVersions يحصل على جميع الإصدارات
func (um *UpgradeManager) GetAllVersions() []*Version {
	um.mu.RLock()
	defer um.mu.RUnlock()

	versions := make([]*Version, 0, len(um.versions))
	for _, version := range um.versions {
		versions = append(versions, version)
	}

	return versions
}

// GetSummary يحصل على ملخص الترقية
func (um *UpgradeManager) GetSummary() map[string]interface{} {
	um.mu.RLock()
	defer um.mu.RUnlock()

	totalVersions := len(um.versions)
	totalUpgrades := len(um.upgrades)
	completedCount := 0
	failedCount := 0

	for _, upgrade := range um.upgrades {
		switch upgrade.Status {
		case UpgradeStatusCompleted:
			completedCount++
		case UpgradeStatusFailed:
			failedCount++
		}
	}

	current, _ := um.GetCurrentVersion()
	latest, _ := um.GetLatestVersion()

	return map[string]interface{}{
		"total_versions":  totalVersions,
		"total_upgrades":  totalUpgrades,
		"completed":      completedCount,
		"failed":         failedCount,
		"current_version": current,
		"latest_version":  latest,
	}
}
