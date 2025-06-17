package factory

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

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
	GetRedisClient() *redis.Client

	GetUserRepository() domain.UserRepository
	GetTransactionRepository() domain.TransactionRepository
	GetBalanceRepository() domain.BalanceRepository
	GetAuditLogRepository() domain.AuditLogRepository
	GetEventStoreRepository() domain.EventStoreRepository

	GetUserService() domain.UserService
	GetTransactionService() domain.TransactionService
	GetBalanceService() domain.BalanceService
	GetAuditLogService() domain.AuditLogService
	GetEventStoreService() domain.EventStoreService
}

type AppFactory struct {
	config      *config.Config
	logger      logger.Logger
	db          *sql.DB
	redisClient *redis.Client

	userRepository        domain.UserRepository
	transactionRepository domain.TransactionRepository
	balanceRepository     domain.BalanceRepository
	auditLogRepository    domain.AuditLogRepository
	eventStoreRepository  domain.EventStoreRepository

	userService        domain.UserService
	transactionService domain.TransactionService
	balanceService     domain.BalanceService
	auditLogService    domain.AuditLogService
	eventStoreService  domain.EventStoreService
}

func NewFactory() (Factory, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	log := logger.New(logger.LogLevel(cfg.LogLevel), nil)

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("veritabanı bağlantısı kurulamadı: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("veritabanı bağlantısı test edilemedi: %w", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("Redis bağlantısı kurulamadı: %w", err)
	}

	factory := &AppFactory{
		config:      cfg,
		logger:      log,
		db:          db,
		redisClient: redisClient,
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
	f.eventStoreRepository = repository.NewEventStoreRepository(f.db, f.logger)
}

func (f *AppFactory) initServices() {
	f.eventStoreService = service.NewEventStoreService(f.eventStoreRepository, f.logger)

	f.auditLogService = service.NewAuditLogService(f.auditLogRepository, f.logger)
	f.balanceService = service.NewBalanceService(
		f.balanceRepository,
		f.auditLogRepository,
		f.eventStoreService,
		f.logger,
		f.redisClient,
	)
	f.userService = service.NewUserService(f.userRepository, f.balanceService, f.auditLogRepository, f.logger)
	f.transactionService = service.NewTransactionService(
		f.transactionRepository,
		f.balanceRepository,
		f.balanceService,
		f.auditLogRepository,
		f.eventStoreService,
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

func (f *AppFactory) GetRedisClient() *redis.Client {
	return f.redisClient
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

func (f *AppFactory) GetEventStoreRepository() domain.EventStoreRepository {
	return f.eventStoreRepository
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

func (f *AppFactory) GetEventStoreService() domain.EventStoreService {
	return f.eventStoreService
}
