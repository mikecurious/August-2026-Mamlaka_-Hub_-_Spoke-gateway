# Sandbox environment

A second instance of the gateway running alongside production on the same host
(`52.204.175.2`), so merchants can integrate and test before going live.

**It is deliberately not a fake environment.** There are no sandbox paybills, so
the sandbox uses the *same live provider credentials as production*. Only two
things differ: the **base URL** and the **database**. A merchant who has
integrated against sandbox moves to production by changing the URL and nothing
else.

The consequence is that **transactions on sandbox move real money**. An STK push
or a payout initiated against the sandbox is a real M-Pesa or Airtel
transaction. Test with small amounts.

Every transaction the sandbox creates is labelled `environment: "sandbox"` in
the database, in API responses and in merchant callbacks, so sandbox activity is
always distinguishable from production activity.

## What is where

| | Production | Sandbox |
|---|---|---|
| Base URL | `https://payments.mamlakapsp.com` | `https://sandbox.payments.mamlakapsp.com` |
| Checkout | `/home/ubuntu/payments.mam-laka.com` | `/home/ubuntu/sandbox.payments.mamlakapsp.com` |
| Binary | `merchant-api-fix` | `merchant-api-sandbox` |
| systemd unit | `merchant-api.service` | `merchant-api-sandbox.service` |
| Port | `127.0.0.1:8090` | `127.0.0.1:8091` |
| Database | `impala_gateway` (user `colls`) | `impala_gateway_sandbox` (user `sandbox`) |
| nginx vhost | `/etc/nginx/sites-available/payments.mamlakapsp.com` | `/etc/nginx/sites-available/sandbox.payments.mamlakapsp.com` |
| Provider credentials | live | **same live credentials** |
| M-Pesa / Airtel rails | live | **live** |
| Transaction label | `production` | `sandbox` |
| Background reconcilers | running, production rows | running, **sandbox rows only** |
| Merchant callbacks | delivered | **delivered** |

The sandbox MySQL user is granted on `impala_gateway_sandbox` only — it cannot
read or write the production database. Its password is on the host at
`/home/ubuntu/.sandbox_db_pw` (mode 600) and in the systemd unit's `DB_DSN`.

## How isolation is implemented

Four environment variables. **Each defaults to the previous hard-coded
behaviour**, so production is unaffected when they are unset:

| Variable | Default (= production) | Sandbox value |
|---|---|---|
| `DB_DSN` | prod `impala_gateway` DSN | sandbox DSN |
| `LISTEN_ADDR` | `127.0.0.1:8090` | `127.0.0.1:8091` |
| `DISABLE_CRONS` | unset — crons run | unset — crons run (kill switch: `1`) |
| `APP_ENV` | unset — labels rows `production` | `sandbox` |
| `AIRTEL_RECON_ENABLED` | unset — reconciler off | `true` |

### Why the reconcilers can run on a copy of production

`StartPesalinkPayoutStatusCron()` and `StartAirtelReconciler()` query live
payment rails about pending rows and then fire callbacks at merchant URLs. The
sandbox database is a **copy of production**, so an unscoped reconciler there
would chase production's pending transactions and re-notify real merchants.

Both queries are therefore wrapped in `transactions.ScopeToEnvironment()`: the
sandbox only ever reconciles rows it created itself, and production also matches
rows written before the `environment` column existed, so no existing row is
stranded when this reaches production.

That scoping — not an off switch — is what makes the sandbox safe. `DISABLE_CRONS=1`
remains as a kill switch but is not set on the sandbox, because the reconciler is
currently the **only** way a sandbox Airtel collection reaches a terminal state
(see the callback note below).

## Transaction labelling

`APP_ENV` is read once at startup and stamped onto every transaction the
process creates, via a `BeforeCreate` hook on `TransactionModel`
(`transactions/model.go`). The hook lives on the model rather than at the ~20
call sites that insert transactions, so every rail — M-Pesa, Airtel, Pesalink,
card, and any added later — is covered without touching any of them.

It is `BeforeCreate`, not `BeforeSave`: a provider callback that later updates a
row must not relabel it. The label reflects where the transaction was *created*.

The label surfaces in three places:

- **Database** — `merchant_transactions.environment` (indexed, so you can filter
  in SQL: `WHERE environment = 'sandbox'`).
- **API responses** — an `environment` field on the transaction object.
- **Merchant callbacks** — an `environment` field in the webhook payload. Added
  centrally in `SendCallback` (the single exit point all 24 call sites use), so
  every rail is covered; the two handlers that POST directly rather than through
  `SendCallback` stamp it themselves. The Flutterwave *forward* POST is not a
  merchant callback and is left alone.

Rows created before the column existed read back as `production`
(`EnvironmentLabel()`), so nothing is silently mislabelled.

### Backfill after first deploy

`AutoMigrate` adds the column on startup, leaving existing rows blank. On the
**sandbox** database, the existing rows are copies of production rows, so label
them honestly:

```sql
UPDATE impala_gateway_sandbox.merchant_transactions
SET environment = 'production'
WHERE environment IS NULL OR environment = '';
```

Run the equivalent on the production database only when this code is deployed
there. (Not required — blank already reads as `production`.)

## Sandbox dashboard

`http://sandbox.merchants-dashboard.mamlakapsp.com` — the merchant portal
running against the sandbox database, so sandbox transactions are visible in
the same UI merchants already know.

| | Production | Sandbox |
|---|---|---|
| Host name | `merchants-dashboard.mamlakapsp.com` | `sandbox.merchants-dashboard.mamlakapsp.com` |
| Portal API | `dashboard-api` :9091 | `dashboard-api-sandbox` :9092 |
| Frontend | `dashboard-fe` :3000 | `dashboard-fe-sandbox` :3001 |
| Database | `impala_gateway` | `impala_gateway_sandbox` |
| Redis DB | `0` | `1` |

**Neither component needed a code change.** Both already read their config from
the environment, and both resolve a `.env` only for keys that are not already
set — so the systemd unit's `Environment=` lines win. Each sandbox unit reuses
the *same working directory and binary* as its production counterpart and
differs only by environment:

- `dashboard-api-sandbox` — `SERVER_ADDR=127.0.0.1:9092`, `DATABASE_DSN` → sandbox DB, `REDIS_DB=1`.
- `dashboard-fe-sandbox` — `PORT=3001`, `API_URL=http://127.0.0.1:9092/api`, `AUTH_URL` → the sandbox host.

`REDIS_DB=1` matters: production uses database `0` on the same Redis instance,
and sharing it would let sandbox and production serve each other cached data.

`AUTH_URL` matters too — NextAuth builds its callback URLs from it, so a wrong
value breaks login. It is set to `https://sandbox.merchants-dashboard.mamlakapsp.com`,
matching the issued certificate. If the host ever changes, change this with it.

Logins work with existing credentials because the user tables came across in the
database copy. Note that the copy also brings production's transaction history,
so the sandbox dashboard shows historical production rows (labelled
`production`) alongside new sandbox ones. Filter on `environment` if you want
only sandbox activity.

## Callback delivery on sandbox

Merchant callbacks are delivered from sandbox exactly as they are from
production — same code path, same signing, same retry behaviour. A merchant
integrating against sandbox receives real webhooks.

**Inbound provider callbacks now work too.** `CALLBACK_BASE_URL` is
`https://sandbox.payments.mamlakapsp.com`, which resolves to this host and
serves a valid certificate, so a provider posting a result callback to the
sandbox reaches it.

The Airtel reconciler stays enabled (`AIRTEL_RECON_ENABLED=true`, though it is
off in production) as a safety net: Airtel delivers its C2B result to a URL the
gateway does not control, so a sandbox collection could still hang `PENDING`
waiting for a callback that never arrives. The reconciler asks Airtel directly.

### Callback signatures differ between sandbox and production — deliberately

Callbacks are signed `X-Mamlaka-Signature: sha256=<hex(HMAC-SHA256(body, secret))>`.
`callbackSigningSecret()` in `merchants/aux.go` resolves the secret **from the
environment, not the database**:

1. `CALLBACK_SIGNING_SECRET_<MERCHANTID>` — a dedicated per-merchant secret
   (merchant id uppercased, non-alphanumerics become `_`). Only `LIPAD` has one.
2. `CALLBACK_SIGNING_SECRET` — the platform fallback, used by **every other
   merchant**.

Two consequences worth understanding:

- **The fallback is shared.** Every merchant without a dedicated secret verifies
  against the same value, so handing it to one merchant hands them the secret
  that signs other merchants' callbacks. Issue a dedicated
  `CALLBACK_SIGNING_SECRET_<MERCHANTID>` before giving a merchant the secret.
- **Sandbox and production differ.** The sandbox unit carries its own
  `CALLBACK_SIGNING_SECRET` / `_LIPAD`, deliberately not production's, so a
  sandbox callback can never carry a signature that validates as production. A
  merchant on the fallback path therefore needs a different verification secret
  per environment — tell them at onboarding. Do not "simplify" this by copying
  production's value into the sandbox unit; that is the property being
  protected.

Settlement via the reconciler goes through the same idempotent
`settleAirtelTransaction` as the callback path, so a late-arriving callback
cannot double-credit a row the reconciler already settled.

## The sandbox carries no production transaction history

The sandbox was seeded from a production dump, then its **financial history was
deliberately cleared** so the sandbox dashboard shows only sandbox activity and
no real customer data (phone numbers, amounts, settlements) sits in it.

Cleared: `merchant_transactions`, `transactions`, `settlement_requests`,
`withdrawal_requests`, `withdrawal_logs`, `platform_earning_models`,
`platform_fee_requests`, `finance_fee_adjustments`, `refund_requests`,
`manual_settlements`, `card_transactions`, `mobile_bulk_payments`,
`bank_bulk_payments`, `balance_topups`, `access_tokens`, `notifications`.

Kept, because the sandbox is unusable without them: `merchants`, `users`,
`merchant_access`, `merchant_balances`, `merchant_collection_balance`, roles,
currencies and forex rates. These still contain real merchant names, emails and
password hashes — treat sandbox database access as production-sensitive.

**The refresh procedure below reimports all of it.** If you re-seed, re-run the
clear afterwards or the sandbox dashboard will show production history again.

## Refreshing sandbox data from production

```bash
sudo mysqldump --single-transaction --quick --lock-tables=false --no-tablespaces \
  impala_gateway | gzip > ~/sandbox-seed-impala_gateway.sql.gz
zcat ~/sandbox-seed-impala_gateway.sql.gz | grep -ciE '^(USE |CREATE DATABASE)'   # must be 0
zcat ~/sandbox-seed-impala_gateway.sql.gz | sudo mysql impala_gateway_sandbox
```

The `grep` check matters: a dump containing `USE`/`CREATE DATABASE` would ignore
the target database argument and overwrite production tables. Do not skip it.

Re-run the backfill above afterwards — a refresh reimports unlabelled prod rows.

## Deploying a change to sandbox

```bash
cd /home/ubuntu/sandbox.payments.mamlakapsp.com
git fetch mine && git checkout <branch>
export PATH=$PATH:/usr/local/go/bin
go build -o merchant-api-sandbox .
sudo systemctl restart merchant-api-sandbox
sudo journalctl -u merchant-api-sandbox -n 50 --no-pager
```

Never build to `-o merchant-api-fix`, and never build inside the production
checkout — that directory's binary is what production runs.

### Back up `.env` before moving a host onto this branch — once

`.env` used to be tracked and no longer is (commit `499d393`). Git deletes a
file that was tracked in the old commit and untracked in the new one, so the
**first** checkout/reset that crosses that commit removes `.env` from the
working directory and the service loses its credentials on restart. This is a
one-time transition, but it applies to every host, **including production**:

```bash
cp -p .env ~/.env.bak.$(date +%F)          # BEFORE
git fetch mine && git reset --hard mine/sandbox/env-configurable
[ -f .env ] || cp -p ~/.env.bak.$(date +%F) .env   # AFTER -- restore if removed
```

Afterwards `.gitignore` keeps `.env` out, and later pulls leave it alone.

Untracking does **not** remove the credentials already in git history — that
still needs rotation (see outstanding items).

Confirm after every restart that the log contains
`DISABLE_CRONS=1: skipping ...` and `listening on 127.0.0.1:8091`.

## Verifying the two are still isolated

```bash
sudo mysql -e 'SELECT user, db, COUNT(*) FROM information_schema.processlist
               WHERE user IN ("sandbox","colls") GROUP BY user, db;'
```

`colls` must appear only against `impala_gateway`, `sandbox` only against
`impala_gateway_sandbox`.

To confirm labelling is live:

```bash
sudo mysql -e 'SELECT environment, COUNT(*) FROM
               impala_gateway_sandbox.merchant_transactions GROUP BY environment;'
```

## Known outstanding items

1. **Secrets in git history** — `.env` is tracked in this repository, so live
   credentials are in the history. Rotation, not deletion, is what fixes that.

Both sandbox hosts now serve HTTPS with Let's Encrypt certificates that certbot
renews automatically:

| Host | Certificate |
|---|---|
| `sandbox.payments.mamlakapsp.com` | issued, HTTP redirects to HTTPS |
| `sandbox.merchants-dashboard.mamlakapsp.com` | issued, HTTP redirects to HTTPS |
