package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"piper_agent/gateway/internal/app"
	"piper_agent/gateway/internal/config"
	"piper_agent/gateway/internal/integrate"
)

// RunAll 内嵌 Piper Runtime + Go web_crawler，拉起其余 Python Worker 与 HTTP 网关。
func RunAll(configPath string, gatewayListen string) error {
	configPath = filepath.Clean(configPath)
	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return err
	}

	gwCfg, err := config.Load(absConfig)
	if err != nil {
		return fmt.Errorf("gateway config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repoRoot := findRepoRoot(absConfig)

	inproc, err := integrate.InprocWorkers(gwCfg, absConfig, repoRoot)
	if err != nil {
		return err
	}
	var serveOpts *app.ServeOptions
	if len(inproc) > 0 {
		serveOpts = &app.ServeOptions{InprocWorkers: inproc}
		log.Printf("web_crawler: in-process Go Worker + embedded Runtime (no piper-runtime / Python worker)")
	}

	var workerCmds []*exec.Cmd
	terminateWorkers := func() {
		for _, c := range workerCmds {
			terminateProcess(c)
		}
	}
	defer terminateWorkers()

	for _, agent := range gwCfg.EnabledAgents() {
		if agent.ID == "web_crawler" {
			continue
		}
		if agent.Address == "" {
			log.Printf("agent %s enabled 但未配置 address，跳过启动", agent.ID)
			continue
		}
		workerCmd, err := startWorker(ctx, absConfig, repoRoot, agent)
		if err != nil {
			return fmt.Errorf("start worker %s: %w", agent.ID, err)
		}
		workerCmds = append(workerCmds, workerCmd)
		log.Printf("worker %s starting at %s", agent.ID, agent.Address)
		if err := waitTCP(agent.Address, 90*time.Second); err != nil {
			return fmt.Errorf("worker %s not ready: %w", agent.ID, err)
		}
		log.Printf("worker %s ready at %s", agent.ID, agent.Address)
	}

	gwErr := make(chan error, 1)
	go func() {
		gwErr <- app.StartHTTP(gwCfg, gatewayListen, serveOpts, absConfig, repoRoot)
	}()

	select {
	case <-ctx.Done():
		log.Printf("shutting down...")
		terminateWorkers()
		return nil
	case err := <-gwErr:
		terminateWorkers()
		return fmt.Errorf("gateway exited: %w", err)
	}
}

func startWorker(ctx context.Context, configPath string, repoRoot string, agent config.AgentSpec) (*exec.Cmd, error) {
	name, args, env, err := workerCommand(configPath, repoRoot, agent)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start worker %s %v: %w", name, args, err)
	}
	return cmd, nil
}

// workerCommand 启动 gRPC Worker（python -m <agent.Module>），不经 CLI。
func workerCommand(configPath string, repoRoot string, agent config.AgentSpec) (string, []string, []string, error) {
	srcDirs := discoverAgentSrcDirs(repoRoot)
	extraEnv := []string{
		fmt.Sprintf("PYTHONPATH=%s", strings.Join(srcDirs, string(os.PathListSeparator))),
	}
	moduleArgs := []string{
		"-m", agent.WorkerModule(),
		"--config", configPath,
		"--listen", agent.Address,
	}

	if override := os.Getenv("PIPER_WORKER_PYTHON"); override != "" {
		return override, moduleArgs, extraEnv, nil
	}

	py, proj, err := resolveWorkerPython(repoRoot, agent)
	if err != nil {
		return "", nil, nil, fmt.Errorf("worker %s: %w", agent.ID, err)
	}
	log.Printf("worker %s using python project %s (%s)", agent.ID, proj, py)
	return py, moduleArgs, extraEnv, nil
}

// RepoRoot 根据配置文件路径推断仓库根目录。
func RepoRoot(configPath string) string {
	return findRepoRoot(configPath)
}

func findRepoRoot(configPath string) string {
	dir := filepath.Dir(configPath)
	for i := 0; i < 6; i++ {
		if fileExists(filepath.Join(dir, "go.work")) ||
			fileExists(filepath.Join(dir, "general_agent", "pyproject.toml")) ||
			fileExists(filepath.Join(dir, "rag_agent", "pyproject.toml")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Dir(filepath.Dir(configPath))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func waitTCP(addr string, timeout time.Duration) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	target := net.JoinHostPort(host, port)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", target, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return errors.New("timeout waiting for " + target)
}

func terminateProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
