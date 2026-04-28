# Cameroon Mobile Money API (XAF)

This document describes Cameroon support added to the existing APIs using the same merchant-facing structure as Benin and Senegal.

Supported providers:

- `MTN`
- `ORANGE-MONEY`

---

## 1) Collection (Payin)

### Endpoint

- **Method:** `POST`
- **URL:** `/api/v1/mobile/initiate`
- **Auth:** `Authorization: Bearer <merchant-jwt>`

### Request (MTN)

```json
{
  "impalaMerchantId": "your-merchant-id",
  "displayName": "app",
  "currency": "XAF",
  "amount": 200,
  "payerPhone": "681461008",
  "mobileMoneySP": "MTN",
  "externalId": "CM-IN-001",
  "callbackUrl": "https://your-domain.com/merchant-callback"
}
```

### Request (Orange Money)

```json
{
  "impalaMerchantId": "your-merchant-id",
  "displayName": "app",
  "currency": "XAF",
  "amount": 200,
  "payerPhone": "681461008",
  "mobileMoneySP": "ORANGE-MONEY",
  "externalId": "CM-IN-002",
  "callbackUrl": "https://your-domain.com/merchant-callback"
}
```

### Service IDs (provider-side mapping)

- Cameroon MTN payin -> `339`
- Cameroon ORANGE-MONEY payin -> `337`

### Phone format

Cameroon numbers are accepted in both formats:

- With country code: `+237681461008` or `237681461008`
- Without country code: `681461008`

### Response (Our API)

Response format remains the same. `redirectUrl` is included only when provider returns `sms_link`.

```json
{
  "message": "Payment initiation successful",
  "externalId": "CM-IN-001",
  "secureId": "qdml8553ZeInavKorBHzLA=="
}
```

---

## 2) Payout (Transfer)

### Endpoint

- **Method:** `POST`
- **URL:** `/api/v1/mobile/transfer`
- **Auth:** `Authorization: Bearer <merchant-jwt>`

### Request (MTN)

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "XAF",
  "amount": 200,
  "recipientPhone": "681461008",
  "mobileMoneySP": "MTN",
  "externalId": "CM-OUT-001",
  "callbackUrl": "https://your-domain.com/merchant-callback"
}
```

### Request (Orange Money)

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "XAF",
  "amount": 200,
  "recipientPhone": "681461008",
  "mobileMoneySP": "ORANGE-MONEY",
  "externalId": "CM-OUT-002",
  "callbackUrl": "https://your-domain.com/merchant-callback"
}
```

### Service IDs (provider-side mapping)

- Cameroon MTN payout -> `338`
- Cameroon ORANGE-MONEY payout -> `336`

---

## 3) Provider Callback

Provider callbacks are received by Mam-laka at:

- `https://payments.mam-laka.com/api/v1/west-africa/callback`

This callback is processed internally and converted to the standard merchant callback structure.

---

## 4) Merchant Callback Format (Our API -> Merchant)

### Success

```json
{
  "amount": 200,
  "currency": "XAF",
  "externalId": "CM-OUT-001",
  "secureId": "i1FHTQgFpHP0g-MRacZXUQ==",
  "transactionReport": "COMPLETE",
  "transactionStatus": "COMPLETE"
}
```

### Failed

```json
{
  "amount": 200,
  "currency": "XAF",
  "externalId": "CM-OUT-001",
  "secureId": "6S4mi15UgO2xYkjjyqH4gA==",
  "transactionReport": "FAILED",
  "transactionStatus": "FAILED"
}
```

`reason` may be included for failed callbacks when available.
