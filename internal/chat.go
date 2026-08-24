package webpaste

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

const ChatDomain = "chatgpt.com"

type chatConfig struct {
	textBoxSelector    string
	sendButtonSelector string
	message            string
}

func NewChatConfig(message string) chatConfig {
	return chatConfig{
		textBoxSelector:    `textarea[name="prompt-textarea"]`,
		sendButtonSelector: `button[data-testid="send-button"]`,
		message:            message,
	}
}

func AutomateChatTask(cfg chatConfig, appURL string, isReusing bool) chromedp.Tasks {
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

func PromptMessage(ctx context.Context, premsg string) string {
	msg, err := getClipboard(ctx)
	if err != nil {
		log.Printf("Error reading clipboard: %v", err)
	}
	if msg == "" {
		msg = "Clipboard is empty"
	}
	return wrapMsg(premsg, msg)
}

func wrapMsg(premsg, msg string) string {
	return premsg + "\n```\n" + msg + "\n```\n"
}
