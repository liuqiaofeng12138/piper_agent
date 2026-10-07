package distributor

import (
	"piper_go/pkg/db/meta"
)

// SyncMetricSeeds ensures Prometheus counters exist (value 0) for Grafana overview panels.
// Without series, Grafana solo panels show "No data in response" even when the pipeline is healthy.
func SyncMetricSeeds(store *meta.Store) {
	Stats.SyncAgentGauges()

	for _, a := range ListAgents("", "") {
		id, _ := a["id"].(string)
		name, _ := a["name"].(string)
		Stats.ensureCounter(metricAgentTokenSuccessCount, id, name)
		Stats.ensureCounter(metricAgentTokenFailureCount, id, name)
	}

	if store == nil {
		return
	}

	seedMetaCounters(store, meta.TableTemplates, metricTemplateTokenSuccessCount, metricTemplateTokenFailureCount)
	seedMetaCounters(store, meta.TableTasks, metricTaskTokenSuccessCount, metricTaskTokenFailureCount)
	seedMetaCounters(store, meta.TableIndices, metricIndexSuccessCount, metricIndexFailureCount)
	seedProxies(store)
}

func seedMetaCounters(store *meta.Store, table, successMetric, failureMetric string) {
	rows, _, err := store.Query(table, meta.QueryOpts{Page: 1, Size: 100000})
	if err != nil {
		return
	}
	for _, row := range rows {
		id, _ := row["id"].(string)
		name, _ := row["name"].(string)
		if name == "" {
			name = id
		}
		if id == "" && name == "" {
			continue
		}
		Stats.ensureCounter(successMetric, id, name)
		Stats.ensureCounter(failureMetric, id, name)
	}
}

func seedProxies(store *meta.Store) {
	rows, _, err := store.Query(meta.TableProxies, meta.QueryOpts{Page: 1, Size: 100000})
	if err != nil {
		return
	}
	for _, row := range rows {
		id, _ := row["id"].(string)
		name, _ := row["name"].(string)
		if name == "" {
			name = id
		}
		if id == "" {
			continue
		}
		Stats.ensureCounter(metricProxyInboundBytes, id, name)
		Stats.ensureCounter(metricProxyOutboundBytes, id, name)
		Stats.ensureCounter(metricProxySuccessCount, id, name)
		Stats.ensureCounter(metricProxyBannedCount, id, name)
		Stats.ensureCounter(metricProxyTimeoutCount, id, name)
	}
}

func (s *stats) ensureCounter(typ, id, name string) {
	if typ == "" {
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
	if _, ok := m[key]; ok {
		return
	}
	m[key] = &statRecord{id: id, name: name}
}
