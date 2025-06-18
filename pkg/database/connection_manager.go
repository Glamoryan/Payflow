package database

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"payflow/internal/config"
	"payflow/pkg/circuitbreaker"
	"payflow/pkg/logger"

	_ "github.com/lib/pq"
)

type ConnectionManager struct {
	masterDB       *sql.DB
	readDBs        []*ReadReplica
	logger         logger.Logger
	circuitBreaker *circuitbreaker.CircuitBreaker
	mutex          sync.RWMutex

	roundRobinIndex int
}

type ReadReplica struct {
	DB        *sql.DB
	Config    config.ReplicaConfig
	IsHealthy bool
	mutex     sync.RWMutex
}

func NewConnectionManager(cfg *config.Config, logger logger.Logger) (*ConnectionManager, error) {
	cm := &ConnectionManager{
		logger:          logger,
		roundRobinIndex: 0,
	}

	cm.circuitBreaker = circuitbreaker.New(circuitbreaker.Settings{
		Name:        "database",
		MaxRequests: 3,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			return counts.ConsecutiveFailures > 3
		},
		OnStateChange: func(name string, from circuitbreaker.State, to circuitbreaker.State) {
			logger.InfoContext(context.Background(), "Circuit breaker state changed", map[string]interface{}{
				"name": name,
				"from": from.String(),
				"to":   to.String(),
			})
		},
	})

	if err := cm.connectMaster(cfg.Database); err != nil {
		return nil, fmt.Errorf("master veritabanı bağlantısı başarısız: %w", err)
	}

	if err := cm.connectReadReplicas(cfg.Database.ReadReplicas); err != nil {
		logger.Error("Read replica bağlantıları başarısız", map[string]interface{}{"error": err.Error()})
	}

	go cm.startHealthCheck()

	return cm, nil
}

func (cm *ConnectionManager) connectMaster(cfg config.DatabaseConfig) error {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	if err := db.Ping(); err != nil {
		return err
	}

	cm.masterDB = db
	cm.logger.InfoContext(context.Background(), "Master veritabanı bağlantısı başarılı", map[string]interface{}{
		"host": cfg.Host,
		"port": cfg.Port,
	})

	return nil
}

func (cm *ConnectionManager) connectReadReplicas(replicas []config.ReplicaConfig) error {
	cm.readDBs = make([]*ReadReplica, 0, len(replicas))

	for _, replicaCfg := range replicas {
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			replicaCfg.Host, replicaCfg.Port, replicaCfg.User, replicaCfg.Password, replicaCfg.Name, replicaCfg.SSLMode)

		db, err := sql.Open("postgres", dsn)
		if err != nil {
			cm.logger.Error("Read replica bağlantısı başarısız", map[string]interface{}{
				"host":  replicaCfg.Host,
				"port":  replicaCfg.Port,
				"error": err.Error(),
			})
			continue
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(5 * time.Minute)

		replica := &ReadReplica{
			DB:        db,
			Config:    replicaCfg,
			IsHealthy: true,
		}

		if err := db.Ping(); err != nil {
			replica.IsHealthy = false
			cm.logger.Error("Read replica ping başarısız", map[string]interface{}{
				"host":  replicaCfg.Host,
				"port":  replicaCfg.Port,
				"error": err.Error(),
			})
		}

		cm.readDBs = append(cm.readDBs, replica)
		cm.logger.InfoContext(context.Background(), "Read replica bağlantısı eklendi", map[string]interface{}{
			"host":    replicaCfg.Host,
			"port":    replicaCfg.Port,
			"healthy": replica.IsHealthy,
		})
	}

	if len(cm.readDBs) == 0 {
		return fmt.Errorf("hiçbir read replica bağlantısı başarılı olamadı")
	}

	return nil
}

func (cm *ConnectionManager) GetWriteDB() *sql.DB {
	return cm.masterDB
}

func (cm *ConnectionManager) GetReadDB() *sql.DB {
	if replica := cm.getHealthyReplica(); replica != nil {
		return replica.DB
	}

	cm.logger.Warn("Hiçbir sağlıklı read replica yok, master kullanılıyor", nil)
	return cm.masterDB
}

func (cm *ConnectionManager) ExecuteWithCircuitBreaker(operation func() (interface{}, error)) (interface{}, error) {
	return cm.circuitBreaker.Execute(operation)
}

func (cm *ConnectionManager) getHealthyReplica() *ReadReplica {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if len(cm.readDBs) == 0 {
		return nil
	}

	var healthyReplicas []*ReadReplica
	for _, replica := range cm.readDBs {
		replica.mutex.RLock()
		if replica.IsHealthy {
			healthyReplicas = append(healthyReplicas, replica)
		}
		replica.mutex.RUnlock()
	}

	if len(healthyReplicas) == 0 {
		return nil
	}

	return cm.selectByWeight(healthyReplicas)
}

func (cm *ConnectionManager) selectByWeight(replicas []*ReadReplica) *ReadReplica {
	if len(replicas) == 1 {
		return replicas[0]
	}

	totalWeight := 0
	for _, replica := range replicas {
		totalWeight += replica.Config.Weight
	}

	if totalWeight == 0 {
		cm.roundRobinIndex = (cm.roundRobinIndex + 1) % len(replicas)
		return replicas[cm.roundRobinIndex]
	}

	random := rand.Intn(totalWeight)
	currentWeight := 0

	for _, replica := range replicas {
		currentWeight += replica.Config.Weight
		if random < currentWeight {
			return replica
		}
	}

	return replicas[0]
}

func (cm *ConnectionManager) startHealthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		cm.checkReplicaHealth()
	}
}

func (cm *ConnectionManager) checkReplicaHealth() {
	for _, replica := range cm.readDBs {
		go func(r *ReadReplica) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := r.DB.PingContext(ctx)

			r.mutex.Lock()
			wasHealthy := r.IsHealthy
			r.IsHealthy = (err == nil)
			r.mutex.Unlock()

			if wasHealthy != r.IsHealthy {
				cm.logger.InfoContext(context.Background(), "Read replica health status changed", map[string]interface{}{
					"host":    r.Config.Host,
					"port":    r.Config.Port,
					"healthy": r.IsHealthy,
					"error":   err,
				})
			}
		}(replica)
	}
}

func (cm *ConnectionManager) Close() error {
	if cm.masterDB != nil {
		if err := cm.masterDB.Close(); err != nil {
			cm.logger.Error("Master DB kapatma hatası", map[string]interface{}{"error": err.Error()})
		}
	}

	for _, replica := range cm.readDBs {
		if err := replica.DB.Close(); err != nil {
			cm.logger.Error("Read replica kapatma hatası", map[string]interface{}{
				"host":  replica.Config.Host,
				"error": err.Error(),
			})
		}
	}

	return nil
}

func (cm *ConnectionManager) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"circuit_breaker_state":  cm.circuitBreaker.State().String(),
		"circuit_breaker_counts": cm.circuitBreaker.Counts(),
	}

	if cm.masterDB != nil {
		dbStats := cm.masterDB.Stats()
		stats["master"] = map[string]interface{}{
			"open_connections": dbStats.OpenConnections,
			"in_use":           dbStats.InUse,
			"idle":             dbStats.Idle,
		}
	}

	replicaStats := make([]map[string]interface{}, len(cm.readDBs))
	for i, replica := range cm.readDBs {
		replica.mutex.RLock()
		dbStats := replica.DB.Stats()
		replicaStats[i] = map[string]interface{}{
			"host":             replica.Config.Host,
			"port":             replica.Config.Port,
			"healthy":          replica.IsHealthy,
			"weight":           replica.Config.Weight,
			"open_connections": dbStats.OpenConnections,
			"in_use":           dbStats.InUse,
			"idle":             dbStats.Idle,
		}
		replica.mutex.RUnlock()
	}
	stats["replicas"] = replicaStats

	return stats
}
