package platform

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return strings.TrimSpace(out.String()), err
	}
	return strings.TrimSpace(out.String()), nil
}

func homeDir() string {
	u, err := user.Current()
	if err != nil {
		return os.Getenv("HOME")
	}
	return u.HomeDir
}

func dirSize(ctx context.Context, path string) int64 {
	out, err := run(ctx, "du", "-sk", path)
	if err != nil {
		return 0
	}
	fields := strings.Fields(out)
	if len(fields) < 1 {
		return 0
	}
	kb, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0
	}
	return kb * 1024
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func sumDirSizes(ctx context.Context, paths []string) (int64, []domainDir) {
	var total int64
	var found []domainDir
	for _, p := range paths {
		if !pathExists(p) {
			continue
		}
		sz := dirSize(ctx, p)
		if sz > 0 {
			total += sz
			found = append(found, domainDir{Path: p, Bytes: sz})
		}
	}
	return total, found
}

type domainDir struct {
	Path  string
	Bytes int64
}

func toDirUsage(d []domainDir) []DirUsageEntry {
	out := make([]DirUsageEntry, len(d))
	for i, x := range d {
		out[i] = DirUsageEntry{Path: x.Path, Bytes: x.Bytes}
	}
	return out
}

type DirUsageEntry struct {
	Path  string
	Bytes int64
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(homeDir(), p[2:])
	}
	return p
}

func dockerSystemDF(ctx context.Context) (*DockerDF, bool) {
	out, err := run(ctx, "docker", "system", "df", "--format", "{{.Type}}\t{{.Size}}")
	if err != nil {
		return nil, false
	}
	df := &DockerDF{}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			continue
		}
		b := parseDockerSize(parts[1])
		switch strings.ToLower(parts[0]) {
		case "images":
			df.Images += b
		case "containers":
			df.Containers += b
		case "local volumes", "volumes":
			df.Volumes += b
		case "build cache":
			df.BuildCache += b
		}
	}
	df.Available = true
	return df, true
}

type DockerDF struct {
	Images, Containers, Volumes, BuildCache int64
	Available                               bool
}

func parseDockerSize(s string) int64 {
	s = strings.TrimSpace(s)
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "GB"):
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "MB"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "KB"), strings.HasSuffix(s, "kB"):
		mult = 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "KB"), "kB")
	case strings.HasSuffix(s, "B"):
		s = strings.TrimSuffix(s, "B")
		mult = 1
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return int64(f * float64(mult))
}

func which(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

const cmdTimeout = 120 * time.Second

func defaultCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cmdTimeout)
}
