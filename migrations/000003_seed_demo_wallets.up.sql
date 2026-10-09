-- Demo wallets for manual testing (Phase 1).
INSERT INTO wallets (id, currency, balance)
VALUES
    ('wallet_1', 'INR', 100000),
    ('wallet_2', 'INR', 50000)
ON CONFLICT (id) DO NOTHING;
