# Top 10 consumption profiles

| # | Profile | Why it grows | Health risks |
|---|---------|--------------|--------------|
| 1 | Endpoint backup | Full/incremental snapshots | Destination full, stale backup |
| 2 | Cloud sync | Local cache + versions | Quota, paused sync |
| 3 | NAS / SMB | Centralized user data | Share >90%, slow SMB |
| 4 | Creative / video | Render caches, proxies | SSD wear, disconnects |
| 5 | Development + Docker | Images, volumes, builds | 10–50GB+ Docker, inode exhaustion |
| 6 | Virtualization | Growing VM disks | Runaway sparse images |
| 7 | Mail caches | OST/PST, attachments | Sync stalls |
| 8 | MDM / package caches | Installers, DNF/brew | Mystery hundreds of GB |
| 9 | System logs | Append-only journals | `/var` full, inode pressure |
| 10 | Local AI models | LLM weights, RAG corpora | Wrong disk tier |

**MVP detection order:** 1 → 2 → 5 → 9 → 3 → 10 → 4 → 6 → 7 → 8

## IT deployment

1. Build or download `hd-health` and `hd-health-agent` for each platform.
2. macOS: run `deploy/macos/install.sh` (LaunchDaemon, hourly snapshots).
3. Fedora: run `deploy/fedora/install.sh` (systemd timer).
4. Collect reports: `hd-health export --format json` via Jamf EA, Ansible, or cron.
5. Alert on exit code: `0` ok, `1` warning (≥85% or forecast <14d), `2` critical (≥95% or full).

See [playbooks](playbooks/) for manual cleanup steps mirrored by `hd-health remediate`.
