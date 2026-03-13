package main

import (
	"strings"
	"testing"
)

func TestDbConfig(t *testing.T) {
	dialect, addr := dbConfig()
	if dialect != "sqlite" {
		t.Fatalf("expected 'sqlite', got %q", dialect)
	}
	if !strings.Contains(addr, "database.db") {
		t.Fatalf("expected 'database.db' in addr, got %q", addr)
	}
	for _, pragma := range []string{"foreign_keys", "journal_mode", "busy_timeout"} {
		if !strings.Contains(addr, pragma) {
			t.Errorf("expected pragma %q in addr, got %q", pragma, addr)
		}
	}
	if strings.Count(addr, "file:") != 1 {
		t.Fatalf("expected exactly one 'file:' in addr, got %q", addr)
	}
}
