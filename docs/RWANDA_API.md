# Rwanda Mobile Money API (RWF)

This document describes Rwanda Flutterwave mobile money collection support added to the existing APIs:

- `POST /api/v1/mobile/initiate` (collection)

The merchant-facing API format remains the same as other channels.

---

## 1) Collection (Payin)

### Endpoint

- **Method:** `POST`
- **URL:** `/api/v1/mobile/initiate`
- **Auth:** `Authorization: Bearer <merchant-jwt>`

### Request

```json
{
  "impalaMerchantId": "your-merchant-id",
  "displayName": "TWD",
  "currency": "RWF",
  "amount": 5000,
  "payerPhone": "2507XXXXXXXX",
  "mobileMoneySP": "MTN",
  "externalId": "RWF-IN-001",
  "callbackUrl": "https://your-domain.com/merchant-callback"
}
```




```json
{
  "message": "Payment initiation successful",
  "transactionId": "....",
  "secureId": "...."
}
```

---

## 4) Merchant Callback Format (Our API -> Merchant)

We preserve the same callback structure used in other channels.

### Success

```json
{
  "transactionStatus": "COMPLETE",
  "transactionReport": "COMPLETE",
  "currency": "RWF",
  "amount": 5000,
  "secureId": "....",
  "externalId": "...."
}
```

### Failed

```json
{
  "transactionStatus": "FAILED",
  "transactionReport": "FAILED",
  "currency": "RWF",
  "amount": 5000,
  "secureId": "....",
  "externalId": "....",
  "reason": "failure reason"
}
```
