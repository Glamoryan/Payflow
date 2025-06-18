package factory

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"payflow/internal/config"
	"payflow/internal/domain"
	"payflow/internal/repository"
	"payflow/internal/service"
	"payflow/pkg/cache"
	"payflow/pkg/database"
	"payflow/pkg/fallback"
	"payflow/pkg/loadbalancer"
	"payflow/pkg/logger"
)

type Factory interface {
	GetLogger() logger.Logger
	GetConfig() *config.Config
	GetDB() *sql.DB
	GetConnectionManager() *database.ConnectionManager
	GetRedisClient() *redis.Client
	GetCache() cache.Cache
	GetCacheManager() cache.CacheStrategy
	GetWarmUpManager() *cache.WarmUpManager
	GetFallbackManager() *fallback.FallbackManager
	GetLoadBalancer() *loadbalancer.LoadBalancer

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
	config            *config.Config
	logger            logger.Logger
	db                *sql.DB
	connectionManager *database.ConnectionManager
	redisClient       *redis.Client
	cache             cache.Cache
	cacheManager      cache.CacheStrategy
	warmUpManager     *cache.WarmUpManager
	fallbackManager   *fallback.FallbackManager
	loadBalancer      *loadbalancer.LoadBalancer

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

	connManager, err := database.NewConnectionManager(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("connection manager oluşturulamadı: %w", err)
	}

	db := connManager.GetWriteDB()

	redisClient := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})

	ctx := context.Background()
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("Redis bağlantısı kurulamadı: %w", err)
	}

	cacheInstance := cache.NewRedisCache(redisClient, log, "payflow")
	cacheManager := cache.NewCacheManager(cacheInstance, log)

	fallbackMgr := fallback.NewFallbackManager(log)

	var loadBal *loadbalancer.LoadBalancer
	if cfg.Server.LoadBalancer.Enabled {
		loadBal = loadbalancer.NewLoadBalancer(cfg.Server.LoadBalancer, log)
		loadBal.StartHealthCheck()
	}

	factory := &AppFactory{
		config:            cfg,
		logger:            log,
		db:                db,
		connectionManager: connManager,
		redisClient:       redisClient,
		cache:             cacheInstance,
		cacheManager:      cacheManager,
		fallbackManager:   fallbackMgr,
		loadBalancer:      loadBal,
	}

	factory.initRepositories()
	factory.initServices()
	factory.initCacheManagers()
	factory.initFallbacks()

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

	baseBalanceService := service.NewBalanceService(
		f.balanceRepository,
		f.auditLogRepository,
		f.eventStoreService,
		f.logger,
		f.redisClient,
	)
	f.balanceService = service.NewCachedBalanceService(baseBalanceService, f.cache, f.cacheManager, f.logger)

	baseUserService := service.NewUserService(f.userRepository, f.balanceService, f.auditLogRepository, f.logger)
	f.userService = service.NewCachedUserService(baseUserService, f.cache, f.cacheManager, f.logger)

	f.transactionService = service.NewTransactionService(
		f.transactionRepository,
		f.balanceRepository,
		f.balanceService,
		f.auditLogRepository,
		f.eventStoreService,
		f.logger,
	)
}

func (f *AppFactory) initCacheManagers() {
	f.warmUpManager = cache.NewWarmUpManager(
		f.cache,
		f.logger,
		f.userService,
		f.balanceService,
		f.transactionService,
	)
}

func (f *AppFactory) initFallbacks() {
	f.fallbackManager.RegisterFallback(&fallback.FallbackConfig{
		Name: "balance_lookup",
		PrimaryFunc: func(ctx context.Context) (interface{}, error) {
			return nil, nil
		},
		Strategy:     fallback.StrategyCache,
		CacheKey:     "fallback_balance",
		CacheTTL:     5 * time.Minute,
		DefaultValue: 0.0,
	})

	f.fallbackManager.RegisterFallback(&fallback.FallbackConfig{
		Name: "user_lookup",
		PrimaryFunc: func(ctx context.Context) (interface{}, error) {
			return nil, nil
		},
		Strategy:      fallback.StrategyRetry,
		MaxRetries:    3,
		RetryInterval: time.Second,
	})

	f.fallbackManager.RegisterFallback(&fallback.FallbackConfig{
		Name: "transaction_create",
		PrimaryFunc: func(ctx context.Context) (interface{}, error) {
			return nil, nil
		},
		Strategy:     fallback.StrategyDegraded,
		DegradedMode: true,
	})
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

func (f *AppFactory) GetConnectionManager() *database.ConnectionManager {
	return f.connectionManager
}

func (f *AppFactory) GetRedisClient() *redis.Client {
	return f.redisClient
}

func (f *AppFactory) GetCache() cache.Cache {
	return f.cache
}

func (f *AppFactory) GetCacheManager() cache.CacheStrategy {
	return f.cacheManager
}

func (f *AppFactory) GetWarmUpManager() *cache.WarmUpManager {
	return f.warmUpManager
}

func (f *AppFactory) GetFallbackManager() *fallback.FallbackManager {
	return f.fallbackManager
}

func (f *AppFactory) GetLoadBalancer() *loadbalancer.LoadBalancer {
	return f.loadBalancer
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
