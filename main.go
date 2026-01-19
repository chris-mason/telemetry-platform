package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type AgentConfigResponse struct {
	Version string      `json:"version"`
	Config  AgentConfig `json:"config"`
}

type AgentConfig struct {
	Sources     []Source    `json:"sources"`
	Destination Destination `json:"destination"`
}

type Source struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
}

type Destination struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	http.HandleFunc("/agents/", func(w http.ResponseWriter, r *http.Request) {
		// Expect: /agents/{id}/config
		// Minimal parsing without extra dependencies
		path := r.URL.Path
		// Path format: /agents/<id>/config
		parts := splitPath(path)
		if len(parts) != 3 || parts[0] != "agents" || parts[2] != "config" {
			http.NotFound(w, r)
			return
		}
		agentID := parts[1]

		log.Printf("serving config for agent %s", agentID)

		resp := AgentConfigResponse{
			Version: "2026-01-18-001",
			Config: AgentConfig{
				Sources: []Source{
					{ID: "linux_auth", Type: "file", Path: "/var/log/auth.log"},
				},
				Destination: Destination{
					Type: "http",
					URL:  "http://localhost:8081/ingest/logs",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	log.Println("control plane listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// splitPath turns "/agents/agent-123/config" into ["agents","agent-123","config"]
func splitPath(p string) []string {
	// trim leading/trailing slashes manually to avoid extra imports
	for len(p) > 0 && p[0] == '/' {
		p = p[1:]
	}
	for len(p) > 0 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	if p == "" {
		return []string{}
	}
	out := []string{}
	cur := ""
	for i := 0; i < len(p); i++ {
		if p[i] == '/' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(p[i])
	}
	out = append(out, cur)
	return out
}
