package mpesa

import "strings"

// C2BCredentials holds M-Pesa C2B OAuth credentials (paybill/STK channel).
type C2BCredentials struct {
	ConsumerKey    string
	ConsumerSecret string
}

// ResolveC2BCredentials returns C2B consumer key/secret for the merchant (used for OAuth + SFC verify).
func ResolveC2BCredentials(merchantID string) C2BCredentials {
	switch {
	case merchantID == "vukaPay_production":
		return C2BCredentials{VukaC2BConsumerKey, VukaC2BConsumerSecret}
	case merchantID == "crayfinance", merchantID == "ncgames_sandbox":
		return C2BCredentials{CrayC2BConsumerKey, CrayC2BConsumerSecret}
	case merchantID == "app":
		return C2BCredentials{AppC2BConsumerKey, AppC2BConsumerSecret}
	case merchantID == "transactworld":
		return C2BCredentials{TWDC2BConsumerKey, TWDC2BConsumerSecret}
	case strings.EqualFold(merchantID, "lipad"):
		return C2BCredentials{LipadC2BConsumerKey, LipadC2BConsumerSecret}
	case strings.EqualFold(merchantID, "shilingibet"):
		return C2BCredentials{ShilingiBetC2BConsumerKey, ShilingiBetC2BConsumerSecret}
	case strings.EqualFold(merchantID, "888starz_production"), strings.EqualFold(merchantID, "kalokalo"), strings.EqualFold(merchantID, "prime_sandbox"):
		return C2BCredentials{NeonC2BConsumerKey, NeonC2BConsumerSecret}
	default:
		return C2BCredentials{AppC2BConsumerKey, AppC2BConsumerSecret}
	}
}
