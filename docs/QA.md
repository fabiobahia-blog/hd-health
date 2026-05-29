# QA checklist

## Automated

```bash
make build
go test ./...
go vet ./...
bash -n deploy/macos/install.sh deploy/fedora/install.sh
./bin/hd-health scan
./bin/hd-health report    # expect exit 0–2
./bin/hd-health export --format json
./bin/hd-health remediate --dry-run
```

CI runs the same on `macos-latest` and `ubuntu-latest` (see `.github/workflows/ci.yml`).

## Manual (macOS)

- [ ] `sudo deploy/macos/install.sh` — LaunchDaemon starts, log at `/var/log/hd-health-agent.log`
- [ ] Jamf EA script returns valid XML with JSON payload
- [ ] `hd-health explain /System/Volumes/Data` lists profiles and dirs
- [ ] With Docker running: profile **Development + containers** shows Docker bytes
- [ ] `hd-health remediate --apply` only runs allowlisted commands (brew/docker/dnf)

## Manual (Fedora)

- [ ] `make build && sudo deploy/fedora/install.sh`
- [ ] `systemctl status hd-health-agent`
- [ ] Inode warning when `df -i` >90% on `/`
- [ ] `journalctl` playbook appears in `remediate --dry-run`

## Known limitations

- macOS APFS reports multiple volumes with similar capacity (df quirk).
- `/dev` inode stats ignored in findings.
- Forecast needs ≥2 snapshots over 7 days for accuracy.
- `remediate --apply` does not run high-risk steps without `--aggressive`.
