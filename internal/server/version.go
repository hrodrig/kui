package server

import (
	"encoding/json"
	"net/http"

	"github.com/hrodrig/kui/internal/version"
)

// VersionPath exposes build metadata for operators.
const VersionPath = "/api/v1/version"

func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(version.BuildInfo())
}
