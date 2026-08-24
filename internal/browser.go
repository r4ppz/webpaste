package webpaste

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

const (
	debugPort = "9222"
	debugURL  = "http://127.0.0.1:" + debugPort

	portProbeTimeout = 300 * time.Millisecond
	portWaitTimeout  = 5 * time.Second
	portPollInterval = 100 * time.Millisecond
	targetTimeout    = 500 * time.Millisecond
)

type BrowserConfig struct {
	ExecPath    string
	UserDataDir string
	ProfileDir  string
	AppURL      string
}

func LoadBrowserConfig() BrowserConfig {
	home, _ := os.UserHomeDir()
	userData := filepath.Join(home, ".config", "BraveSoftware", "Brave-Browser")

	execPath, err := exec.LookPath("brave")
	if err != nil {
		execPath = "/usr/bin/brave"
	}

	return BrowserConfig{
		ExecPath:    execPath,
		UserDataDir: userData,
		ProfileDir:  "Default",
		AppURL:      "https://chatgpt.com/?temporary-chat=true",
	}
}

func EnsureBrowserRunning(cfg BrowserConfig) error {
	if isPortOpen() {
		log.Println("Reusing existing browser window...")
		return nil
	}

	log.Println("Launching background browser instance...")
	cmd := exec.Command(cfg.ExecPath,
		"--user-data-dir="+cfg.UserDataDir,
		"--profile-directory="+cfg.ProfileDir,
		"--app="+cfg.AppURL,
		"--remote-debugging-port="+debugPort,
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}

	return waitForPort(portWaitTimeout)
}

func TaskContext(parentCtx context.Context, domain string) (context.Context, bool) {
	allocCtx, _ := chromedp.NewRemoteAllocator(parentCtx, debugURL)
	browserCtx, _ := chromedp.NewContext(allocCtx)

	targetID := findTargetID(domain)
	if targetID != "" {
		taskCtx, _ := chromedp.NewContext(browserCtx, chromedp.WithTargetID(targetID))
		return taskCtx, true
	}

	return browserCtx, false
}

func isPortOpen() bool {
	client := http.Client{Timeout: portProbeTimeout}
	resp, err := client.Get(debugURL + "/json/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func waitForPort(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isPortOpen() {
			return nil
		}
		time.Sleep(portPollInterval)
	}
	return fmt.Errorf("timeout waiting for port %s", debugPort)
}

func findTargetID(domain string) target.ID {
	client := http.Client{Timeout: targetTimeout}
	resp, err := client.Get(debugURL + "/json/list")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var targets []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return ""
	}

	for _, t := range targets {
		if t.Type == "page" && strings.Contains(t.URL, domain) {
			return target.ID(t.ID)
		}
	}
	return ""
}
