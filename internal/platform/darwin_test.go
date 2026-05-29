package platform

import (
	"strings"
	"testing"
)

func TestParseMacOSDfInodeLine(t *testing.T) {
	line := "/dev/disk3s5 965595304 812817640 75967264 92% 2543176 379836320 1% /System/Volumes/Data"
	fields := splitFields(line)
	if len(fields) != 9 {
		t.Fatalf("fields: %d", len(fields))
	}
	pctStr := strings.TrimSuffix(fields[7], "%")
	if pctStr != "1" {
		t.Fatalf("inode pct want 1 got %s", pctStr)
	}
	if fields[4] == "92%" {
		// capacity column must not be used as inode pct
		capPct := strings.TrimSuffix(fields[4], "%")
		if capPct == "1" {
			t.Fatal("confused capacity with inode")
		}
	}
}
