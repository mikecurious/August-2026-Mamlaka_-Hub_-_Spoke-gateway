-- M-Pesa provider receipt (MpesaReceiptNumber / TransactionReceipt) on successful transactions
ALTER TABLE merchant_transactions
  ADD COLUMN providerReference VARCHAR(64) NULL DEFAULT NULL
  AFTER callbackStatus;
