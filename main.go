package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/chromedp/chromedp"
)

type chatConfig struct {
	execPath           string
	headless           bool
	userDataDir        string
	profileDir         string
	appURL             string
	enableAutomation   bool
	textBoxSelector    string
	sendButtonSelector string
	message            string
}

func defaultConfig() chatConfig {
	return chatConfig{
		execPath:           "/usr/bin/brave",
		headless:           false,
		userDataDir:        "/home/r4ppz/.config/BraveSoftware/Brave-Browser",
		profileDir:         "Default",
		appURL:             "https://chatgpt.com/?temporary-chat=true",
		enableAutomation:   false,
		textBoxSelector:    `[contenteditable="true"][role="textbox"]`,
		sendButtonSelector: `button[data-testid="send-button"]`,
		message:            "Hello, testing!",
	}
}

func buildAllocatorOpts(cfg chatConfig) []chromedp.ExecAllocatorOption {
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
		chromedp.SendKeys(cfg.textBoxSelector, cfg.message, chromedp.ByQuery),
		chromedp.WaitNotPresent(cfg.sendButtonSelector+"[disabled]", chromedp.ByQuery),
		chromedp.Click(cfg.sendButtonSelector, chromedp.ByQuery),
	)
}

func main() {
	cfg := defaultConfig()
	allocOpts := buildAllocatorOpts(cfg)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	if err := runAutomation(ctx, cfg); err != nil {
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
