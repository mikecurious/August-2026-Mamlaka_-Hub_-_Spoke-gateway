# Pesalink Payout API

## Initiate Pesalink Payout

- **Method:** `POST`
- **URL:** `/api/v1/bank/pesalink/payout`
- **Auth:** `Authorization: Bearer <merchant-jwt>`

### Request Body

```json
{
  "externalId": "ME00191",
  "amount": "10",
  "currency": "KES",
  "bankCode": "0057",
  "creditAccount": "00106534176150",
  "callbackUrl": "https://your-domain.com/payment-callback",
  "narration": "pesalink to John Doe"
}
```

### Internal Rules

- `direction` is hardcoded to `A2A`
- `transactionType` is hardcoded to `PESALINK`
- `beneficiaryName` is hardcoded to `mam-laka`
- `receiverPhoneNumber` is hardcoded to `254768899729`
- `transactionReference` is generated internally (12 chars)
- KES balance is checked before initiation
- Transaction is stored as `pending` and finalized by status sync

### Success Response (Our API)

```json
{
  "message": "Pesalink payout initiated successfully. Await final status update.",
  "secureId": "30GY1KKB2CSK",
  "externalId": "ME00191",
  "status": "pending"
}
```

---

## Provider Acceptance Response (Reference)

```json
{
  "RequestID": "CBL296929594011",
  "ResponseData": {
    "requestID": "CBL296929594011",
    "originalRequestID": "CBL296929594011",
    "status": "SUCCESS",
    "statusCode": "00",
    "statusDescription": "Request accepted successfully",
    "statusMessage": "Request accepted. IPSL processing the request. Final Response Via callbackURL",
    "errorCode": "000",
    "errorDesc": "Request Accepted and Processed By IPSL Successfully"
  }
}
```

---

## Pending Status Sync (Cron)

- Background job runs every 60s and checks pending transactions using:
  - `POST https://konnectapigateway.creditbank.co.ke/pesalink-payment-status-check`
  - payload: `{"originalRequestID":"<stored-original-id>"}`

- You can manually trigger one sync run:
  - **Method:** `POST`
  - **URL:** `/api/v1/bank/pesalink/sync`

### Success status from provider

```json
{
  "RequestID": "CBL659320103861",
  "ResponseData": {
    "originalRequestID": "CBL296929594011",
    "endToEndID": "012500572026033011193629594011",
    "status": "SUCCESS",
    "statusCode": "00",
    "statusDescription": "Request was processed successfully by IPSL",
    "statusMessage": "Request was processed successfully by IPSL",
    "errorCode": "ACCP"
  }
}
```

### Failed status from provider

```json
{
  "RequestID": "CBL540312932784",
  "ResponseData": {
    "originalRequestID": "CBL956880323625",
    "endToEndID": "012500472026033010575880323625",
    "status": "FAILED",
    "statusCode": "03",
    "errorCode": "RJCT"
  }
}
```

---

## Merchant Callback (Our API -> Merchant callbackUrl)

### Success

```json
{
  "transactionStatus": "COMPLETE",
  "transactionReport": "COMPLETE",
  "currency": "KES",
  "amount": 10,
  "secureId": "30GY1KKB2CSK",
  "externalId": "ME00191",
  "providerReference": "CBL296929594011"
}
```

### Failed

```json
{
  "transactionStatus": "FAILED",
  "transactionReport": "FAILED",
  "currency": "KES",
  "amount": 10,
  "secureId": "30GY1KKB2CSK",
  "externalId": "ME00191",
  "providerReference": "CBL296929594011",
  "reason": "Request failed"
}
```

