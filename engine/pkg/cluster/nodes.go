package cluster

import (
	"context"
	"fmt"
	"sort"

	"piper_go/internal/config"
	"piper_go/pkg/cluster/model"
	"piper_go/pkg/db/meta"
)

// ListNodes returns all nodes sorted with local first (Java NodeRoute.list).
func ListNodes(store *meta.Store) ([]map[string]any, error) {
	rows, _, err := store.Query(meta.TableNodes, meta.QueryOpts{Page: 1, Size: 10000})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(rows, func(i, j int) bool {
		li, _ := rows[i]["local"].(bool)
		lj, _ := rows[j]["local"].(bool)
		if li == lj {
			return false
		}
		return li
	})
	return rows, nil
}

func GetNode(store *meta.Store, id string) (map[string]any, error) {
	return store.Get(meta.TableNodes, id)
}

func UpsertNode(store *meta.Store, node map[string]any) error {
	id, _ := node["id"].(string)
	if id == "" {
		if inst, _ := node["inst_id"].(string); inst != "" {
			id = inst
			node["id"] = id
		}
	}
	if id == "" {
		return meta.ErrNotFound
	}
	return store.Upsert(meta.TableNodes, node)
}

func DeleteNode(store *meta.Store, id string) error {
	return store.Delete(meta.TableNodes, id)
}

// FetchNodeInfoPublic GETs /misc/info from a peer (Java NodeRoute.create).
func FetchNodeInfoPublic(ctx context.Context, url string) (map[string]any, error) {
	return fetchNodeInfo(ctx, url)
}

// LocalNodeDoc builds a meta row for the local node.
func LocalNodeDoc(n *model.NodeInfo, c config.Config) map[string]any {
	host := c.Host
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	port := c.Port
	if port == 0 {
		port = 8888
	}
	doc := map[string]any{
		"id":                   n.InstID,
		"inst_id":              n.InstID,
		"local":                true,
		"ready":                n.Ready,
		"public_key":           n.PublicKey,
		"es":                   n.ES,
		"s3":                   n.S3,
		"proxies":              n.Proxies,
		"version":              n.Version,
		"name":                 host,
		"service_url":          fmt.Sprintf("http://%s:%d", host, port),
		"prometheus_instance":  fmt.Sprintf("%s:%d", host, port),
	}
	return doc
}
