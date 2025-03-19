package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type AuditLogHandler struct {
	service domain.AuditLogService
	logger  logger.Logger
}

func NewAuditLogHandler(service domain.AuditLogService, logger logger.Logger) *AuditLogHandler {
	return &AuditLogHandler{
		service: service,
		logger:  logger,
	}
}

func (h *AuditLogHandler) GetAllLogs(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	page := 1
	pageSize := 50

	var err error
	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			h.logger.Error("Geçersiz sayfa numarası", map[string]interface{}{"page": pageStr})
			http.Error(w, "Geçersiz sayfa numarası", http.StatusBadRequest)
			return
		}
	}

	if pageSizeStr != "" {
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 || pageSize > 100 {
			h.logger.Error("Geçersiz sayfa boyutu", map[string]interface{}{"page_size": pageSizeStr})
			http.Error(w, "Geçersiz sayfa boyutu. 1-100 arası bir değer olmalı", http.StatusBadRequest)
			return
		}
	}

	logs, err := h.service.GetAllLogs(page, pageSize)
	if err != nil {
		h.logger.Error("Denetim günlükleri alınamadı", map[string]interface{}{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (h *AuditLogHandler) GetEntityLogs(w http.ResponseWriter, r *http.Request) {
	entityTypeStr := r.URL.Query().Get("entity_type")
	entityIDStr := r.URL.Query().Get("entity_id")

	if entityTypeStr == "" {
		h.logger.Error("entity_type parametresi eksik", map[string]interface{}{})
		http.Error(w, "entity_type parametresi eksik", http.StatusBadRequest)
		return
	}

	if entityIDStr == "" {
		h.logger.Error("entity_id parametresi eksik", map[string]interface{}{})
		http.Error(w, "entity_id parametresi eksik", http.StatusBadRequest)
		return
	}

	entityType := domain.EntityType(entityTypeStr)
	switch entityType {
	case domain.EntityTypeUser, domain.EntityTypeTransaction, domain.EntityTypeBalance:
	default:
		h.logger.Error("Geçersiz entity_type", map[string]interface{}{"entity_type": entityTypeStr})
		http.Error(w, "Geçersiz entity_type. Geçerli değerler: user, transaction, balance", http.StatusBadRequest)
		return
	}

	entityID, err := strconv.ParseInt(entityIDStr, 10, 64)
	if err != nil {
		h.logger.Error("Geçersiz entity_id formatı", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz entity_id formatı", http.StatusBadRequest)
		return
	}

	logs, err := h.service.GetEntityLogs(entityType, entityID)
	if err != nil {
		h.logger.Error("Varlık denetim günlükleri alınamadı", map[string]interface{}{
			"entity_type": entityType,
			"entity_id":   entityID,
			"error":       err.Error(),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

type LogActionRequest struct {
	EntityType domain.EntityType `json:"entity_type"`
	EntityID   int64             `json:"entity_id"`
	Action     domain.ActionType `json:"action"`
	Details    string            `json:"details"`
}

func (h *AuditLogHandler) LogAction(w http.ResponseWriter, r *http.Request) {
	var req LogActionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("İstek gövdesi decode edilemedi", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Geçersiz istek gövdesi", http.StatusBadRequest)
		return
	}

	switch req.EntityType {
	case domain.EntityTypeUser, domain.EntityTypeTransaction, domain.EntityTypeBalance:
	default:
		h.logger.Error("Geçersiz entity_type", map[string]interface{}{"entity_type": req.EntityType})
		http.Error(w, "Geçersiz entity_type. Geçerli değerler: user, transaction, balance", http.StatusBadRequest)
		return
	}

	switch req.Action {
	case domain.ActionTypeCreate, domain.ActionTypeUpdate, domain.ActionTypeDelete:
	default:
		h.logger.Error("Geçersiz action", map[string]interface{}{"action": req.Action})
		http.Error(w, "Geçersiz action. Geçerli değerler: create, update, delete", http.StatusBadRequest)
		return
	}

	if req.EntityID <= 0 {
		h.logger.Error("Geçersiz entity_id", map[string]interface{}{"entity_id": req.EntityID})
		http.Error(w, "Geçersiz entity_id", http.StatusBadRequest)
		return
	}

	err := h.service.LogAction(req.EntityType, req.EntityID, req.Action, req.Details)
	if err != nil {
		h.logger.Error("Denetim günlüğü eklenemedi", map[string]interface{}{
			"entity_type": req.EntityType,
			"entity_id":   req.EntityID,
			"action":      req.Action,
			"error":       err.Error(),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *AuditLogHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/audit-logs", h.GetAllLogs)
	mux.HandleFunc("GET /api/entity-logs", h.GetEntityLogs)
	mux.HandleFunc("POST /api/audit-logs", h.LogAction)
}
