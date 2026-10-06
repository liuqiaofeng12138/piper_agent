package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"piper_go/internal/config"

	"github.com/zeromicro/go-zero/core/logx"
)

var defaultRequiredContainers = []string{"elasticsearch", "minio", "prometheus"}

// EnsureDependencies verifies Docker containers and HTTP/TCP endpoints before serving.
func EnsureDependencies(ctx context.Context, c config.Config) error {
	if !c.WebAPI.RequireDeps {
		if err := WaitStorageReady(ctx, c); err != nil {
			return err
		}
		EnsureGrafanaDashboard(ctx, c)
		return nil
	}

	containers := c.WebAPI.RequiredContainers
	if len(containers) == 0 {
		containers = append([]string(nil), defaultRequiredContainers...)
	}

	maxWait := time.Duration(c.WebAPI.DepsMaxWaitSeconds) * time.Second
	interval := time.Duration(c.WebAPI.StorageWaitEvery) * time.Millisecond
	if interval <= 0 {
		interval = 5 * time.Second
	}

	deadline := time.Now().Add(maxWait)

	var lastErr error
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		lastErr = checkAllDeps(ctx, c, containers)
		if lastErr == nil {
			logx.Info("all required dependencies are ready")
			EnsureGrafanaDashboard(ctx, c)
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("startup dependency check failed: %w", lastErr)
		}

		logx.Infof("dependencies not ready (%v), retry in %s...", lastErr, interval)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

func checkAllDeps(ctx context.Context, c config.Config, containers []string) error {
	var problems []string

	for _, name := range containers {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if err := checkDockerContainer(ctx, name); err != nil {
			problems = append(problems, err.Error())
		}
	}

	esURL := fmt.Sprintf("http://%s:%d", c.ES.Host, c.ES.Port)
	if !pingHTTP(ctx, esURL) {
		problems = append(problems, fmt.Sprintf("Elasticsearch unreachable at %s", esURL))
	}

	s3URL := strings.TrimSpace(c.S3.EndpointURL)
	if s3URL != "" && !pingHTTP(ctx, s3URL) {
		problems = append(problems, fmt.Sprintf("S3/MinIO unreachable at %s", s3URL))
	}

	if prom := strings.TrimSpace(c.WebAPI.PrometheusHost); prom != "" {
		if !pingHTTP(ctx, prom) {
			problems = append(problems, fmt.Sprintf("Prometheus unreachable at %s", prom))
		}
	}

	if len(problems) == 0 {
		return nil
	}
	return errors.New(strings.Join(problems, "; "))
}

type dockerContainerState struct {
	Running bool `json:"Running"`
	Health  *struct {
		Status string `json:"Status"`
	} `json:"Health"`
}

func checkDockerContainer(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{json .State}}", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("docker container %q: %s (is Docker running?)", name, msg)
	}

	var st dockerContainerState
	if err := json.Unmarshal(out, &st); err != nil {
		return fmt.Errorf("docker container %q: parse state: %w", name, err)
	}
	if !st.Running {
		return fmt.Errorf("docker container %q is not running", name)
	}
	if st.Health != nil {
		switch strings.ToLower(st.Health.Status) {
		case "healthy", "":
			return nil
		case "starting":
			return fmt.Errorf("docker container %q health status is starting", name)
		default:
			return fmt.Errorf("docker container %q health status is %q", name, st.Health.Status)
		}
	}
	return nil
}
