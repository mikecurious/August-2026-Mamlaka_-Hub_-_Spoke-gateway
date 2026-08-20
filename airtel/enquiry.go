package airtel

import (
	"io"
	"log"
	"net/http"
	"time"
)

// QueryCollectionStatus enquires Airtel for the current status of a collection
// (C2B) by the reference we sent at initiation (transaction.id, our "MLK..." id).
//
// Kenya's collection enquiry lives on GET /merchant/v1/payments/{id} — the
// /standard/v1 path is rejected ("KE opco version [1.0] NOT_ALLOWED", ROUTER007)
// and the /standard/v2,v3 paths reject the bearer token ("Invalid authentication
// credentials"). Read-only: it never moves money.
func QueryCollectionStatus(reference string) (*Result, error) {
	base, err := BaseURL()
	if err != nil {
		return nil, err
	}
	token, err := GetValidAccessToken()
	if err != nil {
		return nil, err
	}
	return airtelGet(base+"/merchant/v1/payments/"+reference, token, reference)
}

func airtelGet(url, token, reference string) (*Result, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Country", "KE")
	req.Header.Set("X-Currency", "KES")

	log.Printf("airtel enquiry: url=%s reference=%s", url, reference)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	log.Printf("airtel enquiry response: url=%s reference=%s http_status=%d body=%s", url, reference, resp.StatusCode, string(respBytes))
	return parseResult(resp.StatusCode, respBytes), nil
}

// ReconcileTerminalStatus is the CONSERVATIVE mapping used by the reconciler.
// Unlike CollectionStatus (which fails-closed to FAILED on a non-2xx initiate),
// an enquiry that is anything other than a DEFINITIVE terminal code returns ""
// so the row is left PENDING and retried on the next tick. A transient network
// error, a ROUTER00x routing reply, or an ambiguous "TA" must never settle money.
//
//	TS  -> COMPLETE (customer paid; credit)
//	TF  -> FAILED
//	else-> "" (TA ambiguous, TIP in-progress, empty): leave PENDING, retry
func (r *Result) ReconcileTerminalStatus() string {
	if !r.parsedOK || !httpStatusOK(r.HTTPStatus) {
		return ""
	}
	switch r.StatusCode {
	case "TS":
		return "COMPLETE"
	case "TF":
		return "FAILED"
	default:
		return ""
	}
}
