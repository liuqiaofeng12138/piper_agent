package impl

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"

	sshpkg "piper_go/pkg/network/ssh"
)

var (
	ErrBroken           = errors.New("proxy broken")
	localPortCounter    int64 = 48000
	defaultRemotePort         = 8888
	defaultPrivateKeyPath     = filepath.Join("pk", "proxy_pk")
)

// Session holds an active SSH tunnel for one proxy doc.
type Session struct {
	Host     *sshpkg.Host
	Listener netCloser
}

type netCloser interface {
	Close() error
}

// Setup validates SSH, ensures tinyproxy on VPS, opens local port forward (Java ProxyPPPD.setup).
func Setup(doc map[string]any) (map[string]any, *Session, error) {
	host := str(doc, "ssh_host")
	user := str(doc, "ssh_user")
	if host == "" || user == "" {
		return nil, nil, ErrBroken
	}
	port := intNum(doc, "ssh_port", 22)
	pass := str(doc, "ssh_passwd")
	var sh *sshpkg.Host
	if pass != "" {
		sh = sshpkg.NewPassword(host, port, user, pass)
	} else {
		pem, err := sshpkg.LoadPrivateKeyFile(defaultPrivateKeyPath)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrBroken, err)
		}
		sh = sshpkg.NewPrivateKey(host, port, user, pem)
	}
	if err := sh.Connect(); err != nil {
		return nil, nil, ErrBroken
	}
	out, _ := sh.Exec("[ -f /run/tinyproxy/tinyproxy.pid ] && echo 1")
	if !strings.Contains(out, "1") {
		sh.Close()
		return nil, nil, fmt.Errorf("%w: tinyproxy not installed on VPS", ErrBroken)
	}
	localHost := str(doc, "host")
	localPort := intNum(doc, "port", 0)
	if localHost == "" || localPort == 0 {
		localPort = int(atomic.AddInt64(&localPortCounter, 1))
		localHost = "127.0.0.1"
	}
	ln, err := sh.StartLocalForward(localPort, fmt.Sprintf("127.0.0.1:%d", defaultRemotePort))
	if err != nil {
		sh.Close()
		return nil, nil, ErrBroken
	}
	doc["host"] = localHost
	doc["port"] = localPort
	doc["status"] = "Free"
	doc["timeout_count"] = 0
	sess := &Session{Host: sh, Listener: ln}
	return doc, sess, nil
}

func Close(sess *Session) {
	if sess == nil {
		return
	}
	if sess.Listener != nil {
		_ = sess.Listener.Close()
	}
	if sess.Host != nil {
		sess.Host.Close()
	}
}

func str(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return v
}

func intNum(m map[string]any, k string, def int) int {
	switch v := m[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		if n > 0 {
			return n
		}
	}
	return def
}
