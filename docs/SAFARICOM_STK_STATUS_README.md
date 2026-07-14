# Safaricom STK Status Issues

## Query ID

Use this ID when checking a pending STK transaction with Safaricom:

```text
CheckoutRequestID
```

Example:

```text
ws_CO_14072026160652563750921195
```

## Issues

| ID / Code | Safaricom Message | Meaning | Recommendation |
|---|---|---|---|
| `ResultCode: 0` | `The service request is processed successfully.` | Payment completed successfully. | Mark transaction as `COMPLETE`, update balance, send merchant callback. |
| `ResultCode: 1025` | `Error Occurred while sending push request.` | Safaricom failed while sending or processing the STK push. | Mark as `FAILED`, save Safaricom message in `responseDescription`, send failed callback. |
| `ResultCode: 2001` | `The initiator information is invalid.` | Shortcode/password/timestamp/credential mismatch. | Mark as `FAILED`, alert technical team, verify shortcode/passkey/password generation. |
| `ResultCode: 1` | `The balance is insufficient for the transaction.` | Customer had insufficient M-Pesa balance. | Mark as `FAILED`, send failed callback with Safaricom message. |
| `errorCode: 500.001.1001` | `The transaction does not Exist` | Safaricom cannot find the `CheckoutRequestID`. | Do not fail immediately. Retry status query, then fail only after retry limit or timeout window. |

## Recommendations

1. Before marking a pending transaction as failed, query Safaricom using `checkoutRequestID`.
2. Treat `ResponseCode: 0` as query acceptance only. Use `ResultCode` for the real transaction status.
3. Store the Safaricom query result in the transaction record:

```text
MerchantRequestID
CheckoutRequestID
ResultCode
ResultDesc
ResponseCode
ResponseDescription
```

4. For `500.001.1001`, retry first because Safaricom records can lag.
5. After the retry limit or 3-hour timeout, mark unresolved transactions as `FAILED` with a clear reason:

```text
Safaricom transaction not found after status query
```

6. The pending timeout flow should become:

```text
Find pending transaction
Query Safaricom with checkoutRequestID
Update status from ResultCode
Send merchant callback
Save Safaricom response for reporting
```
