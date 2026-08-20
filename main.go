package main

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type LoadBalancer struct {
	current  int
	mutex    sync.Mutex
	servers  []*url.URL
	client   *http.Client
}

func NewLoadBalancer(servers []string) (*LoadBalancer, error) {
	if len(servers) == 0 {
		return nil, ErrNoServers
	}

	backends := make([]*url.URL, len(servers))

	for i, server := range servers {
		u, err := url.Parse(server)
		if err != nil {
			return nil, err
		}

		backends[i] = u
	}

	return &LoadBalancer{
		servers: backends,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

var ErrNoServers = &LBError{"no backend servers configured"}

type LBError struct {
	message string
}

func (e *LBError) Error() string {
	return e.message
}

func (l *LoadBalancer) nextServer() *url.URL {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	server := l.servers[l.current]
	l.current = (l.current + 1) % len(l.servers)

	return server
}

func (l *LoadBalancer) handler(w http.ResponseWriter, r *http.Request) {
	backend := l.nextServer()

	target := *backend
	target.Path = strings.TrimRight(backend.Path, "/") + r.URL.Path
	target.RawQuery = r.URL.RawQuery

	req, err := http.NewRequestWithContext(
		r.Context(),
		r.Method,
		target.String(),
		r.Body,
	)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header = r.Header.Clone()

	req.Header.Set("X-Forwarded-Host", r.Host)

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	req.Header.Set("X-Forwarded-For", host)

	resp, err := l.client.Do(req)
	if err != nil {
		http.Error(w, "backend unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		return
	}
}

func (l *LoadBalancer) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", l.handler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return server.ListenAndServe()
}

func main() {
	lb, err := NewLoadBalancer([]string{
		"http://localhost:3000",
		"http://localhost:3001",
		"http://localhost:3002",
	})
	if err != nil {
		panic(err)
	}

	if err := lb.Run(); err != nil {
		panic(err)
	}
}
