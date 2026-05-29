package scan

import (
	"context"
	"fmt"
	"strings"

	"github.com/hd-health/hd-health/internal/platform"
)

type Result struct {
	Volumes    string
	Inodes     string
	TopDirs    map[string][]string
	Suggestions []string
}

func Quick(ctx context.Context, plat platform.Platform, mount string) (*Result, error) {
	vols, err := plat.Volumes(ctx)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Volume inventory:\n")
	for _, v := range vols {
		fmt.Fprintf(&b, "  %s (%s): %.1f%% used, %.1f GB free / %.1f GB total\n",
			v.Name, v.Mount, v.UsedPercent, float64(v.Free)/(1<<30), float64(v.Capacity)/(1<<30))
	}
	inodes, _ := plat.InodeUsage(ctx)
	if len(inodes) > 0 {
		b.WriteString("\nInode usage:\n")
		for _, in := range inodes {
			if in.Mount == "/dev" || in.Mount == "/dev/fd" {
				continue
			}
			if in.Mount == "/" || in.Mount == "/System/Volumes/Data" || in.UsedPercent > 50 {
				fmt.Fprintf(&b, "  %s: %.1f%% inodes used\n", in.Mount, in.UsedPercent)
			}
		}
	}
	top := map[string][]string{}
	target := platform.ResolveMount(mount, vols)
	if target == "" && len(vols) > 0 {
		target = vols[0].Mount
	}
	if mount == "/" && target != "/" && target != mount {
		fmt.Fprintf(&b, "\nNote: user data lives on %s (not APFS root /).\n", target)
	}
	if target != "" {
		dirs, err := plat.TopDirs(ctx, target, 1)
		if err == nil && len(dirs) > 0 {
			b.WriteString(fmt.Sprintf("\nTop directories on %s:\n", target))
			// sort by size - simple bubble for top 10
			sorted := dirs
			for i := 0; i < len(sorted) && i < 10; i++ {
				maxI := i
				for j := i + 1; j < len(sorted); j++ {
					if sorted[j].Bytes > sorted[maxI].Bytes {
						maxI = j
					}
				}
				sorted[i], sorted[maxI] = sorted[maxI], sorted[i]
				d := sorted[i]
				fmt.Fprintf(&b, "  %.1f GB  %s\n", float64(d.Bytes)/(1<<30), d.Path)
				top[target] = append(top[target], fmt.Sprintf("%.1fGB %s", float64(d.Bytes)/(1<<30), d.Path))
			}
		}
	}
	return &Result{
		Volumes:     b.String(),
		Suggestions: plat.SuggestExternalTools(),
		TopDirs:     top,
	}, nil
}
