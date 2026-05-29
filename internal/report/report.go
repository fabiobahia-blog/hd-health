package report

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/hd-health/hd-health/internal/classify"
	"github.com/hd-health/hd-health/internal/domain"
	"github.com/hd-health/hd-health/internal/forecast"
	"github.com/hd-health/hd-health/internal/platform"
	"github.com/hd-health/hd-health/internal/store"
)

type Builder struct {
	Platform     platform.Platform
	Store        *store.Store
	VerbosePaths bool
}

func (b *Builder) Build(ctx context.Context) (*domain.Report, int, error) {
	hostname, _ := b.Platform.Hostname()
	serial, _ := b.Platform.Serial()

	vols, err := b.Platform.Volumes(ctx)
	if err != nil {
		return nil, 0, err
	}
	if b.Store != nil {
		_ = b.Store.RecordVolumes(vols)
	}

	inodes, _ := b.Platform.InodeUsage(ctx)
	docker, _ := b.Platform.DockerUsage(ctx)
	pkg, _ := b.Platform.PackageCacheUsage(ctx)
	journal, _ := b.Platform.JournalUsage(ctx)

	profiles := classify.Run(ctx, classify.Input{
		Platform: b.Platform, Docker: docker, PkgCache: pkg, Journal: journal,
		VerbosePaths: b.VerbosePaths,
	})

	var volReports []domain.VolumeReport
	var maxInode float64
	for _, in := range inodes {
		if in.UsedPercent > maxInode {
			maxInode = in.UsedPercent
		}
	}
	for _, v := range vols {
		growth := 0.0
		if b.Store != nil {
			growth, _ = b.Store.GrowthBytesPerDay(v.Mount, 7)
		}
		volReports = append(volReports, forecast.EnrichVolume(v, growth))
	}

	findings := buildFindings(vols, volReports, inodes, docker, journal, profiles)
	health := domain.HealthSummary{}
	if maxInode > 0 {
		health.InodePercent = &maxInode
	}
	if runtime.GOOS == "darwin" {
		ok := tmutilConfigured()
		health.BackupOK = &ok
	}

	exit := exitCode(findings, volReports)

	r := &domain.Report{
		Hostname:    hostname,
		Serial:      serial,
		CollectedAt: time.Now().UTC(),
		Platform:    platform.PlatformID(),
		Volumes:     volReports,
		Profiles:    profiles,
		Findings:    findings,
		Health:      health,
	}
	return r, exit, nil
}

func tmutilConfigured() bool {
	out, err := exec.Command("tmutil", "destinationinfo").Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

func buildFindings(vols []domain.Volume, reports []domain.VolumeReport, inodes []domain.InodeStat, docker *domain.DockerUsage, journal *domain.JournalUsage, profiles []domain.Profile) []domain.Finding {
	var f []domain.Finding

	for _, v := range vols {
		if v.UsedPercent >= 100 {
			f = append(f, domain.Finding{
				Severity: domain.SeverityCritical, Code: "disk_full",
				Message: fmt.Sprintf("Volume %s (%s) is 100%% full", v.Name, v.Mount),
				Remediation: "Run hd-health remediate --mount " + v.Mount + " --dry-run immediately",
				PlaybookID: "emergency",
				Steps: emergencySteps(),
			})
		} else if v.UsedPercent >= 95 {
			f = append(f, domain.Finding{
				Severity: domain.SeverityCritical, Code: "disk_critical",
				Message: fmt.Sprintf("Volume %s is %.1f%% full", v.Name, v.UsedPercent),
				Remediation: "Free space or expand volume within 24h",
				PlaybookID: "diagnose",
			})
		} else if v.UsedPercent >= 85 {
			f = append(f, domain.Finding{
				Severity: domain.SeverityWarning, Code: "disk_warning",
				Message: fmt.Sprintf("Volume %s is %.1f%% full", v.Name, v.UsedPercent),
				Remediation: "Review top profiles and run remediate --dry-run",
			})
		}
		if v.IsNetwork && v.UsedPercent >= 90 {
			f = append(f, domain.Finding{
				Severity: domain.SeverityWarning, Code: "nas_pressure",
				Message: fmt.Sprintf("Network volume %s is %.1f%% full", v.Mount, v.UsedPercent),
				Remediation: "Coordinate with NAS admin; archive old projects",
			})
		}
	}

	for _, r := range reports {
		if r.ForecastDaysTo85 != nil && *r.ForecastDaysTo85 < 14 {
			f = append(f, domain.Finding{
				Severity: domain.SeverityWarning, Code: "forecast_85",
				Message: fmt.Sprintf("Volume %s may reach 85%% in ~%d days", r.Mount, *r.ForecastDaysTo85),
				Remediation: "Plan cleanup before threshold",
			})
		}
	}

	for _, in := range inodes {
		if in.Mount == "/dev" || in.Mount == "/dev/fd" {
			continue
		}
		if in.UsedPercent >= 90 {
			f = append(f, domain.Finding{
				Severity: domain.SeverityWarning, Code: "inode_pressure",
				Message: fmt.Sprintf("Inode usage on %s is %.1f%%", in.Mount, in.UsedPercent),
				Remediation: "Remove small files or vacuum journals",
				PlaybookID: "logs",
			})
		}
	}

	if docker != nil && docker.Available && docker.TotalBytes > 10*1024*1024*1024 {
		f = append(f, domain.Finding{
			Severity: domain.SeverityWarning, Code: "docker_large",
			Message: fmt.Sprintf("Docker is using ~%.1f GB", float64(docker.TotalBytes)/(1<<30)),
			Remediation: "docker system prune (see remediate --dry-run)",
			PlaybookID: "docker-safe",
		})
	}

	if journal != nil && journal.Available && journal.Bytes > 1024*1024*1024 {
		f = append(f, domain.Finding{
			Severity: domain.SeverityInfo, Code: "journal_large",
			Message: fmt.Sprintf("System journals using significant space"),
			Remediation: "journalctl --vacuum-size=500M",
			PlaybookID: "logs",
		})
	}

	// Top profile hint
	var top domain.Profile
	for _, p := range profiles {
		if p.Bytes > top.Bytes {
			top = p
		}
	}
	if top.Bytes > 5*1024*1024*1024 {
		f = append(f, domain.Finding{
			Severity: domain.SeverityInfo, Code: "top_profile",
			Message: fmt.Sprintf("Largest consumption profile: %s (~%.1f GB)", top.Name, float64(top.Bytes)/(1<<30)),
			Remediation: "Run hd-health explain for path breakdown",
		})
	}

	return f
}

func emergencySteps() []string {
	if runtime.GOOS == "darwin" {
		return []string{
			"brew cleanup",
			"docker system prune -f",
			"du -sh ~/Library/Caches",
		}
	}
	return []string{
		"sudo journalctl --vacuum-size=100M",
		"sudo dnf clean all",
		"docker system prune -f",
	}
}

func exitCode(findings []domain.Finding, vols []domain.VolumeReport) int {
	code := 0
	for _, f := range findings {
		switch f.Severity {
		case domain.SeverityCritical:
			return 2
		case domain.SeverityWarning:
			code = 1
		}
	}
	for _, v := range vols {
		if v.UsedPercent >= 95 {
			return 2
		}
		if v.UsedPercent >= 85 && code < 1 {
			code = 1
		}
	}
	return code
}

func WriteJSON(w io.Writer, r *domain.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func WriteCSV(w io.Writer, r *domain.Report) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"hostname", "platform", "mount", "used_percent", "free_bytes", "growth_bytes_per_day", "forecast_days_to_85"})
	for _, v := range r.Volumes {
		d85 := ""
		if v.ForecastDaysTo85 != nil {
			d85 = fmt.Sprintf("%d", *v.ForecastDaysTo85)
		}
		_ = cw.Write([]string{
			r.Hostname, r.Platform, v.Mount,
			fmt.Sprintf("%.1f", v.UsedPercent),
			fmt.Sprintf("%d", v.Free),
			fmt.Sprintf("%.0f", v.GrowthBytesPerDay),
			d85,
		})
	}
	cw.Flush()
	return nil
}

func FormatHuman(r *domain.Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "hd-health report — %s (%s)\n", r.Hostname, r.Platform)
	fmt.Fprintf(&b, "Collected: %s\n\n", r.CollectedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "Volumes:\n")
	for _, v := range r.Volumes {
		fmt.Fprintf(&b, "  %-20s %6.1f%% used  free=%.1fGB", v.Mount, v.UsedPercent, float64(v.Free)/(1<<30))
		if v.ForecastDaysTo85 != nil {
			fmt.Fprintf(&b, "  →85%% in %dd", *v.ForecastDaysTo85)
		}
		b.WriteString("\n")
	}
	if len(r.Profiles) > 0 {
		fmt.Fprintf(&b, "\nProfiles:\n")
		for _, p := range r.Profiles {
			if p.Bytes < 100<<20 {
				continue
			}
			fmt.Fprintf(&b, "  %-28s %.1f GB  (%.0f%% conf)\n", p.Name, float64(p.Bytes)/(1<<30), p.Confidence*100)
		}
	}
	if len(r.Findings) > 0 {
		fmt.Fprintf(&b, "\nFindings:\n")
		for _, f := range r.Findings {
			fmt.Fprintf(&b, "  [%s] %s\n      → %s\n", f.Severity, f.Message, f.Remediation)
		}
	}
	return b.String()
}
