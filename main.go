package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

func main() {
	premsg := flag.String("premsg", "Explain: ", "Prefix message for the clipboard content")
	flag.Parse()

	if _, err := exec.LookPath("wl-paste"); err != nil {
		log.Fatalf("wl-paste not found in PATH: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	msg, err := getClipboard(ctx)
	if err != nil {
		fmt.Printf("Error reading clipboard: %v\n", err)
	}
	if msg == "" {
		msg = "Clipboard is empty"
	}

	chatCfg, chromeCfg := populateConfig()
	chatCfg.message = wrapMsg(*premsg, msg)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(
		context.Background(),
		buildAllocatorOpts(chromeCfg)...,
	)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	if err := chromedp.Run(ctx, automateChatTask(chatCfg)); err != nil {
		log.Fatalf("wl-paste not found in PATH: %v", err)
	}

	// Wait until either the browser is closed or Ctrl+C is pressed.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	select {
	case <-ctx.Done():
		log.Println("Browser closed.")
	case <-sig:
		log.Println("Ctrl+C received.")
	}
}

type chatConfig struct {
	textBoxSelector    string
	sendButtonSelector string
	message            string
}

type chromeConfig struct {
	execPath         string
	headless         bool
	userDataDir      string
	profileDir       string
	appURL           string
	enableAutomation bool
}

func populateConfig() (chatConfig, chromeConfig) {
	cc := chatConfig{
		textBoxSelector:    `[contenteditable="true"][role="textbox"]`,
		sendButtonSelector: `button[data-testid="send-button"]`,
	}

	home, _ := os.UserHomeDir()
	userData := filepath.Join(home, ".config", "BraveSoftware", "Brave-Browser")

	execPath, err := exec.LookPath("brave")
	if err != nil {
		execPath = "/usr/bin/brave"
	}

	chc := chromeConfig{
		execPath:         execPath,
		headless:         false,
		userDataDir:      userData,
		profileDir:       "Default",
		appURL:           "https://chatgpt.com/?temporary-chat=true",
		enableAutomation: false,
	}

	return cc, chc
}

func buildAllocatorOpts(cfg chromeConfig) []chromedp.ExecAllocatorOption {
	return append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(cfg.execPath),
		chromedp.Flag("headless", cfg.headless),
		chromedp.Flag("user-data-dir", cfg.userDataDir),
		chromedp.Flag("profile-directory", cfg.profileDir),
		chromedp.Flag("app", cfg.appURL),
		chromedp.Flag("enable-automation", cfg.enableAutomation),
	)
}

func automateChatTask(cfg chatConfig) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.WaitVisible(cfg.textBoxSelector, chromedp.ByQuery),
		chromedp.Click(cfg.textBoxSelector, chromedp.ByQuery),
		typeDirectTask(cfg.textBoxSelector, cfg.message),
		chromedp.WaitEnabled(cfg.sendButtonSelector, chromedp.ByQuery),
		chromedp.Click(cfg.sendButtonSelector, chromedp.ByQuery),
		chromedp.SendKeys(cfg.textBoxSelector, kb.Enter, chromedp.ByQuery),
	}
}

func typeDirectTask(selector, text string) chromedp.Tasks {
	js := fmt.Sprintf(`document.querySelector(%q).innerText = %s`, selector, strconv.Quote(text))
	return chromedp.Tasks{
		chromedp.Evaluate(js, nil),
	}
}

func getClipboard(ctx context.Context) (string, error) {
	var out bytes.Buffer

	cmd := exec.CommandContext(ctx, "wl-paste", "-n")
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf(
			"wl-paste failed: %w - output: %s",
			err,
			out.String(),
		)
	}

	return out.String(), nil
}

func wrapMsg(premsg, msg string) string {
	return premsg + "\n```\n" + msg + "\n```\n"
}
