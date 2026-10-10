package agentruntime

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/zeromicro/go-zero/core/conf"
	"piper_go/internal/config"
)

func TestLoadChromeFromUnifiedLocalYAML(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	cfgPath := filepath.Join(root, "deploy", "config", "local.yaml")
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Skip(cfgPath)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	engine, ok := doc["engine"]
	if !ok {
		t.Fatal("no engine")
	}
	engineYAML, err := yaml.Marshal(engine)
	if err != nil {
		t.Fatal(err)
	}
	var c config.Config
	if err := conf.LoadFromYamlBytes(engineYAML, &c); err != nil {
		t.Fatal(err)
	}
	if !c.Chrome.Enabled {
		t.Fatalf("chrome enabled: %#v", c.Chrome)
	}
	if c.Chrome.Headless {
		t.Fatalf("expected headless false, got %#v", c.Chrome)
	}
	if c.Chrome.ManualLoginWaitSeconds != 120 {
		t.Fatalf("manualLoginWaitSeconds want 120 got %d", c.Chrome.ManualLoginWaitSeconds)
	}
}
