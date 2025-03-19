package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type TransactionHandler struct {
	service domain.TransactionService
	logger  logger.Logger
}

func NewTransactionHandler(service domain.TransactionService, logger logger.Logger) *TransactionHandler {
	return &TransactionHandler{
		service: service,
		logger:  logger,
	}
}

func (h *TransactionHandler) GetTransactionByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		h.logger.Error("ID parametresi eksik", map[string]interface{}{})
		http.Error(w, "ID parametresi eksik", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Error("Geçersiz ID formatı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz ID formatı", http.StatusBadRequest)
		return
	}

	transaction, err := h.service.GetTransactionByID(id)
	if err != nil {
		h.logger.Error("İşlem bulunamadı", map[string]interface{}{"id": id, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transaction)
}

func (h *TransactionHandler) GetUserTransactions(w http.ResponseWriter, r *http.Request) {
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

	transactions, err := h.service.GetUserTransactions(userID)
	if err != nil {
		h.logger.Error("Kullanıcı işlemleri alınamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}

type DepositRequest struct {
	UserID int64   `json:"user_id"`
	Amount float64 `json:"amount"`
}

func (h *TransactionHandler) DepositFunds(w http.ResponseWriter, r *http.Request) {
	var req DepositRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("İstek gövdesi decode edilemedi", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz istek gövdesi", http.StatusBadRequest)
		return
	}

	if req.UserID <= 0 {
		h.logger.Error("Geçersiz kullanıcı ID'si", map[string]interface{}{"user_id": req.UserID})
		http.Error(w, "Geçersiz kullanıcı ID'si", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		h.logger.Error("Geçersiz miktar", map[string]interface{}{"amount": req.Amount})
		http.Error(w, "Geçersiz miktar. Pozitif bir değer girilmeli", http.StatusBadRequest)
		return
	}

	transaction, err := h.service.DepositFunds(req.UserID, req.Amount)
	if err != nil {
		h.logger.Error("Para yatırma işlemi başarısız", map[string]interface{}{"user_id": req.UserID, "amount": req.Amount, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
}

type WithdrawRequest struct {
	UserID int64   `json:"user_id"`
	Amount float64 `json:"amount"`
}

func (h *TransactionHandler) WithdrawFunds(w http.ResponseWriter, r *http.Request) {
	var req WithdrawRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("İstek gövdesi decode edilemedi", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz istek gövdesi", http.StatusBadRequest)
		return
	}

	if req.UserID <= 0 {
		h.logger.Error("Geçersiz kullanıcı ID'si", map[string]interface{}{"user_id": req.UserID})
		http.Error(w, "Geçersiz kullanıcı ID'si", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		h.logger.Error("Geçersiz miktar", map[string]interface{}{"amount": req.Amount})
		http.Error(w, "Geçersiz miktar. Pozitif bir değer girilmeli", http.StatusBadRequest)
		return
	}

	transaction, err := h.service.WithdrawFunds(req.UserID, req.Amount)
	if err != nil {
		h.logger.Error("Para çekme işlemi başarısız", map[string]interface{}{"user_id": req.UserID, "amount": req.Amount, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
}

type TransferRequest struct {
	FromUserID int64   `json:"from_user_id"`
	ToUserID   int64   `json:"to_user_id"`
	Amount     float64 `json:"amount"`
}

func (h *TransactionHandler) TransferFunds(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("İstek gövdesi decode edilemedi", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz istek gövdesi", http.StatusBadRequest)
		return
	}

	if req.FromUserID <= 0 || req.ToUserID <= 0 {
		h.logger.Error("Geçersiz kullanıcı ID'si", map[string]interface{}{"from_user_id": req.FromUserID, "to_user_id": req.ToUserID})
		http.Error(w, "Geçersiz kullanıcı ID'si", http.StatusBadRequest)
		return
	}

	if req.FromUserID == req.ToUserID {
		h.logger.Error("Aynı hesaba transfer yapılamaz", map[string]interface{}{"user_id": req.FromUserID})
		http.Error(w, "Aynı hesaba transfer yapılamaz", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		h.logger.Error("Geçersiz miktar", map[string]interface{}{"amount": req.Amount})
		http.Error(w, "Geçersiz miktar. Pozitif bir değer girilmeli", http.StatusBadRequest)
		return
	}

	transaction, err := h.service.TransferFunds(req.FromUserID, req.ToUserID, req.Amount)
	if err != nil {
		h.logger.Error("Transfer işlemi başarısız", map[string]interface{}{
			"from_user_id": req.FromUserID,
			"to_user_id":   req.ToUserID,
			"amount":       req.Amount,
			"error":        err.Error(),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
}

func (h *TransactionHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/transactions", h.GetTransactionByID)
	mux.HandleFunc("GET /api/user-transactions", h.GetUserTransactions)
	mux.HandleFunc("POST /api/transactions/deposit", h.DepositFunds)
	mux.HandleFunc("POST /api/transactions/withdraw", h.WithdrawFunds)
	mux.HandleFunc("POST /api/transactions/transfer", h.TransferFunds)
}
