package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/zeromicro/go-zero/rest/pathvar"

	"piper_go/internal/svc"
	"piper_go/pkg/chrome"
	"piper_go/pkg/cluster"
	"piper_go/pkg/db/meta"
	"piper_go/pkg/distributor"
	piperjson "piper_go/pkg/json"
	"piper_go/pkg/websocket"
)

func NodeListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := cluster.ListNodes(svcCtx.Meta)
		if err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(list))
	}
}

func NodeGetHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		node, err := cluster.GetNode(svcCtx.Meta, id)
		if err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(node))
	}
}

func NodeCreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := r.URL.Query().Get("host")
		if host == "" {
			writeFailureMsg(w, "host required")
			return
		}
		port := svcCtx.Config.Port
		if port == 0 {
			port = 8888
		}
		url := fmt.Sprintf("http://%s:%d/misc/info", host, port)
		node, err := cluster.FetchNodeInfoPublic(r.Context(), url)
		if node != nil {
			node["local"] = false
			node["name"] = host
			node["service_url"] = fmt.Sprintf("http://%s:%d", host, port)
		}
		if err != nil {
			writeFailure(w, err)
			return
		}
		if err := cluster.UpsertNode(svcCtx.Meta, node); err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func NodeDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		if err := cluster.DeleteNode(svcCtx.Meta, id); err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func AgentQueryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		domain := r.URL.Query().Get("domain")
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(distributor.ListAgents(status, domain)))
	}
}

func broadcastAgentListWS() {
	websocket.BroadcastMsg("", map[string]any{
		"type":    "List<Agent>",
		"ref_obj": distributor.AllAgentsJSON(),
	})
}

func AgentCreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		typ := r.URL.Query().Get("type")
		if typ == "" {
			typ = "ChromeAgent"
		}
		var created map[string]any
		switch typ {
		case "ChromeAgent", "Chrome":
			cd := chrome.Default()
			if cd == nil {
				writeFailure(w, errors.New("ChromeDistributor no Agent"))
				return
			}
			node := svcCtx.Node()
			instID := ""
			if node != nil {
				instID = node.InstID
			}
			a := cd.AddAgent(instID)
			created = a.Snapshot()
			distributor.RegisterAgent(created)
		default:
			n := len(distributor.ListAgents("", "")) + 1
			instID := ""
			if node := svcCtx.Node(); node != nil {
				instID = node.InstID
			}
			name := fmt.Sprintf("HA-%d", n)
			created = distributor.NewHttpAgentDoc(instID, name)
			distributor.RegisterAgent(created)
		}
		broadcastAgentListWS()
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(created))
	}
}

func AgentGetHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		a, err := distributor.GetAgentByID(id)
		if err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(a))
	}
}

func AgentDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		if cd := chrome.Default(); cd != nil {
			cd.RemoveAgent(id)
		}
		distributor.UnregisterAgent(id)
		broadcastAgentListWS()
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func AgentAddAccountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		domain := r.URL.Query().Get("domain")
		username := r.URL.Query().Get("username")
		a, err := distributor.GetAgentByID(id)
		if err != nil {
			writeFailure(w, err)
			return
		}
		acct, err := svcCtx.Meta.Get(meta.TableAccounts, accountID(domain, username))
		if err != nil {
			rows, _, _ := svcCtx.Meta.Query(meta.TableAccounts, meta.QueryOpts{Page: 1, Size: 1000})
			for _, row := range rows {
				if row["domain"] == domain && row["username"] == username {
					acct = row
					break
				}
			}
		}
		if acct == nil {
			writeFailure(w, meta.ErrNotFound)
			return
		}
		if st, _ := acct["status"].(string); st != "" && st != "Free" {
			writeFailure(w, errors.New("account not free"))
			return
		}
		accts, _ := a["accounts"].(map[string]any)
		if accts == nil {
			accts = map[string]any{}
		}
		accts[domain] = username
		a["accounts"] = accts
		acct["status"] = "Occupied"
		acct["agent_id"] = id
		_ = svcCtx.Meta.Upsert(meta.TableAccounts, acct)
		distributor.RegisterAgent(a)
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func AgentRemoveAccountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		domain := r.URL.Query().Get("domain")
		username := r.URL.Query().Get("username")
		a, err := distributor.GetAgentByID(id)
		if err != nil {
			writeFailure(w, err)
			return
		}
		accts, _ := a["accounts"].(map[string]any)
		if accts != nil {
			delete(accts, domain)
		}
		a["accounts"] = accts
		distributor.RegisterAgent(a)
		for _, row := range queryAccounts(svcCtx, domain, username) {
			row["status"] = "Free"
			delete(row, "agent_id")
			_ = svcCtx.Meta.Upsert(meta.TableAccounts, row)
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func AgentSetProxyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		proxyID := r.URL.Query().Get("proxy_id")
		a, err := distributor.GetAgentByID(id)
		if err != nil {
			writeFailure(w, err)
			return
		}
		if _, err := svcCtx.Meta.Get(meta.TableProxies, proxyID); err != nil {
			writeFailure(w, err)
			return
		}
		a["proxy_id"] = proxyID
		distributor.RegisterAgent(a)
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func AgentTriggerHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		a, err := distributor.GetAgentByID(id)
		if err != nil {
			writeFailure(w, err)
			return
		}
		if st, _ := a["status"].(string); st != "Hangup" {
			writeFailure(w, errors.New("agent not hangup"))
			return
		}
		a["status"] = "Idle"
		distributor.RegisterAgent(a)
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func MiscRestartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := distributor.SaveAgentsRegistry(svcCtx.Config.H2.Path); err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func MiscExportPackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids := r.URL.Query()["id"]
		pack, err := cluster.ExportPack(svcCtx.Meta, ids)
		if err != nil {
			writeFailure(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pack)
	}
}

func MiscImportPackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := readBody(r)
		if err != nil {
			writeFailure(w, err)
			return
		}
		if err := cluster.ImportPack(svcCtx.Meta, body); err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func DataMigrationConfigGetHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(cluster.DefaultFileTransporter().Config()))
	}
}

func DataMigrationConfigPutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := readBody(r)
		if err != nil {
			writeFailure(w, err)
			return
		}
		if err := cluster.DefaultFileTransporter().SetConfig(body); err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func DataMigrationLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(cluster.DefaultFileTransporter().Logs()))
	}
}

func accountID(domain, username string) string {
	return metaAssignAccountID(domain, username)
}

func metaAssignAccountID(domain, username string) string {
	doc := map[string]any{"domain": domain, "username": username}
	meta.AssignID(meta.TableAccounts, doc)
	id, _ := doc["id"].(string)
	return id
}

func agentsInfoPath(svcCtx *svc.ServiceContext) string {
	return filepath.Join(filepath.Dir(svcCtx.Config.H2.Path), "agents_info.json")
}

func queryAccounts(svcCtx *svc.ServiceContext, domain, username string) []map[string]any {
	rows, _, _ := svcCtx.Meta.Query(meta.TableAccounts, meta.QueryOpts{Page: 1, Size: 1000})
	var out []map[string]any
	for _, row := range rows {
		if row["domain"] == domain && row["username"] == username {
			out = append(out, row)
		}
	}
	return out
}
