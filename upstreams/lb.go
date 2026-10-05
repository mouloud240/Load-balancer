package upstreams

import (
	"fmt"
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
	upstreams []*Upstream
	client   *http.Client
}

func NewLoadBalancer(upstreams []url.URL) (*LoadBalancer, error) {

	
	parsedUpstreams := make([]*Upstream, len(upstreams))
	for idx,upstream:= range upstreams {
		parsedUpstreams[idx]= NewUpstream(upstream)
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

func (l *LoadBalancer) nextUpstream() *Upstream {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	server := l.upstreams[l.current]
	l.current = (l.current + 1) % len(l.upstreams)
	if server.State == open {
		server.TransitionToHalfOpen()
		//If the server is closed we just go to the next one,  for simplicty
		//If last checked is more than our threshold we can set it half p
		server = l.upstreams[l.current]
	  l.current = (l.current + 1) % len(l.upstreams)
	}
	if server.State == halfOpen{

		// If the server is half open we will pass requests as well
		
	}

return server
}

func (l *LoadBalancer) handler(w http.ResponseWriter, r *http.Request) {
	upstream:= l.nextUpstream()
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
