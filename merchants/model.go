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
	DisplayName      string `json:"displayName" binding:"required"`
	PayerPhone       string `json:"payerPhone" binding:"required"`
	MobileMoneySP    string `json:"mobileMoneySP" binding:"required"`
	ExternalID       string `json:"externalId" binding:"required"`
	CallbackURL      string `json:"callbackUrl" binding:"required"`
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
