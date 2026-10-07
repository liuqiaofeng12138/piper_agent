package distributor

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	metricNodeTokenCount              = "node_token_count"
	metricDistributorTokenQueuingSize = "distributor_token_queuing_size"
	metricAgentTokenQueuingSize       = "agent_token_queuing_size"
	metricAgentTokenSuccessCount      = "agent_token_success_count"
	metricAgentTokenFailureCount      = "agent_token_failure_count"
	metricTaskTokenSuccessCount       = "task_token_success_count"
	metricTaskTokenFailureCount       = "task_token_failure_count"
	metricTemplateTokenSuccessCount   = "template_token_success_count"
	metricTemplateTokenFailureCount   = "template_token_failure_count"
	metricIndexSuccessCount           = "index_success_count"
	metricIndexFailureCount           = "index_failure_count"
	metricProxySuccessCount           = "proxy_success_count"
	metricProxyBannedCount            = "proxy_banned_count"
	metricProxyTimeoutCount           = "proxy_timeout_count"
	metricProxyChangeIPSuccessCount   = "proxy_changeIp_success_count"
	metricProxyChangeIPFailureCount   = "proxy_changeIp_failure_count"
	metricProxyInboundBytes           = "proxy_inbound_bytes"
	metricProxyOutboundBytes          = "proxy_outbound_bytes"
	metricAccountSuccessCount         = "account_success_count"
	metricAccountFrozenCount          = "account_frozen_count"
)

type statRecord struct {
	id   string
	name string
	v    atomic.Int64
}

type stats struct {
	queueSize atomic.Int64
	piperUp   atomic.Int64
	mu        sync.RWMutex
	records   map[string]map[string]*statRecord
}

var Stats stats

func (s *stats) SetQueue(n int) {
	s.queueSize.Store(int64(n))
}

func (s *stats) QueueSize() int64 {
	return s.queueSize.Load()
}

func (s *stats) SetUp(v int64) {
	s.piperUp.Store(v)
}

func (s *stats) Count(typ, id, name string, delta int64) {
	if strings.TrimSpace(typ) == "" || delta == 0 {
		return
	}
	key := id
	if key == "" {
		key = name
	}
	if key == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.records == nil {
		s.records = map[string]map[string]*statRecord{}
	}
	m := s.records[typ]
	if m == nil {
		m = map[string]*statRecord{}
		s.records[typ] = m
	}
	rec, ok := m[key]
	if !ok {
		rec = &statRecord{id: id, name: name}
		m[key] = rec
	}
	rec.v.Add(delta)
}

func (s *stats) set(typ, id, name string, v int64) {
	if strings.TrimSpace(typ) == "" {
		return
	}
	key := id
	if key == "" {
		key = name
	}
	if key == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.records == nil {
		s.records = map[string]map[string]*statRecord{}
	}
	m := s.records[typ]
	if m == nil {
		m = map[string]*statRecord{}
		s.records[typ] = m
	}
	rec, ok := m[key]
	if !ok {
		rec = &statRecord{id: id, name: name}
		m[key] = rec
	}
	rec.v.Store(v)
}

// EnsureAgentMetrics registers counters/gauges when an agent is created (does not reset existing counts).
func (s *stats) EnsureAgentMetrics(id, name string) {
	if id == "" && name == "" {
		return
	}
	s.ensureCounter(metricAgentTokenSuccessCount, id, name)
	s.ensureCounter(metricAgentTokenFailureCount, id, name)
	s.set(metricAgentTokenQueuingSize, id, name, 0)
}

func (s *stats) SyncAgentGauges() {
	s.set(metricDistributorTokenQueuingSize, "", "distributor", s.QueueSize())
	for _, a := range ListAgents("", "") {
		id, _ := a["id"].(string)
		name, _ := a["name"].(string)
		pub := agentInt64(a["public_queue_size"])
		loc := agentInt64(a["local_queue_size"])
		q := pub
		if loc > q {
			q = loc
		}
		s.set(metricAgentTokenQueuingSize, id, name, q)
	}
}

func (s *stats) PrometheusText() string {
	// SyncAgentGauges is also invoked from SyncMetricSeeds when store is wired in metrics handler.
	s.SyncAgentGauges()

	up := s.piperUp.Load()
	if up == 0 {
		up = 1
	}
	q := s.queueSize.Load()

	var b strings.Builder
	b.WriteString("# HELP piper_up Piper node ready state.\n# TYPE piper_up gauge\npiper_up ")
	b.WriteString(itoa(up))
	b.WriteString("\n# HELP piper_token_queue Token queue size.\n# TYPE piper_token_queue gauge\npiper_token_queue ")
	b.WriteString(itoa(q))
	b.WriteString("\n")

	s.mu.RLock()
	types := make([]string, 0, len(s.records))
	for typ := range s.records {
		types = append(types, typ)
	}
	sort.Strings(types)
	for _, typ := range types {
		b.WriteString("# HELP ")
		b.WriteString(typ)
		b.WriteString(" Piper metric.\n# TYPE ")
		b.WriteString(typ)
		if strings.Contains(typ, "_size") || strings.HasSuffix(typ, "_bytes") {
			b.WriteString(" gauge\n")
		} else {
			b.WriteString(" counter\n")
		}
		keys := make([]string, 0, len(s.records[typ]))
		for k := range s.records[typ] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rec := s.records[typ][k]
			b.WriteString(typ)
			b.WriteString(formatLabels(rec.id, rec.name))
			b.WriteString(" ")
			b.WriteString(itoa(rec.v.Load()))
			b.WriteString("\n")
		}
	}
	s.mu.RUnlock()
	return b.String()
}

func formatLabels(id, name string) string {
	var parts []string
	if id != "" {
		parts = append(parts, `id="`+escapeLabel(id)+`"`)
	}
	if name != "" {
		parts = append(parts, `name="`+escapeLabel(name)+`"`)
	}
	if len(parts) == 0 {
		return " "
	}
	return "{" + strings.Join(parts, ", ") + "} "
}

func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// RecordTokenStats updates Prometheus counters for a finished token (Java Stats.count).
func RecordTokenStats(token map[string]any, success bool) {
	agentID, _ := token["agent_id"].(string)
	agentName := agentNameForMetrics(agentID)

	if success {
		Stats.Count(metricAgentTokenSuccessCount, agentID, agentName, 1)
	} else {
		Stats.Count(metricAgentTokenFailureCount, agentID, agentName, 1)
	}

	if tplID, _ := token["tpl_id"].(string); tplID != "" {
		if success {
			Stats.Count(metricTemplateTokenSuccessCount, tplID, tplID, 1)
		} else {
			Stats.Count(metricTemplateTokenFailureCount, tplID, tplID, 1)
		}
	}
	if taskID, _ := token["task_id"].(string); taskID != "" {
		if success {
			Stats.Count(metricTaskTokenSuccessCount, taskID, taskID, 1)
		} else {
			Stats.Count(metricTaskTokenFailureCount, taskID, taskID, 1)
		}
	}
	if idx, _ := token["index"].(string); idx != "" {
		if success {
			Stats.Count(metricIndexSuccessCount, idx, idx, 1)
		} else {
			Stats.Count(metricIndexFailureCount, idx, idx, 1)
		}
	}
}

func agentNameForMetrics(agentID string) string {
	if agentID == "" {
		return ""
	}
	if a, err := GetAgentByID(agentID); err == nil {
		if n, _ := a["name"].(string); n != "" {
			return n
		}
	}
	return agentID
}
