package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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
	log.Printf("agent %s starting (DEBUG FILE READER)", agentID)

	cfg := fetchConfig()

	for _, src := range cfg.Config.Sources {
		if src.Type == "file" {
			go debugReadWholeFileLoop(src.Path)
		} else {
			log.Printf("source type %s not implemented yet", src.Type)
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

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("control plane returned status %d", resp.StatusCode)
	}

	var cfg AgentConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		log.Fatalf("error decoding config: %v", err)
	}

	log.Printf("got config version %s", cfg.Version)
	return cfg
}

// debugReadWholeFileLoop repeatedly reads the entire file and prints all lines.
// Super noisy, but great to prove the agent can read the file.
func debugReadWholeFileLoop(path string) {
	for {
		log.Printf("[DEBUG] reading entire file %s", path)

		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[DEBUG] failed to read %s: %v", path, err)
			time.Sleep(5 * time.Second)
			continue
		}

		text := string(data)
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			log.Printf("[AGENT READ][%s] %s", path, line)
		}

		time.Sleep(5 * time.Second)
	}
}
