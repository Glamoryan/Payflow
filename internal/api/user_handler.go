package api

import (
	"encoding/json"
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

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user domain.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		h.logger.Error("İstek gövdesi decode edilemedi", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz istek gövdesi", http.StatusBadRequest)
		return
	}

	if err := h.service.CreateUser(&user); err != nil {
		h.logger.Error("Kullanıcı oluşturma hatası", map[string]interface{}{"error": err.Error()})
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
	mux.HandleFunc("POST /api/users", h.CreateUser)
	mux.HandleFunc("GET /api/users", h.GetUserByID)
	mux.HandleFunc("PUT /api/users", h.UpdateUser)
	mux.HandleFunc("DELETE /api/users", h.DeleteUser)
}
