package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	msg, err := getClipboard(ctx)
	if err != nil {
		fmt.Printf("Error reading clipboard: %v\n", err)
	}
	if msg == "" {
		msg = "Clipboard is empty"
	}

	chatCfg, chromeCfg := defaultConfig()
	chatCfg.mainMessage = chatCfg.preMessage + msg
	allocOpts := buildAllocatorOpts(chromeCfg)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	if err := runAutomation(ctx, chatCfg); err != nil {
		log.Fatalf("automation failed: %v", err)
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
	preMessage         string
	mainMessage        string
}

type chromeConfig struct {
	execPath         string
	headless         bool
	userDataDir      string
	profileDir       string
	appURL           string
	enableAutomation bool
}

func defaultConfig() (chatConfig, chromeConfig) {
	cc := chatConfig{
		textBoxSelector:    `[contenteditable="true"][role="textbox"]`,
		sendButtonSelector: `button[data-testid="send-button"]`,
		preMessage:         "Explain: ",
	}

	chc := chromeConfig{
		execPath:         "/usr/bin/brave",
		headless:         false,
		userDataDir:      "/home/r4ppz/.config/BraveSoftware/Brave-Browser",
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

func runAutomation(ctx context.Context, cfg chatConfig) error {
	return chromedp.Run(ctx,
		chromedp.WaitVisible(cfg.textBoxSelector, chromedp.ByQuery),
		chromedp.Click(cfg.textBoxSelector, chromedp.ByQuery),
		chromedp.SendKeys(cfg.textBoxSelector, cfg.mainMessage, chromedp.ByQuery),
		chromedp.Click(cfg.sendButtonSelector, chromedp.ByQuery),
	)
}

func getClipboard(ctx context.Context) (string, error) {
	var out bytes.Buffer

	cmd := exec.CommandContext(ctx, "wl-paste", "-n", "-t", "text/plain")
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
