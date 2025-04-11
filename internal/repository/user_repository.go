package repository

import (
	"database/sql"
	"fmt"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type UserRepository struct {
	db     *sql.DB
	logger logger.Logger
}

func NewUserRepository(db *sql.DB, logger logger.Logger) domain.UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

func (r *UserRepository) FindByID(id int64) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, created_at, updated_at
		FROM users
		WHERE id = ?
	`

	var user domain.User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.ApiKey,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Kullanıcı ID'ye göre bulunamadı", map[string]interface{}{"id": id, "error": err.Error()})
		return nil, fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) FindByUsername(username string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, created_at, updated_at
		FROM users
		WHERE username = ?
	`

	var user domain.User
	err := r.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.ApiKey,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Kullanıcı adına göre bulunamadı", map[string]interface{}{"username": username, "error": err.Error()})
		return nil, fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User

	query := `
		SELECT id, username, email, password_hash, role, api_key, created_at, updated_at
		FROM users
		WHERE email = ?
	`

	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.ApiKey,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		r.logger.Error("Kullanıcı bulunamadı", map[string]interface{}{"email": email, "error": err.Error()})
		return nil, fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) FindByApiKey(apiKey string) (*domain.User, error) {
	var user domain.User

	query := `
		SELECT id, username, email, password_hash, role, api_key, created_at, updated_at
		FROM users
		WHERE api_key = ?
	`

	err := r.db.QueryRow(query, apiKey).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.ApiKey,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		r.logger.Error("Kullanıcı bulunamadı", map[string]interface{}{"api_key": apiKey, "error": err.Error()})
		return nil, fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) Create(user *domain.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role, api_key, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	if user.Role == "" {
		user.Role = "user"
	}

	err := r.db.QueryRow(
		query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.ApiKey,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		r.logger.Error("Kullanıcı oluşturulamadı", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("kullanıcı oluşturulamadı: %w", err)
	}

	return nil
}

func (r *UserRepository) Update(user *domain.User) error {
	query := `
		UPDATE users
		SET username = ?, email = ?, password_hash = ?, role = ?, api_key = ?, updated_at = ?
		WHERE id = ?
	`

	user.UpdatedAt = time.Now()

	_, err := r.db.Exec(
		query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.ApiKey,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		r.logger.Error("Kullanıcı güncellenemedi", map[string]interface{}{"id": user.ID, "error": err.Error()})
		return fmt.Errorf("kullanıcı güncellenemedi: %w", err)
	}

	return nil
}

func (r *UserRepository) Delete(id int64) error {
	query := `DELETE FROM users WHERE id = ?`

	_, err := r.db.Exec(query, id)

	if err != nil {
		r.logger.Error("Kullanıcı silinemedi", map[string]interface{}{"id": id, "error": err.Error()})
		return fmt.Errorf("kullanıcı silinemedi: %w", err)
	}

	return nil
}
