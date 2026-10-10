package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonProjectDirFromModule(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	mustMkdir(t, filepath.Join(repo, "rag_agent"))
	_ = os.WriteFile(filepath.Join(repo, "rag_agent", "pyproject.toml"), []byte("[project]\n"), 0o644)

	a := AgentSpec{ID: "doc_rag", Module: "rag_agent.workers.doc_rag"}
	dir, err := a.PythonProjectDir(repo)
	if err != nil || dir != "rag_agent" {
		t.Fatalf("dir=%q err=%v", dir, err)
	}
}

func TestPythonProjectDirPiperAgent(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	mustMkdir(t, filepath.Join(repo, "general_agent"))
	_ = os.WriteFile(filepath.Join(repo, "general_agent", "pyproject.toml"), []byte("[project]\n"), 0o644)

	a := AgentSpec{ID: "general_chat", Module: "piper_agent.workers.general_chat"}
	dir, err := a.PythonProjectDir(repo)
	if err != nil || dir != "general_agent" {
		t.Fatalf("dir=%q err=%v", dir, err)
	}
}

func TestPythonProjectDirOverride(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	mustMkdir(t, filepath.Join(repo, "custom_agent"))
	_ = os.WriteFile(filepath.Join(repo, "custom_agent", "pyproject.toml"), []byte("[project]\n"), 0o644)

	a := AgentSpec{
		ID:            "x",
		Module:        "other_pkg.workers.x",
		PythonProject: "custom_agent",
	}
	dir, err := a.PythonProjectDir(repo)
	if err != nil || dir != "custom_agent" {
		t.Fatalf("dir=%q err=%v", dir, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
