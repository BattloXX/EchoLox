package bridge

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestInitLoxBerryLogTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test requires Unix")
	}

	initLog := writeInitLogStub(t, "sleep 10")
	start := time.Now()
	err := initLoxBerryLog(initLog, filepath.Join(t.TempDir(), "loxberry.log"), "test", 300*time.Millisecond)
	if err == nil {
		t.Fatal("initLoxBerryLog() error = nil, want timeout error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("initLoxBerryLog() took %v, want less than 2s", elapsed)
	}
}

func TestInitLoxBerryLogSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test requires Unix")
	}

	initLog := writeInitLogStub(t, "exit 0")
	if err := initLoxBerryLog(initLog, filepath.Join(t.TempDir(), "loxberry.log"), "test", time.Second); err != nil {
		t.Fatalf("initLoxBerryLog() error = %v, want nil", err)
	}
}

func writeInitLogStub(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "initlog.php")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}
