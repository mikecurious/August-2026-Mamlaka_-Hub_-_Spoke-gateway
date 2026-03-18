# Bank Transfer (Payin) API

## Initiate bank transfer

**Endpoint:** `POST /api/v1/bank/payin`  
**Auth:** `Authorization: Bearer <your-jwt>`

### Request body

| Field          | Type   | Required | Description                                      |
|----------------|--------|----------|--------------------------------------------------|
| `externalId`   | string | Yes      | Your unique reference for this transaction       |
| `amount`       | int    | Yes      | Amount (e.g. 100)                                |
| `currency`     | string | Yes      | e.g. `NGN`                                       |
| `callbackUrl`  | string | Yes      | URL we will POST the result to when paid/failed  |
| `customerName` | string | Yes      | Payer name                                       |
| `customerEmail`| string | Yes      | Payer email                                      |
| `accountName`  | string | No       | Label shown to payer (default: "Payment")        |

### Example request

```json
{
  "externalId": "live0008990211932",
  "amount": 100,
  "currency": "NGN",
  "accountName": "Demo account",
  "callbackUrl": "https://your-app.com/callback",
  "customerName": "John Doe",
  "customerEmail": "johndoe@gmail.com"
}
```

### Example success response

```json
{
  "secureId": "PSJRU873PGPI",
  "externalId": "live0008990211932",
  "status": "pending",
  "message": "Bank transfer initiated. Customer should pay to the account below.",
  "amount": 100,
  "currency": "NGN",
  "bankAccountName": "Demo account",
  "bankAccountNumber": "5010875892",
  "bankName": "vfd",
  "bankCode": "566",
  "expiryDate": "2026-03-14T01:26:36.263Z"
}
```

Give the customer the bank details (account number, bank name/code, expiry) so they can complete the payment.

---

## Callback (when payment is done)

We send a `POST` request to your `callbackUrl` when the transfer succeeds or fails.

### Success callback

**Body:**

```json
{
  "amount": 100,
  "currency": "NGN",
  "externalId": "live0008990211932",
  "secureId": "PSJRU873PGPI",
  "transactionReport": "COMPLETE",
  "transactionStatus": "COMPLETE"
}
```

| Field              | Description                          |
|--------------------|--------------------------------------|
| `transactionStatus`| `COMPLETE` on success                |
| `transactionReport`| `COMPLETE` on success                |
| `secureId`         | Our internal reference               |
| `externalId`       | The reference you sent in the request |
| `amount`           | Original amount                      |
| `currency`         | e.g. `NGN`                           |

### Failed callback

On failure we send the same fields with `transactionStatus` and `transactionReport` set to `FAILED`, and an extra `reason` field with the failure message.
