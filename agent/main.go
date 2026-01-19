package main

import (
	"bufio"
	"bytes"
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
	log.Printf("agent %s starting (TAIL -> DESTINATION)", agentID)

	cfg := fetchConfig()
	dest := cfg.Config.Destination

	for _, src := range cfg.Config.Sources {
		if src.Type == "file" && src.Path != "" {
			go tailFile(src.Path, dest)
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
	log.Printf("destination from control plane: %+v", cfg.Config.Destination)
	return cfg
}

func tailFile(path string, dest Destination) {
	log.Printf("tailing %s (new lines only)", path)

	for {
		f, err := os.Open(path)
		if err != nil {
			log.Printf("cannot open %s: %v (retrying in 3s)", path, err)
			time.Sleep(3 * time.Second)
			continue
		}

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
				break
			}

			// still log locally for debugging
			log.Printf("[TAIL][%s] %s", path, line)

			// send to destination (Splunk HEC)
			if err := sendToDestination(dest, path, line); err != nil {
				log.Printf("send error: %v", err)
			}
		}
	}
}

func sendToDestination(dest Destination, sourcePath, line string) error {
	if dest.Type != "splunk_hec" {
		// nothing implemented yet for other types
		return nil
	}

	token := os.Getenv("SPLUNK_HEC_TOKEN")
	if token == "" {
		return fmt.Errorf("SPLUNK_HEC_TOKEN not set; cannot send to Splunk")
	}

	// Splunk HEC event format
	payload := map[string]interface{}{
		"event":      line,
		"source":     sourcePath,
		"sourcetype": "lab:linux:auth",
		"time":       time.Now().Unix(),
		"host":       getHostname(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", dest.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Splunk "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("splunk hec returned status %d", resp.StatusCode)
	}

	return nil
}

func getHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown-host"
	}
	return h
}
