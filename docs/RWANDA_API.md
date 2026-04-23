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

### Notes

- Uses Flutterwave: `POST /v3/charges?type=mobile_money_rwanda`
- The flow matches Zambia collection: pass `phone_number` + normalized `account_bank`.
- `mobileMoneySP`/`account_bank` normalization:
  - `3044`, `AIRTEL` -> `Airtel`
  - `257`, `MPS`, `MOBILE MONEY` -> `MPS`
  - `3045`, `MTN` -> `MTN`
  - `3046`, `ZAMTEL` -> `ZAMTEL`
  - `2236`, `ZM360000`, `ECOBANK ZAMBIA` -> `ZM360000`
- Default customer email sent upstream: `tech@mam-laka.com`
- Internal `tx_ref` is generated from our `secureId`

### Response (Our API)

```json
{
  "message": "Payment initiation successful",
  "transactionId": "....",
  "secureId": "...."
}
```

---

## 2) Flutterwave Callback URL

Set this callback URL in Flutterwave for charge events:

- `https://payments.mam-laka.com/api/v1/flutterwave/callback`

Supported event:

- `charge.completed`

---

## 3) Provider Callback Example

### Collection success callback (from Flutterwave)

```json
{
  "event": "charge.completed",
  "data": {
    "tx_ref": "70skksks",
    "currency": "RWF",
    "amount": 5000,
    "app_fee": 0.5,
    "status": "successful"
  }
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
