package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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
	log.Printf("agent %s starting (TAIL MODE)", agentID)

	cfg := fetchConfig()

	for _, src := range cfg.Config.Sources {
		if src.Type == "file" && src.Path != "" {
			go tailFile(src.Path)
		} else {
			log.Printf("source not supported yet: %+v", src)
		}
	}

	// keep running
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

func tailFile(path string) {
	log.Printf("tailing %s (new lines only)", path)

	for {
		f, err := os.Open(path)
		if err != nil {
			log.Printf("cannot open %s: %v (retrying in 3s)", path, err)
			time.Sleep(3 * time.Second)
			continue
		}

		// Start at end so we only see new lines from now on
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			log.Printf("seek end failed on %s: %v", path, err)
		}

		reader := bufio.NewReader(f)

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				log.Printf("read error on %s: %v (reopening file)", path, err)
				_ = f.Close()
				break // reopen outer loop (handles rotate/truncate too)
			}

			// Print exactly what the agent ingested
			log.Printf("[TAIL][%s] %s", path, line)
		}
	}
}
