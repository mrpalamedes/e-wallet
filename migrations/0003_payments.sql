ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_status_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_status_check
    CHECK (status IN ('pending', 'completed', 'failed', 'expired'));

ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_payment_expiry_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_payment_expiry_check
    CHECK (type <> 'payment' OR expires_at IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_transactions_pending_expiry 
    ON transactions(expires_at)
    WHERE status = 'pending';
