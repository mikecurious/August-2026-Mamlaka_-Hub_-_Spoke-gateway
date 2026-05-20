# Burkina Faso Mobile Money API (XOF)

Burkina Faso is integrated via the Pixel core API using the same merchant-facing payload as Senegal and Benin.

Supported provider:

- `ORANGE-MONEY` only

Service IDs (Pixel):

| Flow | Code |
|------|------|
| Collection (payin) | `167` |
| Payout (disbursement) | `166` |

---

## 1) Collection (Payin)

### Endpoint

- **Method:** `POST`
- **URL:** `/api/v1/mobile/initiate`
- **Auth:** `Authorization: Bearer <merchant-jwt>`

### Request

`om_otp` is **required** for Orange Money collection (same as Senegal).

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "XOF",
  "amount": 200,
  "payerPhone": "56675953",
  "mobileMoneySP": "ORANGE-MONEY",
  "om_otp": "434593",
  "externalId": "BF-IN-001",
  "callbackUrl": "https://your-domain.com/merchant-callback"
}
```

### Phone format

Burkina Faso numbers are accepted as:

- With country code: `+22656675953` or `22656675953`
- Without country code: `56675953` (8-digit local)

### Response

Same shape as other mobile APIs; includes `redirectUrl` when Pixel returns `sms_link`.

---

## 2) Payout (Withdrawal)

### Endpoint

- **Method:** `POST`
- **URL:** `/api/v1/mobile/transfer`
- **Auth:** `Authorization: Bearer <merchant-jwt>`

### Request

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "XOF",
  "amount": 200,
  "recipientPhone": "56675953",
  "mobileMoneySP": "ORANGE-MONEY",
  "externalId": "BF-OUT-001",
  "callbackUrl": "https://your-domain.com/merchant-callback"
}
```

Aliases for `mobileMoneySP`: `ORANGE`, `ORANGE_MONEY`, `OM`, or numeric `166` / `167` for payout/collection respectively.

---

## Callback

Pixel IPN: `https://payments.mam-laka.com/api/v1/west-africa/callback`

Merchant callbacks use the same format as other XOF mobile flows (`transactionStatus`, `secureId`, `externalId`, etc.).
