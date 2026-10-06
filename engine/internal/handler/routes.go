package handler

import (
	"net/http"

	"piper_go/internal/handler/metrics"
	"piper_go/internal/handler/misc"
	"piper_go/internal/svc"
	"piper_go/pkg/db/meta"
	"piper_go/pkg/distributor/cache"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	routes := []rest.Route{
		{Method: http.MethodPost, Path: "/auth/login", Handler: AuthLoginHandler(serverCtx)},
		{Method: http.MethodPost, Path: "/auth/logout", Handler: AuthLogoutHandler(serverCtx)},
		{Method: http.MethodGet, Path: "/auth/me", Handler: AuthMeHandler(serverCtx)},

		{Method: http.MethodGet, Path: "/misc/info", Handler: misc.InfoHandler(serverCtx)},
		{Method: http.MethodGet, Path: "/misc/overview_metrics", Handler: misc.OverviewMetricsHandler(serverCtx)},
		{Method: http.MethodGet, Path: "/misc/config", Handler: MiscConfigGetHandler(serverCtx)},
		{Method: http.MethodPut, Path: "/misc/config", Handler: MiscConfigPutHandler(serverCtx)},
		{Method: http.MethodGet, Path: "/misc/exceptions", Handler: MiscExceptionsHandler(serverCtx)},
		{Method: http.MethodGet, Path: "/misc/global_vars", Handler: MiscGlobalVarsHandler(serverCtx)},
		{Method: http.MethodGet, Path: "/metrics", Handler: metrics.Handler(serverCtx)},
	}

	routes = append(routes, indexRoutes(serverCtx)...)
	routes = append(routes, simpleCRUD(serverCtx, "/funcs", meta.TableFunctions, crudSpec{})...)
	routes = append(routes, varsListRoutes(serverCtx)...)
	routes = append(routes, accountRoutes(serverCtx)...)
	routes = append(routes, proxyRoutes(serverCtx)...)
	routes = append(routes, templateRoutes(serverCtx)...)
	routes = append(routes, taskRoutes(serverCtx)...)

	routes = append(routes, phase2Routes(serverCtx)...)
	routes = append(routes, dataRoutes(serverCtx)...)
	routes = append(routes, phase5Routes(serverCtx)...)
	routes = append(routes, phase6Routes(serverCtx)...)
	server.AddRoutes(routes)
}

func phase6Routes(svcCtx *svc.ServiceContext) []rest.Route {
	return []rest.Route{
		{Method: http.MethodGet, Path: "/notifications", Handler: NotificationQueryHandler(svcCtx)},
		{Method: http.MethodPut, Path: "/notifications/:id/read", Handler: NotificationReadHandler(svcCtx)},
		{Method: http.MethodDelete, Path: "/notifications/:id", Handler: NotificationDeleteHandler(svcCtx)},
	}
}

func phase5Routes(svcCtx *svc.ServiceContext) []rest.Route {
	return []rest.Route{
		{Method: http.MethodGet, Path: "/nodes", Handler: NodeListHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/nodes/:id", Handler: NodeGetHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/nodes", Handler: NodeCreateHandler(svcCtx)},
		{Method: http.MethodDelete, Path: "/nodes/:id", Handler: NodeDeleteHandler(svcCtx)},

		{Method: http.MethodGet, Path: "/agents", Handler: AgentQueryHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/agents", Handler: AgentCreateHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/agents/:id", Handler: AgentGetHandler(svcCtx)},
		{Method: http.MethodDelete, Path: "/agents/:id", Handler: AgentDeleteHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/agents/:id/accounts", Handler: AgentAddAccountHandler(svcCtx)},
		{Method: http.MethodDelete, Path: "/agents/:id/accounts", Handler: AgentRemoveAccountHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/agents/:id/proxy", Handler: AgentSetProxyHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/agents/:id/trigger", Handler: AgentTriggerHandler(svcCtx)},

		{Method: http.MethodPost, Path: "/misc/restart", Handler: MiscRestartHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/misc/export", Handler: MiscExportPackHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/misc/import", Handler: MiscImportPackHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/misc/data_migration/config", Handler: DataMigrationConfigGetHandler(svcCtx)},
		{Method: http.MethodPut, Path: "/misc/data_migration/config", Handler: DataMigrationConfigPutHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/misc/data_migration/logs", Handler: DataMigrationLogsHandler(svcCtx)},
	}
}

func dataRoutes(svcCtx *svc.ServiceContext) []rest.Route {
	return []rest.Route{
		{Method: http.MethodPost, Path: "/data/search", Handler: DataSearchHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/data/search_date_histogram", Handler: DataSearchDateHistogramHandler(svcCtx)},
	}
}

func phase2Routes(svcCtx *svc.ServiceContext) []rest.Route {
	return []rest.Route{
		{Method: http.MethodPost, Path: "/templates/:id/run", Handler: TemplateRunHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/tasks/:id/run", Handler: TaskRunHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/tasks/:id/stop", Handler: TaskStopHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/misc/run_token", Handler: MiscRunTokenHandler(svcCtx)},

		{Method: http.MethodGet, Path: "/tokens", Handler: TokenQueryHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/tokens/:id/descendants", Handler: TokenDescendantsHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/tokens/:id/data", Handler: TokenDataHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/tokens/:id/next", Handler: TokenNextHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/tokens/:id", Handler: TokenGetHandler(svcCtx)},

		{Method: http.MethodGet, Path: "/logs", Handler: LogQueryHandler(svcCtx)},

		{Method: http.MethodGet, Path: "/msg", Handler: WSMsgHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/token_msg", Handler: WSTokenHandler(svcCtx)},
	}
}

func simpleCRUD(svcCtx *svc.ServiceContext, prefix, table string, spec crudSpec) []rest.Route {
	spec.table = table
	return []rest.Route{
		{Method: http.MethodGet, Path: prefix, Handler: metaQuery(svcCtx, table)},
		{Method: http.MethodGet, Path: prefix + "/:id", Handler: metaGet(svcCtx, table, nil)},
		{Method: http.MethodPost, Path: prefix, Handler: metaCreate(svcCtx, spec)},
		{Method: http.MethodPut, Path: prefix + "/:id", Handler: metaUpdate(svcCtx, spec)},
		{Method: http.MethodDelete, Path: prefix + "/:id", Handler: metaDelete(svcCtx, table, nil)},
	}
}

func indexRoutes(svcCtx *svc.ServiceContext) []rest.Route {
	spec := crudSpec{table: meta.TableIndices, onWrite: onIndexWrite}
	return []rest.Route{
		{Method: http.MethodGet, Path: "/indices", Handler: metaQuery(svcCtx, meta.TableIndices)},
		{Method: http.MethodGet, Path: "/indices/:id/ref", Handler: IndexRefHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/indices/:id/views", Handler: IndexViewsHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/indices/:id", Handler: metaGet(svcCtx, meta.TableIndices, cache.GetIndexByIDOrName)},
		{Method: http.MethodPost, Path: "/indices", Handler: metaCreate(svcCtx, spec)},
		{Method: http.MethodPut, Path: "/indices/:id", Handler: metaUpdate(svcCtx, spec)},
		{Method: http.MethodDelete, Path: "/indices/:id", Handler: metaDelete(svcCtx, meta.TableIndices, nil)},
	}
}

func varsListRoutes(svcCtx *svc.ServiceContext) []rest.Route {
	spec := crudSpec{table: meta.TableVarsLists, onWrite: onVarsListWrite}
	r := simpleCRUD(svcCtx, "/vars_lists", meta.TableVarsLists, spec)
	r = append(r,
		rest.Route{Method: http.MethodGet, Path: "/vars_lists/:id/check", Handler: VarsListCheckHandler(svcCtx)},
		rest.Route{Method: http.MethodGet, Path: "/vars_lists/:id/1", Handler: VarsListFirstHandler(svcCtx)},
	)
	return r
}

func accountRoutes(svcCtx *svc.ServiceContext) []rest.Route {
	r := simpleCRUD(svcCtx, "/accounts", meta.TableAccounts, crudSpec{})
	r = append(r,
		rest.Route{Method: http.MethodGet, Path: "/accounts_filter", Handler: AccountFilterHandler(svcCtx)},
		rest.Route{Method: http.MethodPost, Path: "/accounts_domains", Handler: AccountDomainsHandler(svcCtx)},
		rest.Route{Method: http.MethodPost, Path: "/accounts_usernames", Handler: AccountUsernamesHandler(svcCtx)},
	)
	return r
}

func proxyRoutes(svcCtx *svc.ServiceContext) []rest.Route {
	r := simpleCRUD(svcCtx, "/proxies", meta.TableProxies, crudSpec{})
	for i := range r {
		if r[i].Method == http.MethodDelete && r[i].Path == "/proxies/:id" {
			r[i].Handler = metaDelete(svcCtx, meta.TableProxies, proxyDeleteGuard)
		}
	}
	r = append(r, rest.Route{Method: http.MethodPost, Path: "/proxies/:id/init", Handler: ProxyInitHandler(svcCtx)})
	return r
}

func templateRoutes(svcCtx *svc.ServiceContext) []rest.Route {
	spec := crudSpec{table: meta.TableTemplates, onWrite: onTemplateWrite}
	return []rest.Route{
		{Method: http.MethodGet, Path: "/templates", Handler: metaQuery(svcCtx, meta.TableTemplates)},
		{Method: http.MethodGet, Path: "/templates/mapper_types", Handler: TemplateMapperTypesHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/templates/:id/ref", Handler: TemplateRefHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/templates/:id/var_names", Handler: TemplateVarNamesHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/templates/:id", Handler: metaGet(svcCtx, meta.TableTemplates, nil)},
		{Method: http.MethodPost, Path: "/templates", Handler: metaCreate(svcCtx, spec)},
		{Method: http.MethodPut, Path: "/templates/:id", Handler: metaUpdate(svcCtx, spec)},
		{Method: http.MethodDelete, Path: "/templates/:id", Handler: metaDelete(svcCtx, meta.TableTemplates, nil)},
	}
}

func taskRoutes(svcCtx *svc.ServiceContext) []rest.Route {
	spec := crudSpec{table: meta.TableTasks, onWrite: onTaskWrite}
	return []rest.Route{
		{Method: http.MethodGet, Path: "/tasks", Handler: metaQuery(svcCtx, meta.TableTasks)},
		{Method: http.MethodPost, Path: "/tasks/:id/copy", Handler: TaskCopyHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/tasks/:id", Handler: metaGet(svcCtx, meta.TableTasks, nil)},
		{Method: http.MethodPost, Path: "/tasks", Handler: metaCreate(svcCtx, spec)},
		{Method: http.MethodPut, Path: "/tasks/:id", Handler: metaUpdate(svcCtx, spec)},
		{Method: http.MethodDelete, Path: "/tasks/:id", Handler: metaDelete(svcCtx, meta.TableTasks, taskDeleteGuard)},
	}
}
