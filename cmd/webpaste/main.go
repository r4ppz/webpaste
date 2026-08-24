package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/chromedp/chromedp"

	webpaste "webpaste/internal"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	premsg := flag.String("premsg", "Explain: ", "Prefix message for clipboard content")
	flag.Parse()

	if err := webpaste.ClipboardAvailable(); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	message := webpaste.PromptMessage(ctx, *premsg)

	chromeCfg := webpaste.LoadBrowserConfig()
	if err := webpaste.EnsureBrowserRunning(chromeCfg); err != nil {
		return err
	}

	taskCtx, isReusing := webpaste.TaskContext(ctx, webpaste.ChatDomain)

	if err := chromedp.Run(
		taskCtx,
		webpaste.AutomateChatTask(webpaste.NewChatConfig(message), chromeCfg.AppURL, isReusing)); err != nil {
		return fmt.Errorf("chromedp execution failed: %w", err)
	}

	log.Println("Prompt sent successfully!")
	return nil
}
