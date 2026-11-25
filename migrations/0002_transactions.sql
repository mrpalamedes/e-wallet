CREATE TABLE IF NOT EXISTS transactions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users (id),
    type         TEXT        NOT NULL CHECK (type IN ('topup', 'transfer_in', 'transfer_out', 'payment')),
    amount       BIGINT      NOT NULL CHECK (amount > 0),
    from_account TEXT,
    to_account   TEXT,
    status       TEXT        NOT NULL CHECK (status IN ('pending', 'completed', 'failed')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_created
    ON transactions (user_id, created_at DESC, id DESC);