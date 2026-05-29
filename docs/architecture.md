# Architecture

## Components

- **hd-health** — CLI: `scan`, `report`, `forecast`, `explain`, `remediate`, `export`
- **hd-health-agent** — periodic collector writing to SQLite
- **internal/platform** — OS-specific collectors (`darwin`, `linux_fedora`)
- **internal/classify** — rule-based profile plugins
- **internal/forecast** — growth rate and days-to-threshold
- **internal/remediate** — playbook steps per OS
- **internal/store** — SQLite time series
- **internal/report** — JSON/CSV schema

## Data flow

```
Platform.Volumes() → store.RecordSnapshot() → classify.Run() → forecast.Compute() → report.Build()
```

## Report schema

See `internal/report/report.go` for the stable JSON contract (`platform`, `volumes`, `profiles`, `findings`, `health`).

## Platform interface

```go
type Platform interface {
    Name() string
    Volumes(ctx context.Context) ([]Volume, error)
    TopDirs(ctx context.Context, mount string, depth int) ([]DirUsage, error)
    InodeUsage(ctx context.Context) ([]InodeStat, error)
    DockerUsage(ctx context.Context) (*DockerUsage, error)
    PackageCacheUsage(ctx context.Context) (*CacheUsage, error)
    JournalUsage(ctx context.Context) (*JournalUsage, error)
}
```
