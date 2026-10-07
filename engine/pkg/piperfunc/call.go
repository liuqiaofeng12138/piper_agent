package piperfunc

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"piper_go/pkg/db/meta"
	"piper_go/pkg/tpl"
)

// Call executes a function record from meta `functions` table (Java Func.call).
func Call(store *meta.Store, funcID string, vars map[string]any) (any, error) {
	doc, err := store.Get(meta.TableFunctions, funcID)
	if err != nil {
		// try by name
		items, _, qerr := store.Query(meta.TableFunctions, meta.QueryOpts{Page: 1, Size: 1000})
		if qerr != nil {
			return nil, err
		}
		for _, item := range items {
			if name, _ := item["name"].(string); name == funcID {
				doc = item
				break
			}
		}
		if doc == nil {
			return nil, fmt.Errorf("func %s not found", funcID)
		}
	}
	urlTpl, _ := doc["url"].(string)
	method, _ := doc["http_method"].(string)
	if method == "" {
		method = http.MethodGet
	}
	auth, _ := doc["authorization"].(string)
	finalURL := tpl.ReplaceVars(urlTpl, vars, true)
	req, err := http.NewRequest(method, finalURL, nil)
	if err != nil {
		return nil, err
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text") || strings.Contains(ct, "json") || strings.Contains(ct, "xml") {
		return string(b), nil
	}
	return b, nil
}
