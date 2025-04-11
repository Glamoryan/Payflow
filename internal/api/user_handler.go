package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type UserHandler struct {
	service domain.UserService
	logger  logger.Logger
}

func NewUserHandler(service domain.UserService, logger logger.Logger) *UserHandler {
	return &UserHandler{
		service: service,
		logger:  logger,
	}
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("İstek gövdesi decode edilemedi", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz istek gövdesi", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		h.logger.Error("Eksik parametreler", map[string]interface{}{})
		http.Error(w, "Kullanıcı adı, e-posta ve şifre gereklidir", http.StatusBadRequest)
		return
	}

	passwordHash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.Password)))

	user := &domain.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         req.Role,
	}

	if err := h.service.CreateUser(user); err != nil {
		h.logger.Error("Kullanıcı oluşturulamadı", map[string]interface{}{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
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

	user, err := h.service.GetUserByID(id)
	if err != nil {
		h.logger.Error("Kullanıcı bulunamadı", map[string]interface{}{"id": id, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var user domain.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		h.logger.Error("İstek gövdesi decode edilemedi", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz istek gövdesi", http.StatusBadRequest)
		return
	}

	if user.ID == 0 {
		h.logger.Error("Kullanıcı ID'si eksik", map[string]interface{}{})
		http.Error(w, "Kullanıcı ID'si eksik", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateUser(&user); err != nil {
		h.logger.Error("Kullanıcı güncelleme hatası", map[string]interface{}{"id": user.ID, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.DeleteUser(id); err != nil {
		h.logger.Error("Kullanıcı silme hatası", map[string]interface{}{"id": id, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/users", h.GetUserByID)
	mux.HandleFunc("POST /api/users", h.CreateUser)
	mux.HandleFunc("PUT /api/users", h.UpdateUser)
	mux.HandleFunc("DELETE /api/users", h.DeleteUser)
	mux.HandleFunc("POST /api/login", h.Login)
	mux.HandleFunc("POST /api/users/api-key", h.GenerateApiKey)
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	ApiKey   string `json:"api_key"`
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("İstek gövdesi decode edilemedi", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz istek gövdesi", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		h.logger.Error("Eksik parametreler", map[string]interface{}{})
		http.Error(w, "Kullanıcı adı ve şifre gereklidir", http.StatusBadRequest)
		return
	}

	apiKey, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		h.logger.Error("Giriş başarısız", map[string]interface{}{"username": req.Username, "error": err.Error()})
		http.Error(w, "Geçersiz kullanıcı adı veya şifre", http.StatusUnauthorized)
		return
	}

	user, err := h.service.GetUserByUsername(req.Username)
	if err != nil {
		h.logger.Error("Kullanıcı bilgileri alınamadı", map[string]interface{}{"username": req.Username, "error": err.Error()})
		http.Error(w, "Sunucu hatası", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		ApiKey:   apiKey,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *UserHandler) GenerateApiKey(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("X-API-Key")
	if authHeader == "" {
		h.logger.Error("API anahtarı eksik", map[string]interface{}{})
		http.Error(w, "Yetkilendirme gerekli", http.StatusUnauthorized)
		return
	}

	user, err := h.service.GetUserByApiKey(authHeader)
	if err != nil {
		h.logger.Error("API anahtarı geçersiz", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz API anahtarı", http.StatusUnauthorized)
		return
	}

	apiKey, err := h.service.GenerateApiKey(user.ID)
	if err != nil {
		h.logger.Error("API anahtarı oluşturulamadı", map[string]interface{}{"user_id": user.ID, "error": err.Error()})
		http.Error(w, "API anahtarı oluşturulamadı", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"api_key": apiKey,
		"message": "API anahtarı başarıyla yenilendi",
	})
}
