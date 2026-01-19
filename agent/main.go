package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
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
	log.Printf("agent %s starting", agentID)

	cfg := fetchConfig()

	for _, src := range cfg.Config.Sources {
		if src.Type == "file" {
			go tailFile(src.Path)
		}
	}

	// Keep agent running
	select {}
}

func fetchConfig() AgentConfigResponse {
	url := fmt.Sprintf("%s/agents/%s/config", controlPlaneURL, agentID)
	log.Printf("fetching config from %s", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("error calling control plane: %v", err)
	}
	defer resp.Body.Close()

	var cfg AgentConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		log.Fatalf("error decoding config: %v", err)
	}

	log.Printf("got config version %s", cfg.Version)
	return cfg
}

func tailFile(path string) {
	log.Printf("starting file tail for %s", path)

	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("failed to open file %s: %v", path, err)
	}

	// Start at end of file
	file.Seek(0, os.SEEK_END)

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("[file:%s] %s", path, line)
	}
}
