package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type BalanceHandler struct {
	service domain.BalanceService
	logger  logger.Logger
}

func NewBalanceHandler(service domain.BalanceService, logger logger.Logger) *BalanceHandler {
	return &BalanceHandler{
		service: service,
		logger:  logger,
	}
}

func (h *BalanceHandler) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		h.logger.Error("user_id parametresi eksik", map[string]interface{}{})
		http.Error(w, "user_id parametresi eksik", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Error("Geçersiz user_id formatı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz user_id formatı", http.StatusBadRequest)
		return
	}

	balance, err := h.service.GetBalance(userID)
	if err != nil {
		h.logger.Error("Bakiye bilgisi alınamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

func (h *BalanceHandler) InitializeUserBalance(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		h.logger.Error("user_id parametresi eksik", map[string]interface{}{})
		http.Error(w, "user_id parametresi eksik", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Error("Geçersiz user_id formatı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz user_id formatı", http.StatusBadRequest)
		return
	}

	err = h.service.InitializeBalance(userID)
	if err != nil {
		h.logger.Error("Bakiye başlatılamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	balance, err := h.service.GetBalance(userID)
	if err != nil {
		h.logger.Error("Bakiye bilgisi alınamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(balance)
}

func (h *BalanceHandler) GetBalanceHistory(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		h.logger.Error("user_id parametresi eksik", map[string]interface{}{})
		http.Error(w, "user_id parametresi eksik", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Error("Geçersiz user_id formatı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz user_id formatı", http.StatusBadRequest)
		return
	}

	startDateStr := r.URL.Query().Get("start_date")
	if startDateStr == "" {
		h.logger.Error("start_date parametresi eksik", map[string]interface{}{})
		http.Error(w, "start_date parametresi eksik", http.StatusBadRequest)
		return
	}

	endDateStr := r.URL.Query().Get("end_date")
	if endDateStr == "" {
		h.logger.Error("end_date parametresi eksik", map[string]interface{}{})
		http.Error(w, "end_date parametresi eksik", http.StatusBadRequest)
		return
	}

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		h.logger.Error("Geçersiz start_date formatı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz start_date formatı. RFC3339 formatında olmalı (örn: 2023-01-01T00:00:00Z)", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		h.logger.Error("Geçersiz end_date formatı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz end_date formatı. RFC3339 formatında olmalı (örn: 2023-01-01T00:00:00Z)", http.StatusBadRequest)
		return
	}

	history, err := h.service.GetBalanceHistory(userID, startDate, endDate)
	if err != nil {
		h.logger.Error("Bakiye geçmişi alınamadı", map[string]interface{}{
			"user_id":    userID,
			"start_date": startDate,
			"end_date":   endDate,
			"error":      err.Error(),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (h *BalanceHandler) ReplayBalanceEvents(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		h.logger.Error("user_id parametresi eksik", map[string]interface{}{})
		http.Error(w, "user_id parametresi eksik", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Error("Geçersiz user_id formatı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz user_id formatı", http.StatusBadRequest)
		return
	}

	err = h.service.ReplayBalanceEvents(userID)
	if err != nil {
		h.logger.Error("Bakiye eventleri tekrar oynatılamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Balance events replayed successfully"})
}

func (h *BalanceHandler) RebuildBalanceState(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		h.logger.Error("user_id parametresi eksik", map[string]interface{}{})
		http.Error(w, "user_id parametresi eksik", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Error("Geçersiz user_id formatı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz user_id formatı", http.StatusBadRequest)
		return
	}

	err = h.service.RebuildBalanceState(userID)
	if err != nil {
		h.logger.Error("Bakiye durumu yeniden oluşturulamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Balance state rebuilt successfully"})
}

func (h *BalanceHandler) RegisterRoutes(mux *http.ServeMux) {
	h.logger.Info("Balance routes register ediliyor...", map[string]interface{}{})

	mux.HandleFunc("/api/balances/initialize", func(w http.ResponseWriter, r *http.Request) {
		h.logger.Info("Initialize route çağrıldı", map[string]interface{}{"method": r.Method, "path": r.URL.Path})
		if r.Method == http.MethodPost {
			h.InitializeUserBalance(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/balances/history", func(w http.ResponseWriter, r *http.Request) {
		h.logger.Info("History route çağrıldı", map[string]interface{}{"method": r.Method, "path": r.URL.Path})
		if r.Method == http.MethodGet {
			h.GetBalanceHistory(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/balances/replay", func(w http.ResponseWriter, r *http.Request) {
		h.logger.Info("Replay route çağrıldı", map[string]interface{}{"method": r.Method, "path": r.URL.Path})
		if r.Method == http.MethodPost {
			h.ReplayBalanceEvents(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/balances/rebuild", func(w http.ResponseWriter, r *http.Request) {
		h.logger.Info("Rebuild route çağrıldı", map[string]interface{}{"method": r.Method, "path": r.URL.Path})
		if r.Method == http.MethodPost {
			h.RebuildBalanceState(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/balances", func(w http.ResponseWriter, r *http.Request) {
		h.logger.Info("Ana balance route çağrıldı", map[string]interface{}{"method": r.Method, "path": r.URL.Path})
		if r.Method == http.MethodGet {
			h.GetUserBalance(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	h.logger.Info("Balance routes başarıyla register edildi", map[string]interface{}{
		"routes": []string{"/api/balances/initialize", "/api/balances/history", "/api/balances/replay", "/api/balances/rebuild", "/api/balances"},
	})
}
