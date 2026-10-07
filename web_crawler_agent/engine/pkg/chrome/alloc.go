package chrome

import (
	"context"

	"piper_go/internal/config"

	"github.com/chromedp/chromedp"
)

func chromedpAllocatorOptions(cfg config.Config) []chromedp.ExecAllocatorOption {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", cfg.Chrome.Headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)
	if cfg.Chrome.BinaryPath != "" {
		opts = append(opts, chromedp.ExecPath(cfg.Chrome.BinaryPath))
	}
	return opts
}

func chromedpNewExecAllocator(parent context.Context, opts ...chromedp.ExecAllocatorOption) (context.Context, context.CancelFunc) {
	return chromedp.NewExecAllocator(parent, opts...)
}

func chromedpNewContext(parent context.Context) (context.Context, context.CancelFunc) {
	return chromedp.NewContext(parent)
}
