package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"piper_go/internal/config"
	"piper_go/pkg/db/meta"

	"github.com/zeromicro/go-zero/core/logx"
)

var piperJobSuffix = regexp.MustCompile(`(?i)"piper-(.+?)"`)
var ipv4 = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)

// StartPrometheusSync periodically discovers nodes via Prometheus (Java WebAPI scheduled task).
func StartPrometheusSync(ctx context.Context, cfg config.Config, store *meta.Store, localInstID string, apiPort int) {
	host := strings.TrimSpace(cfg.WebAPI.PrometheusHost)
	if host == "" {
		return
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		time.Sleep(10 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := syncOnce(ctx, host, store, localInstID, apiPort); err != nil {
					logx.Infof("prometheus node sync: %v", err)
				}
			}
		}
	}()
}

func syncOnce(ctx context.Context, promHost string, store *meta.Store, localInstID string, apiPort int) error {
	url := strings.TrimRight(promHost, "/") + "/api/v1/query?query=up"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		Data struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
				Value  []any             `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return err
	}
	var hosts []string
	for _, r := range parsed.Data.Result {
		if len(r.Value) < 2 || fmt.Sprint(r.Value[1]) != "1" {
			continue
		}
		job := r.Metric["job"]
		m := piperJobSuffix.FindStringSubmatch(`"` + job + `"`)
		if len(m) < 2 {
			continue
		}
		hosts = append(hosts, resolvePiperHost(r.Metric, m[1]))
	}
	seen := map[string]struct{}{}
	for _, h := range hosts {
		if h == "" {
			continue
		}
		seen[h] = struct{}{}
		infoURL := fmt.Sprintf("http://%s:%d/misc/info", h, apiPort)
		node, err := fetchNodeInfo(ctx, infoURL)
		if err != nil {
			logx.Infof("skip node %s: %v", h, err)
			continue
		}
		if inst, _ := node["inst_id"].(string); inst == localInstID {
			node["local"] = true
		}
		node["service_url"] = fmt.Sprintf("http://%s:%d", h, apiPort)
		node["prometheus_instance"] = fmt.Sprintf("%s:%d", h, apiPort)
		node["name"] = h
		node["id"] = node["inst_id"]
		_ = UpsertNode(store, node)
	}
	rows, _, _ := store.Query(meta.TableNodes, meta.QueryOpts{Page: 1, Size: 10000})
	for _, row := range rows {
		name, _ := row["name"].(string)
		if _, ok := seen[name]; !ok && name != "" {
			id, _ := row["id"].(string)
			_ = DeleteNode(store, id)
		}
	}
	return nil
}

func resolvePiperHost(metric map[string]string, suffix string) string {
	if strings.EqualFold(suffix, "localhost") {
		return "127.0.0.1"
	}
	if ipv4.MatchString(suffix) {
		return suffix
	}
	if inst, ok := metric["instance"]; ok {
		host := inst
		if i := strings.LastIndex(inst, ":"); i > 0 {
			host = inst[:i]
		}
		if strings.EqualFold(host, "host.docker.internal") {
			return "127.0.0.1"
		}
		return host
	}
	return suffix
}

func fetchNodeInfo(ctx context.Context, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var wrap struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(b, &wrap); err != nil {
		return nil, err
	}
	if wrap.Data == nil {
		return nil, fmt.Errorf("empty node info")
	}
	return wrap.Data, nil
}
