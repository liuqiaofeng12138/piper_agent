INSERT INTO dim_fab (fab_id, fab_name, region, node_cap_nm) VALUES
  ('FAB-SH', '上海 Fab12', '华东', 28),
  ('FAB-CD', '成都 Fab8', '西南', 40),
  ('FAB-HS', '合肥 Fab3', '华东', 14);

INSERT INTO dim_product (product_id, product_name, node_nm, category, customer) VALUES
  ('P-N28-001', 'N28 车规 MCU', 28, 'MCU', 'Tier1-A'),
  ('P-N28-002', 'N28 电源管理 IC', 28, 'PMIC', 'Tier1-B'),
  ('P-N14-010', 'N14 AI 推理芯片', 14, 'AI', 'Cloud-X'),
  ('P-N40-020', 'N40 射频前端', 40, 'RF', 'Handset-Y');

-- N28 车规 MCU @ 上海：近 30 天良率（可演示「近一个月 xx 产品良率」）
INSERT INTO fact_yield_daily (stat_date, product_id, fab_id, good_die, total_die, yield_pct)
SELECT
  (CURRENT_DATE - offs)::date,
  'P-N28-001',
  'FAB-SH',
  920000 + (offs % 7) * 3000,
  1000000,
  ROUND((920000 + (offs % 7) * 3000)::numeric / 1000000 * 100, 3)
FROM generate_series(0, 29) AS offs;

-- 其它产品/工厂抽样
INSERT INTO fact_yield_daily (stat_date, product_id, fab_id, good_die, total_die, yield_pct) VALUES
  (CURRENT_DATE - 1, 'P-N28-002', 'FAB-CD', 905000, 1000000, 90.500),
  (CURRENT_DATE - 1, 'P-N14-010', 'FAB-HS', 965000, 1000000, 96.500),
  (CURRENT_DATE - 7, 'P-N40-020', 'FAB-CD', 880000, 1000000, 88.000);

INSERT INTO fact_wip_lot (lot_id, product_id, fab_id, stage, qty_wafers) VALUES
  ('LOT-20260301-A', 'P-N28-001', 'FAB-SH', '光刻', 25),
  ('LOT-20260301-B', 'P-N28-001', 'FAB-SH', '蚀刻', 18),
  ('LOT-20260302-C', 'P-N14-010', 'FAB-HS', 'CMP', 12),
  ('LOT-20260303-D', 'P-N28-002', 'FAB-CD', '薄膜沉积', 20),
  ('LOT-20260304-E', 'P-N40-020', 'FAB-CD', '封装前测', 8);

INSERT INTO fact_equipment_oee (stat_date, equipment_id, fab_id, oee_pct, uptime_hours)
SELECT
  (CURRENT_DATE - offs)::date,
  'EQ-LITHO-01',
  'FAB-SH',
  82.5 + (offs % 5),
  20.0 + (offs % 3) * 0.5
FROM generate_series(0, 29) AS offs;

INSERT INTO fact_equipment_oee (stat_date, equipment_id, fab_id, oee_pct, uptime_hours) VALUES
  (CURRENT_DATE - 1, 'EQ-ETCH-02', 'FAB-SH', 78.20, 19.50),
  (CURRENT_DATE - 1, 'EQ-CMP-01', 'FAB-HS', 88.40, 21.00),
  (CURRENT_DATE - 1, 'EQ-DEP-03', 'FAB-CD', 76.80, 18.75);
