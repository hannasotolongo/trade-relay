CREATE TABLE orders (
    id VARCHAR(128) NOT NULL,
    signal_id VARCHAR(128) NOT NULL,
    account_id VARCHAR(128) NOT NULL,
    broker_account_id VARCHAR(128) NOT NULL,
    broker_order_id VARCHAR(128) NULL,

    symbol VARCHAR(32) NOT NULL,
    side VARCHAR(16) NOT NULL,

    quantity BIGINT NOT NULL,
    filled_quantity BIGINT NOT NULL DEFAULT 0,

    status VARCHAR(32) NOT NULL,

    version BIGINT NOT NULL DEFAULT 1,

    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,

    PRIMARY KEY (id),

    UNIQUE KEY uq_orders_broker_order_id (broker_order_id),

    KEY idx_orders_signal_id (signal_id),
    KEY idx_orders_account_id (account_id),
    KEY idx_orders_status (status),

    CONSTRAINT chk_orders_quantity
        CHECK (quantity > 0),

    CONSTRAINT chk_orders_filled_quantity
        CHECK (
            filled_quantity >= 0
            AND filled_quantity <= quantity
        ),

    CONSTRAINT chk_orders_side
        CHECK (side IN ('BUY', 'SELL')),

    CONSTRAINT chk_orders_status
        CHECK (
            status IN (
                'CREATED',
                'SUBMITTING',
                'ACKNOWLEDGED',
                'PARTIALLY_FILLED',
                'FILLED',
                'REJECTED',
                'UNKNOWN'
            )
        )
);