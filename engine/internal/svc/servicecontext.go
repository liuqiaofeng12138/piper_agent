package svc

import (
	"sync/atomic"

	"piper_go/internal/config"
	"piper_go/pkg/auth"
	"piper_go/pkg/cluster/model"
	"piper_go/pkg/db/es"
	"piper_go/pkg/db/meta"
	"piper_go/pkg/distributor"
	"piper_go/pkg/log"
	"piper_go/pkg/notification"
	"piper_go/pkg/persistence"
)

type ServiceContext struct {
	Config config.Config
	Auth   *auth.Service
	Meta   *meta.Store
	ES     *es.Client
	Dist   *distributor.Engine
	Persist *persistence.Persister
	LogES   *log.Appender
	Notify  *notification.Service

	ready atomic.Bool
	node  atomic.Pointer[model.NodeInfo]
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		Auth:   auth.NewService(c.Auth),
	}
}

func (s *ServiceContext) SetReady(v bool) {
	s.ready.Store(v)
}

func (s *ServiceContext) Ready() bool {
	return s.ready.Load()
}

func (s *ServiceContext) SetNode(n *model.NodeInfo) {
	s.node.Store(n)
}

func (s *ServiceContext) Node() *model.NodeInfo {
	return s.node.Load()
}

func (s *ServiceContext) NodeInfoForAPI() *model.NodeInfo {
	src := s.Node()
	if src == nil {
		return &model.NodeInfo{Ready: s.Ready()}
	}
	copy := *src
	copy.ID = copy.InstID
	copy.Local = false
	copy.Ready = s.Ready()
	return &copy
}
