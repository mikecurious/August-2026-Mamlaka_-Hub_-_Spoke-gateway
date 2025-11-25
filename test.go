package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// -------------------------------
// Response Struct
// -------------------------------
type PayazaResponse struct {
	ResponseCode                    string `json:"response_code"`
	ResponseMessage                 string `json:"response_message"`
	RequiresOTP                     bool   `json:"requires_otp"`
	OTPLength                       int    `json:"otp_length"`
	BeforePaymentInstruction        string `json:"before_payment_instruction"`
	AfterPaymentInstruction         string `json:"after_payment_instruction"`
	PaymentToken                    string `json:"payment_token"`
	Payee                           string `json:"payee"`
	PaymentMethod                   string `json:"payment_method"`
	TransactionChannel              string `json:"transaction_channel"`
	TransactionReference            string `json:"transaction_reference"`
	RedirectCustomerToURLProcessing bool   `json:"redirect_customer_to_url_processing"`
}

// success = response_code == "09"
func (r PayazaResponse) IsSuccess() bool {
	return r.ResponseCode == "09"
}

// -------------------------------
// Payload Struct
// -------------------------------
type PayazaPayload struct {
	Amount                 int    `json:"amount"`
	CustomerNumber         string `json:"customer_number"`
	TransactionReference   string `json:"transaction_reference"`
	TransactionDescription string `json:"transaction_description"`
	CustomerBankCode       string `json:"customer_bank_code"`
	CurrencyCode           string `json:"currency_code"`
	CustomerEmail          string `json:"customer_email"`
	CustomerFirstName      string `json:"customer_first_name"`
	CustomerLastName       string `json:"customer_last_name"`
	CustomerPhoneNumber    string `json:"customer_phone_number"`
	CountryCode            string `json:"country_code"`
}

// -------------------------------------
// Send Payaza API Request
// -------------------------------------
func SendPayazaRequest(payload PayazaPayload) (PayazaResponse, error) {

	url := "https://api.payaza.africa/live/subsidiary/collections/v1/process-collection"

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return PayazaResponse{}, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return PayazaResponse{}, err
	}

	// Load values from ENV
	tenant := os.Getenv("PAYAZA_TENANT_ID")
	productID := os.Getenv("PAYAZA_PRODUCT_ID")
	auth := os.Getenv("PAYAZA_AUTH")

	req.Header.Set("X-TenantID", tenant)
	req.Header.Set("X-ProductID", productID)
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return PayazaResponse{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return PayazaResponse{}, err
	}

	var payResp PayazaResponse
	err = json.Unmarshal(body, &payResp)
	if err != nil {
		return PayazaResponse{}, err
	}
	fmt.Println("Raw Payaza Response:", string(body))

	return payResp, nil
}

// -------------------------------
// Get country code from phone number
// -------------------------------

// DetectCountryCode returns ISO country code for all African phone prefixes
func DetectCountryCode(phone string) string {
	if len(phone) < 2 {
		return ""
	}

	// Most African prefixes are 2–3 digits (E.164)
	prefix2 := phone[:2]
	prefix3 := ""
	if len(phone) >= 3 {
		prefix3 = phone[:3]
	}

	countryMap := map[string]string{
		"213": "DZ", // Algeria
		"244": "AO", // Angola
		"229": "BJ", // Benin
		"267": "BW", // Botswana
		"226": "BF", // Burkina Faso
		"257": "BI", // Burundi
		"238": "CV", // Cape Verde
		"237": "CM", // Cameroon
		"236": "CF", // Central African Republic
		"235": "TD", // Chad
		"269": "KM", // Comoros
		"242": "CG", // Congo
		"243": "CD", // DRC
		"225": "CI", // Côte d'Ivoire
		"253": "DJ", // Djibouti
		"20":  "EG", // Egypt
		"240": "GQ", // Equatorial Guinea
		"291": "ER", // Eritrea
		"251": "ET", // Ethiopia
		"241": "GA", // Gabon
		"220": "GM", // Gambia
		"233": "GH", // Ghana
		"224": "GN", // Guinea
		"245": "GW", // Guinea-Bissau
		"254": "KE", // Kenya
		"266": "LS", // Lesotho
		"231": "LR", // Liberia
		"218": "LY", // Libya
		"261": "MG", // Madagascar
		"265": "MW", // Malawi
		"223": "ML", // Mali
		"222": "MR", // Mauritania
		"230": "MU", // Mauritius
		"212": "MA", // Morocco
		"258": "MZ", // Mozambique
		"264": "NA", // Namibia
		"227": "NE", // Niger
		"234": "NG", // Nigeria
		"250": "RW", // Rwanda
		"239": "ST", // Sao Tome and Principe
		"221": "SN", // Senegal
		"248": "SC", // Seychelles
		"232": "SL", // Sierra Leone
		"27":  "ZA", // South Africa
		"211": "SS", // South Sudan
		"249": "SD", // Sudan
		"268": "SZ", // Eswatini (Swaziland)
		"255": "TZ", // Tanzania
		"228": "TG", // Togo
		"216": "TN", // Tunisia
		"256": "UG", // Uganda
		"260": "ZM", // Zambia
		"263": "ZW", // Zimbabwe
	}

	// Try 3-digit match first
	if country, ok := countryMap[prefix3]; ok {
		return country
	}

	// Try 2-digit match
	if country, ok := countryMap[prefix2]; ok {
		return country
	}

	return ""
}

// -------------------------------
// get bank code
// -------------------------------
func GetBankCode(countryCode, network string) string {
	// Normalize network input
	network = strings.ToUpper(strings.TrimSpace(network))

	bankCodeMap := map[string]map[string]string{
		"BJ": { // Benin
			"MTN":  "MTNBEN",
			"MOOV": "MOOVBJ",
		},
		"NG": { // Nigeria (example)
			"MTN":     "MTNNGN",
			"GLO":     "GLOMGN",
			"AIRTEL":  "AIRTGN",
			"9MOBILE": "9MOBNG",
		},
	}

	if networks, ok := bankCodeMap[countryCode]; ok {
		if bankCode, ok := networks[network]; ok {
			return bankCode
		}
	}

	return ""
}

// -------------------------------
// Main Function
// -------------------------------
func main2() {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: .env file not found, using system env...")
	}

	payload := PayazaPayload{
		Amount:                 500,
		CustomerNumber:         "2290196289492",
		TransactionReference:   "TOO490001E7",
		TransactionDescription: "Test Payment",
		CustomerBankCode:       "MTNBEN",
		CurrencyCode:           "XOF",
		CustomerEmail:          "bigmaitre@blondmail.com",
		CustomerFirstName:      "Robert",
		CustomerLastName:       "Stones",
		CustomerPhoneNumber:    "2290196289492",
		CountryCode:            "BJ",
	}

	// get country code from phone
	payload.CountryCode = DetectCountryCode(payload.CustomerPhoneNumber)
	country := payload.CountryCode
	fmt.Println("Detected Country Code:", country)
	//get bank code from country and network
	payload.CustomerBankCode = GetBankCode(country, "MTN")
	fmt.Println("Determined Bank Code:", payload.CustomerBankCode)

	response, err := SendPayazaRequest(payload)
	if err != nil {
		fmt.Println("Request Error:", err)
		return
	}

	if response.IsSuccess() {
		fmt.Println("SUCCESS (PENDING):")
		fmt.Println("Transaction Ref:", response.TransactionReference)
		fmt.Println("Payment Token:", response.PaymentToken)
	} else {
		fmt.Println("FAILED / DECLINED:")
		fmt.Println("Code:", response.ResponseCode)
		fmt.Println("Message:", response.ResponseMessage)
	}
}
