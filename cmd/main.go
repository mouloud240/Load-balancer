package main

import (
	"github.com/lb/config"
	"github.com/lb/upstreams"
)
func main() {
	conf:=config.LoadConfig()
	lb, err := upstreams.NewLoadBalancer(conf.Upstreams)
	if err != nil {
		panic(err)
	}
	if err := lb.Run(); err != nil {
		panic(err)
	}
}
