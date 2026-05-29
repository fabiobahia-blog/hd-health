package classify

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hd-health/hd-health/internal/domain"
	"github.com/hd-health/hd-health/internal/platform"
)

type Input struct {
	Platform       platform.Platform
	Docker         *domain.DockerUsage
	PkgCache       *domain.CacheUsage
	Journal        *domain.JournalUsage
	VerbosePaths   bool
}

func Run(ctx context.Context, in Input) []domain.Profile {
	var profiles []domain.Profile
	home := ""
	if u, err := os.UserHomeDir(); err == nil {
		home = u
	}

	add := func(id domain.ProfileID, name string, bytes int64, paths []string, conf float64) {
		if bytes <= 0 && conf < 0.3 {
			return
		}
		if !in.VerbosePaths {
			paths = redactPaths(paths)
		}
		profiles = append(profiles, domain.Profile{
			ID: id, Name: name, Confidence: conf, Bytes: bytes, Paths: paths,
		})
	}

	// 1 Backup
	backupPaths := backupPathsForOS(home)
	bBytes, bPaths := measurePaths(ctx, backupPaths)
	conf := 0.5
	if bBytes > 1<<30 {
		conf = 0.85
	}
	if runtime.GOOS == "darwin" && tmutilConfigured() {
		conf = 0.9
	}
	add(domain.ProfileBackup, "Endpoint backup", bBytes, bPaths, conf)

	// 2 Cloud sync
	cPaths := cloudSyncPaths(home)
	cBytes, cP := measurePaths(ctx, cPaths)
	cConf := 0.4
	if cBytes > 500<<20 {
		cConf = 0.8
	}
	add(domain.ProfileCloudSync, "Business cloud sync", cBytes, cP, cConf)

	// 3 NAS - detected via network mounts in volumes (handled in findings)

	// 4 Creative
	crPaths := creativePaths(home)
	crBytes, crP := measurePaths(ctx, crPaths)
	add(domain.ProfileCreative, "Creative / video", crBytes, crP, confidence(crBytes, 2<<30))

	// 5 Development + Docker
	devPaths := devPaths(home)
	dBytes, dP := measurePaths(ctx, devPaths)
	if in.Docker != nil && in.Docker.Available {
		dBytes += in.Docker.TotalBytes
		dConf := 0.85
		if dBytes > 10<<30 {
			dConf = 0.95
		}
		add(domain.ProfileDevelopment, "Development + containers", dBytes, dP, dConf)
	} else {
		add(domain.ProfileDevelopment, "Development + containers", dBytes, dP, confidence(dBytes, 5<<30))
	}

	// 6 VM
	vmPaths := vmPaths(home)
	vmBytes, vmP := measurePaths(ctx, vmPaths)
	add(domain.ProfileVM, "Virtualization", vmBytes, vmP, confidence(vmBytes, 20<<30))

	// 7 Mail
	mPaths := mailPaths(home)
	mBytes, mP := measurePaths(ctx, mPaths)
	add(domain.ProfileMail, "Mail & collaboration", mBytes, mP, confidence(mBytes, 1<<30))

	// 8 MDM / packages
	var mdmScan []string
	if in.PkgCache != nil && in.PkgCache.Path != "" {
		mdmScan = append(mdmScan, in.PkgCache.Path)
	}
	mdmScan = append(mdmScan, mdmExtraPaths()...)
	mdmBytes, mdmPaths := measurePaths(ctx, mdmScan)
	add(domain.ProfileMDM, "MDM / package caches", mdmBytes, dedupeStrings(mdmPaths), confidence(mdmBytes, 500<<20))

	// 9 Logs
	logBytes := int64(0)
	var logPaths []string
	if in.Journal != nil && in.Journal.Bytes > 0 {
		logBytes = in.Journal.Bytes
		logPaths = []string{"/var/log/journal"}
	}
	if runtime.GOOS == "darwin" {
		sz := dirSizeQuick(ctx, "/var/log")
		logBytes += sz
	}
	add(domain.ProfileLogs, "System logs & journals", logBytes, logPaths, confidence(logBytes, 1<<30))

	// 10 AI
	aiPaths := aiPaths(home)
	aiBytes, aiP := measurePaths(ctx, aiPaths)
	add(domain.ProfileAI, "Local AI models", aiBytes, aiP, confidence(aiBytes, 5<<30))

	return profiles
}

func confidence(bytes int64, threshold int64) float64 {
	if bytes <= 0 {
		return 0.2
	}
	if bytes >= threshold {
		return 0.9
	}
	return 0.5 + 0.4*float64(bytes)/float64(threshold)
}

func measurePaths(ctx context.Context, paths []string) (int64, []string) {
	var total int64
	var found []string
	seen := map[string]bool{}
	for _, p := range paths {
		p = expandHome(p)
		if seen[p] {
			continue
		}
		seen[p] = true
		if st, err := os.Stat(p); err != nil {
			continue
		} else if !st.IsDir() {
			continue
		}
		sz := dirSizeQuick(ctx, p)
		if sz > 0 {
			total += sz
			found = append(found, p)
		}
	}
	return total, found
}

func dirSizeQuick(ctx context.Context, path string) int64 {
	// use du via platform helper - simple exec
	cmd := exec.CommandContext(ctx, "du", "-sk", path)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(out))
	if len(fields) < 1 {
		return 0
	}
	var kb int64
	for _, f := range fields {
		if n, err := parseInt64(f); err == nil {
			kb = n
			break
		}
	}
	return kb * 1024
}

func parseInt64(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, os.ErrInvalid
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func dedupeStrings(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func redactPaths(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		home, _ := os.UserHomeDir()
		if strings.HasPrefix(p, home) {
			out[i] = "~" + strings.TrimPrefix(p, home)
		} else {
			out[i] = p
		}
	}
	return out
}

func backupPathsForOS(home string) []string {
	if runtime.GOOS == "darwin" {
		return []string{
			"/Volumes/*/Backups.backupdb",
			filepath.Join(home, "Library", "Application Support", "Backblaze"),
			"/Library/Application Support/CrashPlan",
		}
	}
	return []string{
		"/var/lib/restic",
		"/var/lib/borg",
		"/var/backups",
	}
}

func tmutilConfigured() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	out, err := exec.Command("tmutil", "destinationinfo").Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

func cloudSyncPaths(home string) []string {
	return []string{
		filepath.Join(home, "Library", "CloudStorage"),
		filepath.Join(home, "Google Drive"),
		filepath.Join(home, "OneDrive"),
		filepath.Join(home, "Dropbox"),
		filepath.Join(home, "Nextcloud"),
		filepath.Join(home, ".dropbox"),
	}
}

func creativePaths(home string) []string {
	return []string{
		filepath.Join(home, "Movies"),
		filepath.Join(home, "Videos"),
		filepath.Join(home, "Music"),
	}
}

func devPaths(home string) []string {
	paths := []string{
		filepath.Join(home, "Library", "Developer"),
		filepath.Join(home, ".cache"),
		filepath.Join(home, "go", "pkg"),
		filepath.Join(home, ".npm"),
		filepath.Join(home, ".docker"),
	}
	if runtime.GOOS == "linux" {
		paths = append(paths, "/var/lib/docker")
	}
	return paths
}

func vmPaths(home string) []string {
	if runtime.GOOS == "darwin" {
		return []string{
			filepath.Join(home, "Parallels"),
			filepath.Join(home, "Virtual Machines.localized"),
		}
	}
	return []string{
		"/var/lib/libvirt/images",
		filepath.Join(home, ".local", "share", "libvirt"),
	}
}

func mailPaths(home string) []string {
	if runtime.GOOS == "darwin" {
		return []string{
			filepath.Join(home, "Library", "Mail"),
			filepath.Join(home, "Library", "Group Containers", "UBF8T346G9.Office", "Outlook"),
		}
	}
	return []string{
		filepath.Join(home, ".thunderbird"),
		filepath.Join(home, ".local", "share", "evolution"),
	}
}

func mdmExtraPaths() []string {
	if runtime.GOOS == "darwin" {
		return []string{
			"/Library/Application Support/Jamf",
			"/Library/Application Support/Munki",
		}
	}
	return []string{"/var/cache/dnf", "/var/lib/flatpak"}
}

func aiPaths(home string) []string {
	return []string{
		filepath.Join(home, ".cache", "huggingface"),
		filepath.Join(home, ".ollama"),
		filepath.Join(home, "Library", "Application Support", "LM Studio"),
		filepath.Join(home, ".local", "share", "ollama"),
	}
}
