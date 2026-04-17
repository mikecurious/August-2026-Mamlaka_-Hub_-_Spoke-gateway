# Settlement API Updates

## Overview

This document describes the new manual settlement module and settlement report pagination.

## What Was Added

- New module: manual settlement
- Finance-only settlement execution
- Merchant self-report endpoint
- Pagination for settlement list endpoints
- Settlement audit storage for traceability

## Endpoints

### 1) Finance - Create Manual Settlement

- `POST /api/v1/settlement/manual`
- Auth: Bearer token + finance-only access

#### Request Body

```json
{
  "impalaMerchantId": "app",
  "amount": 1000,
  "currency": "KES",
  "feePercent": 1.5,
  "note": "Manual settlement run for March"
}
```

#### Processing Logic

- Validates amount and currency
- Computes:
  - `feeAmount = amount * feePercent / 100`
  - `totalDeducted = amount + feeAmount`
- Deducts `totalDeducted` from `merchant_collection_balance`
- Writes a row to `manual_settlements` with:
  - merchant, currency, settled amount
  - fee %, fee amount, total deducted
  - balance before/after
  - status and user who settled

#### Response Example

```json
{
  "message": "Manual settlement completed successfully",
  "settlementId": 12,
  "merchantId": "app",
  "currency": "KES",
  "settledAmount": 1000,
  "feePercent": 1.5,
  "feeAmount": 15,
  "totalDeducted": 1015,
  "balanceBefore": 5000,
  "balanceAfter": 3985,
  "status": "COMPLETED",
  "settledBy": "finance"
}
```

### 2) Finance - List Manual Settlements

- `GET /api/v1/settlement/manual/list`
- Auth: Bearer token + finance-only access

#### Query Parameters

- `merchantId` (optional)
- `currency` (optional)
- `page` (optional, default `1`)
- `limit` (optional, default `20`, max `200`)

#### Response Example

```json
{
  "count": 20,
  "pagination": {
    "page": 1,
    "limit": 20,
    "totalPages": 6,
    "totalItems": 103
  },
  "settlements": []
}
```

### 3) Merchant - My Settlement Report

- `GET /api/v1/settlement/report`
- Auth: Bearer token (merchant)

This endpoint automatically uses merchant identity from JWT and returns only that merchant's settlements.

#### Query Parameters

- `currency` (optional)
- `page` (optional, default `1`)
- `limit` (optional, default `20`, max `200`)

#### Response Example

```json
{
  "merchantId": "app",
  "count": 10,
  "pagination": {
    "page": 1,
    "limit": 10,
    "totalPages": 4,
    "totalItems": 38
  },
  "settlements": []
}
```

## Finance Access Configuration

Finance access uses `username` from JWT and checks the `FINANCE_USERS` environment variable.

Example:

```env
FINANCE_USERS=finance,finance_ops,cfo
```

If not set, default allowed username is:

- `finance`

## Notes

- Settlement deduction is done in a DB transaction for safety.
- If merchant is missing or balance is insufficient, settlement is rejected.
- Supported currencies follow existing `merchant_collection_balance` columns.

