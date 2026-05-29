package platform

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hd-health/hd-health/internal/domain"
)

type LinuxFedora struct{}

func (l *LinuxFedora) Name() string { return "linux-fedora" }

func (l *LinuxFedora) Hostname() (string, error) {
	return os.Hostname()
}

func (l *LinuxFedora) Serial() (string, error) {
	if b, err := os.ReadFile("/etc/machine-id"); err == nil {
		return strings.TrimSpace(string(b)), nil
	}
	return "", nil
}

func (l *LinuxFedora) Volumes(ctx context.Context) ([]domain.Volume, error) {
	out, err := run(ctx, "df", "-k", "--output=source,fstype,size,used,avail,pcent,target")
	if err != nil {
		out, err = run(ctx, "df", "-k")
	}
	if err != nil {
		return nil, err
	}
	var vols []domain.Volume
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		var device, fstype, mount string
		var totalK, usedK, availK int64
		var pct float64

		if len(fields) >= 7 && fields[0] != "Filesystem" {
			device = fields[0]
			fstype = fields[1]
			totalK, _ = strconv.ParseInt(fields[2], 10, 64)
			usedK, _ = strconv.ParseInt(fields[3], 10, 64)
			availK, _ = strconv.ParseInt(fields[4], 10, 64)
			pctStr := strings.TrimSuffix(fields[5], "%")
			pct, _ = strconv.ParseFloat(pctStr, 64)
			mount = fields[6]
		} else if len(fields) >= 6 {
			device = fields[0]
			totalK, _ = strconv.ParseInt(fields[1], 10, 64)
			usedK, _ = strconv.ParseInt(fields[2], 10, 64)
			availK, _ = strconv.ParseInt(fields[3], 10, 64)
			pctStr := strings.TrimSuffix(fields[4], "%")
			pct, _ = strconv.ParseFloat(pctStr, 64)
			mount = fields[5]
		} else {
			continue
		}
		if totalK <= 0 {
			continue
		}
		if strings.HasPrefix(device, "tmpfs") || strings.HasPrefix(device, "devtmpfs") ||
			strings.HasPrefix(device, "overlay") || mount == "/dev" ||
			strings.HasPrefix(mount, "/run/user") && mount != "/run" {
			if mount != "/" && mount != "/home" {
				continue
			}
		}
		cap := totalK * 1024
		used := usedK * 1024
		free := availK * 1024
		if pct == 0 && cap > 0 {
			pct = float64(used) / float64(cap) * 100
		}
		name := filepath.Base(mount)
		isNet := strings.HasPrefix(device, "//") || fstype == "nfs" || fstype == "nfs4" || fstype == "cifs"
		vols = append(vols, domain.Volume{
			Name:        name,
			Mount:       mount,
			Device:      device,
			FSType:      fstype,
			Capacity:    cap,
			Free:        free,
			Used:        used,
			UsedPercent: pct,
			IsNetwork:   isNet,
			IsExternal:  strings.HasPrefix(device, "/dev/sd") || strings.HasPrefix(device, "/dev/nvme"),
		})
	}
	return vols, nil
}

func (l *LinuxFedora) TopDirs(ctx context.Context, mount string, depth int) ([]domain.DirUsage, error) {
	if depth < 1 {
		depth = 1
	}
	out, err := run(ctx, "du", "-x", "--max-depth="+strconv.Itoa(depth), "-k", mount)
	if err != nil {
		return nil, err
	}
	var dirs []domain.DirUsage
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		szK, _ := strconv.ParseInt(fields[0], 10, 64)
		path := fields[1]
		dirs = append(dirs, domain.DirUsage{Path: path, Bytes: szK * 1024})
	}
	return dirs, nil
}

func (l *LinuxFedora) InodeUsage(ctx context.Context) ([]domain.InodeStat, error) {
	out, err := run(ctx, "df", "-i")
	if err != nil {
		return nil, err
	}
	var stats []domain.InodeStat
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		mount := fields[len(fields)-1]
		total, _ := strconv.ParseInt(fields[1], 10, 64)
		used, _ := strconv.ParseInt(fields[2], 10, 64)
		free, _ := strconv.ParseInt(fields[3], 10, 64)
		pctStr := strings.TrimSuffix(fields[4], "%")
		pct, _ := strconv.ParseFloat(pctStr, 64)
		stats = append(stats, domain.InodeStat{
			Mount: mount, Total: total, Used: used, Free: free, UsedPercent: pct,
		})
	}
	return stats, nil
}

func (l *LinuxFedora) LargeFiles(ctx context.Context, root string, minBytes int64, limit int) ([]domain.DirUsage, error) {
	minMB := minBytes / (1024 * 1024)
	if minMB < 1 {
		minMB = 100
	}
	out, err := run(ctx, "find", root, "-xdev", "-type", "f", "-size", "+"+strconv.FormatInt(minMB, 10)+"M", "-printf", "%s\t%p\n")
	if err != nil {
		return nil, nil
	}
	var files []domain.DirUsage
	for _, line := range strings.Split(out, "\n") {
		if line == "" || len(files) >= limit {
			break
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		sz, _ := strconv.ParseInt(parts[0], 10, 64)
		files = append(files, domain.DirUsage{Path: parts[1], Bytes: sz})
	}
	return files, nil
}

func (l *LinuxFedora) DockerUsage(ctx context.Context) (*domain.DockerUsage, error) {
	df, ok := dockerSystemDF(ctx)
	if !ok {
		// try podman
		out, err := run(ctx, "podman", "system", "df")
		if err != nil {
			return &domain.DockerUsage{Available: false}, nil
		}
		return &domain.DockerUsage{Raw: out, Available: true}, nil
	}
	total := df.Images + df.Containers + df.Volumes + df.BuildCache
	return &domain.DockerUsage{
		ImagesBytes: df.Images, ContainersBytes: df.Containers,
		VolumesBytes: df.Volumes, BuildCacheBytes: df.BuildCache,
		TotalBytes: total, Available: true,
	}, nil
}

func (l *LinuxFedora) PackageCacheUsage(ctx context.Context) (*domain.CacheUsage, error) {
	cacheDir := "/var/cache/dnf"
	sz := dirSize(ctx, cacheDir)
	if sz == 0 {
		return &domain.CacheUsage{Kind: "dnf"}, nil
	}
	return &domain.CacheUsage{Path: cacheDir, Bytes: sz, Kind: "dnf"}, nil
}

func (l *LinuxFedora) JournalUsage(ctx context.Context) (*domain.JournalUsage, error) {
	out, err := run(ctx, "journalctl", "--disk-usage")
	if err != nil {
		sz := dirSize(ctx, "/var/log/journal")
		return &domain.JournalUsage{Bytes: sz, Available: sz > 0}, nil
	}
	// parse "Archived and active journals take up 1.2G"
	var bytes int64
	lower := strings.ToLower(out)
	for _, suffix := range []struct {
		s string
		m int64
	}{
		{"g", 1024 * 1024 * 1024},
		{"m", 1024 * 1024},
		{"k", 1024},
	} {
		if idx := strings.Index(lower, suffix.s); idx > 0 {
			numStr := ""
			for i := idx - 1; i >= 0; i-- {
				c := lower[i]
				if (c >= '0' && c <= '9') || c == '.' {
					numStr = string(c) + numStr
				} else if numStr != "" {
					break
				}
			}
			if f, err := strconv.ParseFloat(numStr, 64); err == nil {
				bytes = int64(f * float64(suffix.m))
				break
			}
		}
	}
	return &domain.JournalUsage{Bytes: bytes, Raw: out, Available: true}, nil
}

func (l *LinuxFedora) SuggestExternalTools() []string {
	var s []string
	if which("ncdu") {
		s = append(s, "sudo ncdu /")
	} else {
		s = append(s, "sudo dnf install -y ncdu && ncdu /")
	}
	if which("czkawka") {
		s = append(s, "czkawka -d ~ -f duplicates.txt")
	} else {
		s = append(s, "# optional: sudo dnf install -y czkawka-cli")
	}
	return s
}
