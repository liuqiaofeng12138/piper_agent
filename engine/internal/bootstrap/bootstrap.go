package bootstrap

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"piper_go/internal/config"
	"piper_go/internal/svc"
	pkgbootstrap "piper_go/pkg/bootstrap"
	"piper_go/pkg/chrome"
	"piper_go/pkg/cluster"
	"piper_go/pkg/cluster/model"
	"piper_go/pkg/db/es"
	"piper_go/pkg/db/meta"
	"piper_go/pkg/distributor"
	"piper_go/pkg/distributor/cache"
	pkglog "piper_go/pkg/log"
	"piper_go/pkg/notification"
	"piper_go/pkg/persistence"
	"piper_go/pkg/runtimeconfig"
	"piper_go/pkg/storage"

	"github.com/zeromicro/go-zero/core/logx"
)

func Run(ctx context.Context, s *svc.ServiceContext) error {
	if err := pkgbootstrap.EnsureDependencies(ctx, s.Config); err != nil {
		return err
	}

	store, err := meta.Open(s.Config.H2.Path)
	if err != nil {
		return err
	}
	s.Meta = store
	esClient := es.New(s.Config.ES)
	if err := esClient.EnsureTokenLogsMapping(ctx); err != nil {
		return fmt.Errorf("configure token logs mapping: %w", err)
	}
	s.ES = esClient
	s.Dist = distributor.Init(s.Config, esClient)
	distributor.SetAgentsH2Path(s.Config.H2.Path)
	s.Dist.SetMeta(store)
	s3Client, _ := storage.NewS3(s.Config.S3)
	s.Persist = persistence.NewPersister(esClient, s3Client)
	s.Dist.SetPersister(s.Persist)
	runtimeconfig.InitFromConfig(s.Config)
	loadCaches(s)
	distributor.Stats.SetUp(1)

	node := buildLocalNode(s.Config)
	s.Notify = notification.NewService(esClient, node.InstID, s.Config.WebAPI.Notification)
	s.LogES = pkglog.NewAppender(esClient, node.InstID)
	if s.Notify.Enabled() {
		s.LogES.SetDocHook(func(doc map[string]any) {
			s.Notify.FromLog(context.Background(), doc)
		})
	}
	chromeDist := chrome.Init(s.Config, esClient)
	chromeDist.SetHooks(s.Dist.GetTemplate, s.Dist.BuildRunContext, s.Dist.RouteToken, s.Dist.TaskTokenFinished)
	chromeDist.SetPersister(s.Persist)
	if s.Config.Chrome.Enabled {
		chromeDist.StartAgents(node.InstID, s.Config.Chrome.AgentCount)
		logx.Infof("chrome agents started: %d headless=%v manual_login_wait_seconds=%d user_data_dir=%q",
			chromeDist.AgentCount(), s.Config.Chrome.Headless, s.Config.Chrome.ManualLoginWaitSeconds, s.Config.Chrome.UserDataDir)
	}
	node.Ready = true
	s.SetNode(node)
	if err := cluster.UpsertNode(s.Meta, cluster.LocalNodeDoc(node, s.Config)); err != nil {
		logx.Errorf("upsert local node: %v", err)
	}
	distributor.RestoreAgentsRegistry(s.Config.H2.Path)
	registerChromeAgents(chromeDist)
	cluster.StartPrometheusSync(ctx, s.Config, s.Meta, node.InstID, s.Config.Port)
	cluster.StartNodeProxyRefresh(ctx, s.Node, s.SetNode)
	s.SetReady(true)

	logx.Info("bootstrap complete, server ready")
	return nil
}

func registerChromeAgents(cd *chrome.Distributor) {
	if cd == nil {
		return
	}
	cd.EachAgent(func(a *chrome.Agent) {
		distributor.RegisterAgent(a.Snapshot())
	})
}

func loadCaches(s *svc.ServiceContext) {
	load := func(table string) []map[string]any {
		items, _, err := s.Meta.Query(table, meta.QueryOpts{Page: 1, Size: 100000})
		if err != nil {
			logx.Errorf("cache load %s: %v", table, err)
			return nil
		}
		return items
	}
	cache.LoadAll(
		load(meta.TableIndices),
		load(meta.TableTemplates),
		load(meta.TableVarsLists),
		load(meta.TableTasks),
	)
}

func buildLocalNode(c config.Config) *model.NodeInfo {
	host, _ := os.Hostname()
	if host == "" {
		host = "localhost"
	}
	apiHost := strings.TrimSpace(c.Host)
	if apiHost == "" || apiHost == "0.0.0.0" {
		apiHost = "127.0.0.1"
	}
	instID := md5Hex("piper-node:" + host)
	serviceURL := fmt.Sprintf("http://%s:%d", apiHost, c.Port)
	promInst := strings.TrimSpace(c.WebAPI.PrometheusScrapeInstance)
	if promInst == "" {
		promInst = fmt.Sprintf("%s:%d", apiHost, c.Port)
	}
	return &model.NodeInfo{
		InstID:             instID,
		Name:               apiHost,
		ServiceURL:         serviceURL,
		PrometheusInstance: promInst,
		Local:              true,
		Ready:              false,
		PublicKey:          "",
		ES: map[string]any{
			"host": c.ES.Host,
			"port": fmt.Sprintf("%d", c.ES.Port),
		},
		S3: map[string]any{
			"endpointUrl": c.S3.EndpointURL,
		},
		Proxies: []any{},
		Version: "piper_go-phase1",
	}
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
