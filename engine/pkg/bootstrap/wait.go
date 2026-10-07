package bootstrap

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"piper_go/internal/config"

	"github.com/zeromicro/go-zero/core/logx"
)

// WaitStorageReady blocks until ES and S3 endpoints respond or ctx is cancelled.
func WaitStorageReady(ctx context.Context, c config.Config) error {
	if c.WebAPI.SkipStorageWait {
		logx.Info("skipStorageWait=true, not waiting for ES/S3")
		return nil
	}

	interval := time.Duration(c.WebAPI.StorageWaitEvery) * time.Millisecond
	if interval <= 0 {
		interval = 5 * time.Second
	}

	esURL := fmt.Sprintf("http://%s:%d", c.ES.Host, c.ES.Port)
	s3URL := c.S3.EndpointURL

	var esDone, s3Done bool
	for !esDone || !s3Done {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !esDone && pingHTTP(ctx, esURL) {
			logx.Infof("ES ready at %s", esURL)
			esDone = true
		} else if !esDone {
			logx.Infof("ES not ready at %s, wait...", esURL)
		}

		if !s3Done && pingHTTP(ctx, s3URL) {
			logx.Infof("S3 ready at %s", s3URL)
			s3Done = true
		} else if !s3Done {
			logx.Infof("S3 not ready at %s, wait...", s3URL)
		}

		if esDone && s3Done {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
	return nil
}

func pingHTTP(ctx context.Context, rawURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// MinIO/ES may reject GET on root; treat TCP reachability as enough.
		if _, tcpErr := net.DialTimeout("tcp", hostPort(rawURL), 3*time.Second); tcpErr == nil {
			return true
		}
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

func hostPort(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return "127.0.0.1:80"
	}
	return u.Host
}
