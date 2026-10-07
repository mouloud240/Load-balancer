package upstreams

import (
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/lb/config"
)

func parseURL(raw string) url.URL {
	u, _ := url.Parse(raw)
	return *u
}

func TestCheckNextAvailable_WeightedDistribution(t *testing.T) {
	configs := []config.UpstreamConfig{
		{URL: parseURL("http://backend1:8081"), Weight: 1},
		{URL: parseURL("http://backend2:8082"), Weight: 2},
	}

	lb, err := NewLoadBalancer(configs)
	if err != nil {
		t.Fatalf("unexpected error creating load balancer: %v", err)
	}

	expectedOrder := []string{
		"http://backend1:8081",
		"http://backend2:8082",
		"http://backend2:8082",
		"http://backend1:8081",
		"http://backend2:8082",
		"http://backend2:8082",
	}

	for i, want := range expectedOrder {
		server, err := lb.CheckNextAvailable()
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i+1, err)
		}
		if got := server.Server.String(); got != want {
			t.Fatalf("request %d: expected %s, got %s", i+1, want, got)
		}
	}
}

func TestCheckNextAvailable_EqualWeights(t *testing.T) {
	configs := []config.UpstreamConfig{
		{URL: parseURL("http://backend1:8081"), Weight: 1},
		{URL: parseURL("http://backend2:8082"), Weight: 1},
	}

	lb, err := NewLoadBalancer(configs)
	if err != nil {
		t.Fatalf("unexpected error creating load balancer: %v", err)
	}

	expectedOrder := []string{
		"http://backend1:8081",
		"http://backend2:8082",
		"http://backend1:8081",
		"http://backend2:8082",
	}

	for i, want := range expectedOrder {
		server, err := lb.CheckNextAvailable()
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i+1, err)
		}
		if got := server.Server.String(); got != want {
			t.Fatalf("request %d: expected %s, got %s", i+1, want, got)
		}
	}
}

func TestCheckNextAvailable_SingleServer(t *testing.T) {
	configs := []config.UpstreamConfig{
		{URL: parseURL("http://backend1:8081"), Weight: 1},
	}

	lb, err := NewLoadBalancer(configs)
	if err != nil {
		t.Fatalf("unexpected error creating load balancer: %v", err)
	}

	for i := 0; i < 10; i++ {
		server, err := lb.CheckNextAvailable()
		if err != nil {
			t.Fatalf("request %d: expected success, got error %v", i+1, err)
		}
		if server == nil || server.Server.String() != "http://backend1:8081" {
			t.Fatalf("request %d: unexpected server returned", i+1)
		}
	}
}

func TestCheckNextAvailable_BypassOpenServer(t *testing.T) {
	configs := []config.UpstreamConfig{
		{URL: parseURL("http://backend1:8081"), Weight: 1},
		{URL: parseURL("http://backend2:8082"), Weight: 1},
	}

	lb, err := NewLoadBalancer(configs)
	if err != nil {
		t.Fatalf("unexpected error creating load balancer: %v", err)
	}

	// Force backend1 to open (down) with recent lastChecked so it doesn't transition to half-open
	lb.upstreams[0].State = open
	lb.upstreams[0].lastChecked = time.Now()

	for i := 0; i < 5; i++ {
		server, err := lb.CheckNextAvailable()
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i+1, err)
		}
		if server.Server.String() != "http://backend2:8082" {
			t.Fatalf("request %d: expected backend2, got %s", i+1, server.Server.String())
		}
	}
}

func TestCheckNextAvailable_HalfOpenProbe(t *testing.T) {
	configs := []config.UpstreamConfig{
		{URL: parseURL("http://backend1:8081"), Weight: 1},
		{URL: parseURL("http://backend2:8082"), Weight: 1},
	}

	lb, err := NewLoadBalancer(configs)
	if err != nil {
		t.Fatalf("unexpected error creating load balancer: %v", err)
	}

	// Backend 1 is half-open
	lb.upstreams[0].State = halfOpen
	lb.upstreams[0].skippedCount = 0

	// HALF_OPEN_SKIPPED_REQUEST is 5, so backend1 should be skipped 5 times
	for i := 0; i < HALF_OPEN_SKIPPED_REQUEST; i++ {
		server, err := lb.CheckNextAvailable()
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i+1, err)
		}
		if server.Server.String() != "http://backend2:8082" {
			t.Fatalf("request %d: expected backend2 to be picked while backend1 skipped, got %s", i+1, server.Server.String())
		}
	}

	// The 6th request should admit backend1 as probe!
	server, err := lb.CheckNextAvailable()
	if err != nil {
		t.Fatalf("probe request: unexpected error: %v", err)
	}
	if server.Server.String() != "http://backend1:8081" {
		t.Fatalf("expected backend1 as probe, got %s", server.Server.String())
	}

	// Subsequent request should move back to backend2
	server, err = lb.CheckNextAvailable()
	if err != nil {
		t.Fatalf("post-probe request: unexpected error: %v", err)
	}
	if server.Server.String() != "http://backend2:8082" {
		t.Fatalf("expected backend2 after probe, got %s", server.Server.String())
	}
}

func TestCheckNextAvailable_AllOpen(t *testing.T) {
	configs := []config.UpstreamConfig{
		{URL: parseURL("http://backend1:8081"), Weight: 1},
		{URL: parseURL("http://backend2:8082"), Weight: 1},
	}

	lb, err := NewLoadBalancer(configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lb.upstreams[0].State = open
	lb.upstreams[0].lastChecked = time.Now()
	lb.upstreams[1].State = open
	lb.upstreams[1].lastChecked = time.Now()

	_, err = lb.CheckNextAvailable()
	if err != ErrNoServers {
		t.Fatalf("expected ErrNoServers, got %v", err)
	}
}

func TestCheckNextAvailable_ConcurrentAccess(t *testing.T) {
	configs := []config.UpstreamConfig{
		{URL: parseURL("http://backend1:8081"), Weight: 2},
		{URL: parseURL("http://backend2:8082"), Weight: 3},
		{URL: parseURL("http://backend3:8083"), Weight: 1},
	}

	lb, err := NewLoadBalancer(configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				server, err := lb.CheckNextAvailable()
				if err != nil {
					t.Errorf("worker %d: unexpected error: %v", workerID, err)
					return
				}
				if server == nil {
					t.Errorf("worker %d: returned server is nil", workerID)
					return
				}

				if i%10 == 0 {
					server.recordFailure()
				} else if i%5 == 0 {
					server.recordSuccess()
				}
			}
		}(w)
	}

	wg.Wait()
}
