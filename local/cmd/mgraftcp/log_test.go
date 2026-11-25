package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmgle/graftcp/local"
)

func TestDebugLogWritesToStderrAndFile(t *testing.T) {
	oldStderr, oldLogger := os.Stderr, appLogger
	terminal, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	os.Stderr = terminal
	defer func() {
		os.Stderr, appLogger = oldStderr, oldLogger
		local.SetLogger(nil)
	}()
	path := filepath.Join(t.TempDir(), "debug.log")
	file, err := openDebugLog(path)
	if err != nil {
		t.Fatal(err)
	}
	appLogger.Debugf("DNS upstream=%s", "1.1.1.1:53")
	file.Close()
	log, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(terminal.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output, log) || !strings.Contains(string(log), "[DEBUG] DNS upstream=1.1.1.1:53") {
		t.Fatalf("terminal=%q file=%q", output, log)
	}
}
