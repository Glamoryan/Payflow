package domain

import "time"

const (
	UserRoleAdmin = "admin"
	UserRoleUser  = "user"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	ApiKey       string    `json:"api_key,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserRepository interface {
	FindByID(id int64) (*User, error)
	FindByUsername(username string) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByApiKey(apiKey string) (*User, error)
	Create(user *User) error
	Update(user *User) error
	Delete(id int64) error
}

type UserService interface {
	GetUserByID(id int64) (*User, error)
	GetUserByUsername(username string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByApiKey(apiKey string) (*User, error)
	CreateUser(user *User) error
	UpdateUser(user *User) error
	DeleteUser(id int64) error
	GenerateApiKey(userID int64) (string, error)

	HasAdminRole(userID int64) (bool, error)
	CheckPermission(userID int64, requiredRole string) (bool, error)

	Login(username, password string) (string, error)
}
