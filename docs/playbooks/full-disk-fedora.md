# Full disk remediation — Fedora

## 1. Confirm space

```bash
df -h
df -i
df -h /
```

## 2. Find large usage

```bash
sudo du -h --max-depth=1 / 2>/dev/null | sort -hr | head -10
sudo find / -type f -size +100M -exec ls -lh {} \; 2>/dev/null | head -20
sudo dnf install -y ncdu && ncdu /
```

## 3. Common cleanups

```bash
sudo dnf clean all
sudo dnf autoremove
journalctl --disk-usage
sudo journalctl --vacuum-size=500M
sudo journalctl --vacuum-time=7d
sudo rm -rf /tmp/*
rm -rf ~/.cache/*
rpm -qa | grep kernel
sudo dnf remove kernel-<old-version>
```

## 4. Docker

```bash
docker system df
docker system prune
docker system prune -a
docker volume prune
```

## 5. Duplicates (optional)

```bash
sudo dnf install -y czkawka
czkawka -d . -f duplicates.txt
```

## 6. Verify

```bash
df -h
df -h /
```

## Emergency (100% full)

```bash
sudo journalctl --vacuum-size=100M
sudo truncate -s 0 /var/log/messages  # if present and huge
```

Use `hd-health remediate --mount / --dry-run` on Fedora for automated suggestions.
