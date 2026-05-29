package platform

import (
	"context"
	"runtime"

	"github.com/hd-health/hd-health/internal/domain"
)

// Platform collects OS-specific storage facts.
type Platform interface {
	Name() string
	Hostname() (string, error)
	Serial() (string, error)
	Volumes(ctx context.Context) ([]domain.Volume, error)
	TopDirs(ctx context.Context, mount string, depth int) ([]domain.DirUsage, error)
	InodeUsage(ctx context.Context) ([]domain.InodeStat, error)
	LargeFiles(ctx context.Context, root string, minBytes int64, limit int) ([]domain.DirUsage, error)
	DockerUsage(ctx context.Context) (*domain.DockerUsage, error)
	PackageCacheUsage(ctx context.Context) (*domain.CacheUsage, error)
	JournalUsage(ctx context.Context) (*domain.JournalUsage, error)
	SuggestExternalTools() []string
}

func Current() Platform {
	switch runtime.GOOS {
	case "darwin":
		return &Darwin{}
	case "linux":
		return &LinuxFedora{}
	default:
		return &LinuxFedora{}
	}
}

func PlatformID() string {
	switch runtime.GOOS {
	case "darwin":
		return "darwin"
	case "linux":
		return "linux-fedora"
	default:
		return runtime.GOOS
	}
}
