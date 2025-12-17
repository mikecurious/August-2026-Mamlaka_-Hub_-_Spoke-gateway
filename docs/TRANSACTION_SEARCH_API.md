# Transaction Search API Documentation

## Overview

The Transaction Search API allows you to search and filter transactions based on multiple criteria including merchant ID, phone number, amount, currency, status, and more. This endpoint supports pagination and returns detailed transaction information.

## Endpoint

```
GET /api/v1/transactions/search
```

## Authentication

This endpoint requires authentication. Include a Bearer token in the Authorization header:

```
Authorization: Bearer <your_token>
```

## Query Parameters

All parameters are optional. You can combine multiple filters to narrow down your search results.

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `merchantId` | string | Filter by merchant ID. If authenticated, this is automatically set from your token. | `merchant123` |
| `phone` | string | Search by phone number (partial match supported). Searches in the `msisdn` field. | `254712345678` |
| `amount` | integer | Filter by exact transaction amount. | `1000` |
| `currency` | string | Filter by currency code (KES, USD, XOF, UGX, etc.). | `KES` |
| `status` | string | Filter by transaction status (PENDING, COMPLETE, FAILED, processing, etc.). | `COMPLETE` |
| `report` | string | Filter by transaction report type (collection, withdraw, deposit). | `collection` |
| `externalId` | string | Search by exact external ID. | `ImpadlTdest25` |
| `secureId` | string | Search by exact secure ID. | `6S4mi15UgO2xYkjjyqH4gA==` |
| `sourceOfFunds` | string | Filter by source of funds (MPESA, CARD, CRYPTO, korapay, flutterwave, etc.). | `MPESA` |
| `startDate` | integer | Filter transactions from this Unix timestamp (inclusive). | `1704067200` |
| `endDate` | integer | Filter transactions up to this Unix timestamp (inclusive). | `1704153600` |
| `page` | integer | Page number for pagination (default: 1). | `1` |
| `page_size` | integer | Number of results per page (default: 20, max: 100). | `20` |

## Request Examples

### Example 1: Search by Merchant and Status

```bash
curl -X GET "https://api.example.com/api/v1/transactions/search?merchantId=merchant123&status=COMPLETE&page=1&page_size=20" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Example 2: Search by Phone Number

```bash
curl -X GET "https://api.example.com/api/v1/transactions/search?phone=254712345678" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Example 3: Search by Amount and Currency

```bash
curl -X GET "https://api.example.com/api/v1/transactions/search?amount=1000&currency=KES" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Example 4: Search by Date Range

```bash
curl -X GET "https://api.example.com/api/v1/transactions/search?startDate=1704067200&endDate=1704153600&status=COMPLETE" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Example 5: Complex Search with Multiple Filters

```bash
curl -X GET "https://api.example.com/api/v1/transactions/search?merchantId=merchant123&phone=254712345678&amount=1000&currency=KES&status=COMPLETE&report=collection&sourceOfFunds=MPESA&page=1&page_size=50" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Response Format

### Success Response (200 OK)

```json
{
  "transactions": [
    {
      "id": 1234,
      "impalaMerchantId": "merchant123",
      "transactionStatus": "COMPLETE",
      "transactionReport": "collection",
      "currency": "KES",
      "amount": 1000,
      "msisdn": "254712345678",
      "netAmount": 1000.0,
      "secureId": "6S4mi15UgO2xYkjjyqH4gA==",
      "sourceOfFunds": "MPESA",
      "externalId": "ImpadlTdest25",
      "callbackUrl": "https://merchant.com/callback",
      "dateAdded": 1704067200,
      "merchantRequestID": "AG_20240101_123456",
      "checkoutRequestID": "ws_CO_01012024123456",
      "responseCode": "0",
      "responseDescription": "Success",
      "callbackStatus": "SENT"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 5,
    "total_items": 100
  }
}
```

### Error Responses

#### 401 Unauthorized

```json
{
  "error": "Missing authentication details"
}
```

#### 400 Bad Request

```json
{
  "error": "Invalid request format",
  "details": "Invalid page number"
}
```

#### 500 Internal Server Error

```json
{
  "error": "Failed to search transactions",
  "details": "Database connection error"
}
```

## Response Fields

### Transaction Object

| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | Unique transaction ID |
| `impalaMerchantId` | string | Merchant identifier |
| `transactionStatus` | string | Current transaction status (PENDING, COMPLETE, FAILED, processing, etc.) |
| `transactionReport` | string | Transaction type (collection, withdraw, deposit) |
| `currency` | string | Currency code (KES, USD, XOF, UGX, etc.) |
| `amount` | integer | Transaction amount |
| `msisdn` | string | Phone number associated with the transaction |
| `netAmount` | float | Net amount after fees |
| `secureId` | string | Secure transaction identifier |
| `sourceOfFunds` | string | Payment source (MPESA, CARD, CRYPTO, etc.) |
| `externalId` | string | External transaction identifier |
| `callbackUrl` | string | Merchant callback URL |
| `dateAdded` | integer | Unix timestamp when transaction was created |
| `merchantRequestID` | string | Merchant request identifier |
| `checkoutRequestID` | string | Checkout request identifier |
| `responseCode` | string | Response code from payment provider |
| `responseDescription` | string | Response description |
| `callbackStatus` | string | Callback delivery status |

### Pagination Object

| Field | Type | Description |
|-------|------|-------------|
| `current_page` | integer | Current page number |
| `per_page` | integer | Number of items per page |
| `total_pages` | integer | Total number of pages |
| `total_items` | integer | Total number of matching transactions |

## Transaction Status Values

| Status | Description |
|--------|------------|
| `PENDING` | Transaction is pending processing |
| `COMPLETE` | Transaction completed successfully |
| `FAILED` | Transaction failed |
| `processing` | Transaction is being processed |

## Transaction Report Types

| Type | Description |
|------|-------------|
| `collection` | Collection/payment received |
| `withdraw` | Withdrawal/payout |
| `deposit` | Deposit transaction |

## Source of Funds Values

Common values include:
- `MPESA` - M-Pesa mobile money
- `CARD` - Card payment
- `CRYPTO` - Cryptocurrency payment
- `korapay` - Korapay payment
- `flutterwave` - Flutterwave payment
- `payaza` - Payaza payment
- `ECITIZEN` - eCitizen payment

## Notes

1. **Phone Number Search**: The phone parameter uses partial matching (LIKE query), so you can search with partial phone numbers.

2. **Date Range**: Both `startDate` and `endDate` should be Unix timestamps (seconds since epoch).

3. **Pagination**: 
   - Default page size is 20
   - Maximum page size is 100
   - Results are ordered by `dateAdded` in descending order (newest first)

4. **Merchant ID**: If you're authenticated, the `merchantId` parameter is automatically set from your token. You can still override it by providing it explicitly in the query string.

5. **Combining Filters**: All filters are combined with AND logic. All conditions must be met for a transaction to appear in results.

6. **Empty Results**: If no transactions match your criteria, you'll receive an empty array with pagination metadata showing `total_items: 0`.

## Rate Limiting

This endpoint may be subject to rate limiting. Check response headers for rate limit information:
- `X-RateLimit-Limit`: Maximum number of requests per time window
- `X-RateLimit-Remaining`: Remaining requests in current window
- `X-RateLimit-Reset`: Time when the rate limit resets

## Support

For issues or questions, please contact support or refer to the main API documentation.

