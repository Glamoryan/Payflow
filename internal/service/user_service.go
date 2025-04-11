package service

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type UserService struct {
	repo         domain.UserRepository
	balanceSvc   domain.BalanceService
	auditLogRepo domain.AuditLogRepository
	logger       logger.Logger
}

func NewUserService(
	repo domain.UserRepository,
	balanceSvc domain.BalanceService,
	auditLogRepo domain.AuditLogRepository,
	logger logger.Logger,
) domain.UserService {
	return &UserService{
		repo:         repo,
		balanceSvc:   balanceSvc,
		auditLogRepo: auditLogRepo,
		logger:       logger,
	}
}

func (s *UserService) GetUserByID(id int64) (*domain.User, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		s.logger.Error("Kullanıcı ID'ye göre bulunamadı", map[string]interface{}{"id": id, "error": err.Error()})
		return nil, fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("kullanıcı ID'ye göre bulunamadı: %d", id)
	}

	return user, nil
}

func (s *UserService) GetUserByUsername(username string) (*domain.User, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		s.logger.Error("Kullanıcı adına göre bulunamadı", map[string]interface{}{"username": username, "error": err.Error()})
		return nil, fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("kullanıcı adına göre bulunamadı: %s", username)
	}

	return user, nil
}

func (s *UserService) GetUserByEmail(email string) (*domain.User, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		s.logger.Error("Kullanıcı e-posta adresine göre bulunamadı", map[string]interface{}{"email": email, "error": err.Error()})
		return nil, fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("kullanıcı e-posta adresine göre bulunamadı: %s", email)
	}

	return user, nil
}

func (s *UserService) CreateUser(user *domain.User) error {
	existingUser, err := s.repo.FindByEmail(user.Email)
	if err != nil {
		s.logger.Error("E-posta adresi kontrolü sırasında hata oluştu", map[string]interface{}{"email": user.Email, "error": err.Error()})
		return fmt.Errorf("kullanıcı oluşturulamadı: %w", err)
	}

	if existingUser != nil {
		return fmt.Errorf("bu e-posta adresi zaten kullanılıyor: %s", user.Email)
	}

	existingUser, err = s.repo.FindByUsername(user.Username)
	if err != nil {
		s.logger.Error("Kullanıcı adı kontrolü sırasında hata oluştu", map[string]interface{}{"username": user.Username, "error": err.Error()})
		return fmt.Errorf("kullanıcı oluşturulamadı: %w", err)
	}

	if existingUser != nil {
		return fmt.Errorf("bu kullanıcı adı zaten kullanılıyor: %s", user.Username)
	}

	if err := s.repo.Create(user); err != nil {
		s.logger.Error("Kullanıcı oluşturma sırasında hata oluştu", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("kullanıcı oluşturulamadı: %w", err)
	}

	if err := s.balanceSvc.InitializeBalance(user.ID); err != nil {
		s.logger.Error("Bakiye başlatılamadı", map[string]interface{}{"user_id": user.ID, "error": err.Error()})
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeUser,
		EntityID:   user.ID,
		Action:     domain.ActionTypeCreate,
		Details:    fmt.Sprintf("Kullanıcı oluşturuldu: %s", user.Username),
		CreatedAt:  time.Now(),
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"user_id": user.ID, "error": err.Error()})
	}

	return nil
}

func (s *UserService) UpdateUser(user *domain.User) error {
	existingUser, err := s.repo.FindByID(user.ID)
	if err != nil {
		s.logger.Error("Kullanıcı güncellemesi sırasında hata oluştu", map[string]interface{}{"id": user.ID, "error": err.Error()})
		return fmt.Errorf("kullanıcı güncellenemedi: %w", err)
	}

	if existingUser == nil {
		return fmt.Errorf("güncellenecek kullanıcı bulunamadı: %d", user.ID)
	}

	if existingUser.Email != user.Email {
		emailUser, err := s.repo.FindByEmail(user.Email)
		if err != nil {
			s.logger.Error("E-posta adresi kontrolü sırasında hata oluştu", map[string]interface{}{"email": user.Email, "error": err.Error()})
			return fmt.Errorf("kullanıcı güncellenemedi: %w", err)
		}

		if emailUser != nil {
			return fmt.Errorf("bu e-posta adresi zaten kullanılıyor: %s", user.Email)
		}
	}

	if existingUser.Username != user.Username {
		usernameUser, err := s.repo.FindByUsername(user.Username)
		if err != nil {
			s.logger.Error("Kullanıcı adı kontrolü sırasında hata oluştu", map[string]interface{}{"username": user.Username, "error": err.Error()})
			return fmt.Errorf("kullanıcı güncellenemedi: %w", err)
		}

		if usernameUser != nil {
			return fmt.Errorf("bu kullanıcı adı zaten kullanılıyor: %s", user.Username)
		}
	}

	if err := s.repo.Update(user); err != nil {
		s.logger.Error("Kullanıcı güncelleme sırasında hata oluştu", map[string]interface{}{"id": user.ID, "error": err.Error()})
		return fmt.Errorf("kullanıcı güncellenemedi: %w", err)
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeUser,
		EntityID:   user.ID,
		Action:     domain.ActionTypeUpdate,
		Details:    fmt.Sprintf("Kullanıcı güncellendi: %s", user.Username),
		CreatedAt:  time.Now(),
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"user_id": user.ID, "error": err.Error()})
	}

	return nil
}

func (s *UserService) DeleteUser(id int64) error {
	existingUser, err := s.repo.FindByID(id)
	if err != nil {
		s.logger.Error("Kullanıcı silme sırasında hata oluştu", map[string]interface{}{"id": id, "error": err.Error()})
		return fmt.Errorf("kullanıcı silinemedi: %w", err)
	}

	if existingUser == nil {
		return fmt.Errorf("silinecek kullanıcı bulunamadı: %d", id)
	}

	if err := s.repo.Delete(id); err != nil {
		s.logger.Error("Kullanıcı silme sırasında hata oluştu", map[string]interface{}{"id": id, "error": err.Error()})
		return fmt.Errorf("kullanıcı silinemedi: %w", err)
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeUser,
		EntityID:   id,
		Action:     domain.ActionTypeDelete,
		Details:    fmt.Sprintf("Kullanıcı silindi: %s", existingUser.Username),
		CreatedAt:  time.Now(),
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"user_id": id, "error": err.Error()})
	}

	return nil
}

func (s *UserService) HasAdminRole(userID int64) (bool, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return false, fmt.Errorf("yetki kontrolü yapılamadı: %w", err)
	}

	return user.Role == domain.UserRoleAdmin, nil
}

func (s *UserService) CheckPermission(userID int64, requiredRole string) (bool, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return false, fmt.Errorf("yetki kontrolü yapılamadı: %w", err)
	}

	if user.Role == domain.UserRoleAdmin {
		return true, nil
	}

	if requiredRole == domain.UserRoleAdmin && user.Role != domain.UserRoleAdmin {
		return false, nil
	}
	return user.Role == requiredRole, nil
}

func (s *UserService) GenerateApiKey(userID int64) (string, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("API anahtarı oluşturulamadı: %w", err)
	}

	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("API anahtarı oluşturulamadı: %w", err)
	}

	apiKey := fmt.Sprintf("%x", b)

	user.ApiKey = apiKey
	user.UpdatedAt = time.Now()

	if err := s.repo.Update(user); err != nil {
		return "", fmt.Errorf("API anahtarı kaydedilemedi: %w", err)
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeUser,
		EntityID:   userID,
		Action:     domain.ActionTypeUpdate,
		Details:    "API anahtarı yenilendi",
		CreatedAt:  time.Now(),
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
	}

	return apiKey, nil
}

func (s *UserService) GetUserByApiKey(apiKey string) (*domain.User, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API anahtarı boş olamaz")
	}

	user, err := s.repo.FindByApiKey(apiKey)
	if err != nil {
		s.logger.Error("API anahtarı ile kullanıcı bulunamadı", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("geçersiz API anahtarı")
	}

	return user, nil
}

func (s *UserService) Login(username, password string) (string, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return "", fmt.Errorf("giriş yapılamadı: %w", err)
	}

	if user == nil {
		return "", fmt.Errorf("geçersiz kullanıcı adı veya şifre")
	}

	passwordHash := fmt.Sprintf("%x", sha256.Sum256([]byte(password)))

	if user.PasswordHash != passwordHash {
		s.logger.Error("Şifre eşleşmiyor", map[string]interface{}{"username": username})
		return "", fmt.Errorf("geçersiz kullanıcı adı veya şifre")
	}

	if user.ApiKey == "" {
		apiKey, err := s.GenerateApiKey(user.ID)
		if err != nil {
			return "", fmt.Errorf("API anahtarı oluşturulamadı: %w", err)
		}
		return apiKey, nil
	}

	return user.ApiKey, nil
}
