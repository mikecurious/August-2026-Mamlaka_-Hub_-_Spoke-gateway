# Korapay Integration API Documentation

## Overview
This document describes the Korapay mobile money payment integration for the Mam-laka Merchant API. The integration allows merchants to accept mobile money payments through Korapay's payment gateway.

## Authentication
All API endpoints require Bearer token authentication in the Authorization header:
```
Authorization: Bearer YOUR_TOKEN_HERE
```

## Endpoints

### 1. Initiate Payment
**Endpoint:** `POST /korapay/initiate`

**Description:** Initiates a mobile money payment through Korapay

**Request Body:**
```json
{
    "impalaMerchantId": "string (required)",
    "currency": "string (required)",
    "amount": "integer (required)",
    "customerName": "string (required)",
    "customerEmail": "string (required, email format)",
    "payerPhone": "string (required)",
    "description": "string (required)",
    "externalId": "string (required)",
    "callbackUrl": "string (required)",
    "redirectUrl": "string (required)"
}
```

**Example Request:**
```json
{
    "impalaMerchantId": "merchant_123",
    "currency": "KES",
    "amount": 100,
    "customerName": "John Doe",
    "customerEmail": "john@example.com",
    "payerPhone": "254768899729",
    "description": "Payment for services",
    "externalId": "order_12345",
    "callbackUrl": "https://yoursite.com/callback",
    "redirectUrl": "https://yoursite.com/success"
}
```

**Success Response:**
```json
{
    "status": true,
    "message": "Authorization required",
    "secureId": "generated_secure_id",
    "externalId": "order_12345",
    "amount": 100,
    "currency": "KES",
    "provider": "korapay",
    "transactionStatus": "processing",
    "transactionReference": "KPY-CA-Z5pt5TQaN1Hv",
    "paymentReference": "order_12345",
    "fee": 0.29,
    "narration": "Payment for services",
    "authModel": "STK_PROMPT"
}
```

**Error Response:**
```json
{
    "error": "Invalid input",
    "details": "validation error details"
}
```

### 2. Payment Callback
**Endpoint:** `POST /korapay/callback`

**Description:** Handles payment status updates from Korapay (automatically called by Korapay)

**Callback Payload (from Korapay):**
```json
{
    "event": "charge.success",
    "data": {
        "reference": "transaction_reference",
        "payment_reference": "order_12345",
        "currency": "KES",
        "amount": 100,
        "fee": 0.29,
        "payment_method": "mobile_money",
        "status": "success"
    }
}
```

**Callback Events:**
- `charge.success` - Payment completed successfully
- `charge.failed` - Payment failed

## Supported Currencies
- KES (Kenyan Shilling)
- UGX (Ugandan Shilling)
- USD (US Dollar)
- EUR (Euro)
- GBP (British Pound)
- TZS (Tanzanian Shilling)
- XAF (Central African Franc)

## Phone Number Format
Phone numbers should be in international format without the `+` sign:
- Kenya: `254768899729`
- Uganda: `256700000000`
- Tanzania: `255700000000`

## Integration Flow

1. **Payment Initiation**
   - Merchant sends payment request to `/korapay/initiate`
   - System validates merchant and creates transaction record
   - Korapay API is called to initiate payment
   - Customer receives STK prompt on their phone

2. **Payment Processing**
   - Customer enters PIN on their phone
   - Korapay processes the payment

3. **Callback Handling**
   - Korapay sends callback to `/korapay/callback`
   - System updates transaction status
   - Merchant balance is updated (on success)
   - Merchant receives callback notification

## Error Handling

### Common Error Codes
- `400` - Bad Request (invalid input)
- `401` - Unauthorized (invalid token)
- `404` - Not Found (merchant not found)
- `500` - Internal Server Error

### Error Response Format
```json
{
    "error": "Error message",
    "details": "Additional error details"
}
```

## Testing

### Test Phone Numbers
Use Korapay's test phone numbers for development:
- Kenya: `254768899729`
- Uganda: `256700000000`

### Test Amounts
Use small amounts for testing (e.g., 1-10 in local currency)

## Security Notes

1. **API Key**: The Korapay API key is configured in the system
2. **Token Validation**: All requests are validated using Bearer tokens
3. **Secure IDs**: Each transaction gets a unique secure ID
4. **Callback Verification**: Callbacks are processed securely

## Merchant Callback Format

When a payment is completed, the system sends a callback to your specified URL:

**Success Callback:**
```json
{
    "transactionStatus": "success",
    "transactionReport": "charge.success",
    "currency": "KES",
    "amount": 100,
    "netAmount": 99.71,
    "secureId": "generated_secure_id",
    "externalId": "order_12345"
}
```

**Failure Callback:**
```json
{
    "transactionStatus": "failed",
    "transactionReport": "charge.failed",
    "currency": "KES",
    "amount": 100,
    "netAmount": 0,
    "secureId": "generated_secure_id",
    "externalId": "order_12345"
}
```

## Support

For technical support or questions about the Korapay integration, please contact the development team.

---

**Last Updated:** January 2025
**Version:** 1.0
