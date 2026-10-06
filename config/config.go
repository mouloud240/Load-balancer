package config

import (
	"net/url"
	"os"

	"github.com/go-yaml/yaml"
)
type RawConfig struct{
	upstreams[]  string `yaml:"upstreams"`
	listener  string  `yaml:"listener"`
}
type Config struct{
	Upstreams []url.URL
	Listener  string
}

func NewConfig(upstreams []url.URL, listener string) *Config {
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
	yaml.Unmarshal(data,raw)

	rawUrls:=raw.upstreams
	upstreams := make([]url.URL, len(rawUrls))
	for i, ru:= range rawUrls {
    u, _ := url.Parse(ru)
    upstreams[i] = *u // u is nil on failure -> panic: nil pointer dereference
}
	return NewConfig(upstreams,raw.listener)
}
