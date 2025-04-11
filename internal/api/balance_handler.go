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

	balance, err := h.service.GetUserBalance(userID)
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

	balance, err := h.service.GetUserBalance(userID)
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

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	history, err := h.service.GetBalanceHistory(userID, limit, offset)
	if err != nil {
		h.logger.Error("Bakiye geçmişi alınamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (h *BalanceHandler) GetBalanceHistoryByDateRange(w http.ResponseWriter, r *http.Request) {
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

	history, err := h.service.GetBalanceHistoryByDateRange(userID, startDate, endDate)
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

func (h *BalanceHandler) RecalculateBalance(w http.ResponseWriter, r *http.Request) {
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

	balance, err := h.service.RecalculateBalance(userID)
	if err != nil {
		h.logger.Error("Bakiye yeniden hesaplanamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

func (h *BalanceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/balances", h.GetUserBalance)
	mux.HandleFunc("POST /api/balances/initialize", h.InitializeUserBalance)
	mux.HandleFunc("GET /api/balances/history", h.GetBalanceHistory)
	mux.HandleFunc("GET /api/balances/history/date-range", h.GetBalanceHistoryByDateRange)
	mux.HandleFunc("POST /api/balances/recalculate", h.RecalculateBalance)
}
