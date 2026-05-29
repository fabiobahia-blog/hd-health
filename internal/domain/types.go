package domain

import "time"

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type Volume struct {
	Name         string  `json:"name"`
	Mount        string  `json:"mount"`
	Device       string  `json:"device,omitempty"`
	FSType       string  `json:"fs_type,omitempty"`
	Capacity     int64   `json:"capacity_bytes"`
	Free         int64   `json:"free_bytes"`
	Used         int64   `json:"used_bytes"`
	UsedPercent  float64 `json:"used_percent"`
	IsExternal   bool    `json:"is_external"`
	IsNetwork    bool    `json:"is_network"`
}

type InodeStat struct {
	Mount       string  `json:"mount"`
	UsedPercent float64 `json:"used_percent"`
	Total       int64   `json:"total"`
	Used        int64   `json:"used"`
	Free        int64   `json:"free"`
}

type DirUsage struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

type DockerUsage struct {
	ImagesBytes   int64  `json:"images_bytes"`
	ContainersBytes int64 `json:"containers_bytes"`
	VolumesBytes  int64  `json:"volumes_bytes"`
	BuildCacheBytes int64 `json:"build_cache_bytes"`
	TotalBytes    int64  `json:"total_bytes"`
	Raw           string `json:"raw,omitempty"`
	Available     bool   `json:"available"`
}

type CacheUsage struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Kind  string `json:"kind"` // brew, dnf, etc.
}

type JournalUsage struct {
	Bytes     int64  `json:"bytes"`
	Raw       string `json:"raw,omitempty"`
	Available bool   `json:"available"`
}

type ProfileID string

const (
	ProfileBackup      ProfileID = "backup"
	ProfileCloudSync   ProfileID = "cloud_sync"
	ProfileNAS         ProfileID = "nas"
	ProfileCreative    ProfileID = "creative"
	ProfileDevelopment ProfileID = "development"
	ProfileVM          ProfileID = "virtualization"
	ProfileMail        ProfileID = "mail"
	ProfileMDM         ProfileID = "mdm_packages"
	ProfileLogs        ProfileID = "system_logs"
	ProfileAI          ProfileID = "local_ai"
)

type Profile struct {
	ID         ProfileID `json:"id"`
	Name       string    `json:"name"`
	Confidence float64   `json:"confidence"`
	Bytes      int64     `json:"bytes"`
	Paths      []string  `json:"paths_redacted"`
}

type Finding struct {
	Severity    Severity `json:"severity"`
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	Remediation string   `json:"remediation"`
	PlaybookID  string   `json:"playbook_id,omitempty"`
	Steps       []string `json:"steps,omitempty"`
}

type VolumeForecast struct {
	Mount              string  `json:"mount"`
	GrowthBytesPerDay  float64 `json:"growth_bytes_per_day"`
	ForecastDaysTo85   *int    `json:"forecast_days_to_85,omitempty"`
	ForecastDaysTo95   *int    `json:"forecast_days_to_95,omitempty"`
}

type HealthSummary struct {
	SmartStatus   string   `json:"smart_status,omitempty"`
	BackupOK      *bool    `json:"backup_ok,omitempty"`
	SyncOK        *bool    `json:"sync_ok,omitempty"`
	InodePercent  *float64 `json:"inode_percent,omitempty"`
}

type Report struct {
	Hostname    string           `json:"hostname"`
	Serial      string           `json:"serial,omitempty"`
	CollectedAt time.Time        `json:"collected_at"`
	Platform    string           `json:"platform"`
	Volumes     []VolumeReport   `json:"volumes"`
	Profiles    []Profile        `json:"profiles"`
	Findings    []Finding        `json:"findings"`
	Health      HealthSummary    `json:"health"`
}

type VolumeReport struct {
	Volume
	GrowthBytesPerDay float64 `json:"growth_bytes_per_day"`
	ForecastDaysTo85  *int    `json:"forecast_days_to_85,omitempty"`
	ForecastDaysTo95  *int    `json:"forecast_days_to_95,omitempty"`
}

type PlaybookStep struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Commands     []string `json:"commands"`
	Risk         string   `json:"risk"` // low, medium, high
	RequiresRoot bool     `json:"requires_root"`
}
