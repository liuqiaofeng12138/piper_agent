package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"piper_go/internal/config"
)

type Client struct {
	base     string
	hc       *http.Client
	username string
	password string
}

func New(c config.ESConf) *Client {
	return &Client{
		base:     fmt.Sprintf("http://%s:%d", c.Host, c.Port),
		hc:       &http.Client{Timeout: 30 * time.Second},
		username: strings.TrimSpace(c.Username),
		password: c.Password,
	}
}

func (c *Client) authorize(req *http.Request) {
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	c.authorize(req)
	return c.hc.Do(req)
}

func (c *Client) Get(ctx context.Context, index, id string) (map[string]any, error) {
	url := fmt.Sprintf("%s/%s/_doc/%s", c.base, index, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}
	body, _ := io.ReadAll(resp.Body)
	var wrap struct {
		Source map[string]any `json:"_source"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, err
	}
	return wrap.Source, nil
}

func (c *Client) Index(ctx context.Context, index, id string, doc map[string]any) error {
	url := fmt.Sprintf("%s/%s/_doc/%s", c.base, index, id)
	b, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("es index %s: %s", resp.Status, string(out))
	}
	return nil
}

type SearchResult struct {
	Total        int64
	Hits         []map[string]any
	Aggregations AggResult
}

func (c *Client) Search(ctx context.Context, index string, query map[string]any) (*SearchResult, error) {
	return c.searchURL(ctx, fmt.Sprintf("%s/%s/_search", c.base, index), query)
}

func (c *Client) SearchIndices(ctx context.Context, indices []string, query map[string]any) (*SearchResult, error) {
	if len(indices) == 0 {
		return &SearchResult{}, nil
	}
	path := strings.Join(indices, ",")
	return c.searchURL(ctx, fmt.Sprintf("%s/%s/_search", c.base, path), query)
}

func (c *Client) searchURL(ctx context.Context, url string, query map[string]any) (*SearchResult, error) {
	b, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]any `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
		Aggregations map[string]struct {
			Buckets []struct {
				KeyAsString string `json:"key_as_string"`
				Key         any    `json:"key"`
				DocCount    int64  `json:"doc_count"`
			} `json:"buckets"`
		} `json:"aggregations"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("es search decode: %w, body=%s", err, truncate(string(body), 200))
	}
	out := &SearchResult{Total: parsed.Hits.Total.Value}
	for _, h := range parsed.Hits.Hits {
		out.Hits = append(out.Hits, h.Source)
	}
	if agg, ok := parsed.Aggregations["date_histogram"]; ok {
		out.Aggregations = AggResult{}
		for _, b := range agg.Buckets {
			key := b.KeyAsString
			if key == "" {
				key = fmt.Sprint(b.Key)
			}
			out.Aggregations[key] = b.DocCount
		}
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Ping checks cluster reachability.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base, nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("es ping %s", resp.Status)
	}
	return nil
}

const tokenIndexName = "token"

// tokenTotalFieldsLimit avoids ES default 1000-field cap on long-lived token indices
// (historically logs.* procedure keys were mapped dynamically).
const tokenTotalFieldsLimit = 5000

var tokenMappingProperties = []byte(`{"logs":{"type":"object","dynamic":false},"vars":{"type":"object","dynamic":false},"r":{"type":"object","dynamic":false}}`)

// EnsureTokenLogsMapping prevents token.logs and token.vars from creating
// unbounded Elasticsearch mappings. logs is keyed by per-token/procedure IDs,
// while vars contains arbitrary fields emitted by user-defined mappers.
// Both objects remain fully available in _source.
func (c *Client) EnsureTokenLogsMapping(ctx context.Context) error {
	if err := c.ensureTokenIndexFieldLimit(ctx); err != nil {
		return err
	}

	createURL := fmt.Sprintf("%s/%s", c.base, tokenIndexName)
	createBody := bytes.NewBufferString(fmt.Sprintf(
		`{"settings":{"index.mapping.total_fields.limit":%d},"mappings":{"properties":%s}}`,
		tokenTotalFieldsLimit,
		tokenMappingProperties,
	))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, createURL, createBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode < 300 {
		return nil
	}
	if !strings.Contains(string(body), "resource_already_exists_exception") {
		return fmt.Errorf("es create token index %s: %s", resp.Status, string(body))
	}

	// The index already exists in deployments upgraded from an older Piper.
	// Updating dynamic on the existing object field is supported by ES and
	// leaves previously indexed log details intact.
	updateURL := fmt.Sprintf("%s/%s/_mapping", c.base, tokenIndexName)
	updateBody := bytes.NewBuffer([]byte(`{"properties":`))
	updateBody.Write(tokenMappingProperties)
	updateBody.WriteByte('}')
	req, err = http.NewRequestWithContext(ctx, http.MethodPut, updateURL, updateBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("es update token dynamic mappings %s: %s", resp.Status, string(body))
	}
	return nil
}

func (c *Client) ensureTokenIndexFieldLimit(ctx context.Context) error {
	settingsURL := fmt.Sprintf("%s/%s/_settings", c.base, tokenIndexName)
	settingsBody := bytes.NewBufferString(fmt.Sprintf(
		`{"index.mapping.total_fields.limit":%d}`,
		tokenTotalFieldsLimit,
	))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, settingsURL, settingsBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode < 300 {
		return nil
	}
	// Index may not exist yet; create path will apply the same limit.
	if resp.StatusCode == http.StatusNotFound || strings.Contains(string(body), "index_not_found_exception") {
		return nil
	}
	return fmt.Errorf("es update token index settings %s: %s", resp.Status, string(body))
}

func TermQuery(field string, value any) map[string]any {
	return map[string]any{"term": map[string]any{field: value}}
}

func RangeQuery(field string, from, to int64) map[string]any {
	return map[string]any{"range": map[string]any{field: map[string]any{"gte": from, "lte": to}}}
}

func BoolMust(clauses ...map[string]any) map[string]any {
	return map[string]any{"query": map[string]any{"bool": map[string]any{"must": clauses}}}
}

func SearchBody(query map[string]any, from, size int, sortField string, desc bool) map[string]any {
	order := "asc"
	if desc {
		order = "desc"
	}
	body := map[string]any{
		"from": from,
		"size": size,
		"sort": []any{map[string]any{sortField: map[string]any{"order": order}}},
	}
	for k, v := range query {
		body[k] = v
	}
	return body
}

func QueryString(q string) map[string]any {
	return map[string]any{"query_string": map[string]any{"query": q}}
}

func PrefixQuery(field, value string) map[string]any {
	return map[string]any{"prefix": map[string]any{field: value}}
}

func MatchQuery(field, value string) map[string]any {
	return map[string]any{"match": map[string]any{field: value}}
}

type AggResult map[string]int64

func (c *Client) SearchDateHistogram(ctx context.Context, indices []string, query map[string]any, field string, intervalMinutes int, st, et int64) (AggResult, error) {
	if len(indices) == 0 {
		return AggResult{}, nil
	}
	body := map[string]any{}
	for k, v := range query {
		body[k] = v
	}
	body["size"] = 0
	body["aggs"] = map[string]any{
		"date_histogram": map[string]any{
			"date_histogram": map[string]any{
				"field":           field,
				"fixed_interval":  fmt.Sprintf("%dm", intervalMinutes),
				"min_doc_count":   0,
				"extended_bounds": map[string]any{"min": st, "max": et},
			},
		},
	}
	res, err := c.SearchIndices(ctx, indices, body)
	if err != nil {
		return nil, err
	}
	return res.Aggregations, nil
}

func MultiMatchPhrase(kw string, fields ...string) map[string]any {
	return map[string]any{
		"multi_match": map[string]any{
			"query":  kw,
			"fields": strings.Join(fields, ","),
			"type":   "phrase",
		},
	}
}

// UpdateFields partial-updates an ES document (Java ESAppender.updateIndexField).
func (c *Client) UpdateFields(ctx context.Context, index, id string, fields map[string]any) error {
	url := fmt.Sprintf("%s/%s/_update/%s", c.base, index, id)
	body := map[string]any{"doc": fields}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("es update %s: %s", resp.Status, string(out))
	}
	return nil
}

func (c *Client) DeleteDoc(ctx context.Context, index, id string) error {
	url := fmt.Sprintf("%s/%s/_doc/%s", c.base, index, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found")
	}
	if resp.StatusCode >= 300 {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("es delete %s: %s", resp.Status, string(out))
	}
	return nil
}
