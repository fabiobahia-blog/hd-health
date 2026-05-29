package platform

import (
	"context"
	"strings"
)

// SMARTStatus returns a summary if smartctl is installed and a device is given.
func SMARTStatus(ctx context.Context, device string) string {
	if device == "" || !which("smartctl") {
		return ""
	}
	out, err := run(ctx, "smartctl", "-H", device)
	if err != nil {
		return ""
	}
	if strings.Contains(strings.ToUpper(out), "PASSED") {
		return "PASSED"
	}
	if strings.Contains(strings.ToUpper(out), "FAILED") {
		return "FAILED"
	}
	return strings.TrimSpace(out)
}
