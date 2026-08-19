# Mamlaka Mobile Money Integration (KES M-Pesa)

Base URL: `https://payments.mamlakapsp.com`
All paths below are under `/api/v1`.

---

## What changed (read this first)

If you integrated before Aug 2026, three things are new:

0. **The base URL changed to `https://payments.mamlakapsp.com`.** The old host
   `payments.mam-laka.com` is a **separate gateway on a separate database** — any call you
   send there (initiate, transfer, or status) will not be visible on your live environment,
   and a status query there returns "transaction not found" even for a transaction that
   succeeded. Point **every** call — `/mobile/initiate`, `/mobile/transfer`, and
   `/api/v1/transaction` — at `payments.mamlakapsp.com`.
1. **Callbacks are now signed.** Every webhook we POST to your `callbackUrl` carries an
   `X-Mamlaka-Signature` header. You should verify it (see [Callbacks](#3-callbacks-webhooks)).
   Existing callback handling is otherwise unchanged — the JSON body is the same.
2. **There is a status-query endpoint:** `GET /api/v1/transaction`. Use it to poll a
   transaction's status. **Do not run an M-Pesa TSQ / Transaction Status Query against
   Safaricom directly** — collections settle on Mamlaka's short code, so Safaricom will
   return "transaction not found" to a query made with your own short code. Mamlaka is the
   system of record; confirm status via the callback or this endpoint.

---

## 1. Collection — STK Push (customer pays you)

`POST /api/v1/mobile/initiate` — no token required (authenticated by `impalaMerchantId`).

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "KES",
  "amount": 100,
  "payerPhone": "2547XXXXXXXX",
  "mobileMoneySP": "M-PESA",
  "externalId": "your-unique-id-123",
  "callbackUrl": "https://your.domain/callbacks/mpesa"
}
```

| Field | Required | Notes |
|---|---|---|
| `impalaMerchantId` | yes | Your merchant id, exactly as issued. |
| `currency` | yes | `KES`. |
| `amount` | yes | Whole-number KES (integer). |
| `payerPhone` | yes | `2547XXXXXXXX` or `2541XXXXXXXX`. |
| `mobileMoneySP` | yes | `M-PESA`. |
| `externalId` | yes | **Your** unique reference. Must be unique per merchant — a repeat returns `409 DUPLICATE_EXTERNAL_ID`. This is your primary key for status lookups. |
| `callbackUrl` | yes | Where we POST the final result. |

The customer receives an STK prompt. Final outcome is delivered to `callbackUrl`
(and is queryable via [status](#4-status-query)). **The paybill/short code is resolved by
Mamlaka server-side — you do not send it.**

---

## 2. Payout — B2C (you pay a customer)

Payouts require a bearer token.

### 2a. Get a token

`GET /api/v1/` with HTTP **Basic** auth (`impalaMerchantId:password`, base64):

```sh
curl -s https://payments.mamlakapsp.com/api/v1/ \
  -H "Authorization: Basic $(printf '%s:%s' "$MERCHANT_ID" "$PASSWORD" | base64)"
# => {"token":"<JWT>","expires_at":"..."}
```

### 2b. Send the payout

`POST /api/v1/mobile/transfer` with `Authorization: Bearer <token>`:

```json
{
  "impalaMerchantId": "your-merchant-id",
  "currency": "KES",
  "amount": 100,
  "recipientPhone": "2547XXXXXXXX",
  "mobileMoneySP": "M-PESA",
  "externalId": "your-unique-payout-id",
  "callbackUrl": "https://your.domain/callbacks/mpesa"
}
```

Minimum M-Pesa B2C amount is **10 KES**. Result is delivered to `callbackUrl` and is
queryable via [status](#4-status-query).

---

## 3. Callbacks (webhooks)

We POST `application/json` to your `callbackUrl` when a transaction reaches a final state.

```json
{
  "amount": 100,
  "currency": "KES",
  "externalId": "your-unique-id-123",
  "secureId": "mamlaka-internal-id",
  "transactionStatus": "COMPLETE",
  "transactionReport": "The service request is processed successfully.",
  "reference": "SLJ7XXXXXX"
}
```

| Field | Notes |
|---|---|
| `transactionStatus` | `COMPLETE` or `FAILED`. |
| `transactionReport` | Human-readable result description. |
| `externalId` | The reference **you** sent at initiation — match on this. |
| `secureId` | Mamlaka's internal id. |
| `reference` | **M-Pesa receipt number** on success (empty on failure). |

Respond `200 OK` to acknowledge. Non-200 is treated as a delivery failure.

### Verifying the signature

Every callback includes:

```
X-Mamlaka-Signature: sha256=<hex HMAC-SHA256(rawRequestBody, sharedSecret)>
```

The HMAC is computed over the **exact raw bytes** of the request body. Verify against the
shared secret Mamlaka issued you, using a constant-time comparison. Node.js example:

```js
const crypto = require('crypto')

function verifyMamlakaSignature(rawBody, header, secret) {
  const expected = 'sha256=' +
    crypto.createHmac('sha256', secret).update(rawBody).digest('hex')
  const a = Buffer.from(header || '', 'utf8')
  const b = Buffer.from(expected, 'utf8')
  return a.length === b.length && crypto.timingSafeEqual(a, b)
}

// Express: capture the raw body for verification
app.use('/callbacks/mpesa', express.raw({ type: '*/*' }))
app.post('/callbacks/mpesa', (req, res) => {
  const raw = req.body // Buffer
  if (!verifyMamlakaSignature(raw, req.get('X-Mamlaka-Signature'), process.env.MAMLAKA_CALLBACK_SECRET)) {
    return res.sendStatus(401)
  }
  const payload = JSON.parse(raw.toString('utf8'))
  // ... reconcile using payload.externalId + payload.reference ...
  res.sendStatus(200)
})
```

Compute the HMAC over the body **as received**, before any JSON re-serialization —
re-stringifying can change bytes and break the signature.

---

## 4. Status query

`GET /api/v1/transaction?reference=<externalId>`

Use this to poll a transaction instead of querying Safaricom.

- `reference` — matches any of: **your `externalId`**, Mamlaka `secureId`,
  `merchantRequestID`, or `checkoutRequestID`. Query by **`externalId`** — it's the id you
  control and always have.
- `merchant` (optional) — if supplied it must **exactly** equal the `impalaMerchantId` the
  transaction was booked under. If unsure, **omit it** and match on `reference` alone.
- The **M-Pesa receipt number is NOT a lookup key** — do not pass it as `reference`.

```sh
curl -s "https://payments.mamlakapsp.com/api/v1/transaction?reference=your-unique-id-123"
```

```json
{
  "transaction": {
    "impalaMerchantId": "your-merchant-id",
    "transaction_status": "COMPLETE",
    "transaction_report": "The service request is processed successfully.",
    "currency": "KES",
    "amount": 100,
    "secure_id": "mamlaka-internal-id",
    "external_id": "your-unique-id-123",
    "callback_url": "https://your.domain/callbacks/mpesa",
    "date_added": 1787138041
  }
}
```

`404 Transaction not found` means the `reference`/`merchant` combination matched nothing —
almost always a wrong `merchant` value or querying by the receipt number. Retry with just
`?reference=<externalId>`.

> The status response returns the final state but **not** the M-Pesa receipt. The receipt
> (`reference`) is delivered on the **callback**. Persist it from there.

---

## Reconciliation — recommended

1. Treat the **signed callback** as the source of truth; store `externalId` → `reference`
   (receipt) + `transactionStatus`.
2. If a callback is ever missed, **poll `GET /api/v1/transaction?reference=<externalId>`**.
3. **Never** call Safaricom's TSQ directly for these transactions — it will report
   "transaction not found" because the transaction belongs to Mamlaka's short code.
