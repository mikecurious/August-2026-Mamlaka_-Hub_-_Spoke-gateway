# Sandbox environment

A second, isolated instance of the gateway running alongside production on the
same host (`52.204.175.2`), for testing changes against production-shaped data
without touching production or live money rails.

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
| Background crons | running | **disabled** (`DISABLE_CRONS=1`) |
| Airtel rail | `production` | `uat` |

The sandbox MySQL user is granted on `impala_gateway_sandbox` only — it cannot
read or write the production database. Its password is on the host at
`/home/ubuntu/.sandbox_db_pw` (mode 600) and in the systemd unit's `DB_DSN`.

## How isolation is implemented

Three environment variables were added to the code. **Each defaults to the
previous hard-coded value**, so production behaves identically with them unset:

| Variable | Default (= production) | Sandbox value |
|---|---|---|
| `DB_DSN` | prod `impala_gateway` DSN | sandbox DSN |
| `LISTEN_ADDR` | `127.0.0.1:8090` | `127.0.0.1:8091` |
| `DISABLE_CRONS` | unset — crons run | `1` — crons skipped |

`DISABLE_CRONS` is the safety-critical one. `StartPesalinkPayoutStatusCron()`
and `StartAirtelReconciler()` query live payment rails about rows in the
database and fire signed callbacks at merchant URLs. The sandbox runs on a
**copy of production data**, so leaving them on would re-notify real merchants
about real transactions. They are gated off.

The sandbox also gets its own `CALLBACK_SIGNING_SECRET` values, distinct from
production's, so a callback signed by sandbox cannot be mistaken for a
production one.

## Refreshing sandbox data from production

```bash
sudo mysqldump --single-transaction --quick --lock-tables=false --no-tablespaces \
  impala_gateway | gzip > ~/sandbox-seed-impala_gateway.sql.gz
zcat ~/sandbox-seed-impala_gateway.sql.gz | grep -ciE '^(USE |CREATE DATABASE)'   # must be 0
zcat ~/sandbox-seed-impala_gateway.sql.gz | sudo mysql impala_gateway_sandbox
```

The `grep` check matters: a dump containing `USE`/`CREATE DATABASE` would ignore
the target database argument and overwrite production tables. Do not skip it.

## Deploying a change to sandbox

```bash
cd /home/ubuntu/sandbox.payments.mamlakapsp.com
git fetch mine && git checkout <branch>
export PATH=$PATH:/usr/local/go/bin
go build -o merchant-api-sandbox .
sudo systemctl restart merchant-api-sandbox
sudo journalctl -u merchant-api-sandbox -n 50 --no-pager
```

Confirm after every restart that the log contains
`DISABLE_CRONS=1: skipping ...` and `listening on 127.0.0.1:8091`.

## Verifying the two are still isolated

```bash
sudo mysql -e 'SELECT user, db, COUNT(*) FROM information_schema.processlist
               WHERE user IN ("sandbox","colls") GROUP BY user, db;'
```

`colls` must appear only against `impala_gateway`, `sandbox` only against
`impala_gateway_sandbox`.

## Known outstanding items

1. **DNS** — `sandbox.payments.mamlakapsp.com` resolves to two A records in
   Cloudflare: `52.204.175.2` (this host, serves the sandbox) and
   `52.204.58.147` (a different host, returns 404 for this name). Remove the
   `52.204.58.147` record so the name resolves only here.
2. **TLS** — no certificate yet; the vhost is HTTP-only. After the DNS record is
   corrected, run
   `sudo certbot --nginx -d sandbox.payments.mamlakapsp.com`.
   Certbot's HTTP-01 challenge cannot succeed while the name round-robins to a
   host that does not serve it.
3. **Provider credentials** — the sandbox currently carries a copy of
   production's `.env`. Airtel is pointed at UAT, but the M-Pesa credentials are
   live Daraja production apps: an STK push or B2C call made against the sandbox
   will move real money. Swap in Daraja sandbox apps per brand before treating
   this as a safe place to test payments.
4. **Secrets in git history** — `.env` is tracked in this repository, so live
   credentials are in the history. Rotation, not deletion, is what fixes that.
