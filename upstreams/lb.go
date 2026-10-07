package upstreams

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lb/config"
)

type LoadBalancer struct {
	current  int
	mutex    sync.Mutex
	upstreams []*Upstream
	client   *http.Client
}

func NewLoadBalancer(upstreams []config.UpstreamConfig) (*LoadBalancer, error) {
	
	parsedUpstreams := make([]*Upstream, len(upstreams))
	for idx,upstream:= range upstreams {
		parsedUpstreams[idx]= NewUpstream(upstream.URL, upstream.Weight)
	}
	for idx, upstream := range parsedUpstreams {
		if upstream == nil {
			fmt.Print("Upstream at index ", idx, " is nil\n")

			continue
		}
		fmt.Printf("Upstream: %s, State: %d\n", upstream.Server.String(), upstream.State)
	}
	if len(upstreams) == 0 {
		return nil, ErrNoServers
	}

	return &LoadBalancer{
		upstreams:parsedUpstreams,
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

//Scan from snapshotted start so multiple halfOpen/open upstreams can't force a blind pick, we only return closed or admitted halfOpen probe
func (l *LoadBalancer) CheckNextAvailable() (*Upstream, error) {

	l.mutex.Lock()
	defer l.mutex.Unlock()

	start := l.current
	for i := 0; i < len(l.upstreams); i++ {
		idx := (start + i) % len(l.upstreams)
		server := l.upstreams[idx]

		admitted, exhausted := server.ClaimRequest()
		if !admitted {
			continue
		}

		if exhausted {
			l.current = (idx + 1) % len(l.upstreams)
		} else {
			l.current = idx
		}
		return server, nil
	}
	return nil, ErrNoServers
}

func (l *LoadBalancer) handler(w http.ResponseWriter, r *http.Request) {
	upstream, err := l.CheckNextAvailable()
	if err != nil {
		http.Error(w, "no backend servers available", http.StatusServiceUnavailable)
		return
	}
	backend:=upstream.Server

	target := backend
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
	//TODO: Maybe I can use context cloning instead of full headers clone

	req.Header = r.Header.Clone()

	req.Header.Set("X-Forwarded-Host", r.Host)

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	req.Header.Set("X-Forwarded-For", host)

	resp, err := l.client.Do(req)
	if err != nil {
		fmt.Print(err.Error())
		http.Error(w, "backend unavailable", http.StatusBadGateway)
		upstream.recordFailure()
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		upstream.recordSuccess()
	}
	
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
