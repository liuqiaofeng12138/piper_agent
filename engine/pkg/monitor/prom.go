package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type promQueryResp struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// QueryInstant runs a Prometheus instant query and returns the first scalar value.
func QueryInstant(ctx context.Context, promBase, promQL string) (float64, bool, error) {
	promBase = strings.TrimRight(strings.TrimSpace(promBase), "/")
	if promBase == "" {
		return 0, false, nil
	}
	u := promBase + "/api/v1/query?query=" + url.QueryEscape(promQL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed promQueryResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, false, err
	}
	if parsed.Status != "success" || len(parsed.Data.Result) == 0 {
		return 0, false, nil
	}
	v := parsed.Data.Result[0].Value
	if len(v) < 2 {
		return 0, false, nil
	}
	f, err := strconv.ParseFloat(fmt.Sprint(v[1]), 64)
	if err != nil {
		return 0, false, nil
	}
	return f, true, nil
}

// ResolveNodeInstance picks a node_exporter instance label present in Prometheus.
func ResolveNodeInstance(ctx context.Context, promBase string, candidates []string) string {
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		q := `count(node_cpu_seconds_total{instance="` + escapePromLabel(c) + `"})`
		if _, ok, _ := QueryInstant(ctx, promBase, q); ok {
			return c
		}
	}
	// Any node_cpu series
	if f, ok, _ := QueryInstant(ctx, promBase, `count(node_cpu_seconds_total) by (instance)`); ok && f > 0 {
		// need label not count - use label_values via query
		u := strings.TrimRight(promBase, "/") + `/api/v1/query?query=` + url.QueryEscape(`group by (instance) (node_cpu_seconds_total)`)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var parsed promQueryResp
			if json.Unmarshal(body, &parsed) == nil && len(parsed.Data.Result) > 0 {
				if inst := parsed.Data.Result[0].Metric["instance"]; inst != "" {
					return inst
				}
			}
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return "127.0.0.1:9100"
}

func escapePromLabel(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// OverviewStats loads dashboard header metrics (same PromQL as web global-item.vue).
func OverviewStats(ctx context.Context, promBase, nodeInst, piperInst string) map[string]any {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	out := map[string]any{
		"node_instance":   nodeInst,
		"piper_instance":  piperInst,
		"prometheus_ok":   false,
	}

	type q struct {
		key string
		ql  string
	}
	queries := []q{
		{"cores", `count(node_schedstat_running_seconds_total{instance="` + escapePromLabel(nodeInst) + `"}) by (instance)`},
		{"cpu", `avg(rate(node_cpu_seconds_total{instance="` + escapePromLabel(nodeInst) + `"}[2m])) by (instance) * 100`},
		{"mem_available", `node_memory_MemAvailable_bytes{instance="` + escapePromLabel(nodeInst) + `"}`},
		{"mem_total", `node_memory_MemTotal_bytes{instance="` + escapePromLabel(nodeInst) + `"}`},
		{"net_down", `max(rate(node_network_receive_bytes_total{instance="` + escapePromLabel(nodeInst) + `"}[5m])*8) by (instance)`},
		{"net_up", `max(rate(node_network_transmit_bytes_total{instance="` + escapePromLabel(nodeInst) + `"}[5m])*8) by (instance)`},
		{"token_success_5m", `sum(increase(agent_token_success_count{instance="` + escapePromLabel(piperInst) + `"}[5m]))`},
		{"token_failure_5m", `sum(increase(agent_token_failure_count{instance="` + escapePromLabel(piperInst) + `"}[5m]))`},
		{"token_success_1h", `sum(increase(agent_token_success_count{instance="` + escapePromLabel(piperInst) + `"}[1h]))`},
		{"token_failure_1h", `sum(increase(agent_token_failure_count{instance="` + escapePromLabel(piperInst) + `"}[1h]))`},
		{"token_success_24h", `sum(increase(agent_token_success_count{instance="` + escapePromLabel(piperInst) + `"}[24h]))`},
		{"token_failure_24h", `sum(increase(agent_token_failure_count{instance="` + escapePromLabel(piperInst) + `"}[24h]))`},
	}
	gotAny := false
	for _, item := range queries {
		v, ok, err := QueryInstant(ctx, promBase, item.ql)
		if err != nil {
			out["prometheus_error"] = err.Error()
			continue
		}
		if ok {
			gotAny = true
			out[item.key] = v
		}
	}
	out["prometheus_ok"] = gotAny
	return out
}
