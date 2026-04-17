# Transaction API Updates

## Overview

This document covers the latest transaction API updates focused on pagination consistency and client usability.

## Updated Endpoints

- `GET /api/v1/transactions`
- `GET /api/v1/transactions/search`
- `GET /api/v1/transaction?merchant=<merchantId>&secureId=<secureId>` (single transaction lookup, unchanged)

## Pagination Parameters

Both list endpoints now support:

- `page` (default: `1`)
- `limit` (default: `20`, max: `100`)

Backward compatibility is retained for:

- `page_size` (still accepted on list/search endpoints)

If both `limit` and `page_size` are provided, `limit` is used.

## Transaction List Example

### Request

```bash
curl --location 'https://payments.mam-laka.com/api/v1/transactions?page=1&limit=20' \
--header 'Authorization: Bearer <token>'
```

### Response

```json
{
  "transactions": [
    {
      "amount": 100,
      "currency": "NGN",
      "externalId": "ME001915",
      "secureId": "97ZHIDQODOUG",
      "transactionReport": "withdraw",
      "transactionStatus": "COMPLETE"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "totalPages": 3,
    "totalItems": 54,
    "current_page": 1,
    "per_page": 20,
    "total_pages": 3,
    "total_items": 54
  }
}
```

## Transaction Search Example

### Request

```bash
curl --location 'https://payments.mam-laka.com/api/v1/transactions/search?status=COMPLETE&currency=NGN&page=1&limit=50' \
--header 'Authorization: Bearer <token>'
```

### Notes

- Search still supports existing filters (`status`, `currency`, `externalId`, `secureId`, `report`, `sourceOfFunds`, date range, etc.).
- Results are ordered by newest first (`dateAdded DESC`).
- Max page size is capped to `100`.

## Pagination Fields

Standard fields:

- `pagination.page`
- `pagination.limit`
- `pagination.totalPages`
- `pagination.totalItems`

Compatibility fields (kept for existing integrations):

- `pagination.current_page`
- `pagination.per_page`
- `pagination.total_pages`
- `pagination.total_items`

