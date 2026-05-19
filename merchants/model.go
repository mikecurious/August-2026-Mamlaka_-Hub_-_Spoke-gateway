package merchants

type STKResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

/*
 go run main.go
VgN8mZljJJg4ycWhVTKxeOrWEqRlResponse: {
            "ConversationID": "AG_20250201_206051c643ca6389833f",
            "OriginatorConversationID": "9021-4f51-8691-8604c07e13de14061111",
            "ResponseCode":"0",
            "ResponseDescription": "Accept the service request successfully."
        }

*/

type B2BResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

//WITHdrawal sample request

/*
	{
	    "impalaMerchantId":"{{username}}",
	    "currency":"KES",
	    "amount":10,
	    "recipientPhone":"254112299271",
	    "mobileMoneySP":"M-Pesa",
	    "externalId":"joeltest4",
	    "callbackUrl":""
	}
*/
type MobileWithdrawalRequest struct {
	ImpalaMerchantId string  `json:"impalaMerchantId" binding:"required"`
	Currency         string  `json:"currency" binding:"required"`
	Amount           float32 `json:"amount" binding:"required"`
	RecipientPhone   string  `json:"recipientPhone" binding:"required"`
	MobileMoneySP    string  `json:"mobileMoneySP" binding:"required"`
	ExternalID       string  `json:"externalId" binding:"required"`
	CallbackURL      string  `json:"callbackUrl" binding:"required"`
	OMOTP            int     `json:"om_otp"`
}

// bank-pesa link request struct containing the followign, amount, destinationAccount, destinationBankCode, customerName string)
type PesalinkBankTransferRequest struct {
	Amount              float32 `json:"amount"`
	DestinationAccount  string  `json:"destinationAccount"`
	DestinationBankCode string  `json:"destinationBankCode"`
	CustomerName        string  `json:"customerName"`
}

// MobilePaymentRequest structure to bind incoming JSON request
type MobilePaymentRequest struct {
	ImpalaMerchantId string `json:"impalaMerchantId" binding:"required"`
	Currency         string `json:"currency" binding:"required"`
	Amount           int    `json:"amount" binding:"required"`
	DisplayName      string `json:"displayName"`
	PayerPhone       string `json:"payerPhone" binding:"required"`
	MobileMoneySP    string `json:"mobileMoneySP" binding:"required"`
	ExternalID       string `json:"externalId" binding:"required"`
	CallbackURL      string `json:"callbackUrl" binding:"required"`
	OMOTP            string `json:"om_otp"`
	// TestTransaction, when true, applies a 10 KES max for non-lipad merchants on shared paybill 4130455 (default KES collection).
	TestTransaction bool `json:"testTransaction"`
}
type CardPaymentRequest struct {
	ImpalaMerchantId string  `json:"impalaMerchantId" binding:"required"`
	Currency         string  `json:"currency" binding:"required"`
	Amount           float32 `json:"amount" binding:"required"`
	MobileMoneySP    string  `json:"mobileMoneySP" binding:"required"`
	ExternalID       string  `json:"externalId" binding:"required"`
	CallbackURL      string  `json:"callbackUrl" binding:"required"`
	RedirectURL      string  `json:"redirectUrl"`
}

// to convert the merchant balances
type ConvertBalance struct {
	OriginCurrency      string  `json:"originCurrency" binding:"required"`
	DestinationCurrency string  `json:"destinationCurrency" binding:"required"`
	Amount              float64 `json:"amount" binding:"required"`
	ExchangeRate        float64 `json:"exchangeRate" binding:"required"`
}

// virtual card handler
type CreateCardHolderRequest struct {
	CardTypeID int    `json:"cardTypeId" binding:"required"`
	AreaCode   string `json:"areaCode" binding:"required"`
	Mobile     string `json:"mobile" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	FirstName  string `json:"firstName" binding:"required"`
	LastName   string `json:"lastName" binding:"required"`
	BirthDay   string `json:"birthDay" binding:"required"`
	Country    string `json:"country" binding:"required"`
	Town       string `json:"town" binding:"required"`
	Address    string `json:"address" binding:"required"`
	PostCode   string `json:"postCode" binding:"required"`
}

// west africa model
type WestAfricaCallbackResponse struct {
	TransactionID           string  `json:"transaction_id"`
	Amount                  int     `json:"amount"`
	Benefice                int     `json:"benefice"`
	Commission              int     `json:"comission"`
	Destination             string  `json:"destination"`
	Fee                     int     `json:"fee"`
	Response                string  `json:"response"`
	Error                   *string `json:"error"`
	ServiceID               int     `json:"service_id"`
	CustomerName            string  `json:"customer_name"`
	State                   string  `json:"state"`
	CustomData              string  `json:"custom_data"`
	IPNUrl                  string  `json:"ipn_url"`
	TransactionChannel      string  `json:"transaction_channel"`
	ProviderID              string  `json:"provider_id"`
	SMSLink                 int     `json:"sms_link"`
	CreatedAt               string  `json:"created_at"`
	UpdatedAt               string  `json:"updated_at"`
	IPNState                int     `json:"ipn_state"`
	WAmountAfterTransaction string  `json:"w_amount_after_transaction"`
	PLastWalletAmount       int     `json:"p_last_wallet_amount"`
	PNewWalletAmount        int     `json:"p_new_wallet_amount"`
	PID                     int     `json:"p_id"`
	Hash                    string  `json:"hash"`
	Currency                string  `json:"currency"`
}

// eCitizen structures
type ECitizenValidateRequest struct {
	RefNo       string  `json:"ref_no" binding:"required"`
	Currency    string  `json:"currency" binding:"required"`
	Amount      float32 `json:"amount" binding:"required"`
	CallbackURL string  `json:"callback_url" binding:"required"`
}

type ECitizenValidateResponse struct {
	Status string `json:"status"`
	Desc   string `json:"desc"`
	Data   struct {
		Name     string `json:"name"`
		Currency string `json:"currency"`
		Amount   string `json:"amount"`
	} `json:"data"`
}

type ECitizenConfirmRequest struct {
	RefNo                  string `json:"ref_no" binding:"required"`
	Amount                 int    `json:"amount" binding:"required"`
	Currency               string `json:"currency" binding:"required"`
	GatewayTransactionID   string `json:"gateway_transaction_id" binding:"required"`
	GatewayTransactionDate string `json:"gateway_transaction_date" binding:"required"`
	CustomerName           string `json:"customer_name" binding:"required"`
	CustomerAccountNumber  string `json:"customer_account_number" binding:"required"`
}

type ECitizenConfirmResponse struct {
	Status string `json:"status"`
	Desc   string `json:"desc"`
}

// Korapay structures
type KorapayPaymentRequest struct {
	ImpalaMerchantId string `json:"impalaMerchantId" binding:"required"`
	Currency         string `json:"currency" binding:"required"`
	Amount           int    `json:"amount" binding:"required"`
	CustomerName     string `json:"customerName" binding:"required"`
	CustomerEmail    string `json:"customerEmail" binding:"required,email"`
	PayerPhone       string `json:"payerPhone" binding:"required"`
	Description      string `json:"description" binding:"required"`
	ExternalID       string `json:"externalId" binding:"required"`
	CallbackURL      string `json:"callbackUrl" binding:"required"`
	RedirectURL      string `json:"redirectUrl" binding:"required"`
}

type KorapayCallbackRequest struct {
	Event string              `json:"event"`
	Data  KorapayCallbackData `json:"data"`
}

type KorapayCallbackData struct {
	Reference        string  `json:"reference"`
	PaymentReference string  `json:"payment_reference"`
	Currency         string  `json:"currency"`
	Amount           int     `json:"amount"`
	Fee              float64 `json:"fee"`
	PaymentMethod    string  `json:"payment_method"`
	Status           string  `json:"status"`
}

// KorapayBankPayinRequest is the request body for initiating a Korapay bank-transfer (payin).
type KorapayBankPayinRequest struct {
	ExternalID   string `json:"externalId" binding:"required"`
	Amount       int    `json:"amount" binding:"required,gt=0"`
	Currency     string `json:"currency" binding:"required"`
	AccountName  string `json:"accountName"` // Optional; displayed to payer (e.g. "Demo account")
	CallbackURL  string `json:"callbackUrl" binding:"required"`
	CustomerName string `json:"customerName" binding:"required"`
	CustomerEmail string `json:"customerEmail" binding:"required,email"`
}

// KorapayPayoutRequest is the request body for Korapay bank payout (disburse).
type KorapayPayoutRequest struct {
	ExternalID     string `json:"externalId" binding:"required"`
	Amount         string `json:"amount" binding:"required"`
	Currency       string `json:"currency" binding:"required"`
	Narration      string `json:"narration"`
	BankCode       string `json:"bankCode" binding:"required"`
	AccountNumber  string `json:"accountNumber" binding:"required"`
	CustomerName   string `json:"customerName" binding:"required"`
	CustomerEmail  string `json:"customerEmail" binding:"required,email"`
	CallbackURL    string `json:"callbackUrl" binding:"required"`
}

// TillPaymentRequest is the request body for CreditBank till payment.
type TillPaymentRequest struct {
	ImpalaMerchantId string `json:"impalaMerchantId" binding:"required"`
	Currency         string `json:"currency" binding:"required"`
	Amount           string `json:"amount" binding:"required"`
	CreditAccount    string `json:"creditAccount" binding:"required"`
	Narration        string `json:"narration"`
	ExternalID       string `json:"externalId" binding:"required"`
	CallbackURL      string `json:"callbackUrl" binding:"required"`
}

type PesalinkPayoutRequest struct {
	ExternalID    string `json:"externalId" binding:"required"`
	Amount        string `json:"amount" binding:"required"`
	Currency      string `json:"currency" binding:"required"`
	BankCode      string `json:"bankCode" binding:"required"`
	CreditAccount string `json:"creditAccount" binding:"required"`
	CallbackURL   string `json:"callbackUrl" binding:"required"`
	Narration     string `json:"narration"`
}

// Flutterwave structures
type FlutterwavePaymentRequest struct {
	ImpalaMerchantId string `json:"impalaMerchantId" binding:"required"`
	Currency         string `json:"currency" binding:"required"`
	Amount           int    `json:"amount" binding:"required"`
	CustomerEmail    string `json:"customerEmail" binding:"required,email"`
	PayerPhone       string `json:"payerPhone" binding:"required"`
	ExternalID       string `json:"externalId" binding:"required"`
	CallbackURL      string `json:"callbackUrl" binding:"required"`
	RedirectURL      string `json:"redirectUrl" binding:"required"`
}

type FlutterwaveCallbackRequest struct {
	Event     string                  `json:"event"`
	Data      FlutterwaveCallbackData `json:"data"`
	EventType string                  `json:"event.type"`
}

type FlutterwaveCallbackData struct {
	ID                int                 `json:"id"`
	TxRef             string              `json:"tx_ref"`
	FlwRef            string              `json:"flw_ref"`
	DeviceFingerprint string              `json:"device_fingerprint"`
	Amount            int                 `json:"amount"`
	Currency          string              `json:"currency"`
	ChargedAmount     int                 `json:"charged_amount"`
	AppFee            float64             `json:"app_fee"`
	MerchantFee       int                 `json:"merchant_fee"`
	ProcessorResponse string              `json:"processor_response"`
	AuthModel         string              `json:"auth_model"`
	IP                string              `json:"ip"`
	Narration         string              `json:"narration"`
	Status            string              `json:"status"`
	PaymentType       string              `json:"payment_type"`
	CreatedAt         string              `json:"created_at"`
	AccountID         int                 `json:"account_id"`
	Customer          FlutterwaveCustomer `json:"customer"`
	Reference         string              `json:"reference"`
	CompleteMessage   string              `json:"complete_message"`
	TransferFee       float64             `json:"fee"`
}

type FlutterwaveCustomer struct {
	ID          int    `json:"id"`
	PhoneNumber string `json:"phone_number"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	CreatedAt   string `json:"created_at"`
}

// Unified Payment Request - dynamically routes to Flutterwave or Korapay
type UnifiedPaymentRequest struct {
	ImpalaMerchantId string `json:"impalaMerchantId" binding:"required"`
	Country          string `json:"country" binding:"required"` // Country code (e.g., "KE", "UG", "CI", "NG")
	Currency         string `json:"currency" binding:"required"`
	Amount           int    `json:"amount" binding:"required"`
	CustomerName     string `json:"customerName"` // Required for Korapay
	CustomerEmail    string `json:"customerEmail" binding:"required,email"`
	PayerPhone       string `json:"payerPhone" binding:"required"`
	Description      string `json:"description"` // Required for Korapay
	ExternalID       string `json:"externalId" binding:"required"`
	CallbackURL      string `json:"callbackUrl" binding:"required"`
	RedirectURL      string `json:"redirectUrl"` // Optional
}

// Transfer structures
type TransferRequest struct {
	ImpalaMerchantId string  `json:"impalaMerchantId" binding:"required"`
	Currency         string  `json:"currency" binding:"required"`
	Amount           float64 `json:"amount" binding:"required"`
}
