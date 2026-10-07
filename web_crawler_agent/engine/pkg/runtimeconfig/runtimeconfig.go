package runtimeconfig

import (
	"encoding/json"
	"sync"

	"piper_go/internal/config"
)

var (
	mu   sync.RWMutex
	web  map[string]any
)

func InitFromConfig(c config.Config) {
	mu.Lock()
	defer mu.Unlock()
	web = map[string]any{
		"noAuth":                 c.WebAPI.NoAuth,
		"notification":           c.WebAPI.Notification,
		"onlyVideo":              c.WebAPI.OnlyVideo,
		"prometheusHost":         c.WebAPI.PrometheusHost,
		"esHost":                 c.ES.Host,
		"esPort":                 c.ES.Port,
		"s3EndpointUrl":          c.S3.EndpointURL,
		"connectTimeout":         c.Requester.ConnectTimeout,
		"readTimeout":            c.Requester.ReadTimeout,
		"tokenTimeout":           c.Requester.TokenTimeout,
		"requestPerSecondLimit":  c.Requester.RequestPerSecondLimit,
	}
}

func WebAPIRoot() map[string]any {
	mu.RLock()
	defer mu.RUnlock()
	out := make(map[string]any, len(web))
	for k, v := range web {
		out[k] = v
	}
	return out
}

func MergeWebAPI(body []byte) error {
	var patch map[string]any
	if err := json.Unmarshal(body, &patch); err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	for k, v := range patch {
		web[k] = v
	}
	return nil
}
