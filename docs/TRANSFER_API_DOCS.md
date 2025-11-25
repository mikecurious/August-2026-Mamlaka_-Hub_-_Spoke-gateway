# Transfer API Documentation

## Overview
This document describes the Transfer API for moving funds from merchant collection balances to merchant balances, with automatic fee deduction and earnings tracking.

## Authentication
All API endpoints require Bearer token authentication in the Authorization header:
```
Authorization: Bearer YOUR_TOKEN_HERE
```

## Endpoint

### Transfer Funds
**Endpoint:** `POST /transfer`

**Description:** Transfers funds from merchant collection balance to merchant balance, deducting a 1.5% platform fee

**Request Body:**
```json
{
    "impalaMerchantId": "string (required)",
    "currency": "string (required)",
    "amount": "number (required)"
}
```

**Example Request:**
```json
{
    "impalaMerchantId": "ncgames_sandbox",
    "currency": "KES",
    "amount": 1000.00
}
```

**Success Response:**
```json
{
    "status": "success",
    "message": "Transfer completed successfully",
    "amount": 1000.00,
    "currency": "KES",
    "fee": 15.00,
    "netAmount": 985.00,
    "transferDate": "2025-01-22 15:30:45",
    "merchantId": "ncgames_sandbox"
}
```

**Error Responses:**

**Insufficient Collection Balance:**
```json
{
    "error": "INSUFFICIENT_BALANCE",
    "message": "Insufficient collection balance. Available: 500.00 KES"
}
```

**Unsupported Currency:**
```json
{
    "error": "Unsupported currency"
}
```

**Merchant Not Found:**
```json
{
    "error": "Merchant not found"
}
```

## Supported Currencies
- KES (Kenyan Shilling)
- UGX (Ugandan Shilling)
- USD (US Dollar)
- EUR (Euro)
- GBP (British Pound)
- TZS (Tanzanian Shilling)
- XAF (Central African Franc)

## Fee Structure
- **Platform Fee:** 1.5% of transfer amount
- **Net Amount:** Transfer amount minus fee
- **Fee Tracking:** All fees are recorded in the platform earnings table

## Transfer Process

1. **Validation**
   - Verify merchant authentication
   - Check merchant exists
   - Validate currency support
   - Verify sufficient collection balance

2. **Database Transaction**
   - Deduct full amount from collection balance
   - Add net amount (amount - fee) to merchant balance
   - Record platform earnings entry

3. **Response**
   - Return success with transfer details
   - Include fee breakdown

## Database Changes

### Collection Balance (merchant_collection_balance)
- Deducts full amount from appropriate currency field
- Example: `kesBalance = kesBalance - 1000.00`

### Merchant Balance (merchant_balances)
- Adds net amount to appropriate currency field
- Example: `kesBalance = kesBalance + 985.00`

### Platform Earnings (platform_earnings)
- Records new entry with:
  - `impalaMerchantId`: Merchant ID
  - `amountTransferred`: Original transfer amount
  - `transactionCharges`: Fee amount (1.5%)
  - `transferDate`: Current timestamp

## Error Handling

### Common Error Codes
- `400` - Bad Request (invalid input, insufficient balance, unsupported currency)
- `401` - Unauthorized (invalid token)
- `404` - Not Found (merchant not found, balance not found)
- `500` - Internal Server Error (database errors)

### Transaction Safety
- All operations are wrapped in database transactions
- If any step fails, all changes are rolled back
- Ensures data consistency

## Example Usage

### Transfer 1000 KES
```bash
curl -X POST http://localhost:8090/api/v1/transfer \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "impalaMerchantId": "ncgames_sandbox",
    "currency": "KES",
    "amount": 1000.00
  }'
```

### Response
```json
{
    "status": "success",
    "message": "Transfer completed successfully",
    "amount": 1000.00,
    "currency": "KES",
    "fee": 15.00,
    "netAmount": 985.00,
    "transferDate": "2025-01-22 15:30:45",
    "merchantId": "ncgames_sandbox"
}
```

## Business Logic

1. **Collection Balance** - Where payments are received
2. **Merchant Balance** - Where merchants can withdraw funds
3. **Platform Fee** - 1.5% charged for transfer service
4. **Earnings Tracking** - All fees recorded for platform revenue

## Security Notes

1. **Authentication Required** - All requests must include valid Bearer token
2. **Merchant Validation** - Verifies merchant exists before processing
3. **Transaction Safety** - Database transactions ensure data integrity
4. **Balance Verification** - Checks sufficient funds before transfer

---

**Last Updated:** January 2025
**Version:** 1.0
