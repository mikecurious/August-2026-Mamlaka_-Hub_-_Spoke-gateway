# Flutterwave Integration API Documentation

## Overview
This document describes the Flutterwave M-Pesa payment integration for the Mam-laka Merchant API. The integration allows merchants to accept mobile money payments through Flutterwave's payment gateway.

## Authentication
All API endpoints require Bearer token authentication in the Authorization header:
```
Authorization: Bearer YOUR_TOKEN_HERE
```

## Endpoints

### 1. Initiate Payment
**Endpoint:** `POST /flutterwave/initiate`

**Description:** Initiates a mobile money payment through Flutterwave

**Request Body:**
```json
{
    "impalaMerchantId": "string (required)",
    "currency": "string (required)",
    "amount": "integer (required)",
    "customerEmail": "string (required, email format)",
    "payerPhone": "string (required)",
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
    "customerEmail": "john@example.com",
    "payerPhone": "254768899729",
    "externalId": "order_12345",
    "callbackUrl": "https://yoursite.com/callback",
    "redirectUrl": "https://yoursite.com/success"
}
```

**Success Response:**
```json
{
    "status": "success",
    "message": "Charge initiated",
    "secureId": "generated_secure_id",
    "externalId": "order_12345",
    "amount": 100,
    "currency": "KES",
    "provider": "flutterwave",
    "transactionStatus": "processing",
    "transactionReference": "DRT2185193369827039",
    "paymentReference": "generated_secure_id",
    "fee": 0.49,
    "narration": "FLW-PBF MPESA Transaction",
    "authModel": "LIPA_MPESA",
    "processorResponse": "Successful, pending customer validation"
}
```

**Error Response:**
```json
{
    "status": "error",
    "message": "Transaction Reference already exist. Try again in 2 minutes time to use the same ref for a new transaction",
    "data": null
}
```

### 2. Payment Callback
**Endpoint:** `POST /flutterwave/callback`

**Description:** Handles payment status updates from Flutterwave (automatically called by Flutterwave)

**Callback Payload (from Flutterwave):**
```json
{
    "event": "charge.completed",
    "data": {
        "id": 1902562078,
        "tx_ref": "generated_secure_id",
        "flw_ref": "DRT2185193369827039",
        "device_fingerprint": "N/A",
        "amount": 100,
        "currency": "KES",
        "charged_amount": 100,
        "app_fee": 0.49,
        "merchant_fee": 0,
        "processor_response": "The service request is processed successfully.",
        "auth_model": "LIPA_MPESA",
        "ip": "54.154.184.168",
        "narration": "FLW-PBF MPESA Transaction",
        "status": "successful",
        "payment_type": "mpesa",
        "created_at": "2025-10-22T10:46:22.000Z",
        "account_id": 3287467,
        "customer": {
            "id": 1239815700,
            "name": "Anonymous customer",
            "phone_number": "254768899729",
            "email": "john@example.com",
            "created_at": "2025-10-21T09:54:40.000Z"
        }
    },
    "event.type": "MPESA_TRANSACTION"
}
```

**Callback Events:**
- `charge.completed` with `status: "successful"` - Payment completed successfully
- `charge.completed` with `status: "failed"` - Payment failed

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
   - Merchant sends payment request to `/flutterwave/initiate`
   - System validates merchant and creates transaction record
   - Flutterwave API is called to initiate payment
   - Customer receives STK prompt on their phone

2. **Payment Processing**
   - Customer enters PIN on their phone
   - Flutterwave processes the payment

3. **Callback Handling**
   - Flutterwave sends callback to `/flutterwave/callback`
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

### Flutterwave Specific Errors
- **Duplicate Reference**: "Transaction Reference already exist. Try again in 2 minutes time to use the same ref for a new transaction"
- **Invalid Phone**: Phone number format errors
- **Insufficient Balance**: Customer doesn't have enough balance

## Testing

### Test Phone Numbers
Use Flutterwave's test phone numbers for development:
- Kenya: `254768899729`
- Uganda: `256700000000`

### Test Amounts
Use small amounts for testing (e.g., 1-10 in local currency)

## Security Notes

1. **API Key**: The Flutterwave API key is configured in the system
2. **Token Validation**: All requests are validated using Bearer tokens
3. **Secure IDs**: Each transaction gets a unique secure ID
4. **Callback Verification**: Callbacks are processed securely

## Merchant Callback Format

When a payment is completed, the system sends a callback to your specified URL:

**Success Callback:**
```json
{
    "transactionStatus": "success",
    "transactionReport": "charge.completed",
    "currency": "KES",
    "amount": 100,
    "netAmount": 99.51,
    "secureId": "generated_secure_id",
    "externalId": "order_12345"
}
```

**Failure Callback:**
```json
{
    "transactionStatus": "failed",
    "transactionReport": "charge.completed",
    "currency": "KES",
    "amount": 100,
    "netAmount": 0,
    "secureId": "generated_secure_id",
    "externalId": "order_12345"
}
```

## Flutterwave vs Other Providers

### Key Differences:
- **Reference Handling**: Flutterwave uses `tx_ref` instead of `reference`
- **Callback Structure**: Different callback payload structure
- **Fee Structure**: Uses `app_fee` instead of `fee`
- **Status Values**: Uses `successful`/`failed` instead of `success`/`failed`

### Response Fields:
- `flw_ref`: Flutterwave's internal reference
- `tx_ref`: Your transaction reference
- `app_fee`: Application fee charged
- `processor_response`: Detailed response from M-Pesa

## Support

For technical support or questions about the Flutterwave integration, please contact the development team.

---

**Last Updated:** January 2025
**Version:** 1.0
