package loadbalancer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"payflow/internal/config"
	"payflow/pkg/circuitbreaker"
	"payflow/pkg/logger"
)

type LoadBalancer struct {
	backends  []*Backend
	algorithm Algorithm
	logger    logger.Logger
	mutex     sync.RWMutex

	current uint64

	healthCheckInterval time.Duration
	healthCheckPath     string

	circuitBreakers map[string]*circuitbreaker.CircuitBreaker
}

type Backend struct {
	URL          *url.URL
	Alive        bool
	Weight       int
	Connections  int64
	ReverseProxy *httputil.ReverseProxy
	mutex        sync.RWMutex
}

type Algorithm string

const (
	RoundRobin         Algorithm = "round_robin"
	WeightedRoundRobin Algorithm = "weighted_round_robin"
	LeastConnections   Algorithm = "least_connections"
	IPHash             Algorithm = "ip_hash"
)

func NewLoadBalancer(cfg config.LoadBalancerConfig, logger logger.Logger) *LoadBalancer {
	lb := &LoadBalancer{
		algorithm:           Algorithm(cfg.Algorithm),
		logger:              logger,
		healthCheckInterval: time.Duration(cfg.HealthCheckInterval) * time.Second,
		healthCheckPath:     cfg.HealthCheckPath,
		circuitBreakers:     make(map[string]*circuitbreaker.CircuitBreaker),
	}

	return lb
}

func (lb *LoadBalancer) AddBackend(urlStr string, weight int) error {
	serverURL, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	backend := &Backend{
		URL:          serverURL,
		Alive:        true,
		Weight:       weight,
		ReverseProxy: httputil.NewSingleHostReverseProxy(serverURL),
	}

	cb := circuitbreaker.New(circuitbreaker.Settings{
		Name:        fmt.Sprintf("backend-%s", serverURL.Host),
		MaxRequests: 3,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			return counts.ConsecutiveFailures > 2
		},
		OnStateChange: func(name string, from circuitbreaker.State, to circuitbreaker.State) {
			lb.logger.InfoContext(context.Background(), "Backend circuit breaker state changed", map[string]interface{}{
				"backend": name,
				"from":    from.String(),
				"to":      to.String(),
			})
		},
	})

	lb.mutex.Lock()
	lb.backends = append(lb.backends, backend)
	lb.circuitBreakers[serverURL.Host] = cb
	lb.mutex.Unlock()

	lb.logger.InfoContext(context.Background(), "Backend added", map[string]interface{}{
		"url":    urlStr,
		"weight": weight,
	})

	return nil
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := lb.getNextBackend(r)
	if backend == nil {
		http.Error(w, "Hiçbir sağlıklı backend bulunamadı", http.StatusServiceUnavailable)
		return
	}

	atomic.AddInt64(&backend.Connections, 1)
	defer atomic.AddInt64(&backend.Connections, -1)

	cb := lb.circuitBreakers[backend.URL.Host]

	_, err := cb.Execute(func() (interface{}, error) {
		backend.ReverseProxy.ServeHTTP(w, r)
		return nil, nil
	})

	if err != nil {
		if err == circuitbreaker.ErrCircuitBreakerOpen {
			backend.setAlive(false)
			lb.logger.Error("Backend marked unhealthy due to circuit breaker", map[string]interface{}{
				"backend": backend.URL.Host,
			})
		}
		http.Error(w, "Backend unavailable", http.StatusBadGateway)
	}
}

func (lb *LoadBalancer) getNextBackend(r *http.Request) *Backend {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	aliveBackends := lb.getAliveBackends()
	if len(aliveBackends) == 0 {
		return nil
	}

	switch lb.algorithm {
	case RoundRobin:
		return lb.roundRobin(aliveBackends)
	case WeightedRoundRobin:
		return lb.weightedRoundRobin(aliveBackends)
	case LeastConnections:
		return lb.leastConnections(aliveBackends)
	case IPHash:
		return lb.ipHash(aliveBackends, r)
	default:
		return lb.roundRobin(aliveBackends)
	}
}

func (lb *LoadBalancer) getAliveBackends() []*Backend {
	var aliveBackends []*Backend
	for _, backend := range lb.backends {
		if backend.isAlive() {
			aliveBackends = append(aliveBackends, backend)
		}
	}
	return aliveBackends
}

func (lb *LoadBalancer) roundRobin(backends []*Backend) *Backend {
	next := atomic.AddUint64(&lb.current, 1)
	return backends[(next-1)%uint64(len(backends))]
}

func (lb *LoadBalancer) weightedRoundRobin(backends []*Backend) *Backend {
	totalWeight := 0
	for _, backend := range backends {
		totalWeight += backend.Weight
	}

	if totalWeight == 0 {
		return lb.roundRobin(backends)
	}

	next := atomic.AddUint64(&lb.current, 1)
	target := int(next) % totalWeight

	currentWeight := 0
	for _, backend := range backends {
		currentWeight += backend.Weight
		if target < currentWeight {
			return backend
		}
	}

	return backends[0]
}

func (lb *LoadBalancer) leastConnections(backends []*Backend) *Backend {
	var selected *Backend
	minConnections := int64(^uint64(0) >> 1)

	for _, backend := range backends {
		connections := atomic.LoadInt64(&backend.Connections)
		if connections < minConnections {
			minConnections = connections
			selected = backend
		}
	}

	return selected
}

func (lb *LoadBalancer) ipHash(backends []*Backend, r *http.Request) *Backend {
	hash := hashString(getClientIP(r))
	return backends[hash%uint32(len(backends))]
}

func (lb *LoadBalancer) StartHealthCheck() {
	if lb.healthCheckInterval <= 0 {
		return
	}

	ticker := time.NewTicker(lb.healthCheckInterval)
	go func() {
		for range ticker.C {
			lb.checkBackendsHealth()
		}
	}()

	lb.logger.InfoContext(context.Background(), "Health check started", map[string]interface{}{
		"interval": lb.healthCheckInterval,
		"path":     lb.healthCheckPath,
	})
}

func (lb *LoadBalancer) checkBackendsHealth() {
	for _, backend := range lb.backends {
		go lb.checkBackendHealth(backend)
	}
}

func (lb *LoadBalancer) checkBackendHealth(backend *Backend) {
	timeout := 5 * time.Second
	client := &http.Client{Timeout: timeout}

	url := fmt.Sprintf("%s%s", backend.URL.String(), lb.healthCheckPath)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		backend.setAlive(false)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		backend.setAlive(false)
		lb.logger.Error("Backend health check failed", map[string]interface{}{
			"backend": backend.URL.Host,
			"error":   err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	alive := resp.StatusCode >= 200 && resp.StatusCode < 300
	backend.setAlive(alive)

	if !alive {
		lb.logger.Error("Backend health check failed", map[string]interface{}{
			"backend": backend.URL.Host,
			"status":  resp.StatusCode,
		})
	}
}

func (b *Backend) isAlive() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.Alive
}

func (b *Backend) setAlive(alive bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.Alive = alive
}

func (lb *LoadBalancer) GetStats() map[string]interface{} {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	stats := map[string]interface{}{
		"algorithm":       string(lb.algorithm),
		"total_backends":  len(lb.backends),
		"alive_backends":  len(lb.getAliveBackends()),
		"current_counter": atomic.LoadUint64(&lb.current),
	}

	backendStats := make([]map[string]interface{}, len(lb.backends))
	for i, backend := range lb.backends {
		cb := lb.circuitBreakers[backend.URL.Host]
		backendStats[i] = map[string]interface{}{
			"url":                    backend.URL.String(),
			"alive":                  backend.isAlive(),
			"weight":                 backend.Weight,
			"connections":            atomic.LoadInt64(&backend.Connections),
			"circuit_breaker":        cb.State().String(),
			"circuit_breaker_counts": cb.Counts(),
		}
	}
	stats["backends"] = backendStats

	return stats
}

func hashString(s string) uint32 {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}

func getClientIP(r *http.Request) string {
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		return xForwardedFor
	}

	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	return r.RemoteAddr
}
