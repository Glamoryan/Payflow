-- +migrate Up
CREATE TABLE IF NOT EXISTS balance_history (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    amount DECIMAL(15,2) NOT NULL,
    previous_amount DECIMAL(15,2) NOT NULL,
    transaction_type VARCHAR(50) NOT NULL,
    transaction_id INTEGER REFERENCES transactions(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_balance_history_user_id ON balance_history(user_id);
CREATE INDEX idx_balance_history_transaction_id ON balance_history(transaction_id);
CREATE INDEX idx_balance_history_created_at ON balance_history(created_at);

-- +migrate Down
DROP INDEX IF EXISTS idx_balance_history_created_at;
DROP INDEX IF EXISTS idx_balance_history_transaction_id;
DROP INDEX IF EXISTS idx_balance_history_user_id;
DROP TABLE IF EXISTS balance_history; 