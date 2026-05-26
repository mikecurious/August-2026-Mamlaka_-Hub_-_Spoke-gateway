# Gambia Mobile Money API (GMD)

Gambia is integrated via the Pixel core API (separate API key from West Africa XOF routes).

Supported operators:

| Operator | `mobileMoneySP` | Collection (payin) | Payout |
|----------|-----------------|-------------------|--------|
| QMoney | `QMONEY` | **331** | **330** |
| AfriMoney | `AFRIMONEY` | **375** | **374** |

Test numbers:

- QMoney: `3655332`
- AfriMoney: `7215283`

---

## Environment

Set on the server (never expose in API responses):

```env
PIXEL_GMD_API_KEY=your_gambia_pixel_key
```

West Africa XOF/BF/SN routes continue to use `PIXEL_CORE_API_KEY`.

Pixel IPN (production): `https://payments.mam-laka.com/api/v1/west-africa/callback`

---

## 1) Collection (Payin)

- **Method:** `POST`
- **URL:** `/api/v1/mobile/initiate`
- **Auth:** `Authorization: Bearer <merchant-jwt>`

### QMoney

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "GMD",
  "amount": 200,
  "payerPhone": "3655332",
  "mobileMoneySP": "QMONEY",
  "om_otp": "123456",
  "externalId": "GM-IN-QM-001",
  "callbackUrl": "https://webhook.site/57cfdc77-6414-4a0b-9486-b9dafade3aea"
}
```

QMoney may require `om_otp` to confirm after initiation (provider message: resend OTP to confirm).

### AfriMoney

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "GMD",
  "amount": 200,
  "payerPhone": "7215283",
  "mobileMoneySP": "AFRIMONEY",
  "externalId": "GM-IN-AF-001",
  "callbackUrl": "https://webhook.site/57cfdc77-6414-4a0b-9486-b9dafade3aea"
}
```

### Phone format

- Local: `3655332`, `7215283`
- With country code: `2203655332` or `+2203655332`

### Success response (our API)

```json
{
  "message": "Payment initiation successful",
  "externalId": "GM-IN-QM-001",
  "secureId": "qdml8553ZeInavKorBHzLA=="
}
```

---

## 2) Payout (Withdrawal)

- **Method:** `POST`
- **URL:** `/api/v1/mobile/transfer`
- **Auth:** `Authorization: Bearer <merchant-jwt>`

### QMoney payout

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "GMD",
  "amount": 200,
  "recipientPhone": "3655332",
  "mobileMoneySP": "QMONEY",
  "externalId": "GM-OUT-QM-001",
  "callbackUrl": "https://webhook.site/57cfdc77-6414-4a0b-9486-b9dafade3aea"
}
```

### AfriMoney payout

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "GMD",
  "amount": 200,
  "recipientPhone": "7215283",
  "mobileMoneySP": "AFRIMONEY",
  "externalId": "GM-OUT-AF-001",
  "callbackUrl": "https://webhook.site/57cfdc77-6414-4a0b-9486-b9dafade3aea"
}
```

---

## Balances

| Wallet | Table | Column |
|--------|-------|--------|
| Collection (payins) | `merchant_collection_balance` | `gmdBalance` |
| Payout (disbursements) | `merchant_balances` | `gmdBalance` |

---

## Merchant callback

Same shape as other mobile flows (`transactionStatus`, `secureId`, `externalId`, `amount`, `currency`).
