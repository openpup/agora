CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE finance_market_data (
    time TIMESTAMPTZ NOT NULL,
    ticker VARCHAR(20) NOT NULL,
    market VARCHAR(20) NOT NULL,
    open DOUBLE PRECISION,
    high DOUBLE PRECISION,
    low DOUBLE PRECISION,
    close DOUBLE PRECISION,
    volume DOUBLE PRECISION,
    metadata JSONB NOT NULL DEFAULT '{}',
    PRIMARY KEY (time, ticker, market)
);

SELECT create_hypertable('finance_market_data', 'time', if_not_exists => TRUE);
CREATE INDEX idx_finance_market_data_ticker ON finance_market_data (ticker, time DESC);
