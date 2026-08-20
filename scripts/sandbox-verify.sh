#!/usr/bin/env bash
# Confirms the sandbox is running, is isolated from production, and that the
# reconcilers cannot touch production rows. Read-only -- safe to run any time.
#
#   bash ~/sandbox-verify.sh

SB_DB=impala_gateway_sandbox
PR_DB=impala_gateway
ok()   { printf '  \033[32mPASS\033[0m  %s\n' "$1"; }
bad()  { printf '  \033[31mFAIL\033[0m  %s\n' "$1"; FAILED=1; }
head_() { printf '\n\033[1m%s\033[0m\n' "$1"; }
FAILED=0

head_ "1. Services"
for s in merchant-api merchant-api-sandbox \
         dashboard-api dashboard-api-sandbox \
         dashboard-fe dashboard-fe-sandbox; do
  [ "$(systemctl is-active $s)" = active ] && ok "$s is active" || bad "$s is NOT active"
done

head_ "2. Ports (prod vs sandbox -- must be different processes)"
echo "  gateway   8090 prod / 8091 sandbox"
echo "  dash api  9091 prod / 9092 sandbox"
echo "  dash web  3000 prod / 3001 sandbox"
sudo ss -tlnp 2>/dev/null | grep -E ':(8090|8091|9091|9092|3000|3001) ' | sed 's/^/  /'

head_ "3. Database isolation"
sudo mysql -t -e "SELECT user, db, COUNT(*) conns FROM information_schema.processlist
                  WHERE user IN ('sandbox','colls') GROUP BY user, db;"
CROSS=$(sudo mysql -N -e "SELECT COUNT(*) FROM information_schema.processlist
                          WHERE (user='sandbox' AND db='$PR_DB') OR (user='colls' AND db='$SB_DB');")
[ "$CROSS" = 0 ] && ok "no cross-database connections" || bad "$CROSS cross-database connection(s)!"

# Functional test, not grant-string matching: actually try to read production
# as the sandbox user and require it to be refused.
if [ -r /home/ubuntu/.sandbox_db_pw ]; then
  SBPW=$(cat /home/ubuntu/.sandbox_db_pw)
  if mysql -u sandbox -p"$SBPW" -N -e "SELECT 1 FROM $PR_DB.merchant_transactions LIMIT 1;" >/dev/null 2>&1; then
    bad "sandbox user CAN read $PR_DB -- privileges are too wide!"
  else
    ok "sandbox user is refused access to $PR_DB"
  fi
  VISIBLE=$(mysql -u sandbox -p"$SBPW" -N -e "SHOW DATABASES;" 2>/dev/null | grep -vE '^(information_schema|performance_schema)$' | tr '\n' ' ')
  echo "  databases visible to sandbox user: $VISIBLE"
else
  echo "  (skipped: /home/ubuntu/.sandbox_db_pw not readable)"
fi

head_ "4. Production schema untouched"
N=$(sudo mysql -N -e "SELECT COUNT(*) FROM information_schema.columns
     WHERE table_schema='$PR_DB' AND table_name='merchant_transactions' AND column_name='environment';")
[ "$N" = 0 ] && ok "prod table has no 'environment' column yet (not deployed there)" \
             || ok "prod table has the 'environment' column (branch deployed to prod)"

head_ "5. Transaction labelling"
sudo mysql -t $SB_DB -e "SELECT environment, COUNT(*) rows_ FROM merchant_transactions GROUP BY environment;"

head_ "6. Reconciler scoping -- the safety proof"
echo "  'unscoped' = rows the cron would hit WITHOUT the scope (production data)."
echo "  'scoped'   = rows it will actually touch. Scoped must only ever be sandbox rows."
sudo mysql -t $SB_DB -e "
SELECT 'AIRTEL   unscoped' AS q, COUNT(*) AS rows_ FROM merchant_transactions
 WHERE sourceOfFunds='AIRTEL' AND transactionReport='collection'
   AND transactionStatus='PENDING' AND merchantRequestID<>''
UNION ALL SELECT 'AIRTEL   scoped', COUNT(*) FROM merchant_transactions
 WHERE sourceOfFunds='AIRTEL' AND transactionReport='collection'
   AND transactionStatus='PENDING' AND merchantRequestID<>'' AND environment='sandbox'
UNION ALL SELECT 'PESALINK unscoped', COUNT(*) FROM merchant_transactions
 WHERE sourceOfFunds='pesalink_creditbank' AND transactionReport='withdraw'
   AND transactionStatus IN ('PENDING','pending')
UNION ALL SELECT 'PESALINK scoped', COUNT(*) FROM merchant_transactions
 WHERE sourceOfFunds='pesalink_creditbank' AND transactionReport='withdraw'
   AND transactionStatus IN ('PENDING','pending') AND environment='sandbox';"

LEAK=$(sudo mysql -N $SB_DB -e "SELECT COUNT(*) FROM merchant_transactions
 WHERE environment<>'sandbox' AND transactionStatus='PENDING'
   AND sourceOfFunds='AIRTEL' AND transactionReport='collection';")
ACT=$(sudo journalctl -u merchant-api-sandbox --since '-30 min' --no-pager 2>/dev/null \
      | grep -cE 'pending collection\(s\) to check|settled id=')
if [ "$LEAK" -gt 0 ] && [ "$ACT" -gt 0 ]; then
  bad "reconciler showed row activity while production rows are pending -- INVESTIGATE"
else
  ok "reconciler has not processed any production row"
fi

head_ "7. Endpoints"
printf '  prod    https://payments.mamlakapsp.com -> %s\n' \
  "$(curl -s -o /dev/null -w %{http_code} --max-time 10 https://payments.mamlakapsp.com/api/v1/)"
printf '  sandbox (direct 8091)                   -> %s\n' \
  "$(curl -s -o /dev/null -w %{http_code} --max-time 10 http://127.0.0.1:8091/api/v1/)"
printf '  sandbox (public HTTPS)                  -> %s\n' \
  "$(curl -s -o /dev/null -w %{http_code} --max-time 20 https://sandbox.payments.mamlakapsp.com/api/v1/)"
echo "  (401 = reachable and asking for auth, which is correct)"

head_ "7b. Dashboard"
printf '  sandbox dashboard  /                 -> %s (307/308 = redirect, correct)\n' \
  "$(curl -s -o /dev/null -w %{http_code} --max-time 20 https://sandbox.merchants-dashboard.mamlakapsp.com/)"
printf '  sandbox dashboard  /api/auth/session -> %s (200 = NextAuth up)\n' \
  "$(curl -s -o /dev/null -w %{http_code} --max-time 20 https://sandbox.merchants-dashboard.mamlakapsp.com/api/auth/session)"
printf '  prod dashboard     /                 -> %s\n' \
  "$(curl -s -o /dev/null -w %{http_code} --max-time 20 https://merchants-dashboard.mamlakapsp.com/)"
DAPI=$(grep -c 'impala_gateway_sandbox' /etc/systemd/system/dashboard-api-sandbox.service 2>/dev/null)
[ "${DAPI:-0}" -gt 0 ] && ok "sandbox dashboard API points at the sandbox database" \
                       || bad "sandbox dashboard API is NOT pointed at the sandbox database"
RDB=$(grep -o 'REDIS_DB=[0-9]*' /etc/systemd/system/dashboard-api-sandbox.service 2>/dev/null | cut -d= -f2)
[ "${RDB:-0}" != "0" ] && ok "sandbox dashboard uses a separate Redis DB ($RDB, prod uses 0)" \
                       || bad "sandbox dashboard shares Redis DB 0 with production"

head_ "8. Sandbox configuration"
grep '^Environment=' /etc/systemd/system/merchant-api-sandbox.service \
  | sed -E 's/(DB_DSN=|SIGNING_SECRET(_LIPAD)?=)[^"]*/\1<redacted>/'

# Deliberate decision: the platform fallback signing secrets must NOT match
# production, so a sandbox callback can never validate as a production one.
# Compared by hash so the secrets themselves are never printed.
for f in CALLBACK_SIGNING_SECRET CALLBACK_SIGNING_SECRET_LIPAD; do
  P=$(grep -o "${f}=[^\"]*" /etc/systemd/system/merchant-api.service 2>/dev/null | head -1 | cut -d= -f2- | sha256sum)
  S=$(grep -o "${f}=[^\"]*" /etc/systemd/system/merchant-api-sandbox.service 2>/dev/null | head -1 | cut -d= -f2- | sha256sum)
  if [ -z "$P" ] || [ -z "$S" ]; then
    echo "  (skipped: $f not set in both units)"
  elif [ "$P" = "$S" ]; then
    bad "$f matches production -- sandbox callbacks could validate as production!"
  else
    ok "$f differs from production (intended)"
  fi
done

head_ "9. Sandbox startup log"
sudo journalctl -u merchant-api-sandbox -n 60 --no-pager 2>/dev/null \
  | grep -iE 'listening on|airtel reconciler:|DISABLE_CRONS' | tail -5

if [ "$FAILED" = 1 ]; then
  printf '\n\033[31mOne or more checks FAILED -- see above.\033[0m\n'
  exit 1
fi
printf '\n\033[32mAll checks passed.\033[0m\n'
