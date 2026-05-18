package logbuf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotationAt5MB(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	bakPath := logPath + ".bak"

	l := &Logger{}
	if err := l.SetFile(logPath); err != nil {
		t.Fatalf("SetFile: %v", err)
	}
	t.Cleanup(func() { l.SetFile("") })

	// Write enough to exceed 5 MB (each line ~1052 bytes with timestamp prefix)
	line := strings.Repeat("x", 1024)
	for i := 0; i < 5200; i++ {
		l.Info("%s", line)
	}

	// After first rotation: .bak must exist and the current log must be fresh (small)
	if _, err := os.Stat(bakPath); err != nil {
		t.Fatalf("expected %s to exist after first rotation: %v", bakPath, err)
	}
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("log file missing after first rotation: %v", err)
	}
	if info.Size() >= maxFileSize {
		t.Fatalf("log file not rotated: size=%d >= %d", info.Size(), maxFileSize)
	}

	// Second rotation: write another 5+ MB; the old .bak should be overwritten
	bak1, _ := os.Stat(bakPath)
	for i := 0; i < 5200; i++ {
		l.Info("%s", line)
	}
	bak2, err := os.Stat(bakPath)
	if err != nil {
		t.Fatalf(".bak missing after second rotation: %v", err)
	}
	// The .bak must have been rewritten (its ModTime advances, or at minimum it exists)
	if !bak2.ModTime().After(bak1.ModTime()) {
		t.Fatalf(".bak was not overwritten on second rotation (ModTime unchanged)")
	}
	// The current log must again be small (freshly rotated)
	info2, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("log file missing after second rotation: %v", err)
	}
	if info2.Size() >= maxFileSize {
		t.Fatalf("log file not rotated second time: size=%d >= %d", info2.Size(), maxFileSize)
	}
}

func TestNoRotationBelow5MB(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	l := &Logger{}
	if err := l.SetFile(logPath); err != nil {
		t.Fatalf("SetFile: %v", err)
	}
	t.Cleanup(func() { l.SetFile("") })

	l.Info("small entry")

	if _, err := os.Stat(logPath + ".bak"); err == nil {
		t.Fatal("unexpected .bak created for small log")
	}
}
