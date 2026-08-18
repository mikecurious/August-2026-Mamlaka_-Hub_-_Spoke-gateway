package mpesa

import (
	"fmt"
	"os"
	"strings"
)

// Brand identifies a Safaricom credential set. Brands are business
// configuration, not secrets: the merchantID -> Brand mapping stays in code,
// while the credential values themselves are read from the environment.
type Brand string

const (
	BrandApp         Brand = "APP"
	BrandVuka        Brand = "VUKA"
	BrandCray        Brand = "CRAY"
	BrandTWD         Brand = "TWD"
	BrandLipad       Brand = "LIPAD"
	BrandShilingiBet Brand = "SHILINGIBET"
	BrandNeon        Brand = "NEON"
	BrandCheza       Brand = "CHEZAMONSTA"
)

// Environment variable scheme:
//
//	MPESA_<BRAND>_STK_CONSUMER_KEY
//	MPESA_<BRAND>_STK_CONSUMER_SECRET
//	MPESA_<BRAND>_STK_SHORTCODE
//	MPESA_<BRAND>_STK_PASSKEY
//	MPESA_<BRAND>_B2C_CONSUMER_KEY
//	MPESA_<BRAND>_B2C_CONSUMER_SECRET
//	MPESA_<BRAND>_B2C_SHORTCODE
//	MPESA_<BRAND>_B2C_INITIATOR_NAME
//	MPESA_<BRAND>_B2C_SECURITY_CREDENTIAL
//
// There are no fallbacks. A variable that is unset, or set to an empty or
// blank string, is a hard error naming the brand and the variable.

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

func envVarName(brand Brand, channel, field string) string {
	return fmt.Sprintf("MPESA_%s_%s_%s", string(brand), channel, field)
}

// requireEnv reads one credential value. Set-but-empty counts as missing.
// The error names the variable but never its value.
func requireEnv(brand Brand, channel, field string) (string, error) {
	name := envVarName(brand, channel, field)
	raw, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("mpesa: missing required environment variable %s (brand %s)", name, brand)
	}
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("mpesa: environment variable %s is set but empty (brand %s)", name, brand)
	}
	return raw, nil
}

// STKCredentialsForBrand loads a brand's STK/C2B collection credentials.
func STKCredentialsForBrand(brand Brand) (STKCredentials, error) {
	var creds STKCredentials
	var err error
	if creds.ConsumerKey, err = requireEnv(brand, "STK", "CONSUMER_KEY"); err != nil {
		return STKCredentials{}, err
	}
	if creds.ConsumerSecret, err = requireEnv(brand, "STK", "CONSUMER_SECRET"); err != nil {
		return STKCredentials{}, err
	}
	if creds.BusinessShortCode, err = requireEnv(brand, "STK", "SHORTCODE"); err != nil {
		return STKCredentials{}, err
	}
	if creds.PassKey, err = requireEnv(brand, "STK", "PASSKEY"); err != nil {
		return STKCredentials{}, err
	}
	return creds, nil
}

// B2CCredentialsForBrand loads a brand's B2C payout credentials.
func B2CCredentialsForBrand(brand Brand) (B2CCredentials, error) {
	var creds B2CCredentials
	var err error
	if creds.ConsumerKey, err = requireEnv(brand, "B2C", "CONSUMER_KEY"); err != nil {
		return B2CCredentials{}, err
	}
	if creds.ConsumerSecret, err = requireEnv(brand, "B2C", "CONSUMER_SECRET"); err != nil {
		return B2CCredentials{}, err
	}
	if creds.BusinessShortCode, err = requireEnv(brand, "B2C", "SHORTCODE"); err != nil {
		return B2CCredentials{}, err
	}
	if creds.InitiatorName, err = requireEnv(brand, "B2C", "INITIATOR_NAME"); err != nil {
		return B2CCredentials{}, err
	}
	if creds.SecurityCredential, err = requireEnv(brand, "B2C", "SECURITY_CREDENTIAL"); err != nil {
		return B2CCredentials{}, err
	}
	return creds, nil
}

// STKShortCodeForBrand returns a brand's collection paybill for logging only.
// A paybill is not a secret; callers that actually move money go through
// STKCredentialsForBrand and surface the missing-variable error there.
func STKShortCodeForBrand(brand Brand) string {
	return os.Getenv(envVarName(brand, "STK", "SHORTCODE"))
}

// envSafeMerchantID renders a merchant ID as an environment variable fragment:
// upper-cased, with every character outside [A-Z0-9] replaced by an underscore,
// since merchant IDs may contain characters env var names cannot.
func envSafeMerchantID(merchantID string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(merchantID) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// brandOverride lets a deployment route one merchant to a different credential
// brand without a code change, so the same binary can serve a partially
// migrated fleet — one box sending a merchant's traffic through a new paybill
// while another still uses the old one.
//
//	MPESA_BRAND_OVERRIDE_STK_<MERCHANT>   collections only
//	MPESA_BRAND_OVERRIDE_B2C_<MERCHANT>   payouts only
//	MPESA_BRAND_OVERRIDE_<MERCHANT>       both channels
//
// Channel-specific wins over the general form, which allows a merchant's
// collections to move before their payouts do. The value is returned as given:
// an unrecognised brand fails loudly at credential resolution rather than
// silently falling back to the default brand, which would route money to the
// wrong short code.
func brandOverride(merchantID, channel string) (Brand, bool) {
	id := envSafeMerchantID(merchantID)
	for _, name := range []string{
		"MPESA_BRAND_OVERRIDE_" + channel + "_" + id,
		"MPESA_BRAND_OVERRIDE_" + id,
	} {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return Brand(strings.ToUpper(v)), true
		}
	}
	return "", false
}

// STKBrandForMerchant maps a merchant ID to its collection brand.
// The matching semantics here (which cases use == and which use EqualFold)
// are load-bearing and must not be "tidied up".
func STKBrandForMerchant(merchantID string) Brand {
	if b, ok := brandOverride(merchantID, "STK"); ok {
		return b
	}
	switch {
	case strings.EqualFold(merchantID, "chezamonsta"):
		return BrandCheza
	case merchantID == "vukaPay_production":
		return BrandVuka
	case merchantID == "crayfinance", merchantID == "ncgames_sandbox":
		return BrandCray
	case merchantID == "app":
		return BrandApp
	case merchantID == "transactworld":
		return BrandTWD
	case strings.EqualFold(merchantID, "lipad"):
		return BrandLipad
	case strings.EqualFold(merchantID, "shilingibet"):
		return BrandShilingiBet
	case strings.EqualFold(merchantID, "888starz_production"), strings.EqualFold(merchantID, "kalokalo"), strings.EqualFold(merchantID, "prime_sandbox"):
		return BrandNeon
	default:
		return BrandApp
	}
}

// B2CBrandForMerchant maps a merchant ID to its payout brand.
func B2CBrandForMerchant(merchantID string) Brand {
	if b, ok := brandOverride(merchantID, "B2C"); ok {
		return b
	}
	switch {
	case strings.EqualFold(merchantID, "chezamonsta"):
		return BrandCheza
	case merchantID == "vukaPay_production":
		return BrandVuka
	case merchantID == "crayfinance", merchantID == "ncgames_sandbox":
		return BrandCray
	case merchantID == "app":
		return BrandApp
	case merchantID == "transactworld":
		return BrandTWD
	case strings.EqualFold(merchantID, "lipad"):
		return BrandLipad
	case strings.EqualFold(merchantID, "shilingibet"):
		return BrandShilingiBet
	case strings.EqualFold(merchantID, "888starz_production"), strings.EqualFold(merchantID, "kalokalo"), strings.EqualFold(merchantID, "prime_sandbox"):
		return BrandNeon
	default:
		return BrandApp
	}
}

// ResolveC2BCredentials returns C2B consumer key/secret for the merchant (used for OAuth + SFC verify).
func ResolveC2BCredentials(merchantID string) (C2BCredentials, error) {
	stk, err := ResolveSTKCredentials(merchantID)
	if err != nil {
		return C2BCredentials{}, fmt.Errorf("merchant %q: %w", merchantID, err)
	}
	return C2BCredentials{stk.ConsumerKey, stk.ConsumerSecret}, nil
}

// ResolveSTKCredentials returns the C2B/STK credentials for a merchant.
func ResolveSTKCredentials(merchantID string) (STKCredentials, error) {
	creds, err := STKCredentialsForBrand(STKBrandForMerchant(merchantID))
	if err != nil {
		return STKCredentials{}, fmt.Errorf("merchant %q: %w", merchantID, err)
	}
	return creds, nil
}

// ResolveB2CCredentials returns the B2C credentials for a merchant.
func ResolveB2CCredentials(merchantID string) (B2CCredentials, error) {
	creds, err := B2CCredentialsForBrand(B2CBrandForMerchant(merchantID))
	if err != nil {
		return B2CCredentials{}, fmt.Errorf("merchant %q: %w", merchantID, err)
	}
	return creds, nil
}

// StkPushForBrand is StkPush with the brand's credentials loaded from the
// environment. Credential arguments are passed by field name so the
// positional contract of StkPush cannot drift.
func StkPushForBrand(phoneNumber string, amount int, callbackURL, accountReference string, brand Brand) (*StkPushResponse, error) {
	creds, err := STKCredentialsForBrand(brand)
	if err != nil {
		return nil, err
	}
	return StkPush(
		phoneNumber,
		amount,
		callbackURL,
		accountReference,
		creds.ConsumerKey,
		creds.ConsumerSecret,
		creds.BusinessShortCode,
		creds.PassKey,
	)
}

// GenerateB2CRequestForBrand is GenerateB2CRequest with the brand's
// credentials loaded from the environment. Note that GenerateB2CRequest takes
// the security credential in its `password` parameter, which sits *before*
// businessShortCode and initiatorName.
func GenerateB2CRequestForBrand(phoneNumber string, amount float64, callbackURL, externalID, identifier string, brand Brand) (*B2BResponse, error) {
	creds, err := B2CCredentialsForBrand(brand)
	if err != nil {
		return nil, err
	}
	return GenerateB2CRequest(
		phoneNumber,
		amount,
		callbackURL,
		externalID,
		identifier,
		creds.ConsumerKey,
		creds.ConsumerSecret,
		creds.SecurityCredential,
		creds.BusinessShortCode,
		creds.InitiatorName,
	)
}
