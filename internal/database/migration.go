package database

import (
	"database/sql"
	"fmt"
	"time"

	"payflow/pkg/logger"
)

type Migration struct {
	ID        int64
	Name      string
	AppliedAt time.Time
}

type MigrationService struct {
	db     *sql.DB
	logger logger.Logger
}

func NewMigrationService(db *sql.DB, logger logger.Logger) *MigrationService {
	return &MigrationService{
		db:     db,
		logger: logger,
	}
}

func (m *MigrationService) InitMigrationTable() error {
	query := `
    CREATE TABLE IF NOT EXISTS migrations (
        id SERIAL PRIMARY KEY,
        name TEXT NOT NULL UNIQUE,
        applied_at TIMESTAMP NOT NULL
    )
    `

	_, err := m.db.Exec(query)
	if err != nil {
		m.logger.Error("Migration tablosu oluşturulamadı", map[string]interface{}{"error": err.Error()})
		return err
	}

	return nil
}

func (m *MigrationService) IsMigrationApplied(name string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM migrations WHERE name = $1"
	err := m.db.QueryRow(query, name).Scan(&count)
	if err != nil {
		m.logger.Error("Migration durumu kontrol edilemedi", map[string]interface{}{"name": name, "error": err.Error()})
		return false, err
	}

	return count > 0, nil
}

func (m *MigrationService) RecordMigration(name string) error {
	query := "INSERT INTO migrations (name, applied_at) VALUES ($1, $2)"
	_, err := m.db.Exec(query, name, time.Now())
	if err != nil {
		m.logger.Error("Migration kaydedilemedi", map[string]interface{}{"name": name, "error": err.Error()})
		return err
	}

	return nil
}

func (m *MigrationService) ApplyMigration(name string, migrationFunc func(*sql.DB) error) error {
	applied, err := m.IsMigrationApplied(name)
	if err != nil {
		return err
	}

	if applied {
		m.logger.Info("Migration zaten uygulanmış", map[string]interface{}{"name": name})
		return nil
	}

	m.logger.Info("Migration uygulanıyor", map[string]interface{}{"name": name})

	tx, err := m.db.Begin()
	if err != nil {
		m.logger.Error("Transaction başlatılamadı", map[string]interface{}{"error": err.Error()})
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			m.logger.Error("Migration geri alındı", map[string]interface{}{"name": name, "error": err.Error()})
		}
	}()

	if err = migrationFunc(m.db); err != nil {
		m.logger.Error("Migration uygulanamadı", map[string]interface{}{"name": name, "error": err.Error()})
		return err
	}

	if err = m.RecordMigration(name); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		m.logger.Error("Transaction commit edilemedi", map[string]interface{}{"error": err.Error()})
		return err
	}

	m.logger.Info("Migration başarıyla uygulandı", map[string]interface{}{"name": name})
	return nil
}

func (m *MigrationService) RunMigrations() error {
	m.logger.Info("Migrationlar başlatılıyor", map[string]interface{}{})

	if err := m.InitMigrationTable(); err != nil {
		return fmt.Errorf("migration tablosu oluşturulamadı: %w", err)
	}

	migrations := []struct {
		Name string
		Func func(*sql.DB) error
	}{
		{"create_users_table", CreateUsersTable},
		{"create_transactions_table", CreateTransactionsTable},
		{"create_balances_table", CreateBalancesTable},
		{"create_audit_logs_table", CreateAuditLogsTable},
		{"create_balance_history_table", CreateBalanceHistoryTable},
	}

	for _, migration := range migrations {
		if err := m.ApplyMigration(migration.Name, migration.Func); err != nil {
			return fmt.Errorf("migration uygulanamadı %s: %w", migration.Name, err)
		}
	}

	return nil
}

func CreateUsersTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        username TEXT NOT NULL UNIQUE,
        email TEXT NOT NULL UNIQUE,
        password_hash TEXT NOT NULL,
        role TEXT NOT NULL DEFAULT 'user',
        api_key TEXT UNIQUE,
        created_at TIMESTAMP NOT NULL,
        updated_at TIMESTAMP NOT NULL
    )
    `

	_, err := db.Exec(query)
	return err
}

func CreateTransactionsTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS transactions (
        id SERIAL PRIMARY KEY,
        from_user_id INTEGER,
        to_user_id INTEGER,
        amount NUMERIC(18,2) NOT NULL,
        type TEXT NOT NULL,
        status TEXT NOT NULL,
        created_at TIMESTAMP NOT NULL,
        FOREIGN KEY (from_user_id) REFERENCES users (id),
        FOREIGN KEY (to_user_id) REFERENCES users (id)
    )
    `

	_, err := db.Exec(query)
	return err
}

func CreateBalancesTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS balances (
        user_id INTEGER PRIMARY KEY,
        amount NUMERIC(18,2) NOT NULL DEFAULT 0,
        last_updated_at TIMESTAMP NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users (id)
    )
    `

	_, err := db.Exec(query)
	return err
}

func CreateAuditLogsTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS audit_logs (
        id SERIAL PRIMARY KEY,
        entity_type TEXT NOT NULL,
        entity_id INTEGER NOT NULL,
        action TEXT NOT NULL,
        details TEXT,
        created_at TIMESTAMP NOT NULL
    )
    `

	_, err := db.Exec(query)
	return err
}

func CreateBalanceHistoryTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS balance_history (
        id SERIAL PRIMARY KEY,
        user_id INTEGER NOT NULL,
        amount NUMERIC(18,2) NOT NULL,
        previous_amount NUMERIC(18,2) NOT NULL,
        transaction_id INTEGER,
        operation TEXT NOT NULL,
        created_at TIMESTAMP NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users (id),
        FOREIGN KEY (transaction_id) REFERENCES transactions (id)
    );
    
    CREATE INDEX IF NOT EXISTS balance_history_user_id_idx ON balance_history (user_id);
    CREATE INDEX IF NOT EXISTS balance_history_created_at_idx ON balance_history (created_at);
    `

	_, err := db.Exec(query)
	return err
}
