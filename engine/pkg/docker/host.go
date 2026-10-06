package docker

import (
	"fmt"
	"sync"
)

// Host is a minimal DockerHost registry (Java nio.docker.DockerHost subset).
type Host struct {
	IP   string
	Port int
}

var (
	localIP = "127.0.0.1"
	hosts   = map[string]*Host{}
	mu      sync.RWMutex
)

func GetHost(ip string) *Host {
	if ip == "" || ip == localIP {
		return getLocal()
	}
	mu.RLock()
	h := hosts[ip]
	mu.RUnlock()
	return h
}

func getLocal() *Host {
	mu.Lock()
	defer mu.Unlock()
	if h, ok := hosts[localIP]; ok {
		return h
	}
	h := &Host{IP: localIP, Port: 2375}
	hosts[localIP] = h
	return h
}

func AddHost(h *Host) {
	if h == nil || h.IP == "" {
		return
	}
	mu.Lock()
	hosts[h.IP] = h
	mu.Unlock()
}

// ChromeContainer placeholder for Java Docker ChromeContainer (Phase 6 baseline: chromedp local).
type ChromeContainer struct {
	ID     string
	HostIP string
}

func (h *Host) CreateChromeContainer(name string) (*ChromeContainer, error) {
	if h == nil {
		return nil, fmt.Errorf("docker host not configured")
	}
	return &ChromeContainer{ID: name, HostIP: h.IP}, nil
}
