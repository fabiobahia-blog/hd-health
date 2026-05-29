package report

import (
	"testing"

	"github.com/hd-health/hd-health/internal/domain"
)

func TestExitCodeCritical(t *testing.T) {
	f := []domain.Finding{{Severity: domain.SeverityCritical, Code: "disk_full"}}
	if exitCode(f, nil) != 2 {
		t.Fatalf("expected 2 got %d", exitCode(f, nil))
	}
}

func TestExitCodeWarning(t *testing.T) {
	f := []domain.Finding{{Severity: domain.SeverityWarning, Code: "disk_warning"}}
	if exitCode(f, nil) != 1 {
		t.Fatalf("expected 1 got %d", exitCode(f, nil))
	}
}
