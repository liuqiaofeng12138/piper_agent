package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"piper_go/internal/config"

	"github.com/zeromicro/go-zero/core/logx"
)

const grafanaDashboardUID = "MM3UsuH7z"

var grafanaDashboardSearchPaths = []string{
	"docker/grafana/provisioning/dashboards/piper-dashboard.json",
	"../docker/grafana/provisioning/dashboards/piper-dashboard.json",
	"../../docker/grafana/provisioning/dashboards/piper-dashboard.json",
}

// EnsureGrafanaDashboard imports the Piper Grafana dashboard when Grafana is up but the dashboard is missing.
func EnsureGrafanaDashboard(ctx context.Context, c config.Config) {
	if !c.WebAPI.ProvisionGrafanaDashboard {
		return
	}
	host := strings.TrimSpace(c.WebAPI.GrafanaHost)
	if host == "" {
		host = "http://127.0.0.1:3000"
	}
	host = strings.TrimRight(host, "/")

	if !grafanaHealthy(ctx, host) {
		logx.Infof("Grafana not reachable at %s, skip dashboard provisioning", host)
		return
	}
	path, err := locateGrafanaDashboardJSON()
	if err != nil {
		logx.Errorf("could not locate piper-dashboard.json: %v", err)
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		logx.Errorf("read Grafana dashboard %s: %v", path, err)
		return
	}
	existed := grafanaDashboardExists(ctx, host, c)
	if err := importGrafanaDashboard(ctx, host, c, raw); err != nil {
		logx.Errorf("import Grafana dashboard: %v", err)
		return
	}
	if existed {
		logx.Infof("Grafana dashboard %q updated (overwrite) from %s", grafanaDashboardUID, path)
	} else {
		logx.Infof("Grafana dashboard %q provisioned from %s", grafanaDashboardUID, path)
	}
}

func grafanaHealthy(ctx context.Context, host string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, host+"/api/health", nil)
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func grafanaDashboardExists(ctx context.Context, host string, c config.Config) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, host+"/api/dashboards/uid/"+grafanaDashboardUID, nil)
	if err != nil {
		return false
	}
	applyGrafanaAuth(req, c)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return true
	}
	io.Copy(io.Discard, resp.Body)
	return false
}

func importGrafanaDashboard(ctx context.Context, host string, c config.Config, dashboardJSON []byte) error {
	var dashboard map[string]any
	if err := json.Unmarshal(dashboardJSON, &dashboard); err != nil {
		return fmt.Errorf("parse dashboard json: %w", err)
	}
	payload, err := json.Marshal(map[string]any{
		"dashboard": dashboard,
		"overwrite": true,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, host+"/api/dashboards/db", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	applyGrafanaAuth(req, c)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("grafana API %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func applyGrafanaAuth(req *http.Request, c config.Config) {
	user := strings.TrimSpace(c.WebAPI.GrafanaAdminUser)
	pass := c.WebAPI.GrafanaAdminPassword
	if user == "" {
		return
	}
	req.SetBasicAuth(user, pass)
}

func locateGrafanaDashboardJSON() (string, error) {
	if p := strings.TrimSpace(os.Getenv("PIPER_GRAFANA_DASHBOARD_PATH")); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		return "", fmt.Errorf("PIPER_GRAFANA_DASHBOARD_PATH=%q: %w", p, os.ErrNotExist)
	}
	for _, rel := range grafanaDashboardSearchPaths {
		if _, err := os.Stat(rel); err == nil {
			abs, _ := filepath.Abs(rel)
			return abs, nil
		}
	}
	return "", fmt.Errorf("searched %v (set PIPER_GRAFANA_DASHBOARD_PATH)", grafanaDashboardSearchPaths)
}
