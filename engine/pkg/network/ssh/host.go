package ssh

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

const defaultTimeout = 15 * time.Second

// Host is a minimal JSch SshHost equivalent for proxy/docker ops.
type Host struct {
	Addr     string
	Port     int
	User     string
	Password string
	KeyPEM   []byte

	client *ssh.Client
}

func NewPassword(host string, port int, user, password string) *Host {
	return &Host{Addr: host, Port: port, User: user, Password: password}
}

func NewPrivateKey(host string, port int, user string, pem []byte) *Host {
	return &Host{Addr: host, Port: port, User: user, KeyPEM: pem}
}

func LoadPrivateKeyFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (h *Host) Connect() error {
	if h.client != nil {
		return nil
	}
	var auth []ssh.AuthMethod
	if len(h.KeyPEM) > 0 {
		signer, err := ssh.ParsePrivateKey(h.KeyPEM)
		if err != nil {
			return err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	if h.Password != "" {
		auth = append(auth, ssh.Password(h.Password))
	}
	cfg := &ssh.ClientConfig{
		User:            h.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         defaultTimeout,
	}
	addr := fmt.Sprintf("%s:%d", h.Addr, h.Port)
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return err
	}
	h.client = client
	return nil
}

func (h *Host) Close() {
	if h.client != nil {
		_ = h.client.Close()
		h.client = nil
	}
}

func (h *Host) Exec(cmd string) (string, error) {
	if h.client == nil {
		return "", fmt.Errorf("ssh not connected")
	}
	sess, err := h.client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	var buf bytes.Buffer
	sess.Stdout = &buf
	sess.Stderr = &buf
	if err := sess.Run(cmd); err != nil {
		return buf.String(), err
	}
	return buf.String(), nil
}

// StartLocalForward binds localPort on 127.0.0.1 and forwards to remoteAddr via SSH.
func (h *Host) StartLocalForward(localPort int, remoteAddr string) (net.Listener, error) {
	if h.client == nil {
		return nil, fmt.Errorf("ssh not connected")
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			local, err := ln.Accept()
			if err != nil {
				return
			}
			go h.forwardConn(local, remoteAddr)
		}
	}()
	return ln, nil
}

func (h *Host) forwardConn(local net.Conn, remoteAddr string) {
	defer local.Close()
	remote, err := h.client.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remote.Close()
	go func() { _, _ = io.Copy(remote, local) }()
	_, _ = io.Copy(local, remote)
}
