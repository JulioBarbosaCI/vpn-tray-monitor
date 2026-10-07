package winsys

import (
	"os"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lendo %s: %v", path, err)
	}
	return string(raw)
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
