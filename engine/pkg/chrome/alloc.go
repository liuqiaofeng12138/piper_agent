package chrome

import (
	"context"
	"os"

	"piper_go/internal/config"

	"github.com/chromedp/chromedp"
)

func chromedpAllocatorOptions(cfg config.Config) []chromedp.ExecAllocatorOption {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", cfg.Chrome.Headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("exclude-switches", "enable-automation"),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
	)
	if cfg.Chrome.BinaryPath != "" {
		opts = append(opts, chromedp.ExecPath(cfg.Chrome.BinaryPath))
	}
	if dir := cfg.Chrome.UserDataDir; dir != "" {
		_ = os.MkdirAll(dir, 0o755)
		opts = append(opts, chromedp.Flag("user-data-dir", dir))
	}
	return opts
}

func chromedpNewExecAllocator(parent context.Context, opts ...chromedp.ExecAllocatorOption) (context.Context, context.CancelFunc) {
	return chromedp.NewExecAllocator(parent, opts...)
}

func chromedpNewContext(parent context.Context) (context.Context, context.CancelFunc) {
	return chromedp.NewContext(parent)
}
