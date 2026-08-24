package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

const (
	debugPort = "9222"
	debugURL  = "http://127.0.0.1:" + debugPort
)

type chatConfig struct {
	textBoxSelector    string
	sendButtonSelector string
	message            string
}

type chromeConfig struct {
	execPath    string
	userDataDir string
	profileDir  string
	appURL      string
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	premsg := flag.String("premsg", "Explain: ", "Prefix message for clipboard content")
	flag.Parse()

	if _, err := exec.LookPath("wl-paste"); err != nil {
		return fmt.Errorf("wl-paste not found in PATH: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	chatCfg, chromeCfg := populateConfig()
	chatCfg.message = getPromptMessage(ctx, *premsg)

	if err := ensureBrowserRunning(chromeCfg); err != nil {
		return err
	}

	taskCtx, isReusing := getTaskContext(ctx, "chatgpt.com")

	if err := chromedp.Run(
		taskCtx,
		automateChatTask(chatCfg, chromeCfg.appURL, isReusing)); err != nil {
		return fmt.Errorf("chromedp execution failed: %w", err)
	}

	log.Println("Prompt sent successfully!")
	return nil
}

func ensureBrowserRunning(cfg chromeConfig) error {
	if isPortOpen(debugPort) {
		log.Println("Reusing existing browser window...")
		return nil
	}

	log.Println("Launching background browser instance...")
	cmd := exec.Command(cfg.execPath,
		"--user-data-dir="+cfg.userDataDir,
		"--profile-directory="+cfg.profileDir,
		"--app="+cfg.appURL,
		"--remote-debugging-port="+debugPort,
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}

	return waitForPort(debugPort, 5*time.Second)
}

func getTaskContext(parentCtx context.Context, domain string) (context.Context, bool) {
	allocCtx, _ := chromedp.NewRemoteAllocator(parentCtx, debugURL)
	browserCtx, _ := chromedp.NewContext(allocCtx)

	targetID := findExistingTargetID(domain)
	if targetID != "" {
		taskCtx, _ := chromedp.NewContext(browserCtx, chromedp.WithTargetID(targetID))
		return taskCtx, true
	}

	return browserCtx, false
}

func getPromptMessage(ctx context.Context, premsg string) string {
	msg, err := getClipboard(ctx)
	if err != nil {
		log.Printf("Error reading clipboard: %v", err)
	}
	if msg == "" {
		msg = "Clipboard is empty"
	}
	return wrapMsg(premsg, msg)
}

func automateChatTask(cfg chatConfig, appURL string, isReusing bool) chromedp.Tasks {
	var refreshTask chromedp.Tasks

	if isReusing {
		refreshTask = chromedp.Tasks{
			chromedp.Evaluate(fmt.Sprintf(`window.location.href = %s;`, strconv.Quote(appURL)), nil),
			chromedp.Sleep(1 * time.Second),
		}
	} else {
		refreshTask = chromedp.Tasks{
			chromedp.Navigate(appURL),
		}
	}

	return chromedp.Tasks{
		refreshTask,
		chromedp.WaitVisible(cfg.textBoxSelector, chromedp.ByQuery),
		chromedp.Focus(cfg.textBoxSelector, chromedp.ByQuery),
		typeDirectTask(cfg.textBoxSelector, cfg.message),
		chromedp.WaitEnabled(cfg.sendButtonSelector, chromedp.ByQuery),
		chromedp.SendKeys(cfg.textBoxSelector, kb.Enter, chromedp.ByQuery),
		chromedp.Click(cfg.sendButtonSelector, chromedp.ByQuery),
	}
}

func typeDirectTask(selector, text string) chromedp.Tasks {
	js := fmt.Sprintf(`document.querySelector(%q).value = %s`, selector, strconv.Quote(text))
	return chromedp.Tasks{
		chromedp.Evaluate(js, nil),
	}
}

func isPortOpen(port string) bool {
	client := http.Client{Timeout: 300 * time.Millisecond}
	resp, err := client.Get("http://127.0.0.1:" + port + "/json/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func waitForPort(port string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isPortOpen(port) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for port %s", port)
}

func findExistingTargetID(domain string) target.ID {
	client := http.Client{Timeout: 500 * time.Millisecond}
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

func getClipboard(parentCtx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wl-paste", "-n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("wl-paste failed: %w - output: %s", err, string(out))
	}

	return string(out), nil
}

func populateConfig() (chatConfig, chromeConfig) {
	cc := chatConfig{
		textBoxSelector:    `textarea[name="prompt-textarea"]`,
		sendButtonSelector: `button[data-testid="send-button"]`,
	}

	home, _ := os.UserHomeDir()
	userData := filepath.Join(home, ".config", "BraveSoftware", "Brave-Browser")
	// userData := filepath.Join(home, ".config", "BraveSoftware", "Brave-Automation")

	execPath, err := exec.LookPath("brave")
	if err != nil {
		execPath = "/usr/bin/brave"
	}

	chc := chromeConfig{
		execPath:    execPath,
		userDataDir: userData,
		profileDir:  "Default",
		appURL:      "https://chatgpt.com/?temporary-chat=true",
	}

	return cc, chc
}

func wrapMsg(premsg, msg string) string {
	return premsg + "\n```\n" + msg + "\n```\n"
}
