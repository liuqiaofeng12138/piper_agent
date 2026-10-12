package api

import (
	"net/http"
	"os"
	"path/filepath"

	"piper_agent/gateway/internal/artifacts"
)

func (s *Server) downloadArtifact(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path, err := artifacts.PPTXPath(s.deps.ArtifactsDir, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	name := artifacts.SanitizeFilename(r.URL.Query().Get("filename"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	http.ServeFile(w, r, filepath.Clean(path))
}
