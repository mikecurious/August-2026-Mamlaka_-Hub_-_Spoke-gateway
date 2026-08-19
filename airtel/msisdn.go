package airtel

import "strings"

// NormalizeMSISDN strips the "+" prefix, the 254 country code and any leading
// zero so Airtel receives the bare 9-digit subscriber number (Airtel's wire
// format). Validation of the raw input is the caller's responsibility.
func NormalizeMSISDN(phoneNumber string) string {
	msisdn := strings.TrimSpace(phoneNumber)
	msisdn = strings.TrimPrefix(msisdn, "+")
	msisdn = strings.TrimPrefix(msisdn, "254")
	msisdn = strings.TrimPrefix(msisdn, "0")
	return msisdn
}
