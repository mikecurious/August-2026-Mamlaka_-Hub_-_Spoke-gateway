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
| Host name | `payments.mamlakapsp.com` | `sandbox.payments.mamlakapsp.com` |
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
value breaks login. **It is currently `http://...`; change it to `https://...`
when TLS is issued.**

Logins work with existing credentials because the user tables came across in the
database copy. Note that the copy also brings production's transaction history,
so the sandbox dashboard shows historical production rows (labelled
`production`) alongside new sandbox ones. Filter on `environment` if you want
only sandbox activity.

## Callback delivery on sandbox

Merchant callbacks are delivered from sandbox exactly as they are from
production — same code path, same signing, same retry behaviour. A merchant
integrating against sandbox receives real webhooks.

**Inbound provider callbacks are a different matter.** `CALLBACK_BASE_URL` on
sandbox is `https://sandbox.payments.mamlakapsp.com`, which providers cannot
reach until the DNS record is corrected and TLS is issued (see outstanding
items). Until then, an Airtel or M-Pesa result callback aimed at sandbox will
not arrive, and the **Airtel reconciler is the only path** by which a sandbox
Airtel collection reaches a terminal state. That is why it is enabled here
(`AIRTEL_RECON_ENABLED=true`) though it is off in production.

Settlement via the reconciler goes through the same idempotent
`settleAirtelTransaction` as the callback path, so a late-arriving callback
cannot double-credit a row the reconciler already settled.

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

1. **DNS** — `sandbox.payments.mamlakapsp.com` resolves to two A records in
   Cloudflare: `52.204.175.2` (this host, serves the sandbox) and
   `52.204.58.147` (a different host, returns 404 for this name). Remove the
   `52.204.58.147` record so the name resolves only here.
   (`sandbox.merchants-dashboard.mamlakapsp.com` is already correct — a single
   A record to this host.)
2. **TLS** — neither sandbox vhost has a certificate; both are HTTP-only.
   The dashboard can be issued immediately, its DNS is already correct:
   ```
   sudo certbot --nginx -d sandbox.merchants-dashboard.mamlakapsp.com
   sudo certbot --nginx -d sandbox.payments.mamlakapsp.com   # after the DNS fix above
   ```
   Certbot's HTTP-01 challenge cannot succeed for the gateway while that name
   round-robins to a host that does not serve it. After issuing the dashboard
   certificate, update `AUTH_URL` in `dashboard-fe-sandbox.service` to `https://`
   and restart it, or login will break.
3. **Callback signing secrets** — per-merchant secrets come from the database,
   which is a copy of production, so those already match. The *platform fallback*
   secret is currently sandbox-specific, so a merchant on the fallback path gets
   a different signature on sandbox than on production. Decide whether to keep
   them distinct (safer) or match production (zero-friction migration).
4. **Secrets in git history** — `.env` is tracked in this repository, so live
   credentials are in the history. Rotation, not deletion, is what fixes that.
