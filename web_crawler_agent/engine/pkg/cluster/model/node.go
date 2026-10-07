package model

// NodeInfo JSON shape aligned with one.rewind.nio.cluster.model.NodeInfo (API /misc/info).
type NodeInfo struct {
	ID                  string         `json:"id,omitempty"`
	InstID              string         `json:"inst_id"`
	ServiceURL          string         `json:"service_url,omitempty"`
	PrometheusInstance  string         `json:"prometheus_instance,omitempty"`
	Name                string         `json:"name,omitempty"`
	Local               bool           `json:"local"`
	PublicKey           string         `json:"public_key,omitempty"`
	Ready               bool           `json:"ready"`
	ES                  map[string]any `json:"es,omitempty"`
	S3                  map[string]any `json:"s3,omitempty"`
	Proxies             []any          `json:"proxies,omitempty"`
	Version             string         `json:"version,omitempty"`
}
