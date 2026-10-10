package chrome

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"piper_go/internal/config"

	"github.com/chromedp/chromedp"
)

func TestChromedpStartsWithAllocatorOptions(t *testing.T) {
	if os.Getenv("CHROME_SMOKE") != "1" {
		t.Skip("set CHROME_SMOKE=1 to run")
	}
	dir := filepath.Join(t.TempDir(), "profile")
	cfg := config.Config{Chrome: config.ChromeConf{Headless: true, UserDataDir: dir}}
	opts := chromedpAllocatorOptions(cfg)
	alloc, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	ctx, cancel := chromedp.NewContext(alloc)
	defer cancel()
	if err := chromedp.Run(ctx); err != nil {
		t.Fatalf("chromedp.Run: %v", err)
	}
}
