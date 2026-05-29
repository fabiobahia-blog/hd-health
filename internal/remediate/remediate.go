package remediate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/hd-health/hd-health/internal/domain"
)

type Options struct {
	Mount      string
	DryRun     bool
	Apply      bool
	Aggressive bool
}

type Result struct {
	Steps []domain.PlaybookStep
	Run   []string
	Errors []string
}

func Playbooks(mount string) []domain.PlaybookStep {
	if runtime.GOOS == "darwin" {
		return darwinPlaybooks(mount)
	}
	return fedoraPlaybooks(mount)
}

func Execute(ctx context.Context, opt Options) (*Result, error) {
	steps := Playbooks(opt.Mount)
	res := &Result{Steps: steps}
	if opt.DryRun || !opt.Apply {
		for _, s := range steps {
			res.Run = append(res.Run, "# "+s.Description)
			for _, c := range s.Commands {
				res.Run = append(res.Run, c)
			}
		}
		return res, nil
	}
	for _, s := range steps {
		if s.Risk == "high" && !opt.Aggressive {
			continue
		}
		if s.ID == "docker-aggressive" && !opt.Aggressive {
			continue
		}
		for _, c := range s.Commands {
			if strings.HasPrefix(c, "#") {
				continue
			}
			if err := runStep(ctx, c, s.RequiresRoot); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", c, err))
			} else {
				res.Run = append(res.Run, "OK: "+c)
			}
		}
	}
	return res, nil
}

func runStep(ctx context.Context, command string, needsRoot bool) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}
	allowlist := map[string]bool{
		"brew": true, "docker": true, "dnf": true, "journalctl": true,
	}
	if !allowlist[parts[0]] {
		return fmt.Errorf("command not in apply allowlist: %s", parts[0])
	}
	if needsRoot && os.Geteuid() != 0 {
		parts = append([]string{"sudo"}, parts...)
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func darwinPlaybooks(mount string) []domain.PlaybookStep {
	return []domain.PlaybookStep{
		{ID: "diagnose", Description: "Confirm disk space", Commands: []string{"df -h", "df -i", "df -h " + mount}, Risk: "low"},
		{ID: "pkg-cache", Description: "Homebrew cache cleanup", Commands: []string{"brew cleanup"}, Risk: "low"},
		{ID: "logs", Description: "Check unified logging /var/log size", Commands: []string{"du -sh /var/log"}, Risk: "low"},
		{ID: "temp", Description: "Clear user Library caches (review first)", Commands: []string{"du -sh ~/Library/Caches", "# rm -rf ~/Library/Caches/*"}, Risk: "high", RequiresRoot: false},
		{ID: "xcode-archives", Description: "Remove old Xcode archives", Commands: []string{"du -sh ~/Library/Developer/Xcode/Archives", "rm -rf ~/Library/Developer/Xcode/Archives/*"}, Risk: "medium"},
		{ID: "docker-safe", Description: "Docker prune unused", Commands: []string{"docker system df", "docker system prune -f"}, Risk: "low"},
		{ID: "docker-aggressive", Description: "Docker prune all unused images", Commands: []string{"docker system prune -a -f"}, Risk: "high"},
		{ID: "docker-volumes", Description: "Docker volume prune", Commands: []string{"docker volume prune -f"}, Risk: "high"},
		{ID: "ncdu", Description: "Interactive analyzer", Commands: []string{"brew install ncdu", "ncdu " + mount}, Risk: "low"},
		{ID: "verify", Description: "Verify space freed", Commands: []string{"df -h", "df -h " + mount}, Risk: "low"},
	}
}

func fedoraPlaybooks(mount string) []domain.PlaybookStep {
	return []domain.PlaybookStep{
		{ID: "diagnose", Description: "Confirm disk space", Commands: []string{"df -h", "df -i", "df -h " + mount}, Risk: "low"},
		{ID: "pkg-cache", Description: "DNF cache and autoremove", Commands: []string{"sudo dnf clean all", "sudo dnf autoremove -y"}, Risk: "low", RequiresRoot: true},
		{ID: "logs", Description: "Journal vacuum", Commands: []string{"journalctl --disk-usage", "sudo journalctl --vacuum-size=500M"}, Risk: "low", RequiresRoot: true},
		{ID: "temp", Description: "Temp and user cache (high risk)", Commands: []string{"# sudo rm -rf /tmp/*", "# rm -rf ~/.cache/*"}, Risk: "high"},
		{ID: "kernels", Description: "List kernels — remove old manually", Commands: []string{"rpm -qa | grep kernel", "# sudo dnf remove kernel-<old>"}, Risk: "medium", RequiresRoot: true},
		{ID: "docker-safe", Description: "Docker prune unused", Commands: []string{"docker system df", "docker system prune -f"}, Risk: "low"},
		{ID: "docker-aggressive", Description: "Docker prune all unused images", Commands: []string{"docker system prune -a -f"}, Risk: "high"},
		{ID: "docker-volumes", Description: "Docker volume prune", Commands: []string{"docker volume prune -f"}, Risk: "high"},
		{ID: "ncdu", Description: "Interactive analyzer", Commands: []string{"sudo dnf install -y ncdu", "sudo ncdu " + mount}, Risk: "low"},
		{ID: "emergency", Description: "Emergency journal shrink", Commands: []string{"sudo journalctl --vacuum-size=100M"}, Risk: "medium", RequiresRoot: true},
		{ID: "verify", Description: "Verify space freed", Commands: []string{"df -h", "df -h " + mount}, Risk: "low"},
	}
}
