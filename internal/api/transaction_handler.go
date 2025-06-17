package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type TransactionHandler struct {
	service     domain.TransactionService
	userService domain.UserService
	logger      logger.Logger
}

func NewTransactionHandler(service domain.TransactionService, userService domain.UserService, logger logger.Logger) *TransactionHandler {
	return &TransactionHandler{
		service:     service,
		userService: userService,
		logger:      logger,
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

func (h *TransactionHandler) GetWorkerPoolStats(w http.ResponseWriter, r *http.Request) {

	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		h.logger.Error("API anahtarı eksik", map[string]interface{}{})
		http.Error(w, "Yetkilendirme gerekli", http.StatusUnauthorized)
		return
	}

	user, err := h.userService.GetUserByApiKey(apiKey)
	if err != nil {
		h.logger.Error("API anahtarı geçersiz", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz API anahtarı", http.StatusUnauthorized)
		return
	}

	isAdmin, err := h.userService.HasAdminRole(user.ID)
	if err != nil {
		h.logger.Error("Yetki kontrolü yapılamadı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Yetki kontrolü yapılamadı", http.StatusInternalServerError)
		return
	}

	if !isAdmin {
		h.logger.Warn("Yetkisiz erişim", map[string]interface{}{"user_id": user.ID})
		http.Error(w, "Bu işlemi yapmak için admin yetkisi gerekiyor", http.StatusForbidden)
		return
	}

	stats, err := h.service.GetWorkerPoolStats()
	if err != nil {
		h.logger.Error("Worker pool istatistikleri alınamadı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "İstatistikler alınamadı: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *TransactionHandler) RollbackTransaction(w http.ResponseWriter, r *http.Request) {

	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		h.logger.Error("API anahtarı eksik", map[string]interface{}{})
		http.Error(w, "Yetkilendirme gerekli", http.StatusUnauthorized)
		return
	}

	user, err := h.userService.GetUserByApiKey(apiKey)
	if err != nil {
		h.logger.Error("API anahtarı geçersiz", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz API anahtarı", http.StatusUnauthorized)
		return
	}

	isAdmin, err := h.userService.HasAdminRole(user.ID)
	if err != nil {
		h.logger.Error("Yetki kontrolü yapılamadı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Yetki kontrolü yapılamadı", http.StatusInternalServerError)
		return
	}

	if !isAdmin {
		h.logger.Warn("Yetkisiz erişim", map[string]interface{}{"user_id": user.ID})
		http.Error(w, "Bu işlemi yapmak için admin yetkisi gerekiyor", http.StatusForbidden)
		return
	}

	transactionIDStr := r.URL.Query().Get("id")
	if transactionIDStr == "" {
		h.logger.Error("İşlem ID'si eksik", map[string]interface{}{})
		http.Error(w, "İşlem ID'si gerekli", http.StatusBadRequest)
		return
	}

	transactionID, err := strconv.ParseInt(transactionIDStr, 10, 64)
	if err != nil {
		h.logger.Error("Geçersiz işlem ID'si", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz işlem ID'si", http.StatusBadRequest)
		return
	}

	if err := h.service.RollbackTransaction(transactionID); err != nil {
		h.logger.Error("İşlem geri alınamadı", map[string]interface{}{
			"transaction_id": transactionID,
			"error":          err.Error(),
		})
		http.Error(w, "İşlem geri alınamadı: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "success",
		"message":        "İşlem başarıyla geri alındı",
		"transaction_id": transactionID,
	})
}

type BatchTransactionRequest struct {
	Transactions []struct {
		SenderID    int64   `json:"sender_id"`
		ReceiverID  int64   `json:"receiver_id"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	} `json:"transactions"`
}

type BatchTransactionResponse struct {
	Processed int    `json:"processed"`
	Failed    int    `json:"failed"`
	Message   string `json:"message"`
}

func (h *TransactionHandler) ProcessBatchTransactions(w http.ResponseWriter, r *http.Request) {
	var req BatchTransactionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("İstek gövdesi decode edilemedi", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz istek gövdesi", http.StatusBadRequest)
		return
	}

	if len(req.Transactions) == 0 {
		h.logger.Error("Boş işlem listesi", map[string]interface{}{})
		http.Error(w, "İşlem listesi boş olamaz", http.StatusBadRequest)
		return
	}

	transactions := make([]*domain.Transaction, 0, len(req.Transactions))
	for _, t := range req.Transactions {
		if t.SenderID <= 0 || t.ReceiverID <= 0 {
			h.logger.Error("Geçersiz kullanıcı ID'si", map[string]interface{}{"sender_id": t.SenderID, "receiver_id": t.ReceiverID})
			http.Error(w, "Geçersiz kullanıcı ID'si", http.StatusBadRequest)
			return
		}

		if t.Amount <= 0 {
			h.logger.Error("Geçersiz miktar", map[string]interface{}{"amount": t.Amount})
			http.Error(w, "Geçersiz miktar. Pozitif bir değer girilmeli", http.StatusBadRequest)
			return
		}

		senderID := t.SenderID
		receiverID := t.ReceiverID
		transaction := &domain.Transaction{
			FromUserID: &senderID,
			ToUserID:   &receiverID,
			Amount:     t.Amount,
			Type:       domain.TransactionTypeTransfer,
			Status:     domain.TransactionStatusPending,
			CreatedAt:  time.Now(),
		}
		transactions = append(transactions, transaction)
	}

	h.logger.Info("Toplu işlem başlatılıyor", map[string]interface{}{"count": len(transactions)})
	processed, failed, err := h.service.ProcessBatchTransactions(transactions)

	var response BatchTransactionResponse
	if err != nil {
		h.logger.Error("Toplu işlem başarısız", map[string]interface{}{"error": err.Error()})
		response = BatchTransactionResponse{
			Processed: processed,
			Failed:    failed,
			Message:   "Bazı işlemler başarısız oldu: " + err.Error(),
		}
	} else {
		h.logger.Info("Toplu işlem tamamlandı", map[string]interface{}{"processed": processed, "failed": failed})
		response = BatchTransactionResponse{
			Processed: processed,
			Failed:    failed,
			Message:   "İşlem başarıyla tamamlandı",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if failed > 0 {
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(response)
}

func (h *TransactionHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/transactions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetTransactionByID(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/user-transactions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetUserTransactions(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/transactions/deposit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.DepositFunds(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/transactions/withdraw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.WithdrawFunds(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/transactions/transfer", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.TransferFunds(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/transactions/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.ProcessBatchTransactions(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/transactions/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetWorkerPoolStats(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/transactions/rollback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.RollbackTransaction(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
