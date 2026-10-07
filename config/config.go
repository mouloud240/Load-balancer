package config

import (
	"net/url"
	"os"

	"github.com/go-yaml/yaml"
)
type weightedUpstream struct{
	Upstream string `yaml:"upstream"`
	Weight  int   `yaml:"weight"`
}
type RawConfig struct{
	Upstreams[]  weightedUpstream `yaml:"upstreams"`
	Listener  string  `yaml:"listener"`
}
type UpstreamConfig struct{
	URL url.URL
	Weight int
}
type Config struct{
	Upstreams []UpstreamConfig
	Listener  string
}

func NewConfig(upstreams []UpstreamConfig, listener string) *Config {
	return &Config{
		Upstreams: upstreams,
		Listener:listener,
	}
}
func LoadConfig() *Config{	
	data,err:=os.ReadFile("config.yaml")
	if err!=nil{
		panic("No config file was found")

	}
	raw:=RawConfig{}
	yaml.Unmarshal(data,&raw)

	rawUrls:=raw.Upstreams
	upstreams := make([]UpstreamConfig, len(rawUrls))
	for i, ru:= range rawUrls {
    u, _ := url.Parse(ru.Upstream)
    weight := ru.Weight
    if weight <= 0 {
      weight = 1
    }
    upstreams[i] = UpstreamConfig{URL: *u, Weight: weight} // u is nil on failure -> panic: nil pointer dereference
}
	return NewConfig(upstreams,raw.Listener)
}
