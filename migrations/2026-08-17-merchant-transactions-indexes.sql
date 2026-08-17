-- Incident 2026-08-17: b2c callbacks timing out (499) — merchantRequestID lookups
-- were full-scanning merchant_transactions (~400k rows, ~193s/query).
-- Columns are longtext (GORM default), so prefix lengths are required.

ALTER TABLE merchant_transactions
  ADD INDEX idx_mt_merchant_request_id (merchantRequestID(100)),
  ALGORITHM=INPLACE, LOCK=NONE;

ALTER TABLE merchant_transactions
  ADD INDEX idx_mt_sof_report_status (sourceOfFunds(50), transactionReport(50), transactionStatus(20)),
  ALGORITHM=INPLACE, LOCK=NONE;

ALTER TABLE merchant_transactions
  ADD INDEX idx_mt_checkout_request_id (checkoutRequestID(100)),
  ALGORITHM=INPLACE, LOCK=NONE;
