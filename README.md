# hd-health

Storage health monitoring for IT and small business: local disks, external drives, and cloud sync footprint on **macOS** (v1) and **Fedora Linux** (v1.1).

## Features

- Volume inventory with inode usage and growth forecasting
- Top-10 consumption profile detection (backup, cloud sync, Docker/dev, logs, NAS, AI models, and more)
- Safe remediation playbooks with `--dry-run` (default) and `--apply`
- JSON/CSV reports for Jamf, Ansible, or manual fleet collection
- Background agent with SQLite metrics history

## Quick start

```bash
# Build
make build

# One-shot scan
./bin/hd-health scan

# Full report (exit 1=warning, 2=critical)
./bin/hd-health report

# Remediation cheat sheet (dry-run)
./bin/hd-health remediate --mount / --dry-run

# Export for RMM
./bin/hd-health export --format json > report.json
```

## Install agent (macOS)

```bash
sudo ./deploy/macos/install.sh
```

## Install agent (Fedora)

```bash
sudo ./deploy/fedora/install.sh
```

## Documentation

- [Use cases & top-10 profiles](docs/use-cases.md)
- [Architecture](docs/architecture.md)
- [Full disk playbook — macOS](docs/playbooks/full-disk-macos.md)
- [Full disk playbook — Fedora](docs/playbooks/full-disk-fedora.md)

## Privacy

By default, reports use path categories and aggregate sizes only. Use `--verbose-paths` for on-site IT troubleshooting.

## License

MIT
