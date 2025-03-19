package api

import (
	"encoding/json"
	"net/http"
	"strconv"

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

func (h *BalanceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/balances", h.GetUserBalance)
	mux.HandleFunc("POST /api/balances/initialize", h.InitializeUserBalance)
}
