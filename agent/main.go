package main

import (
	"encoding/json"
	"fmt"
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

const (
	agentID         = "ubuntu-01"
	controlPlaneURL = "http://localhost:8080"
)

func main() {
	log.Printf("agent %s starting up", agentID)

	// Build URL like: http://localhost:8080/agents/ubuntu-01/config
	url := fmt.Sprintf("%s/agents/%s/config", controlPlaneURL, agentID)
	log.Printf("fetching config from %s", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("error calling control plane: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("control plane returned status %d", resp.StatusCode)
	}

	var cfg AgentConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		log.Fatalf("error decoding config: %v", err)
	}

	log.Printf("got config version %s", cfg.Version)
	log.Printf("sources: %+v", cfg.Config.Sources)
	log.Printf("destination: %+v", cfg.Config.Destination)

	// For now, just exit after printing config.
	// Next step: we'll actually start ingestion based on cfg.
}
