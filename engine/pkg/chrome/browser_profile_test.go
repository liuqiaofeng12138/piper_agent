package chrome

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"piper_go/internal/config"

	"github.com/chromedp/chromedp"
)

func TestChromedpWithProjectProfile(t *testing.T) {
	if os.Getenv("CHROME_SMOKE") != "1" {
		t.Skip("set CHROME_SMOKE=1")
	}
	root := filepath.Join("..", "..", "..")
	profile := filepath.Join(root, "data", "chrome_profile")
	cfg := config.Config{Chrome: config.ChromeConf{Headless: false, UserDataDir: profile}}
	opts := chromedpAllocatorOptions(cfg)
	alloc, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	ctx, cancel := chromedp.NewContext(alloc)
	defer cancel()
	if err := chromedp.Run(ctx); err != nil {
		t.Fatalf("headed profile %q: %v", profile, err)
	}
}

func TestChromedpFreshPiperProfile(t *testing.T) {
	if os.Getenv("CHROME_SMOKE") != "1" {
		t.Skip("set CHROME_SMOKE=1")
	}
	profile := filepath.Join(t.TempDir(), "chrome_piper", "CA-1")
	cfg := config.Config{Chrome: config.ChromeConf{Headless: true, UserDataDir: profile}}
	opts := chromedpAllocatorOptions(cfg)
	alloc, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	ctx, cancel := chromedp.NewContext(alloc)
	defer cancel()
	if err := chromedp.Run(ctx); err != nil {
		t.Fatalf("chrome_piper profile %q: %v", profile, err)
	}
}

func TestChromedpHeadlessProjectProfile(t *testing.T) {
	if os.Getenv("CHROME_SMOKE") != "1" {
		t.Skip("set CHROME_SMOKE=1")
	}
	root := filepath.Join("..", "..", "..")
	profile := filepath.Join(root, "data", "chrome_profile")
	cfg := config.Config{Chrome: config.ChromeConf{Headless: true, UserDataDir: profile}}
	opts := chromedpAllocatorOptions(cfg)
	alloc, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	ctx, cancel := chromedp.NewContext(alloc)
	defer cancel()
	if err := chromedp.Run(ctx); err != nil {
		t.Fatalf("headless profile %q: %v", profile, err)
	}
}
