# Airtel Money Integration Plan

Integrating the standalone **`mam-laka-airtel`** service into the Mamlaka gateway
(`merchant-api`, module `com.mam-laka/main`) as a first-class mobile-money rail,
a sibling to `mpesa/` (KES reference) and `westafrica/` (Pixel aggregator).

Paths in this doc are `file:line` anchors in one of two repos:
- **Airtel repo**: `/Users/michaelbrian/Mamlaka/mam-laka-airtel`
- **Gateway**: `/Users/michaelbrian/Code/merchant-api`

---

## 1. What `mam-laka-airtel` provides today

A **standalone, mature, tested Go service** (`module airtel-gateway`, Go 1.25.4)
for **Airtel Money Kenya** — KES only, every request hardcodes `X-Country: KE` /
`X-Currency: KES` (`handlers/payments.go:124-133,156-157,295-296`). It uses the
Go **stdlib `net/http`** router (not gin) and owns a **MongoDB v2 ledger** for
its own transaction + callback audit trail. It also bundles an unrelated
**Comviva PreTUPS airtime** product (XML/HTTP) — out of scope here (see §11).

### Airtel API surface it already implements

| Concern | Method + Airtel path | Code |
| --- | --- | --- |
| OAuth2 token | `POST /auth/oauth2/token` (JSON `client_credentials`) | `services/airtel.go:30-107` |
| Collection (STK / C2B) | `POST /merchant/v1/payments/` | `handlers/payments.go:144` |
| Disbursement (B2C / B2B) | `POST /standard/v2/disbursements/` | `handlers/payments.go:284` |
| Merchant balance | `GET /standard/v1/users/balance` | `services/airtel.go:137` |
| Inbound collection callback | (Airtel → us) | `handlers/payments.go:505` |
| Inbound disbursement callback | (Airtel → us) | `handlers/payments.go:510` |

- **Base URL** resolves from `AIRTEL_ENV` (`uat`→`https://openapiuat.airtelkenya.com`,
  `live`→`https://openapi.airtelkenya.com`) or explicit `AIRTEL_BASE_URL`
  (`config/config.go:15-26`). Default is **UAT** — a fail-open default we must
  not carry over (see §7).
- **Token caching** with a scaled refresh margin (`services/airtel.go:112-126`),
  in-process `sync.Mutex`. Good; portable as-is.
- **MSISDN normalization**: strips `+`, `254`, leading `0` → bare 9-digit
  subscriber number (`handlers/payments.go:350-357`).
- **PIN encryption for B2C** (`handlers/payments.go:672-719`): prefers
  pre-encrypted `AIRTEL_ENCRYPTED_PIN`; falls back to RSA PKCS1v15 over
  `AIRTEL_B2C_PIN` with `AIRTEL_B2C_PUBLIC_KEY`.
- **Status codes**: Airtel `TS`→SUCCESS, `TF`→FAILED, everything else
  (`TA`,`TIP`,…) stays PENDING (`handlers/payments.go:362-371`).
- **Disbursements can settle synchronously** — the sync response may already
  carry `TS`/`TF`; a 2xx with no status means the money did **not** move and the
  row is FAILED (`buildDisbursementUpdate`, `handlers/payments.go:418-430`).
  Collections are only an *acknowledgement*; they stay PENDING for the callback
  (`buildCollectionUpdate`, `handlers/payments.go:436-450`).
- **Inbound callback auth**: HMAC-SHA256 over the **raw `transaction` object
  bytes**, keyed with `AIRTEL_CALLBACK_PRIVATE_KEY`, base64-compared
  (`verifyCallbackHash`, `handlers/payments.go:635-657`). **Fail-open**: when the
  key is unset, callbacks are accepted unverified (`:533-537`). Must become
  fail-closed in the gateway (§6).
- Callback envelope shape: `{ transaction: { id, message, status_code,
  airtel_money_id }, hash }` (`AirtelCallback`, `handlers/payments.go:494-502`).
  Note the collection `id` echoed back is the **`reference` / `transaction.id`
  we sent** (`:132`, `:273`) — this is our correlation key.

### Maturity

Complete and self-consistent for UAT. Tests are **pure-function + stubbed
Airtel**, run with no Mongo and no creds (`README` "Tests"). Never run against
Airtel production (consistent with the project's "never on mainnet" posture for
new rails). Two **must-fix-on-port** issues:
1. **Token logged in clear** — `services/airtel.go:80` logs the full token
   response body (the access token) and `client_id`. Violates the gateway's
   "secrets never in logs" invariant.
2. **No API auth** — the standalone relies on network isolation (`README`
   "Deployment"). Irrelevant once ported (the gateway's middleware/token gating
   applies), but the initiate/disburse *library* funcs must not be exposed
   except through the gateway handlers.

---

## 2. Gap analysis — standalone vs. what the gateway needs

| Capability | Standalone | Gateway needs | Action |
| --- | --- | --- | --- |
| Airtel token + request builders | ✅ stdlib | Library funcs (no HTTP handlers) | **Port** into `airtel/` package |
| Persistence | Own Mongo ledger | `merchant_transactions` (gorm/MySQL) | **Drop Mongo**; reuse `transactions.TransactionModel` |
| Merchant callback out | ❌ ("owns its ledger") | Signed `SendCallback` to `callbackUrl` | **Add** via existing `SendCallback` |
| Balance accounting | ❌ | Per-currency collection/payout balances | **Add** via `balances` (KES) |
| Inbound callback auth | Fail-open HMAC | Fail-closed HMAC / allowlist | **Harden** |
| Config | Fail-open UAT default | Fail-closed, no fallbacks | **Harden** to mpesa's `requireEnv` style |
| Routing/dedupe/idempotency | reference-unique index | `externalId` dedupe + `callbackStatus<>SENT` guard | **Reuse** gateway patterns |
| HTTP framework | `net/http` + `"GET /path"` mux | gin | **Reshape** — handlers become gin, mux patterns dropped |
| Status enquiry (Airtel Money) | ❌ (only airtime polls) | Reconcile stale PENDING | **Phase 2**, endpoint TBD from Airtel docs |

Net: port the **Airtel client logic** (token, payload builders, PIN encryption,
MSISDN normalize, status mapping, hash verify) as a **pure library**; rebuild
persistence, balances, callbacks, config and routing on the **gateway's existing
conventions**. Nothing in the portable code needs Go 1.25 once the `"GET /path"`
mux patterns are gone — target the workspace's `go 1.23.x`.

---

## 3. Where Airtel slots in

### 3.1 New package `airtel/` (workspace module)

Create `merchant-api/airtel/` mirroring `mpesa/`:
- `go.mod` → `module com.mam-laka/airtel`, `go 1.23.x` (match `go.work`, not 1.25).
- Add `./airtel` to the `use (...)` list in `go.work`.
- Files:
  - `airtel/config.go` — fail-closed env resolution (`requireEnv`-style, §7).
  - `airtel/token.go` — port `GetValidAccessToken` + `refreshMargin`
    (`services/airtel.go`), **with the token log removed**.
  - `airtel/collect.go` — `InitiateSTKPush(...) (*CollectResponse, error)`:
    the payload builder + POST from `handlers/payments.go:121-187`, minus Mongo
    and minus `http.ResponseWriter`.
  - `airtel/disburse.go` — `Disburse(...) (*DisburseResponse, error)`: the
    builder + PIN encryption + POST from `handlers/payments.go:191-333`.
  - `airtel/pin.go` — port `EncryptPIN` (`handlers/payments.go:672-719`).
  - `airtel/callback.go` — port `AirtelCallback` struct, `verifyCallbackHash`,
    and `mapAirtelTransactionStatus` (pure helpers the gateway handlers call).
  - `airtel/msisdn.go` — port `normalizeMSISDN` (or reuse the gateway's
    `normalizeKenyaMobileMSISDN`, `routes.go:1296` — see open question §10).

The package exposes **typed request/response structs and functions**, matching
how `mpesa.StkPushForBrand` (`mpesa/credentials.go:275`) and
`mpesa.GenerateB2CRequestForBrand` (`:296`) are called from `routes.go`. No gin,
no DB, no globals beyond the token cache.

### 3.2 Routing — by `mobileMoneySP`, scoped to KES

Both handlers switch on `req.Currency`, and **KES currently routes to M-Pesa
unconditionally**. Airtel Money is *also* KES, so currency alone cannot
discriminate — route on **`mobileMoneySP` inside the KES branch**.

- **Collections** — `MobilePaymentHandler`, `routes.go:1295` (`if req.Currency
  == "KES"`): at the top of this branch, after `normalizeKenyaMobileMSISDN`,
  branch on `normalizeAirtelSP(req.MobileMoneySP)`:
  - `AIRTEL` / `AIRTELMONEY` → new Airtel collection path (§4).
  - else → existing M-Pesa STK path (`:1314-1373`) unchanged.
- **Withdrawals** — `MobileWithdrawalHandler`, `routes.go:1932` (`case "KES"`):
  same SP branch before the M-Pesa `GenerateB2CRequestForBrand` chain
  (`:1971-2003`).

**Reconciling existing AIRTEL handling — no conflict:**
- `routes.go:1096` (`if req.MobileMoneySP != "AIRTEL"`) is inside
  `AirtimeDisbursementHandler` (`:1082`) — **airtime top-up** via the airtime
  provider (`sendAirtelAirtime`, `:960`), a *different product* on a different
  endpoint (`POST mobile/airtime`, `:8888`). Not Airtel Money. Leave untouched.
- `payaza/payaza.go:203,214` maps `"AIRTEL"` → `AIRUGA`/`AIRTGN` for **Uganda /
  other markets**, reached from the `UGX`/other-currency branches. This plan
  **only** claims `currency==KES && SP==AIRTEL`; AIRTEL under any other currency
  is explicitly untouched.

Add a small helper in `merchants/`:
```go
func isAirtelSP(sp string) bool {
    s := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(sp), "-", ""))
    return s == "AIRTEL" || s == "AIRTELMONEY"
}
```

---

## 4. Collections flow (pay-in / STK)

Mirror the M-Pesa KES path (`routes.go:1295-1373` + persistence `:1797-1830`) and
its callback (`MobileCallbackHandler`, `routes.go:3120`).

**Request mapping** (`MobilePaymentRequest`, `merchants/model.go:90-102`):
`impalaMerchantId`, `amount` (int), `payerPhone`, `externalId`, `callbackUrl`,
`mobileMoneySP`. Reused validation up-front is currency-agnostic and already
runs before the branch: `ExternalIDExists` dedupe (`:1234-1245`),
`GetUserByMerchantId` (`:1248`), `secureID := mpesa.GenerateSecureID()`
(`:1263`).

**Correlation key (critical).** Generate a gateway reference (see §10 — a
**sanitized/hex** value, not raw `externalId`, not base64url `secureID`) and pass
it to Airtel as **both** `reference` and `transaction.id`
(`handlers/payments.go:122,132`). Airtel echoes it back as
`callback.transaction.id` (`handlers/payments.go:569`). Store it in
`merchantRequestID` (or `checkoutRequestID`) so the callback lookup
(`GetTransactionByMerchantRequestID`) and the status query's OR-clause
(`GetTransactionByMerchantIDAndReference`, `transactions/model.go:288-304`) both
resolve it.

**Steps:**
1. `airtel.GetValidAccessToken()`; on failure `respondPaymentFailed`.
2. `airtel.InitiateSTKPush(msisdn, amount, reference)` → 2xx = prompt sent.
3. Persist a `transactions.TransactionModel` exactly like `routes.go:1797-1814`
   but with `SourceOfFunds: "AIRTEL"`, `MerchantRequestID: reference`,
   `TransactionReport: "collection"`, `TransactionStatus: "PENDING"`,
   `Currency: "KES"`, `CallbackURL: req.CallbackURL`. A non-2xx / parse-fail
   response → persist FAILED with the raw body (mirror `buildCollectionUpdate`).
4. Respond `{ "message": "...", "externalId", "secureId" }` (`:1822-1830`).

**Async settlement → signed merchant callback** (new gin handler,
`AirtelCollectionsCallbackHandler`, §6). On terminal success, inside a
`db.Transaction`, mirroring `MobileCallbackHandler` `routes.go:3227-3245`:
- Update the row to COMPLETE with the `id = ? AND transactionStatus <> COMPLETE
  AND callbackStatus <> SENT` idempotency guard (`:3229`).
- Set `providerReference = callback.transaction.airtel_money_id` (mirrors the
  M-Pesa `MpesaReceiptNumber` convention, `merchants/aux.go:89-91`).
- **Credit the collection balance**: `balances.MerchantCollectionBalance`
  `kesBalance += amount` (`routes.go:3239-3243`) — the pay-in balance, distinct
  from the payout `MerchantBalance`.
- `SendCallback(tx.ID, buildMpesaMerchantCallback(&tx, "COMPLETE", desc, amount,
  providerRef))` (`merchants/aux.go:135`) — signs with `X-Mamlaka-Signature`
  (`aux.go:173-177`) and marks `callbackStatus=SENT`.

---

## 5. Payouts flow (B2C / B2B disbursement)

Mirror the KES M-Pesa withdrawal (`routes.go:1932-2041`) and `B2CCallbackHandler`
(`routes.go:3541`). `MobileWithdrawalHandler` already Bearer-gates the whole
route (`routes.go:1837-1855`) and fetches the balance up-front (`:1921`).

**Request mapping** (`MobileWithdrawalRequest`, `merchants/model.go:70-79`):
`impalaMerchantId`, `amount` (float32), `recipientPhone`, `externalId`,
`callbackUrl`, `mobileMoneySP`.

**Balance model — follow the KES M-Pesa precedent, not UGX/XOF.** M-Pesa KES
does **not** deduct up-front; it checks sufficiency at initiation (`:1946-1953`)
and **deducts inside the callback on success**, in a gorm transaction
(`routes.go:3653-3679`). UGX/XOF/XAF deduct-then-refund (`:2054-2063`) because
those rails settle differently. Airtel Money shares `kesBalance`, so:
1. Sufficiency check against `balance.KESBalance` (`:1946`); insufficient →
   `{status:"FAILED", error:"INSUFFICIENT_BALANCE"}`.
2. Token gate: `airtel.GetValidAccessToken()`.
3. `airtel.Disburse(msisdn, amount, reference, type, name)` — `type` defaults to
   `B2C` (`resolveDisbursementType`, `handlers/payments.go:337-346`); PIN
   encryption inside (`EncryptPIN`).
4. Persist the PENDING row (`SourceOfFunds:"AIRTEL"`, `TransactionReport:
   "withdraw"`, `MerchantRequestID: reference`), like `routes.go:2011-2028`.
   Note `Amount` is stored as `int(req.Amount)` (`req.Amount` is `float32`,
   `routes.go:1903,2018`).

**Synchronous settlement is possible.** Unlike M-Pesa B2C (always async), Airtel
disbursements may return `TS`/`TF` in the sync body (`buildDisbursementUpdate`,
`handlers/payments.go:418-430`). Therefore implement **one idempotent settlement
function** used from *both* the initiate path (when the sync response is
terminal) and the disbursement callback:
```
settleAirtelPayout(reference, status, providerRef, desc):
  db.Transaction:
    UPDATE ... WHERE id=? AND transactionStatus<>'COMPLETE' AND callbackStatus<>'SENT'
    if RowsAffected==0 -> already settled (idempotent no-op)
    if status==COMPLETE: deduct kesBalance (MerchantBalance) by amount   // routes.go:3664
    SendCallback(...)  // signed, mark SENT
```
- On terminal **FAILED**, no deduction (`routes.go:3768`); send the FAILED
  signed callback.
- A `TA`/`TIP`/no-status sync response leaves the row PENDING for the callback.
- **Refund-on-failure** only applies if a design later deducts up-front; since we
  follow deduct-on-success, "refund" is simply *not deducting* on the FAILED
  branch — matching M-Pesa B2C exactly.

---

## 6. Airtel inbound callbacks / IPN

Add two gin handlers + routes (public, like the M-Pesa callbacks
`routes.go:8891-8892`):
```
router.POST("airtel/callback/collections",   AirtelCollectionsCallbackHandler)
router.POST("airtel/callback/disbursements", AirtelDisbursementCallbackHandler)
```
Both parse the `AirtelCallback` envelope (ported struct) and:
1. **Authenticate (fail-closed).** Verify the HMAC-SHA256 `hash` over the raw
   `transaction` bytes with `AIRTEL_CALLBACK_PRIVATE_KEY` (`verifyCallbackHash`,
   `handlers/payments.go:635-657`). Unlike the standalone's fail-open default
   (`:533-537`), **require the key when the rail is enabled** and **401 on a bad
   or missing hash** — a collections callback credits money (satisfies the
   gateway's "every inbound third-party callback is authenticated" invariant).
   If Airtel additionally IP-restricts, add a `TRUSTED_PROXY_IPS`-aware allowlist
   (confirm from Airtel — §10).
2. **Look up** the row by `callback.transaction.id` via
   `GetTransactionByMerchantRequestID` (`transactions/model.go:313`).
3. **Normalize status**: `mapAirtelTransactionStatus(status_code)` → `TS`→COMPLETE,
   `TF`→FAILED, else PENDING (leave un-downgraded — a non-terminal callback must
   never revert a settled row; see `buildCallbackSettlement`,
   `handlers/payments.go:596-598`).
4. **Settle** via the shared idempotent function (§4 for collections credit, §5
   for payout deduct) and fire the signed `SendCallback`.
5. Map identifiers: `airtel_money_id` → `providerReference`; keep the raw
   `message` in `responseDescription`.

Field → column mapping into `merchant_transactions`
(`transactions/model.go:13-38`):

| Airtel field | Column |
| --- | --- |
| our `reference` (= `transaction.id`) | `merchantRequestID` |
| `airtel_money_id` | `providerReference` |
| `status_code` (mapped) | `transactionStatus` |
| `message` | `responseDescription` |
| `secureID` (gateway) | `secureId` |
| `externalId` | `externalId` |

---

## 7. Config / secrets (fail-closed, no fallbacks)

New env vars, resolved through an `airtel.requireEnv`-style helper modeled on
`mpesa/credentials.go:69-79` (set-but-empty is a hard error naming the var):

| Var | Purpose | Rule |
| --- | --- | --- |
| `AIRTEL_CLIENT_ID` | OAuth2 client id | required when rail enabled |
| `AIRTEL_CLIENT_SECRET` | OAuth2 secret | required |
| `AIRTEL_ENV` | `uat`/`live` | **required — no silent UAT default** |
| `AIRTEL_BASE_URL` | explicit override | optional; if set, must be non-empty |
| `AIRTEL_ENCRYPTED_PIN` | pre-encrypted B2C PIN | required *or* the B2C_* pair below |
| `AIRTEL_B2C_PIN` / `AIRTEL_B2C_PUBLIC_KEY` | local RSA fallback | required if no `AIRTEL_ENCRYPTED_PIN` |
| `AIRTEL_CALLBACK_PRIVATE_KEY` | inbound HMAC key | **required — callback is fail-closed** |

Deviations from the standalone's defaults, to honour the gateway's security
invariants:
- **Drop the UAT default** in `config.AirtelBaseURL()` (`config/config.go:20-25`)
  — require explicit `AIRTEL_ENV`/`AIRTEL_BASE_URL`.
- **Callback key is mandatory** (standalone made it optional, `:533-537`).
- **Remove the token log** (`services/airtel.go:80`).
- Outbound merchant callback signing reuses the existing
  `CALLBACK_SIGNING_SECRET[_<MERCHANT>]` (`merchants/aux.go:118-133`) — nothing
  new. Add per-merchant Airtel brand overrides later only if a second Airtel
  short code appears (§10).

---

## 8. Persistence / balances

- **No new columns.** `transactions.TransactionModel`
  (`transactions/model.go:13-38`) already carries every field Airtel needs:
  `merchantRequestID`, `checkoutRequestID`, `providerReference`,
  `transactionStatus`, `callbackStatus`, `secureId`, `externalId`, `msisdn`,
  `recipientName`, `sourceOfFunds`, `responseCode`, `responseDescription`.
  Set `sourceOfFunds="AIRTEL"` to distinguish the rail (queryable via
  `SourceOfFunds` filter, `transactions/model.go:457-459`).
- **Balances (KES)**: collections credit `balances.MerchantCollectionBalance`
  (`routes.go:3239`); payouts deduct `balances.MerchantBalance`
  (`routes.go:3664`) via `kesBalance`. No new balance currency.
- **Idempotency**: reuse the `callbackStatus <> "SENT"` + `transactionStatus <>`
  guard (`routes.go:3229,3655,3732`) and the per-merchant `ExternalIDExists`
  dedupe (`:1234`).

---

## 9. Status query (`GET /api/v1/transaction`)

`GetTransactionHandler` (`routes.go:4227`) resolves any of `reference`,
`secureId`, `externalId` through `GetTransactionByMerchantIDAndReference`, whose
OR-clause matches `secureId | externalId | merchantRequestID | checkoutRequestID`
(`transactions/model.go:295-298`). Because we store the Airtel `reference` in
`merchantRequestID` and the gateway `secureID` in `secureId`, **Airtel
transactions are already findable with no handler change** — merchants query by
their `externalId`/`secureId` exactly as for M-Pesa.

A **Phase-2** Airtel Money *status enquiry* (to actively reconcile stale PENDING
rows, the way the M-Pesa sweepers do — `SyncPendingB2CWithdrawalsHandler`,
`routes.go:8905`) is **not present in the standalone** (it only polls *airtime*).
The Airtel Money transaction-status endpoint URL must be confirmed from Airtel's
developer docs before building it — do not invent it.

---

## 10. Open questions / risks

1. **Reference charset.** `secureID` is base64url (`mpesa.GenerateSecureID()`,
   `routes.go:1263`) containing `-`/`_`, which may violate
   Airtel's reference constraints. **Recommendation:** generate a dedicated
   sanitized/hex Airtel reference (globally unique) and store it in
   `merchantRequestID`. Do **not** reuse raw `externalId` — `ExternalIDExists`
   is *per-merchant* (`transactions/model.go:370-381`), so it is not globally
   unique across the Airtel ledger. **Confirm Airtel's `transaction.id` format
   rules.**
2. **UAT vs prod creds.** The service has never hit Airtel production
   (consistent with project posture). Sandbox go-live needs real
   `AIRTEL_CLIENT_ID/SECRET`, the correct `AIRTEL_ENCRYPTED_PIN` (Airtel support
   provides), and a portal-registered callback URL.
3. **Callback URL is portal-registered, not per-request.** The STK/disburse
   payloads carry **no** callback URL — Airtel invokes a URL configured per
   application in the developer portal. Deployment must register
   `/api/v1/airtel/callback/{collections,disbursements}` there, and we must
   confirm **whether Airtel IP-restricts** its callbacks (drives the §6
   allowlist decision). `req.CallbackURL` remains the *merchant's* URL for our
   outbound signed callback only.
4. **MSISDN normalizer choice.** The standalone's `normalizeMSISDN` strips to a
   bare 9-digit number (`handlers/payments.go:350-357`); the gateway's
   `normalizeKenyaMobileMSISDN` yields `2547XXXXXXXX` (`routes.go:1296`). Airtel
   wants the **bare** subscriber number — keep the ported `normalizeMSISDN` for
   the Airtel wire, but run the gateway validator first for input validation and
   for the stored `msisdn`.
5. **Module import cleanliness.** The standalone's `go.mod` (`airtel-gateway`,
   Go 1.25.4, `mongo-driver/v2`) does **not** import cleanly — Mongo is dropped
   and the module is renamed. The portable code is stdlib-only (`crypto/*`,
   `net/http`, `encoding/*`) → no new gateway dependencies, no `go.sum` churn.
   The `"GET /path"` mux patterns (Go 1.22+) are discarded with the handlers.
6. **Two-leg disbursement synchronicity.** Because a disbursement can settle
   synchronously *or* via callback, the shared idempotent `settleAirtelPayout`
   (§5) is essential — without it, a sync-`TS` plus a later callback would
   double-deduct. The `callbackStatus <> SENT` guard prevents this.
7. **Second Airtel short code.** If Airtel later provisions multiple
   collection/disbursement identities (as M-Pesa did with brands), generalise
   `airtel/config.go` to a brand key like `mpesa`'s `MPESA_<BRAND>_*`
   (`mpesa/credentials.go:63-64`). Start single-tenant.

---

## 11. PreTUPS airtime — out of scope

`mam-laka-airtel` also bundles a Comviva **PreTUPS airtime** product
(`pretups/pretups.go`, `handlers/airtime.go`, XML/HTTP, `PRETUPS_*` config). It
is a *different product* from Airtel Money and the gateway already routes
airtime elsewhere (`AirtimeDisbursementHandler` → `sendAirtelAirtime`,
`routes.go:1082,960`). **Do not port PreTUPS** as part of this integration.

---

## 12. Phased task list

**Phase 0 — scaffold (offline)**
1. Create `airtel/` module; add to `go.work`; `go.mod` at `go 1.23.x`.
2. Port token (`token.go`, **strip the token log**), config (`config.go`,
   fail-closed), PIN (`pin.go`), MSISDN + status/hash helpers, callback structs.
   Reshape STK/disburse into library funcs returning typed structs.
3. Port the standalone's pure-function unit tests
   (`handlers/payments_test.go`, `services/airtel_test.go`) into `airtel/`.
   → *Testable offline (stubbed HTTP).*

**Phase 1 — collections (needs Airtel sandbox)**
4. Add `isAirtelSP` + the SP branch in `MobilePaymentHandler` KES block
   (`routes.go:1295`); persist PENDING with `sourceOfFunds="AIRTEL"`.
5. Add `AirtelCollectionsCallbackHandler` + route; fail-closed HMAC; settle +
   credit `MerchantCollectionBalance` + signed `SendCallback`.
6. Register the collections callback URL in the Airtel portal; end-to-end STK in
   UAT. → *Needs sandbox + tunnel.*

**Phase 2 — payouts (needs Airtel sandbox + funded wallet)**
7. Add the SP branch in `MobileWithdrawalHandler` KES block (`routes.go:1932`);
   sufficiency check; `airtel.Disburse`.
8. Implement the shared idempotent `settleAirtelPayout` (deduct `MerchantBalance`
   on success); call it from both the sync response and
   `AirtelDisbursementCallbackHandler`.
9. Register the disbursements callback URL; end-to-end B2C in UAT.

**Phase 3 — reconciliation & ops**
10. Add an Airtel Money status-enquiry client (URL from Airtel docs) and wire the
    stale-PENDING sweeper, mirroring `SyncPendingB2CWithdrawalsHandler`.
11. `GET /api/v1/transaction` verified (no change expected); balance endpoint
    (`GET /standard/v1/users/balance`) exposed for ops if useful.
12. Prod credentials, `AIRTEL_ENV=live`, callback IP allowlist, load/security
    review.

**Dependency order:** 0 → 1 → 2 → 3. Phase 0 is fully offline; Phases 1-3 each
need Airtel UAT reachability + a public tunnel for callbacks.
