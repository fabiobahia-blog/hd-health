package platform

import (
	"runtime"
	"strings"

	"github.com/hd-health/hd-health/internal/domain"
)

// ResolveMount returns the best mount for scans/explain when the user passes "/" on macOS.
func ResolveMount(requested string, vols []domain.Volume) string {
	if requested != "" && requested != "/" {
		return requested
	}
	if runtime.GOOS == "darwin" {
		for _, v := range vols {
			if v.Mount == "/System/Volumes/Data" {
				return v.Mount
			}
		}
		for _, v := range vols {
			if strings.Contains(v.Mount, "Data") && v.UsedPercent > 0 {
				return v.Mount
			}
		}
	}
	if len(vols) > 0 {
		best := vols[0]
		for _, v := range vols {
			if v.Mount == "/" {
				best = v
			}
			if v.UsedPercent > best.UsedPercent && !v.IsNetwork {
				best = v
			}
		}
		if best.Mount != "" {
			return best.Mount
		}
	}
	return requested
}
