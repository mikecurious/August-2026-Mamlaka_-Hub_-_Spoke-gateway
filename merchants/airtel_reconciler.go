package merchants

import (
	"log"
	"os"
	"strings"
	"time"

	"com.mam-laka/airtel"
	"com.mam-laka/database"
	"com.mam-laka/transactions"
)

// StartAirtelReconciler polls Airtel for the true status of Airtel collections
// that are still PENDING and settles the ones that have reached a terminal state.
//
// Why this exists: Airtel delivers its async C2B result callback to a URL we do
// not control (the standalone airtime service), so a collection that is ACCEPTED
// at push time would otherwise hang PENDING forever in the gateway. This job asks
// Airtel directly ("did this actually pay?") and settles the answer.
//
// Safety:
//   - disabled unless AIRTEL_RECON_ENABLED=true (off by default);
//   - settlement goes through the idempotent settleAirtelTransaction, so it can
//     never double-credit or downgrade a row already settled by a callback;
//   - only a DEFINITIVE TS/TF settles a row — TA (ambiguous), in-progress, and
//     any transport/routing error leave the row PENDING for the next tick.
func StartAirtelReconciler() {
	if strings.ToLower(strings.TrimSpace(os.Getenv("AIRTEL_RECON_ENABLED"))) != "true" {
		log.Println("airtel reconciler: disabled (set AIRTEL_RECON_ENABLED=true to enable)")
		return
	}

	const (
		interval = 60 * time.Second
		grace    = int64(120)            // ignore pushes younger than 2 min (still awaiting the customer)
		maxAge   = int64(7 * 24 * 3600)  // stop chasing rows older than 7 days
		batch    = 50
	)

	log.Printf("airtel reconciler: enabled (every %s, grace %ds, maxAge %ds, batch %d)", interval, grace, maxAge, batch)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			reconcileAirtelCollectionsOnce(grace, maxAge, batch)
		}
	}()
}

func reconcileAirtelCollectionsOnce(grace, maxAge int64, batch int) {
	db := database.GetConnection()
	now := time.Now().Unix()

	var rows []transactions.TransactionModel
	err := db.Model(&transactions.TransactionModel{}).
		Where("sourceOfFunds = ? AND transactionReport = ? AND transactionStatus = ?", "AIRTEL", "collection", "PENDING").
		Where("dateAdded <= ? AND dateAdded >= ?", now-grace, now-maxAge).
		Where("merchantRequestID <> ''").
		Order("id DESC").
		Limit(batch).
		Find(&rows).Error
	if err != nil {
		log.Printf("airtel reconciler: query failed: %v", err)
		return
	}
	if len(rows) == 0 {
		return
	}
	log.Printf("airtel reconciler: %d pending collection(s) to check", len(rows))

	for _, row := range rows {
		ref := strings.TrimSpace(row.MerchantRequestID)
		result, err := airtel.QueryCollectionStatus(ref)
		if err != nil {
			log.Printf("airtel reconciler: enquiry failed id=%d ref=%s: %v", row.ID, ref, err)
			continue
		}
		status := result.ReconcileTerminalStatus()
		if status == "" {
			// Still ambiguous / in-progress — leave PENDING, retry next tick.
			continue
		}
		if serr := settleAirtelTransaction(ref, status, result.AirtelMoneyID, result.Message); serr != nil {
			// Note: settleAirtelTransaction returns the merchant-callback error
			// even when the DB settlement itself succeeded, so this is not
			// necessarily a settlement failure — the row may well be settled and
			// only the outbound webhook failed (that path is logged separately).
			log.Printf("airtel reconciler: settle for id=%d ref=%s status=%s returned: %v", row.ID, ref, status, serr)
			continue
		}
		log.Printf("airtel reconciler: settled id=%d ref=%s -> %s", row.ID, ref, status)
	}
}
