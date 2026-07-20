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

// B2CCredentials holds credentials required for B2C payouts and transaction-status queries.
type B2CCredentials struct {
	ConsumerKey        string
	ConsumerSecret     string
	BusinessShortCode  string
	InitiatorName      string
	SecurityCredential string
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

func ResolveB2CCredentials(merchantID string) B2CCredentials {
	switch {
	case merchantID == "vukaPay_production":
		return B2CCredentials{VukaPayB2CConsumerKey, VukaPayB2CConsumerSecret, VukaPayB2CShortCode, VukaPayB2CInitiatorName, VukaPayB2CPassword}
	case merchantID == "crayfinance", merchantID == "ncgames_sandbox":
		return B2CCredentials{CrayPayB2CConsumerKey, CrayPayB2CConsumerSecret, CrayPayB2CShortCode, CrayPayB2CInitiatorName, CrayPayB2CPassword}
	case merchantID == "app":
		return B2CCredentials{AppPayB2CConsumerKey, AppPayB2CConsumerSecret, AppPayB2CShortCode, AppPayB2CInitiatorName, AppPayB2CPassword}
	case merchantID == "transactworld":
		return B2CCredentials{TWDPayB2CConsumerKey, TWDPayB2CConsumerSecret, TWDPayB2CShortCode, TWDPayB2CInitiatorName, TWDPayB2CPassword}
	case strings.EqualFold(merchantID, "lipad"):
		return B2CCredentials{LipadPayB2CConsumerKey, LipadPayB2CConsumerSecret, LipadPayB2CShortCode, LipadPayB2CInitiatorName, LipadPayB2CPassword}
	case strings.EqualFold(merchantID, "shilingibet"):
		return B2CCredentials{ShilingiBetB2CConsumerKey, ShilingiBetB2CConsumerSecret, ShilingiBetB2CShortCode, ShilingiBetB2CInitiatorName, ShilingiBetB2CPassword}
	case strings.EqualFold(merchantID, "888starz_production"), strings.EqualFold(merchantID, "kalokalo"), strings.EqualFold(merchantID, "prime_sandbox"):
		return B2CCredentials{NeonPayB2CConsumerKey, NeonPayB2CConsumerSecret, NeonPayB2CShortCode, NeonPayB2CInitiatorName, NeonPayB2CPassword}
	default:
		return B2CCredentials{AppPayB2CConsumerKey, AppPayB2CConsumerSecret, AppPayB2CShortCode, AppPayB2CInitiatorName, AppPayB2CPassword}
	}
}
