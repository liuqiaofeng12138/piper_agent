package artifacts

import (
	"path/filepath"
	"testing"
)

func TestPPTXPathRejectsBadID(t *testing.T) {
	_, err := PPTXPath("/tmp", "../evil")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSanitizeFilename(t *testing.T) {
	if SanitizeFilename("") != "presentation.pptx" {
		t.Fatal(SanitizeFilename(""))
	}
	if SanitizeFilename("报告.pptx") != "报告.pptx" {
		t.Fatal(SanitizeFilename("报告.pptx"))
	}
}

func TestResolveDirRelative(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "local.yaml")
	dir, err := ResolveDir(cfg, "artifacts")
	if err != nil {
		t.Fatal(err)
	}
	if dir == "" {
		t.Fatal("empty dir")
	}
}
