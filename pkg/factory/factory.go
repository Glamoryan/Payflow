package factory

import (
	"database/sql"
	"fmt"

	"payflow/internal/config"
	"payflow/internal/domain"
	"payflow/internal/repository"
	"payflow/internal/service"
	"payflow/pkg/logger"
)

type Factory interface {
	GetLogger() logger.Logger
	GetConfig() *config.Config
	GetDB() *sql.DB

	GetUserRepository() domain.UserRepository
	GetTransactionRepository() domain.TransactionRepository
	GetBalanceRepository() domain.BalanceRepository
	GetAuditLogRepository() domain.AuditLogRepository

	GetUserService() domain.UserService
	GetTransactionService() domain.TransactionService
	GetBalanceService() domain.BalanceService
	GetAuditLogService() domain.AuditLogService
}

type AppFactory struct {
	config *config.Config
	logger logger.Logger
	db     *sql.DB

	userRepository        domain.UserRepository
	transactionRepository domain.TransactionRepository
	balanceRepository     domain.BalanceRepository
	auditLogRepository    domain.AuditLogRepository

	userService        domain.UserService
	transactionService domain.TransactionService
	balanceService     domain.BalanceService
	auditLogService    domain.AuditLogService
}

func NewFactory() (Factory, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	log := logger.New(logger.LogLevel(cfg.LogLevel), nil)

	db, err := sql.Open("sqlite3", "payflow.db")
	if err != nil {
		return nil, fmt.Errorf("veritabanı bağlantısı kurulamadı: %w", err)
	}

	factory := &AppFactory{
		config: cfg,
		logger: log,
		db:     db,
	}

	factory.initRepositories()
	factory.initServices()

	return factory, nil
}

func (f *AppFactory) initRepositories() {
	f.userRepository = repository.NewUserRepository(f.db, f.logger)
	f.transactionRepository = repository.NewTransactionRepository(f.db, f.logger)
	f.balanceRepository = repository.NewBalanceRepository(f.db, f.logger)
	f.auditLogRepository = repository.NewAuditLogRepository(f.db, f.logger)
}

func (f *AppFactory) initServices() {
	f.auditLogService = service.NewAuditLogService(f.auditLogRepository, f.logger)
	f.balanceService = service.NewBalanceService(f.balanceRepository, f.auditLogRepository, f.logger)
	f.userService = service.NewUserService(f.userRepository, f.balanceService, f.auditLogRepository, f.logger)
	f.transactionService = service.NewTransactionService(
		f.transactionRepository,
		f.balanceRepository,
		f.balanceService,
		f.auditLogRepository,
		f.logger,
	)
}

func (f *AppFactory) GetLogger() logger.Logger {
	return f.logger
}

func (f *AppFactory) GetConfig() *config.Config {
	return f.config
}

func (f *AppFactory) GetDB() *sql.DB {
	return f.db
}

func (f *AppFactory) GetUserRepository() domain.UserRepository {
	return f.userRepository
}

func (f *AppFactory) GetTransactionRepository() domain.TransactionRepository {
	return f.transactionRepository
}

func (f *AppFactory) GetBalanceRepository() domain.BalanceRepository {
	return f.balanceRepository
}

func (f *AppFactory) GetAuditLogRepository() domain.AuditLogRepository {
	return f.auditLogRepository
}

func (f *AppFactory) GetUserService() domain.UserService {
	return f.userService
}

func (f *AppFactory) GetTransactionService() domain.TransactionService {
	return f.transactionService
}

func (f *AppFactory) GetBalanceService() domain.BalanceService {
	return f.balanceService
}

func (f *AppFactory) GetAuditLogService() domain.AuditLogService {
	return f.auditLogService
}
