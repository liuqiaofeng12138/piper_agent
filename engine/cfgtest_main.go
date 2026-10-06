//go:build ignore

package main

import (
	"fmt"

	"piper_go/internal/config"

	"github.com/zeromicro/go-zero/core/conf"
)

func main() {
	var c config.Config
	conf.MustLoad("etc/pipergo-api.yaml", &c)
	fmt.Printf("NoAuth=%v PrometheusHost=%q\n", c.WebAPI.NoAuth, c.WebAPI.PrometheusHost)
}
