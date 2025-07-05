package server

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	"time"
)

func (s *Server) infoHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"app":       s.appName,
		"version":   s.appVersion,
		"env":       s.appEnv,
		"commit":    s.appCommit,
		"timestamp": time.Now(),
		"uptime":    time.Since(s.start).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(info); err != nil {
		s.logger.Error("failed to encode info response", "error", err)
	}
}

func getCommit() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	return ""
}
