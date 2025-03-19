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
	query := `SELECT id, username, email, password_hash, role, created_at, updated_at FROM users WHERE id = $1`

	var user domain.User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
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
	query := `SELECT id, username, email, password_hash, role, created_at, updated_at FROM users WHERE username = $1`

	var user domain.User
	err := r.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
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
	query := `SELECT id, username, email, password_hash, role, created_at, updated_at FROM users WHERE email = $1`

	var user domain.User
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Kullanıcı e-posta adresine göre bulunamadı", map[string]interface{}{"email": email, "error": err.Error()})
		return nil, fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) Create(user *domain.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
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
		SET username = $1, email = $2, password_hash = $3, role = $4, updated_at = $5
		WHERE id = $6
	`

	user.UpdatedAt = time.Now()

	_, err := r.db.Exec(
		query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
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
	query := `DELETE FROM users WHERE id = $1`

	_, err := r.db.Exec(query, id)

	if err != nil {
		r.logger.Error("Kullanıcı silinemedi", map[string]interface{}{"id": id, "error": err.Error()})
		return fmt.Errorf("kullanıcı silinemedi: %w", err)
	}

	return nil
}
