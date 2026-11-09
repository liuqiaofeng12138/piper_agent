-- 半导体演示数仓（NL2SQL Agent）
CREATE TABLE dim_fab (
    fab_id       VARCHAR(16) PRIMARY KEY,
    fab_name     VARCHAR(64) NOT NULL,
    region       VARCHAR(32) NOT NULL,
    node_cap_nm  INT NOT NULL
);

CREATE TABLE dim_product (
    product_id   VARCHAR(16) PRIMARY KEY,
    product_name VARCHAR(128) NOT NULL,
    node_nm      INT NOT NULL,
    category     VARCHAR(32) NOT NULL,
    customer     VARCHAR(64)
);

CREATE TABLE fact_yield_daily (
    stat_date    DATE NOT NULL,
    product_id   VARCHAR(16) NOT NULL REFERENCES dim_product(product_id),
    fab_id       VARCHAR(16) NOT NULL REFERENCES dim_fab(fab_id),
    good_die     BIGINT NOT NULL,
    total_die    BIGINT NOT NULL,
    yield_pct    NUMERIC(6, 3) NOT NULL,
    PRIMARY KEY (stat_date, product_id, fab_id)
);

CREATE TABLE fact_wip_lot (
    lot_id       VARCHAR(24) PRIMARY KEY,
    product_id   VARCHAR(16) NOT NULL REFERENCES dim_product(product_id),
    fab_id       VARCHAR(16) NOT NULL REFERENCES dim_fab(fab_id),
    stage        VARCHAR(48) NOT NULL,
    qty_wafers   INT NOT NULL,
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE fact_equipment_oee (
    stat_date     DATE NOT NULL,
    equipment_id  VARCHAR(24) NOT NULL,
    fab_id        VARCHAR(16) NOT NULL REFERENCES dim_fab(fab_id),
    oee_pct       NUMERIC(6, 2) NOT NULL,
    uptime_hours  NUMERIC(8, 2) NOT NULL,
    PRIMARY KEY (stat_date, equipment_id)
);

CREATE INDEX idx_yield_product_date ON fact_yield_daily (product_id, stat_date DESC);
CREATE INDEX idx_wip_product ON fact_wip_lot (product_id, stage);
CREATE INDEX idx_oee_fab_date ON fact_equipment_oee (fab_id, stat_date DESC);
