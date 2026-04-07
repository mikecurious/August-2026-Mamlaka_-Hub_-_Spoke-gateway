# Zambia Mobile Money API (ZMW)

This document describes Zambia support added to the existing APIs:

- `POST /api/v1/mobile/initiate` (collection)
- `POST /api/v1/mobile/transfer` (withdrawal/transfer)

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
  "currency": "ZMW",
  "amount": 10,
  "payerPhone": "260978363554",
  "mobileMoneySP": "AIRTEL",
  "externalId": "ImpadlTdest2909",
  "callbackUrl": "https://your-domain.com/merchant-callback"
}
```

### Notes

- Uses Flutterwave: `POST /v3/charges?type=mobile_money_zambia`
- `mobileMoneySP` normalization:
  - `AIRTEL` -> `Airtel`
  - `MTN` -> `MTN`
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

## 2) Transfer (Payout)

### Endpoint

- **Method:** `POST`
- **URL:** `/api/v1/mobile/transfer`
- **Auth:** `Authorization: Bearer <merchant-jwt>`

### Request

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "ZMW",
  "amount": 10,
  "recipientPhone": "+260978363554",
  "mobileMoneySP": "MTN",
  "externalId": "ZMW-OUT-001",
  "callbackUrl": "https://your-domain.com/merchant-callback"
}
```

### Notes

- Uses Flutterwave: `POST /v3/transfers/`
- `mobileMoneySP` normalization:
  - `AIRTEL` -> `Airtel`
  - `MTN` -> `MTN`
- Transaction is stored as pending and finalized on callback.

### Response (Our API)

```json
{
  "message": "Payment initiation successful",
  "transactionId": "ZMW-OUT-001",
  "secureId": "...."
}
```

---

## 3) Flutterwave Callback URL (Single URL)

Set this callback URL in Flutterwave for both charge and transfer events:

- `https://payments.mam-laka.com/api/v1/flutterwave/callback`

Supported events:

- `charge.completed`
- `transfer.completed`

---

## 4) Provider Callback Examples

### Collection success callback (from Flutterwave)

```json
{
  "event": "charge.completed",
  "data": {
    "tx_ref": "70skksks",
    "currency": "ZMW",
    "amount": 10,
    "app_fee": 0.5,
    "status": "successful"
  }
}
```

### Transfer completed callback (from Flutterwave)

```json
{
  "event": "transfer.completed",
  "data": {
    "reference": "e4630d77b4f0dd00",
    "currency": "ZMW",
    "amount": 10,
    "fee": 0.2,
    "status": "SUCCESSFUL",
    "complete_message": "Transaction was successful"
  }
}
```

---

## 5) Merchant Callback Format (Our API -> Merchant)

We preserve the same callback structure used in other channels.

### Success

```json
{
  "transactionStatus": "COMPLETE",
  "transactionReport": "COMPLETE",
  "currency": "ZMW",
  "amount": 10,
  "secureId": "....",
  "externalId": "...."
}
```

### Failed

```json
{
  "transactionStatus": "FAILED",
  "transactionReport": "FAILED",
  "currency": "ZMW",
  "amount": 10,
  "secureId": "....",
  "externalId": "....",
  "reason": "failure reason"
}
```

