# Pixel pending transaction sync (XOF / XAF / GMD)

Polls Pixel for transactions stuck in `PENDING` when the IPN callback was not received, then completes them and sends the merchant webhook (same logic as `POST /api/v1/west-africa/callback`).

## Authentication

Set on the server:

```bash
PIXEL_SYNC_CRON_SECRET=your-long-random-secret
PIXEL_CORE_API_KEY=...   # XOF, XAF, and most Pixel routes
PIXEL_GMD_API_KEY=...    # Gambia (GMD); also tried as fallback for other currencies
```

Request must include:

```
Authorization: Bearer <PIXEL_SYNC_CRON_SECRET>
```

or header:

```
X-Cron-Secret: <PIXEL_SYNC_CRON_SECRET>
```

## Endpoint

- **Method:** `POST`
- **URL:** `https://payments.mam-laka.com/api/v1/west-africa/sync-pending`
- **Body (optional JSON):**

```json
{
  "limit": 100,
  "minAgeSeconds": 60
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `limit` | 100 | Max rows per run (cap 500) |
| `minAgeSeconds` | 60 | Only sync transactions at least this old (avoids racing new payins) |

Query params `?limit=50&minAgeSeconds=120` override body values.

## What is scanned

```sql
currency IN ('XOF','XAF','GMD')
AND transactionStatus IN ('PENDING','pending')
AND merchantRequestID LIKE 'PIX_%'
```

Provider id used for Pixel: `merchantRequestID` (e.g. `PIX_37068773`).

## API keys

| Currency | Keys tried (in order) |
|----------|------------------------|
| GMD | `PIXEL_GMD_API_KEY`, then `PIXEL_CORE_API_KEY` |
| XOF, XAF | `PIXEL_CORE_API_KEY`, then `PIXEL_GMD_API_KEY` |

If Pixel returns `transaction not found` with one key, the next key is tried automatically.

## Pixel status API

`POST https://proxy-coreapi.pixelinnov.net/api_v1/transaction/status`

```json
{
  "api_key": "...",
  "transaction_ids": "PIX_37068773"
}
```

## Final states

| Pixel `state` | Action |
|---------------|--------|
| `SUCCESSFUL`, `SUCCESS`, `COMPLETED` | Credit collection / complete payout, merchant callback |
| `FAILED`, `FAILURE`, `CANCELLED` | Mark failed, refund payout if needed, merchant callback |
| Other (e.g. `PENDING1`) | Left pending; counted in `stillPending` |

## Response

```json
{
  "status": "success",
  "message": "Pixel pending transaction sync completed",
  "result": {
    "scanned": 12,
    "completed": 3,
    "failed": 1,
    "stillPending": 7,
    "notFound": 1,
    "skipped": 0,
    "errors": []
  }
}
```

## Log files (server working directory)

| File | Contents |
|------|----------|
| `pixel_sync.log` | Each run: started, per-txn status, completed/failed |
| `pixel_not_found.log` | `PIX_*` ids not found on Pixel after trying all keys |

## Google Apps Script (every 5 minutes)

```javascript
function pixelPendingSync() {
  var secret = 'YOUR_PIXEL_SYNC_CRON_SECRET';
  var url = 'https://payments.mam-laka.com/api/v1/west-africa/sync-pending';
  var res = UrlFetchApp.fetch(url, {
    method: 'post',
    muteHttpExceptions: true,
    headers: {
      'Authorization': 'Bearer ' + secret,
      'Content-Type': 'application/json'
    },
    payload: JSON.stringify({ limit: 100, minAgeSeconds: 60 })
  });
  Logger.log(res.getResponseCode() + ' ' + res.getContentText());
}
```

Trigger: time-driven, every 5 or 10 minutes.
