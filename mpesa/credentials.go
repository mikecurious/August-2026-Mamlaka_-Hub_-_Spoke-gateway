package mpesa

import "strings"

// C2BCredentials holds M-Pesa C2B OAuth credentials (paybill/STK channel).
type C2BCredentials struct {
	ConsumerKey    string
	ConsumerSecret string
}

// STKCredentials holds all credentials required for STK push and STK status query.
type STKCredentials struct {
	ConsumerKey       string
	ConsumerSecret    string
	BusinessShortCode string
	PassKey           string
}

// ResolveC2BCredentials returns C2B consumer key/secret for the merchant (used for OAuth + SFC verify).
func ResolveC2BCredentials(merchantID string) C2BCredentials {
	stk := ResolveSTKCredentials(merchantID)
	return C2BCredentials{stk.ConsumerKey, stk.ConsumerSecret}
}

// ResolveSTKCredentials returns the C2B/STK credentials for a merchant.
func ResolveSTKCredentials(merchantID string) STKCredentials {
	switch {
	case merchantID == "vukaPay_production":
		return STKCredentials{VukaC2BConsumerKey, VukaC2BConsumerSecret, VukaC2BBusinessShortCode, VukaC2BPassKey}
	case merchantID == "crayfinance", merchantID == "ncgames_sandbox":
		return STKCredentials{CrayC2BConsumerKey, CrayC2BConsumerSecret, CrayC2BBusinessShortCode, CrayC2BPassKey}
	case merchantID == "app":
		return STKCredentials{AppC2BConsumerKey, AppC2BConsumerSecret, AppC2BBusinessShortCode, AppC2BPassKey}
	case merchantID == "transactworld":
		return STKCredentials{TWDC2BConsumerKey, TWDC2BConsumerSecret, TWDC2BBusinessShortCode, TWDC2BPassKey}
	case strings.EqualFold(merchantID, "lipad"):
		return STKCredentials{LipadC2BConsumerKey, LipadC2BConsumerSecret, LipadC2BBusinessShortCode, LipadC2BPassKey}
	case strings.EqualFold(merchantID, "shilingibet"):
		return STKCredentials{ShilingiBetC2BConsumerKey, ShilingiBetC2BConsumerSecret, ShilingiBetC2BBusinessShortCode, ShilingiBetC2BPassKey}
	case strings.EqualFold(merchantID, "888starz_production"), strings.EqualFold(merchantID, "kalokalo"), strings.EqualFold(merchantID, "prime_sandbox"):
		return STKCredentials{NeonC2BConsumerKey, NeonC2BConsumerSecret, NeonC2BBusinessShortCode, NeonC2BPassKey}
	default:
		return STKCredentials{AppC2BConsumerKey, AppC2BConsumerSecret, AppC2BBusinessShortCode, AppC2BPassKey}
	}
}
