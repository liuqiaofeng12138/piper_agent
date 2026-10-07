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
	"runtime"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"piper_agent/gateway/internal/app"
	"piper_agent/gateway/internal/config"
)

// RunAll 启动 Runtime、Python Worker 与 HTTP 网关（Runtime/Worker 为子进程，网关阻塞于当前进程）。
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
	runtimeListen := yamlListen(absConfig, "listen", ":50051")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repoRoot := findRepoRoot(absConfig)

	rtCmd, err := startRuntime(ctx, absConfig, repoRoot)
	if err != nil {
		return err
	}
	defer terminateProcess(rtCmd)
	log.Printf("piper-runtime starting on %s", runtimeListen)
	if err := waitTCP(runtimeListen, 180*time.Second); err != nil {
		return fmt.Errorf("runtime not ready: %w", err)
	}
	log.Printf("piper-runtime ready at %s", runtimeListen)

	var workerCmds []*exec.Cmd
	terminateWorkers := func() {
		for _, c := range workerCmds {
			terminateProcess(c)
		}
	}
	defer terminateWorkers()

	for _, agent := range gwCfg.EnabledAgents() {
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
		gwErr <- app.StartHTTP(gwCfg, gatewayListen)
	}()

	select {
	case <-ctx.Done():
		log.Printf("shutting down...")
		terminateWorkers()
		terminateProcess(rtCmd)
		return nil
	case err := <-gwErr:
		terminateWorkers()
		terminateProcess(rtCmd)
		return fmt.Errorf("gateway exited: %w", err)
	}
}

func startRuntime(ctx context.Context, configPath string, repoRoot string) (*exec.Cmd, error) {
	runtimeDir := filepath.Join(repoRoot, "runtime")
	if override := os.Getenv("PIPER_RUNTIME_CMD"); override != "" {
		cmd := exec.CommandContext(ctx, override, "-f", configPath)
		cmd.Dir = runtimeDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return cmd, nil
	}

	exeName := "piper-runtime.exe"
	if runtime.GOOS != "windows" {
		exeName = "piper-runtime"
	}
	built := filepath.Join(runtimeDir, exeName)
	if fileExists(built) {
		cmd := exec.CommandContext(ctx, built, "-f", configPath)
		cmd.Dir = runtimeDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("start %s: %w", built, err)
		}
		return cmd, nil
	}

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/piper-runtime", "-f", configPath)
	cmd.Dir = runtimeDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("go run piper-runtime: %w", err)
	}
	return cmd, nil
}

func startWorker(ctx context.Context, configPath string, repoRoot string, agent config.AgentSpec) (*exec.Cmd, error) {
	agentsDir := filepath.Join(repoRoot, "agents")
	name, args, env := workerCommand(configPath, agentsDir, agent)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = agentsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start worker %s %v: %w", name, args, err)
	}
	return cmd, nil
}

// workerCommand 启动 gRPC Worker（python -m <agent.Module>），不经 piper-agent CLI。
func workerCommand(configPath string, agentsDir string, agent config.AgentSpec) (string, []string, []string) {
	extraEnv := []string{
		fmt.Sprintf("PYTHONPATH=%s", filepath.Join(agentsDir, "src")),
	}
	moduleArgs := []string{
		"-m", agent.WorkerModule(),
		"--config", configPath,
		"--listen", agent.Address,
	}

	if override := os.Getenv("PIPER_WORKER_PYTHON"); override != "" {
		return override, moduleArgs, extraEnv
	}

	venvPy := filepath.Join(agentsDir, ".venv", "Scripts", "python.exe")
	if runtime.GOOS != "windows" {
		venvPy = filepath.Join(agentsDir, ".venv", "bin", "python")
	}
	if fileExists(venvPy) {
		return venvPy, moduleArgs, extraEnv
	}

	py := "python"
	if p, err := exec.LookPath("python3"); err == nil {
		py = p
	} else if p, err := exec.LookPath("python"); err == nil {
		py = p
	}
	return py, moduleArgs, extraEnv
}

func yamlListen(configPath string, key string, fallback string) string {
	b, err := os.ReadFile(configPath)
	if err != nil {
		return fallback
	}
	var doc map[string]any
	if yaml.Unmarshal(b, &doc) != nil {
		return fallback
	}
	if v, ok := doc[key].(string); ok && v != "" {
		return v
	}
	return fallback
}

func findRepoRoot(configPath string) string {
	dir := filepath.Dir(configPath)
	for i := 0; i < 6; i++ {
		if fileExists(filepath.Join(dir, "go.work")) || fileExists(filepath.Join(dir, "agents", "pyproject.toml")) {
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
