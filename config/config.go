package config

import (
	"net/url"
)
type Config struct{
	Upstreams []url.URL
}



func NewConfig(upstreams []url.URL) *Config {
	return &Config{
		Upstreams: upstreams,
	}
}
func LoadConfig() *Config{
	

	rawUrls:= []string{
		"http://localhost:3002",
		"http://localhost:3001",
		"http://localhost:3000",
		"http://localhost:3003",
	}
	upstreams := make([]url.URL, len(rawUrls))
	for i, raw := range rawUrls {
    u, _ := url.Parse(raw)
    upstreams[i] = *u // u is nil on failure -> panic: nil pointer dereference
}
	return NewConfig(upstreams)
}
