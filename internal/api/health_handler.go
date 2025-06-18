package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"payflow/pkg/factory"
	"payflow/pkg/logger"
)

type HealthHandler struct {
	factory factory.Factory
	logger  logger.Logger
}

type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Services  map[string]interface{} `json:"services"`
	Version   string                 `json:"version"`
}

func NewHealthHandler(factory factory.Factory, logger logger.Logger) *HealthHandler {
	return &HealthHandler{
		factory: factory,
		logger:  logger,
	}
}

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	services := make(map[string]interface{})

	services["database"] = h.checkDatabaseHealth()

	services["redis"] = h.checkRedisHealth()

	if cm := h.factory.GetConnectionManager(); cm != nil {
		services["connection_manager"] = cm.GetStats()
	}

	if lb := h.factory.GetLoadBalancer(); lb != nil {
		services["load_balancer"] = lb.GetStats()
	}

	if fm := h.factory.GetFallbackManager(); fm != nil {
		services["fallback_manager"] = fm.GetStats()
	}

	services["cache"] = h.checkCacheHealth()

	status := "healthy"
	for _, service := range services {
		if serviceMap, ok := service.(map[string]interface{}); ok {
			if serviceStatus, exists := serviceMap["status"]; exists {
				if serviceStatus != "healthy" {
					status = "degraded"
					break
				}
			}
		}
	}

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Services:  services,
		Version:   "1.0.0",
	}

	if status == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

func (h *HealthHandler) checkDatabaseHealth() map[string]interface{} {
	db := h.factory.GetDB()
	if db == nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  "database connection is nil",
		}
	}

	if err := db.Ping(); err != nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	}

	stats := db.Stats()
	return map[string]interface{}{
		"status":           "healthy",
		"open_connections": stats.OpenConnections,
		"in_use":           stats.InUse,
		"idle":             stats.Idle,
		"wait_count":       stats.WaitCount,
		"wait_duration":    stats.WaitDuration.String(),
	}
}

func (h *HealthHandler) checkRedisHealth() map[string]interface{} {
	client := h.factory.GetRedisClient()
	if client == nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  "redis client is nil",
		}
	}

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	}

	poolStats := client.PoolStats()
	return map[string]interface{}{
		"status":      "healthy",
		"hits":        poolStats.Hits,
		"misses":      poolStats.Misses,
		"timeouts":    poolStats.Timeouts,
		"total_conns": poolStats.TotalConns,
		"idle_conns":  poolStats.IdleConns,
		"stale_conns": poolStats.StaleConns,
	}
}

func (h *HealthHandler) checkCacheHealth() map[string]interface{} {
	cache := h.factory.GetCache()
	if cache == nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  "cache is nil",
		}
	}

	testKey := "health_check_test"
	testValue := "test"
	ctx := context.Background()

	err := cache.Set(ctx, testKey, testValue, time.Minute)
	if err != nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  "cache set failed: " + err.Error(),
		}
	}

	var result interface{}
	err = cache.Get(ctx, testKey, &result)
	if err != nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  "cache get failed: " + err.Error(),
		}
	}

	cache.Delete(ctx, testKey)

	return map[string]interface{}{
		"status": "healthy",
	}
}

func (h *HealthHandler) LivenessCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
	})
}

func (h *HealthHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ready := true
	issues := make([]string, 0)

	if db := h.factory.GetDB(); db != nil {
		if err := db.Ping(); err != nil {
			ready = false
			issues = append(issues, "database: "+err.Error())
		}
	} else {
		ready = false
		issues = append(issues, "database: connection is nil")
	}

	if client := h.factory.GetRedisClient(); client != nil {
		if _, err := client.Ping(r.Context()).Result(); err != nil {
			ready = false
			issues = append(issues, "redis: "+err.Error())
		}
	} else {
		ready = false
		issues = append(issues, "redis: client is nil")
	}

	response := map[string]interface{}{
		"timestamp": time.Now(),
	}

	if ready {
		response["status"] = "ready"
		w.WriteHeader(http.StatusOK)
	} else {
		response["status"] = "not_ready"
		response["issues"] = issues
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}
