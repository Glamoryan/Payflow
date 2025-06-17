package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"payflow/pkg/cache"
	"payflow/pkg/logger"
)

type CacheHandler struct {
	cache         cache.Cache
	warmUpManager *cache.WarmUpManager
	logger        logger.Logger
}

type CacheStatsResponse struct {
	CacheType  string                 `json:"cache_type"`
	Uptime     time.Duration          `json:"uptime"`
	TotalKeys  int                    `json:"total_keys"`
	CacheStats map[string]interface{} `json:"cache_stats"`
	Timestamp  time.Time              `json:"timestamp"`
}

type WarmUpRequest struct {
	UserID *int64 `json:"user_id,omitempty"`
	Type   string `json:"type"` // "user", "top_users", "frequent_data"
	Limit  *int   `json:"limit,omitempty"`
}

type CacheInvalidateRequest struct {
	Pattern *string  `json:"pattern,omitempty"`
	Keys    []string `json:"keys,omitempty"`
	UserID  *int64   `json:"user_id,omitempty"`
}

func NewCacheHandler(cache cache.Cache, warmUpManager *cache.WarmUpManager, logger logger.Logger) *CacheHandler {
	return &CacheHandler{
		cache:         cache,
		warmUpManager: warmUpManager,
		logger:        logger,
	}
}

func (h *CacheHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/cache/stats", h.handleCacheStats)
	mux.HandleFunc("/api/cache/warmup", h.handleWarmUp)
	mux.HandleFunc("/api/cache/invalidate", h.handleInvalidate)
	mux.HandleFunc("/api/cache/keys", h.handleKeys)
	mux.HandleFunc("/api/cache/health", h.handleHealth)
}

func (h *CacheHandler) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// Get all keys matching payflow prefix
	keys, err := h.cache.GetKeys(ctx, "*")
	if err != nil {
		h.logger.Error("Cache keys alınamadı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Cache stats could not be retrieved", http.StatusInternalServerError)
		return
	}

	stats := CacheStatsResponse{
		CacheType: "Redis",
		TotalKeys: len(keys),
		CacheStats: map[string]interface{}{
			"user_keys":        countKeysByPrefix(keys, "user:"),
			"balance_keys":     countKeysByPrefix(keys, "balance:"),
			"transaction_keys": countKeysByPrefix(keys, "transaction:"),
			"event_keys":       countKeysByPrefix(keys, "event:"),
			"dashboard_keys":   countKeysByPrefix(keys, "dashboard:"),
		},
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *CacheHandler) handleWarmUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req WarmUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	var err error

	switch req.Type {
	case "user":
		if req.UserID == nil {
			http.Error(w, "user_id is required for user warm-up", http.StatusBadRequest)
			return
		}
		err = h.warmUpManager.WarmUpUserData(ctx, *req.UserID)

	case "top_users":
		limit := 10
		if req.Limit != nil {
			limit = *req.Limit
		}
		err = h.warmUpManager.WarmUpTopUsers(ctx, limit)

	case "frequent_data":
		err = h.warmUpManager.WarmUpFrequentlyAccessedData(ctx)

	default:
		http.Error(w, "Invalid warm-up type. Use: user, top_users, frequent_data", http.StatusBadRequest)
		return
	}

	if err != nil {
		h.logger.Error("Cache warm-up hatası", map[string]interface{}{
			"type":  req.Type,
			"error": err.Error(),
		})
		http.Error(w, fmt.Sprintf("Warm-up failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":    "success",
		"type":      req.Type,
		"timestamp": time.Now(),
	}

	if req.UserID != nil {
		response["user_id"] = *req.UserID
	}
	if req.Limit != nil {
		response["limit"] = *req.Limit
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *CacheHandler) handleInvalidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CacheInvalidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	var err error
	var deletedCount int

	if req.Pattern != nil {
		// Delete by pattern
		keys, getErr := h.cache.GetKeys(ctx, *req.Pattern)
		if getErr != nil {
			http.Error(w, fmt.Sprintf("Error getting keys: %v", getErr), http.StatusInternalServerError)
			return
		}
		deletedCount = len(keys)
		err = h.cache.DeletePattern(ctx, *req.Pattern)

	} else if len(req.Keys) > 0 {
		// Delete specific keys
		deletedCount = len(req.Keys)
		err = h.cache.DeleteMultiple(ctx, req.Keys)

	} else if req.UserID != nil {
		// Delete user-related cache
		err = cache.InvalidateUserCache(ctx, h.cache, *req.UserID)
		if err == nil {
			deletedCount = 5 // Approximate number of user-related keys
		}

	} else {
		http.Error(w, "Either pattern, keys, or user_id must be provided", http.StatusBadRequest)
		return
	}

	if err != nil {
		h.logger.Error("Cache invalidation hatası", map[string]interface{}{"error": err.Error()})
		http.Error(w, fmt.Sprintf("Cache invalidation failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":        "success",
		"deleted_count": deletedCount,
		"timestamp":     time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *CacheHandler) handleKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		pattern = "*"
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100 // Default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	ctx := context.Background()
	keys, err := h.cache.GetKeys(ctx, pattern)
	if err != nil {
		h.logger.Error("Cache keys alınamadı", map[string]interface{}{
			"pattern": pattern,
			"error":   err.Error(),
		})
		http.Error(w, "Error retrieving cache keys", http.StatusInternalServerError)
		return
	}

	// Apply limit
	if len(keys) > limit {
		keys = keys[:limit]
	}

	response := map[string]interface{}{
		"keys":      keys,
		"count":     len(keys),
		"pattern":   pattern,
		"limit":     limit,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *CacheHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	err := h.cache.Ping(ctx)

	response := map[string]interface{}{
		"timestamp": time.Now(),
	}

	if err != nil {
		response["status"] = "unhealthy"
		response["error"] = err.Error()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}

	response["status"] = "healthy"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to count keys by prefix
func countKeysByPrefix(keys []string, prefix string) int {
	count := 0
	for _, key := range keys {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			count++
		}
	}
	return count
}
