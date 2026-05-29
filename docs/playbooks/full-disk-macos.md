# Full disk remediation — macOS

## 1. Confirm space

```bash
df -h
df -i
df -h /
```

## 2. Find large usage

```bash
sudo du -hd 1 / 2>/dev/null | sort -hr | head -10
find / -type f -size +100M 2>/dev/null | head -20
brew install ncdu && ncdu /
```

## 3. Common cleanups

```bash
rm -rf ~/Library/Caches/*
brew cleanup
rm -rf ~/Library/Developer/Xcode/Archives/*
```

## 4. Docker

```bash
docker system df
docker system prune
docker system prune -a   # aggressive
docker volume prune      # confirm first
```

## 5. Verify

```bash
df -h
```

Use `hd-health remediate --mount / --dry-run` for the same steps with risk labels.
