package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf

	WebAPI    WebAPIConf    `json:"WebAPI"`
	ES        ESConf        `json:"ES"`
	S3        S3Conf        `json:"S3"`
	H2        H2Conf        `json:"H2"`
	Auth      AuthConf      `json:"Auth"`
	Requester RequesterConf `json:"Requester"`
	Chrome    ChromeConf    `json:"Chrome"`
}

type ChromeConf struct {
	Enabled    bool   `json:"enabled,default=false"`
	Headless   bool   `json:"headless,default=true"`
	BinaryPath string `json:"binaryPath,optional"`
	AgentCount int    `json:"agentCount,default=1"`
}

type H2Conf struct {
	Path string `json:"path,default=./db/raw_meta.db"`
}

type WebAPIConf struct {
	NoAuth                    bool     `json:"noAuth,default=true"`
	Notification              bool     `json:"notification,default=false"`
	OnlyVideo                 bool     `json:"onlyVideo,default=false"`
	PrometheusHost            string   `json:"prometheusHost,optional"`
	// Instance label Prometheus uses when scraping this API (e.g. host.docker.internal:8888).
	PrometheusScrapeInstance  string   `json:"prometheusScrapeInstance,optional"`
	GrafanaHost               string   `json:"grafanaHost,optional"`
	ProvisionGrafanaDashboard bool     `json:"provisionGrafanaDashboard,default=true"`
	GrafanaAdminUser          string   `json:"grafanaAdminUser,optional"`
	GrafanaAdminPassword      string   `json:"grafanaAdminPassword,optional"`
	SkipStorageWait           bool     `json:"skipStorageWait,default=false"`
	StorageWaitEvery          int      `json:"storageWaitEveryMs,default=5000"`
	RequireDeps               bool     `json:"requireDeps,default=false"`
	RequiredContainers        []string `json:"requiredContainers,optional"`
	DepsMaxWaitSeconds        int      `json:"depsMaxWaitSeconds,default=120"`
}

type ESConf struct {
	Host     string `json:"host,default=127.0.0.1"`
	Port     int    `json:"port,default=9200"`
	Username string `json:"username,optional"`
	Password string `json:"password,optional"`
}

type S3Conf struct {
	EndpointURL string `json:"endpointUrl,default=http://127.0.0.1:9000"`
	AccessKey   string `json:"accessKey,optional"`
	SecretKey   string `json:"secretKey,optional"`
}

type AuthConf struct {
	TokenTTLSeconds int64      `json:"tokenTtlSeconds,default=86400"`
	ServiceToken    string     `json:"serviceToken,default=piper-internal-service-token"`
	Users           []AuthUser `json:"users,optional"`
}

type AuthUser struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Email    string   `json:"email,optional"`
	Roles    []string `json:"roles,optional"`
}

type RequesterConf struct {
	ConnectTimeout        int64 `json:"connectTimeout,default=80000"`
	ReadTimeout           int64 `json:"readTimeout,default=120000"`
	TokenTimeout          int64 `json:"tokenTimeout,default=120000"`
	RequestPerSecondLimit int64 `json:"requestPerSecondLimit,default=20"`
	PageLoadTimeout       int64 `json:"pageLoadTimeout,optional"`
}
