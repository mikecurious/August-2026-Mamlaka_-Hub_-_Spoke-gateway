# Till Payment API

## Initiate Till Payment

- **Method:** `POST`
- **URL:** `/api/v1/till/payment`
- **Auth:** `Authorization: Bearer <merchant-jwt>`

### Request Body

```json
{
  "impalaMerchantId": "app",
  "currency": "KES",
  "amount": "10",
  "creditAccount": "4509102",
  "narration": "TEST PAYMENT",
  "externalId": "B21E23KAJ126",
  "callbackUrl": "https://your-domain.com/payment-callback"
}
```

### Internal Behavior

- `transactionType` is hardcoded to `MPESAB2B_TILL`.
- `recieverPhoneNumber` is hardcoded to `254768899729`.
- `timestamp` is generated internally.
- `transactionReference` is generated internally (12 alphanumeric chars).
- `callBackUrl` and `errorCallBackUrl` sent to provider are internal:
  - `https://payments.mam-laka.com/api/v1/till/callback`
- KES balance is checked before initiation.
- KES deduction happens only on successful callback.

### Success Response (Our API)

```json
{
  "message": "Till payment initiated successfully. Await callback for final status.",
  "secureId": "AB12CD34EF56",
  "externalId": "B21E23KAJ126",
  "status": "pending"
}
```

---

## Provider Callback (CreditBank -> Our API)

- **Method:** `POST`
- **URL:** `/api/v1/till/callback`

### Sample Callback Payload

```json
{
  "ResultType": 0,
  "ResultCode": 0,
  "ResultDesc": "The service request is processed successfully.",
  "OriginatorConversationID": "fed2-4d7c-a178-8005140b24485451",
  "ConversationID": "AG_20260326_010020230n2qsuegn9dx",
  "TransactionID": "UCQSF8OSF3"
}
```

### Callback Handling

- Finds transaction by `OriginatorConversationID`.
- If `ResultCode == 0`:
  - marks transaction as success/complete,
  - deducts KES balance.
- Otherwise marks transaction as failed.
- Sends merchant callback to the original `callbackUrl`.

---

## Merchant Callback (Our API -> Merchant)

### Success

```json
{
  "transactionStatus": "COMPLETE",
  "transactionReport": "COMPLETE",
  "currency": "KES",
  "amount": 10,
  "secureId": "AB12CD34EF56",
  "externalId": "B21E23KAJ126"
}
```

### Failed

```json
{
  "transactionStatus": "FAILED",
  "transactionReport": "FAILED",
  "currency": "KES",
  "amount": 10,
  "secureId": "AB12CD34EF56",
  "externalId": "B21E23KAJ126",
  "reason": "The service request failed."
}
```

