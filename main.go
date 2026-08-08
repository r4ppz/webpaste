package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/chromedp/chromedp"
)

func main() {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/usr/bin/brave"),
		chromedp.Flag("headless", false),
		chromedp.Flag(
			"user-data-dir",
			"/home/r4ppz/.config/BraveSoftware/Brave-Browser",
		),
		chromedp.Flag(
			"profile-directory",
			"Default",
		),
		chromedp.Flag(
			"app",
			"https://chatgpt.com/?temporary-chat=true",
		),
		chromedp.Flag("enable-automation", false),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(
		context.Background(),
		opts...,
	)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	const (
		textBox       = `[contenteditable="true"][role="textbox"]`
		sendButton    = `button[data-testid="send-button"]`
		messageToSend = `Hello, testing!`
	)

	err := chromedp.Run(
		ctx,
		chromedp.WaitVisible(
			textBox,
			chromedp.ByQuery,
		),

		chromedp.Click(
			textBox,
			chromedp.ByQuery,
		),

		chromedp.SendKeys(
			textBox,
			messageToSend,
			chromedp.ByQuery,
		),

		chromedp.WaitNotPresent(
			sendButton+"[disabled]",
			chromedp.ByQuery,
		),

		chromedp.Click(
			sendButton,
			chromedp.ByQuery,
		),
	)

	if err != nil {
		log.Fatal(err)
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
