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
export PATH="/opt/homebrew/bin:$PATH"
make build

./bin/hd-health scan
./bin/hd-health report
./bin/hd-health explain /System/Volumes/Data
./bin/hd-health remediate --mount /System/Volumes/Data --dry-run
./bin/hd-health export --format json > report.json
```

Exit codes: `0` ok, `1` warning, `2` critical.

On macOS, user data is on `/System/Volumes/Data` — `explain /` resolves there automatically.

**zsh:** run commands without inline `#` comments unless `setopt interactivecomments` is on (otherwise zsh may print `unknown file attribute`).

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
