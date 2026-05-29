package platform

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hd-health/hd-health/internal/domain"
)

type Darwin struct{}

func (d *Darwin) Name() string { return "darwin" }

func (d *Darwin) Hostname() (string, error) {
	h, err := os.Hostname()
	return h, err
}

func (d *Darwin) Serial() (string, error) {
	out, err := run(context.Background(), "ioreg", "-l")
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`"IOPlatformSerialNumber"\s*=\s*"([^"]+)"`)
	if m := re.FindStringSubmatch(out); len(m) > 1 {
		return m[1], nil
	}
	return "", nil
}

func (d *Darwin) Volumes(ctx context.Context) ([]domain.Volume, error) {
	out, err := run(ctx, "diskutil", "list", "-plist")
	if err != nil {
		return d.volumesFromDF(ctx)
	}
	_ = out
	return d.volumesFromDF(ctx)
}

func (d *Darwin) volumesFromDF(ctx context.Context) ([]domain.Volume, error) {
	out, err := run(ctx, "df", "-k")
	if err != nil {
		return nil, err
	}
	var vols []domain.Volume
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := splitFields(line)
		if len(fields) < 6 {
			continue
		}
		mount := fields[len(fields)-1]
		if mount == "/dev" || strings.HasPrefix(mount, "/dev") && len(fields) == 6 {
			continue
		}
		totalK, _ := strconv.ParseInt(fields[1], 10, 64)
		usedK, _ := strconv.ParseInt(fields[2], 10, 64)
		availK, _ := strconv.ParseInt(fields[3], 10, 64)
		if totalK <= 0 {
			continue
		}
		// skip pseudo filesystems
		if strings.HasPrefix(fields[0], "map ") || fields[0] == "devfs" {
			continue
		}
		cap := totalK * 1024
		used := usedK * 1024
		free := availK * 1024
		pct := float64(used) / float64(cap) * 100
		name := filepath.Base(mount)
		if name == "" || name == "." {
			name = mount
		}
		vols = append(vols, domain.Volume{
			Name:        name,
			Mount:       mount,
			Device:      fields[0],
			Capacity:    cap,
			Free:        free,
			Used:        used,
			UsedPercent: pct,
			IsNetwork:   strings.Contains(fields[0], "@") || strings.Contains(fields[0], "//"),
			IsExternal:  strings.HasPrefix(fields[0], "/dev/disk") && !strings.Contains(mount, "Macintosh HD") && mount != "/",
		})
	}
	return vols, nil
}

func splitFields(s string) []string {
	var f []string
	var cur strings.Builder
	inQuote := false
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == ' ' && !inQuote:
			if cur.Len() > 0 {
				f = append(f, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		f = append(f, cur.String())
	}
	return f
}

func (d *Darwin) TopDirs(ctx context.Context, mount string, depth int) ([]domain.DirUsage, error) {
	if depth < 1 {
		depth = 1
	}
	args := []string{"-d", strconv.Itoa(depth), "-x", mount}
	out, err := run(ctx, "du", args...)
	if err != nil {
		return nil, err
	}
	var dirs []domain.DirUsage
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, "\t")
		if idx < 0 {
			continue
		}
		szStr := strings.TrimSpace(line[:idx])
		path := strings.TrimSpace(line[idx+1:])
		sz, _ := strconv.ParseInt(szStr, 10, 64)
		dirs = append(dirs, domain.DirUsage{Path: path, Bytes: sz * 512}) // du -d uses 512 blocks on macOS
	}
	return dirs, nil
}

func (d *Darwin) InodeUsage(ctx context.Context) ([]domain.InodeStat, error) {
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
		fields := splitFields(line)
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
			Mount:       mount,
			Total:       total,
			Used:        used,
			Free:        free,
			UsedPercent: pct,
		})
	}
	return stats, nil
}

func (d *Darwin) LargeFiles(ctx context.Context, root string, minBytes int64, limit int) ([]domain.DirUsage, error) {
	minMB := minBytes / (1024 * 1024)
	if minMB < 1 {
		minMB = 100
	}
	out, err := run(ctx, "find", root, "-type", "f", "-size", "+"+strconv.FormatInt(minMB, 10)+"M", "-print0")
	if err != nil {
		return nil, nil
	}
	var files []domain.DirUsage
	parts := strings.Split(out, "\x00")
	for _, p := range parts {
		if p == "" || len(files) >= limit {
			break
		}
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		files = append(files, domain.DirUsage{Path: p, Bytes: fi.Size()})
	}
	return files, nil
}

func (d *Darwin) DockerUsage(ctx context.Context) (*domain.DockerUsage, error) {
	df, ok := dockerSystemDF(ctx)
	if !ok {
		return &domain.DockerUsage{Available: false}, nil
	}
	total := df.Images + df.Containers + df.Volumes + df.BuildCache
	return &domain.DockerUsage{
		ImagesBytes:     df.Images,
		ContainersBytes: df.Containers,
		VolumesBytes:    df.Volumes,
		BuildCacheBytes: df.BuildCache,
		TotalBytes:      total,
		Available:       true,
	}, nil
}

func (d *Darwin) PackageCacheUsage(ctx context.Context) (*domain.CacheUsage, error) {
	home := homeDir()
	brewCache := filepath.Join(home, "Library", "Caches", "Homebrew")
	if !pathExists(brewCache) {
		return &domain.CacheUsage{Kind: "brew"}, nil
	}
	sz := dirSize(ctx, brewCache)
	return &domain.CacheUsage{Path: brewCache, Bytes: sz, Kind: "brew"}, nil
}

func (d *Darwin) JournalUsage(ctx context.Context) (*domain.JournalUsage, error) {
	out, err := run(ctx, "log", "show", "--last", "1s", "--style", "json")
	if err != nil {
		// estimate /var/log
		sz := dirSize(ctx, "/var/log")
		return &domain.JournalUsage{Bytes: sz, Available: sz > 0}, nil
	}
	var entries []json.RawMessage
	_ = json.Unmarshal([]byte(out), &entries)
	sz := dirSize(ctx, "/var/log")
	return &domain.JournalUsage{Bytes: sz, Raw: "unified logging", Available: true}, nil
}

func (d *Darwin) SuggestExternalTools() []string {
	var s []string
	if which("ncdu") {
		s = append(s, "ncdu /")
	} else {
		s = append(s, "brew install ncdu && ncdu /")
	}
	if which("czkawka") {
		s = append(s, "czkawka -d ~ -f duplicates.txt")
	}
	return s
}

func tmutilDestination() string {
	out, _ := exec.Command("tmutil", "destinationinfo").Output()
	return string(out)
}
