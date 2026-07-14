# Senegal Mobile Money API (XOF)

This document describes Senegal support added to the existing APIs using the same merchant-facing structure.

Supported providers:

- `WAVE`
- `ORANGE-MONEY`

---

## 1) Collection (Payin)

### Endpoint

- **Method:** `POST`
- **URL:** `/api/v1/mobile/initiate`
- **Auth:** `Authorization: Bearer <merchant-jwt>`

### Request (Wave)

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "XOF",
  "amount": 200,
  "payerPhone": "776389999",
  "mobileMoneySP": "WAVE",
  "externalId": "SN-IN-001",
  "callbackUrl": "https://your-domain.com/merchant-callback"
}
```

### Request (Orange Money)

`om_otp` is required for Orange Money collection.

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "XOF",
  "amount": 200,
  "payerPhone": "776389999",
  "mobileMoneySP": "ORANGE-MONEY",
  "om_otp": "434593",
  "externalId": "SN-IN-002",
  "callbackUrl": "https://your-domain.com/merchant-callback"
}
```


### Phone format

Senegal numbers are accepted in both formats:

- With country code: `+221776389999` or `221776389999`
- Without country code: `776389999`

### Response (Our API)

The response format stays the same and includes `redirectUrl` when the provider returns `sms_link`.

```json
{
  "message": "Payment initiation successful",
  "externalId": "SN-IN-001",
  "secureId": "qdml8553ZeInavKorBHzLA==",
  "redirectUrl": "https://pay.wave.com/..."
}
```

---

## 2) Payout (Withdrawal)

### Endpoint

- **Method:** `POST`
- **URL:** `/api/v1/mobile/transfer`
- **Auth:** `Authorization: Bearer <merchant-jwt>`

### Request

For Senegal payouts, pass `mobileMoneySP` as `WAVE` or `ORANGE-MONEY`.
`om_otp` is required for Orange Money payout and should be sent as a string so leading zeros are preserved.



```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "XOF",
  "amount": 200,
  "recipientPhone": "776389999",
  "mobileMoneySP": "ORANGE-MONEY",
  "om_otp": "091842",
  "externalId": "SN-OUT-001",
  "callbackUrl": "https://your-domain.com/merchant-callback"
}
```


```json
{
  "message": "Payment initiation successful",
  "externalId": "SN-OUT-001",
  "secureId": "i1FHTQgFpHP0g-MRacZXUQ==",
  "redirectUrl": "https://pay.wave.com/..."
}
```


---



## 4) Merchant Callback Format (Our API -> Merchant)


### Success

```json
{
  "amount": 10,
  "currency": "XOF",
  "externalId": "SN-OUT-001",
  "secureId": "i1FHTQgFpHP0g-MRacZXUQ==",
  "transactionReport": "COMPLETE",
  "transactionStatus": "COMPLETE"
}
```

### Failed / Declined

```json
{
  "amount": 10,
  "currency": "XOF",
  "externalId": "SN-OUT-001",
  "secureId": "6S4mi15UgO2xYkjjyqH4gA==",
  "transactionReport": "FAILED",
  "transactionStatus": "FAILED"
}
```

`reason` may be included for failed callbacks when available.
